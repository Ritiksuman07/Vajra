package verify

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ritiksuman07/Vajra/internal/config"
	"github.com/Ritiksuman07/Vajra/internal/llm"
)

func TestFastChecksClean(t *testing.T) {
	rep := Run(context.Background(), "Here is a clean answer. It is correct and clear.", Options{})
	l1 := rep.Layers[0]
	if !l1.Passed {
		t.Fatalf("expected fast checks to pass: %s", l1.Detail)
	}
	if len(rep.PII) != 0 {
		t.Errorf("unexpected PII: %v", rep.PII)
	}
}

func TestFastChecksEmpty(t *testing.T) {
	rep := Run(context.Background(), "   ", Options{})
	if rep.Layers[0].Passed {
		t.Error("empty responses must fail fast checks")
	}
	if rep.Verified {
		t.Error("empty responses must not be verified")
	}
}

func TestFastChecksPII(t *testing.T) {
	rep := Run(context.Background(), "Contact me at user@example.com or 123-45-6789.", Options{})
	if len(rep.PII) == 0 {
		t.Error("expected PII to be flagged")
	}
}

func TestFastChecksUnbalancedFence(t *testing.T) {
	rep := Run(context.Background(), "```go\nfmt.Println()\n", Options{})
	if !strings.Contains(rep.Layers[0].Detail, "unbalanced") {
		t.Errorf("expected unbalanced fence note, got %q", rep.Layers[0].Detail)
	}
}

func TestCitationRequirement(t *testing.T) {
	rep := Run(context.Background(), "Paris is the capital. [cite:geo:1:0-20]", Options{RequireCitations: true})
	if !rep.Layers[0].Passed {
		t.Errorf("expected pass with a citation: %s", rep.Layers[0].Detail)
	}
	rep = Run(context.Background(), "Paris is the capital.", Options{RequireCitations: true})
	if rep.Layers[0].Passed {
		t.Error("expected failure when a citation is required but missing")
	}
}

func TestCosine(t *testing.T) {
	if got := cosine([]float64{1, 0}, []float64{1, 0}); math.Abs(got-1) > 1e-9 {
		t.Errorf("cosine identical = %v, want 1", got)
	}
	if got := cosine([]float64{1, 0}, []float64{0, 1}); math.Abs(got) > 1e-9 {
		t.Errorf("cosine orthogonal = %v, want 0", got)
	}
	if got := cosine(nil, nil); got != 0 {
		t.Errorf("cosine nil = %v, want 0", got)
	}
}

// TestJudgeLayer exercises layer 3 against a fake Ollama server.
func TestJudgeLayer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/chat":
			verdict := `{"correctness":9,"completeness":8,"faithfulness":9,"coherence":9,"verdict":"pass","notes":"solid"}`
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"role": "assistant", "content": verdict},
				"done":    true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.OllamaHost = srv.URL
	cfg.Model = "judge"
	router := llm.NewRouter(cfg)

	rep := Run(context.Background(), "The sky is blue.", Options{Router: router, JudgeModel: "judge", MinScore: 70})
	if rep.Judge == nil {
		t.Fatal("expected the judge to run")
	}
	if rep.Judge.Verdict != "pass" {
		t.Errorf("verdict = %q", rep.Judge.Verdict)
	}
	if !rep.Verified {
		t.Errorf("expected verified, score=%.1f", rep.Score)
	}
	if !strings.Contains(rep.Summary, "Verified") {
		t.Errorf("summary = %q", rep.Summary)
	}
}

func TestJudgeMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"content": "I cannot produce JSON"},
			"done":    true,
		})
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.OllamaHost = srv.URL
	router := llm.NewRouter(cfg)
	rep := Run(context.Background(), "answer", Options{Router: router, JudgeModel: "j"})
	l3 := rep.Layers[2]
	if !l3.Skipped {
		t.Errorf("expected judge layer to be skipped on malformed output, got %+v", l3)
	}
}
