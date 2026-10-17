// Package verify implements Vajra's four-layer anti-hallucination pipeline.
//
// The pipeline is a set of heuristics, not a guarantee. It is designed to
// run entirely on your machine:
//
//	Layer 1  fast checks        (regex, structure, PII, citations)  ~µs
//	Layer 2  semantic similarity (local embeddings, cosine)          ~50ms
//	Layer 3  LLM-as-a-judge      (local model, 4-dimension rubric)   ~1s
//	Layer 4  human escalation    (flag low-confidence output)        instant
//
// Every layer degrades gracefully: if a model is unavailable the layer is
// reported as "skipped" and the report still returns.
package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/Ritiksuman07/Vajra/internal/llm"
)

// LayerResult is the outcome of one pipeline layer.
type LayerResult struct {
	Name    string  `json:"name"`
	Passed  bool    `json:"passed"`
	Score   float64 `json:"score"`
	Detail  string  `json:"detail,omitempty"`
	Skipped bool    `json:"skipped,omitempty"`
}

// Judge holds the LLM-as-a-judge rubric scores (0-10 each).
type Judge struct {
	Correctness  float64 `json:"correctness"`
	Completeness float64 `json:"completeness"`
	Faithfulness float64 `json:"faithfulness"`
	Coherence    float64 `json:"coherence"`
	Verdict      string  `json:"verdict"`
	Notes        string  `json:"notes"`
}

// Report is the complete verification result.
type Report struct {
	Layers      []LayerResult `json:"layers"`
	Score       float64       `json:"score"`
	Verified    bool          `json:"verified"`
	NeedsReview bool          `json:"needs_review"`
	Citations   bool          `json:"citations"`
	PII         []string      `json:"pii,omitempty"`
	Similarity  float64       `json:"similarity,omitempty"`
	Judge       *Judge        `json:"judge,omitempty"`
	Summary     string        `json:"summary"`
}

// Options configures a verification run.
type Options struct {
	Router           *llm.Router
	Context          []string // grounding sources, may be empty
	RequireCitations bool
	JudgeModel       string
	// MinScore is the pass threshold (0-100). Defaults to 70.
	MinScore float64
}

// ---------------------------------------------------------------------------
// Layer 1: fast checks
// ---------------------------------------------------------------------------

var (
	reEmail = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	rePhone = regexp.MustCompile(`\b(?:\+?\d{1,3}[\s.\-]?)?(?:\(?\d{3}\)?[\s.\-]?)\d{3}[\s.\-]?\d{4}\b`)
	reSSN   = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	reCard  = regexp.MustCompile(`\b(?:\d[ -]?){13,16}\b`)
	reCite  = regexp.MustCompile(`\[cite:[^\]]+\]`)
)

var refusalMarkers = []string{
	"i cannot", "i can't", "as an ai language model", "i'm sorry, but i",
	"i am unable to", "i don't have access to",
}

func fastChecks(response string, opts Options) LayerResult {
	res := LayerResult{Name: "fast_checks", Passed: true, Score: 1.0}
	var notes []string

	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return LayerResult{Name: "fast_checks", Passed: false, Score: 0, Detail: "empty response"}
	}
	if len(trimmed) < 2 {
		notes = append(notes, "suspiciously short")
	}

	// PII scan.
	var pii []string
	if m := reEmail.FindAllString(response, -1); len(m) > 0 {
		pii = append(pii, fmt.Sprintf("%d email(s)", len(m)))
	}
	if m := reSSN.FindAllString(response, -1); len(m) > 0 {
		pii = append(pii, fmt.Sprintf("%d SSN-like", len(m)))
	}
	if m := reCard.FindAllString(response, -1); len(m) > 0 && !strings.Contains(response, "```") {
		pii = append(pii, fmt.Sprintf("%d card-like", len(m)))
	}
	if m := rePhone.FindAllString(response, -1); len(m) > 0 && len(m) < 6 {
		_ = m
	}
	if len(pii) > 0 {
		notes = append(notes, "possible PII: "+strings.Join(pii, ", "))
	}

	// Code fence balance.
	if c := strings.Count(response, "```"); c%2 != 0 {
		notes = append(notes, "unbalanced code fences")
	}

	// Refusals.
	low := strings.ToLower(response)
	for _, marker := range refusalMarkers {
		if strings.Contains(low, marker) {
			notes = append(notes, "possible refusal")
			break
		}
	}

	// Citation requirement.
	citations := reCite.MatchString(response)
	if opts.RequireCitations && !citations {
		res.Passed = false
		notes = append(notes, "missing [cite:...] annotation")
	}

	res.Score = 1.0
	if !res.Passed {
		res.Score = 0.4
	}
	res.Detail = strings.Join(notes, "; ")
	if res.Detail == "" {
		res.Detail = "ok"
	}
	return res
}

// ---------------------------------------------------------------------------
// Layer 2: semantic similarity
// ---------------------------------------------------------------------------

