package llm

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"ab", 1},       // always at least one token for non-empty text
		{"abcd", 1},     // 4 chars per token
		{"abcdefgh", 2}, // 8 chars
		{strings.Repeat("x", 400), 100},
	}
	for _, c := range cases {
		if got := EstimateTokens(c.in); got != c.want {
			t.Errorf("EstimateTokens(%q) = %d, want %d", truncateForTest(c.in), got, c.want)
		}
	}
}

func truncateForTest(s string) string {
	if len(s) > 20 {
		return s[:20] + "…"
	}
	return s
}

func TestBudgetKeepsSystemAndNewest(t *testing.T) {
	b := Budget{Max: 100, Reserve: 20} // prompt budget = 80 tokens = 320 chars
	msgs := []Message{
		{Role: RoleSystem, Content: "system prompt"},
		{Role: RoleUser, Content: strings.Repeat("a", 400)},      // 100 tokens — too big, oldest
		{Role: RoleAssistant, Content: strings.Repeat("b", 200)}, // 50 tokens
		{Role: RoleUser, Content: "newest question"},             // 4 tokens
	}
	res := b.Trim(msgs)

	if len(res.Messages) == 0 {
		t.Fatal("nothing kept")
	}
	if res.Messages[0].Role != RoleSystem {
		t.Errorf("system prompt must be kept first, got %q", res.Messages[0].Role)
	}
	if !res.Trimmed || res.Dropped != 1 {
		t.Errorf("expected 1 dropped message, got dropped=%d trimmed=%v", res.Dropped, res.Trimmed)
	}
	last := res.Messages[len(res.Messages)-1]
	if last.Content != "newest question" {
		t.Errorf("newest message must be kept, got %q", last.Content)
	}
	if res.Used > res.Limit {
		t.Errorf("used %d tokens > limit %d", res.Used, res.Limit)
	}
}

func TestBudgetUnlimitedKeepsEverything(t *testing.T) {
	msgs := []Message{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleUser, Content: strings.Repeat("x", 100000)},
	}
	res := Budget{}.Trim(msgs)
	if res.Trimmed || res.Dropped != 0 {
		t.Errorf("unlimited budget should not trim: %+v", res)
	}
	if len(res.Messages) != 2 {
		t.Errorf("kept %d messages, want 2", len(res.Messages))
	}
}

func TestBudgetKeepsOversizedFinalMessage(t *testing.T) {
	b := Budget{Max: 40, Reserve: 10} // prompt budget = 30 tokens = 120 chars
	long := "context\n\n" + strings.Repeat("y", 2000) + "\n\nthe actual question?"
	res := b.Trim([]Message{{Role: RoleUser, Content: long}})

	if len(res.Messages) != 1 {
		t.Fatalf("the only message must be kept, got %d", len(res.Messages))
	}
	got := res.Messages[0].Content
	if len(got) > 200 {
		t.Errorf("message not trimmed: %d bytes", len(got))
	}
	// The tail is kept on purpose, because the question is appended last.
	if !strings.Contains(got, "question") {
		t.Errorf("tail (the question) should survive trimming, got %q", got)
	}
}

func TestTailAndHeadTokens(t *testing.T) {
	s := "0123456789abcdefghij" // 20 chars
	if got := TailTokens(s, 1); !strings.HasSuffix(got, "hij") && !strings.HasSuffix(got, "ghij") {
		t.Errorf("TailTokens = %q", got)
	}
	if got := HeadTokens(s, 1); !strings.Contains(got, "0123") || !strings.Contains(got, "truncated") {
		t.Errorf("HeadTokens = %q", got)
	}
	if TailTokens(s, 100) != s || HeadTokens(s, 100) != s {
		t.Error("text under the limit must be returned unchanged (so a tail is not double-cut)")
	}
}

// The budget must never cut a multi-byte rune in half.
func TestTrimDoesNotSplitRunes(t *testing.T) {
	b := Budget{Max: 12, Reserve: 4}
	msg := strings.Repeat("日本語テキスト", 50)
	res := b.Trim([]Message{{Role: RoleUser, Content: msg}})
	got := res.Messages[0].Content
	for i := 0; i < len(got); {
		r, size := decodeRune(got[i:])
		if r == 0xFFFD && size == 1 {
			t.Fatalf("broken rune at byte %d of %q", i, got)
		}
		i += size
	}
}

func decodeRune(s string) (rune, int) {
	for i, r := range s {
		if i == 0 {
			return r, len(string(r))
		}
		break
	}
	return 0, 1
}
