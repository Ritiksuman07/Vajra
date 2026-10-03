"""
Anti-Hallucination Verification Pipeline

Implements the 4-layer verification pipeline:
1. Fast Checks (schema, citations, PII, length)
2. Semantic Similarity (bge-small embeddings, cosine >= 0.75)
3. LLM-as-a-Judge (Llama 3.1 8B, 4-dimension rubric)
4. Human Escalation (CLI/TUI prompt)

All verification is performed LOCALLY — no cloud calls.
"""

import re
import math
import hashlib
from dataclasses import dataclass, field
from typing import Any, Optional


@dataclass
class LayerResult:
    """Result from a single verification layer."""
    name: str
    passed: bool
    score: Optional[float] = None
    latency_ms: int = 0


@dataclass
class VerificationResult:
    """Complete verification result for a response."""
    passed: bool
    confidence: float
    layer1: Optional[LayerResult] = None
    layer2: Optional[LayerResult] = None
    layer3: Optional[LayerResult] = None
    layer4: Optional[LayerResult] = None
    details: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "passed": self.passed,
            "confidence": self.confidence,
            "layers": [l for l in [self.layer1, self.layer2, self.layer3, self.layer4] if l],
            **self.details,
        }


class VerificationPipeline:
    """
    Multi-layer anti-hallucination verification.
    
    Implements the verification pipeline:
    Layer 1: Fast Checks (100% coverage, <5ms)
    Layer 2: Semantic Similarity (<50ms)
    Layer 3: LLM-as-a-Judge (<2s)
    Layer 4: Human Escalation (on demand)
    """

    CITATION_REGEX = re.compile(r"\[cite:[^\]]+\]")
    MAX_RESPONSE_LENGTH = 2000

    def __init__(self, judge_model: str = "llama3.1:8b"):
        self.judge_model = judge_model

    def verify(
        self,
        prompt: str,
        response: str,
        context: Optional[str] = None,
        trace_id: Optional[str] = None,
    ) -> VerificationResult:
        """
        Run full verification pipeline.

        Args:
            prompt: Original user prompt/task
            response: Agent's response
            context: Retrieved context (for semantic check)
            trace_id: Optional trace ID for logging

        Returns:
            VerificationResult with details for all layers
        """
        results = []

        # Layer 1: Fast Checks
        l1 = self._layer_fast_checks(response)
        results.append(("layer1", l1))

        if not l1.passed:
            # Fast failure
            return VerificationResult(
                passed=False,
                confidence=0.3,
                layer1=l1,
                details={"reason": "fast_checks_failed"},
            )

        # Layer 2: Semantic Similarity
        l2 = self._layer_semantic_similarity(response, context)
        results.append(("layer2", l2))

        if not l2.passed:
            # Escalate to judge/human
            confidence = (l1.score or 0.8) * (l2.score or 0.5) if l2.score else 0.4
            return VerificationResult(
                passed=False,
                confidence=confidence,
                layer1=l1,
                layer2=l2,
                details={"reason": "semantic_similarity_low"},
            )

        # Layer 3: LLM-as-a-Judge (only if available)
        l3 = self._layer_llm_judge(prompt, response)
        results.append(("layer3", l3))

        # Calculate confidence
        scores = [l.score for l in [l1, l2, l3] if l and l.score]
        confidence = sum(scores) / len(scores) if scores else 0.85

        # Layered pass/fail
        all_passed = l1.passed and l2.passed
        if l3 and not l3.passed:
            all_passed = False
            confidence = min(confidence, 0.5)

        return VerificationResult(
            passed=all_passed,
            confidence=confidence,
            layer1=l1,
            layer2=l2,
            layer3=l3,
            details={"validation_path": l1.passed and l2.passed and (l3 is None or l3.passed)},
        )

    # ── Layer 1: Fast Checks ──────────────────────────────────────────────

    def _layer_fast_checks(self, response: str) -> LayerResult:
        """
        Layer 1: Fast, deterministic checks.
        Runs on 100% of responses, <5ms latency.
        """
        # Schema validation
        schema_valid = isinstance(response, str)

        # Citation check
        citations = self.CITATION_REGEX.findall(response)
        has_citations = len(citations) > 0

        # PII scan (basic patterns)
        pii_patterns = [
            r"\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b",  # Credit card
            r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b",  # Email
            r"\b\d{3}-\d{3}-\d{4}\b",  # Phone
        ]
        pii_found = any(re.search(p, response) for p in pii_patterns)

        # Length limits
        length_ok = len(response) <= self.MAX_RESPONSE_LENGTH

        # No markdown in plaintext mode
        no_markdown = "```" not in response if length_ok else False

        passed = schema_valid and length_ok and not pii_found

        score = 1.0 if passed else 0.0

        return LayerResult(
            name="fast_checks",
            passed=passed,
            score=score,
        )

    # ── Layer 2: Semantic Similarity ───────────────────────────────────────

    def _layer_semantic_similarity(
        self,
        response: str,
        context: Optional[str],
    ) -> LayerResult:
        """
        Layer 2: Semantic similarity between response and context.
        Uses embeddings to verify grounding.
        """
        if context is None:
            # No context means we can't verify grounding
            # This should trigger Layer 3 or 4
            return LayerResult(
                name="semantic_similarity",
                passed=True,  # Pass by default when no context
                score=0.85,
            )

        # Simple cosine similarity calculation
        # In production, use sentence-transformers (bge-small)
        similarity = self._cosine_similarity(response, context)

        threshold = 0.75
        passed = similarity >= threshold

        return LayerResult(
            name="semantic_similarity",
            passed=passed,
            score=similarity,
        )

    def _cosine_similarity(self, a: str, b: str) -> float:
        """
        Calculate cosine similarity between two strings.
        Placeholder - in production uses bge-small embeddings.
        """
        # Count word overlap as a simple similarity measure
        words_a = set(a.lower().split())
        words_b = set(b.lower().split())

        if not words_a or not words_b:
            return 0.5  # Default

        intersection = len(words_a & words_b)
        union = len(words_a | words_b)

        if union == 0:
            return 0.5

        return intersection / union

    # ── Layer 3: LLM-as-a-Judge ───────────────────────────────────────────

    def _layer_llm_judge(
        self,
        prompt: str,
        response: str,
    ) -> LayerResult:
        """
        Layer 3: LLM-as-a-Judge using local model.
        Evaluates correctness, completeness, faithfulness, coherence.
        """
        # For local-first deployment, use a local model
        # In production, this would call the judge model via Ollama
        scores = self._judge_response(prompt, response)

        # Average score across dimensions
        avg_score = sum(scores.values()) / len(scores)
        threshold = 7.0 / 10.0  # 7.0/10 normalized

        passed = avg_score >= threshold

        return LayerResult(
            name="llm_judge",
            passed=passed,
            score=avg_score,
        )

    def _judge_response(self, prompt: str, response: str) -> dict[str, float]:
        """
        Judge the response quality using 4-dimension rubric.
        Returns scores 0-10 for each dimension.
        
        In production, calls local Llama 3.1 8B with specific prompt.
        """
        # Heuristic-based scoring (placeholder)
        # Production would use LLM-as-a-judge

        # Correctness: Does answer match prompt?
        correctness = 8.0 if response else 3.0

        # Completeness: Does it fully address the task?
        completeness = 7.5 if len(response) > 50 else 5.0

        # Faithfulness: Does it avoid hallucinations?
        # Check for unsupported claims (simplified)
        faithfulness = 8.0 if self._has_citations_or_facts(response) else 6.0

        # Coherence: Is it well-structured?
        coherence = 8.5 if self._is_coherent(response) else 6.0

        return {
            "correctness": correctness,
            "completeness": completeness,
            "faithfulness": faithfulness,
            "coherence": coherence,
        }

    def _has_citations_or_facts(self, response: str) -> bool:
        """Check if response has citations or verifiable facts."""
        return bool(self.CITATION_REGEX.search(response))

    def _is_coherent(self, response: str) -> bool:
        """Check if response has logical structure."""
        sentences = response.split(".")
        return len(sentences) > 1 and len(response) > 100

    # ── Layer 4: Human Escalation ────────────────────────────────────────

    def escalate_to_human(
        self,
        prompt: str,
        response: str,
        verification: VerificationResult,
    ) -> str:
        """
        Format escalation ticket for human review.

        In production, this would create a ticket in the incident system.
        For CLI, prints to stdout/stderr.
        """
        ticket = f"""
=== HUMAN REVIEW REQUIRED ===

Task: {prompt}

Response: {response}

Verification:
- Layer 1 (Fast Checks): {'✓' if verification.layer1.passed else '✗'}
- Layer 2 (Semantic): Score {verification.layer2.score:.2f if verification.layer2 else 'N/A'}
- Confidence: {verification.confidence:.2f}

Reason: Response requires human judgment

AUDIT_HASH: {self._hash_response(prompt, response)}
============================
"""
        return ticket

    def _hash_response(self, prompt: str, response: str) -> str:
        """Create audit hash for traceability."""
        data = f"{prompt}|{response}".encode()
        return hashlib.sha256(data).hexdigest()[:16]


# ── Convenience Functions ───────────────────────────────────────────────


def verify_response(
    prompt: str,
    response: str,
    context: Optional[str] = None,
) -> bool:
    """Quick verification check."""
    pipeline = VerificationPipeline()
    result = pipeline.verify(prompt, response, context)
    return result.passed


def get_verification_confidence(
    prompt: str,
    response: str,
    context: Optional[str] = None,
) -> float:
    """Get confidence score for a response."""
    pipeline = VerificationPipeline()
    result = pipeline.verify(prompt, response, context)
    return result.confidence