func semanticCheck(ctx context.Context, response string, sources []string, r *llm.Router) LayerResult {
	res := LayerResult{Name: "semantic_similarity", Passed: true, Score: 1.0}
	if r == nil || len(sources) == 0 {
		res.Skipped = true
		res.Detail = "no grounding sources"
		return res
	}
	respVec, err := r.Embed(ctx, response)
	if err != nil {
		res.Skipped = true
		res.Detail = "embedding model unavailable"
		return res
	}
	best := 0.0
	for _, src := range sources {
		vec, err := r.Embed(ctx, src)
		if err != nil {
			continue
		}
		if s := cosine(respVec, vec); s > best {
			best = s
		}
	}
	res.Score = best
	res.Passed = best >= 0.75
	res.Detail = fmt.Sprintf("cosine %.2f (threshold 0.75)", best)
	return res
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// ---------------------------------------------------------------------------
// Layer 3: LLM-as-a-judge
// ---------------------------------------------------------------------------

const judgeSystem = "You are a strict evaluator of AI answers. " +
	"Score each dimension from 0 to 10. Reply with ONLY a JSON object."

var jsonObjRe = regexp.MustCompile(`(?s)\{.*\}`)

func judgeCheck(ctx context.Context, response string, sources []string, r *llm.Router, model string) (LayerResult, *Judge) {
	res := LayerResult{Name: "llm_judge", Passed: true, Score: 1.0}
	if r == nil {
		res.Skipped = true
		res.Detail = "no judge model"
		return res, nil
	}
	grounding := "(none provided)"
	if len(sources) > 0 {
		grounding = strings.Join(sources, "\n---\n")
	}
	prompt := fmt.Sprintf(`Evaluate the ANSWER against the SOURCES.

SOURCES:
%s

ANSWER:
%s

Return JSON with keys: correctness, completeness, faithfulness, coherence (0-10),
verdict ("pass" or "fail"), notes (one short sentence).`,
		truncate(grounding, 4000), truncate(response, 6000))

	chatResp, err := r.Chat(ctx, llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: judgeSystem},
			{Role: llm.RoleUser, Content: prompt},
		},
		Temperature: 0,
	})
	if err != nil {
		res.Skipped = true
		res.Detail = "judge unavailable: " + err.Error()
		return res, nil
	}

	m := jsonObjRe.FindString(chatResp.Content)
	if m == "" {
		res.Skipped = true
		res.Detail = "judge returned no JSON"
		return res, nil
	}
	var j Judge
	if err := json.Unmarshal([]byte(m), &j); err != nil {
		res.Skipped = true
		res.Detail = "judge JSON parse error"
		return res, nil
	}
	avg := (j.Correctness + j.Completeness + j.Faithfulness + j.Coherence) / 4
	res.Score = avg / 10
	res.Passed = avg >= 7 && !strings.EqualFold(j.Verdict, "fail")
	res.Detail = fmt.Sprintf("%.1f/10 (%s)", avg, j.Verdict)
	return res, &j
}

// ---------------------------------------------------------------------------
// Pipeline
// ---------------------------------------------------------------------------

// Run executes the pipeline and returns a report.
func Run(ctx context.Context, response string, opts Options) *Report {
	if opts.MinScore == 0 {
		opts.MinScore = 70
	}
	rep := &Report{}

	// Layer 1
	l1 := fastChecks(response, opts)
	rep.Layers = append(rep.Layers, l1)
	rep.PII = extractPII(response)
	rep.Citations = reCite.MatchString(response)

	// Layer 2
	l2 := semanticCheck(ctx, response, opts.Context, opts.Router)
	rep.Layers = append(rep.Layers, l2)
	rep.Similarity = l2.Score

	// Layer 3
	l3, judge := judgeCheck(ctx, response, opts.Context, opts.Router, opts.JudgeModel)
	rep.Layers = append(rep.Layers, l3)
	rep.Judge = judge

	// Layer 4: combine into a trust score.
	weights := map[string]float64{"fast_checks": 0.25, "semantic_similarity": 0.35, "llm_judge": 0.40}
	var total, weight float64
	for _, l := range rep.Layers {
		if l.Skipped {
			continue
		}
		w := weights[l.Name]
		total += l.Score * w
		weight += w
	}
	score := 0.5
	if weight > 0 {
		score = total / weight
	}
	rep.Score = math.Round(score*10000) / 100
	rep.Verified = rep.Score >= opts.MinScore && l1.Passed
	rep.NeedsReview = !rep.Verified
	rep.Summary = summarise(rep)
	return rep
}

func extractPII(s string) []string {
	var out []string
	if m := reEmail.FindAllString(s, -1); len(m) > 0 {
		out = append(out, "email")
	}
	if m := reSSN.FindAllString(s, -1); len(m) > 0 {
		out = append(out, "ssn")
	}
	return out
}

func summarise(rep *Report) string {
	if rep.NeedsReview {
		return fmt.Sprintf("Needs review — trust %.0f%% (threshold not met)", rep.Score)
	}
	return fmt.Sprintf("Verified — trust %.0f%%", rep.Score)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
