package llm

import "strings"

// ── Context budgeting ────────────────────────────────────────────────────────
//
// Local models have a fixed context window. Push past it and Ollama silently
// truncates from the front — which means the system prompt and your question
// can fall off. Vajra trims deliberately instead, and tells you when it did.
//
// Token counting for local models cannot be exact: every family ships its own
// tokeniser, and Vajra does not bundle them. The estimate below is the usual
// ~4 characters per token heuristic — deliberately simple, documented, and
// good enough to keep a conversation inside the window.

// charsPerToken is the heuristic used by EstimateTokens.
const charsPerToken = 4

// EstimateTokens approximates the token count of a string.
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	n := len(s) / charsPerToken
	if n < 1 {
		n = 1
	}
	return n
}

// EstimateMessageTokens approximates the cost of one message including a
// small per-message overhead for role and framing tokens.
func EstimateMessageTokens(m Message) int {
	return EstimateTokens(m.Content) + 4
}

// Budget describes how much context the user wants Vajra to use.
type Budget struct {
	// Max is the total context window in tokens (0 means "do not trim").
	Max int
	// Reserve is the number of tokens kept free so the reply has room.
	Reserve int
}

// DefaultBudget returns a budget for a given context size, reserving a quarter
// of the window for the model's answer.
func DefaultBudget(max int) Budget {
	if max <= 0 {
		return Budget{}
	}
	return Budget{Max: max, Reserve: max / 4}
}

// limit is the number of tokens the prompt may occupy.
func (b Budget) limit() int {
	if b.Max <= 0 {
		return 0
	}
	l := b.Max - b.Reserve
	if l < 1 {
		l = b.Max
	}
	return l
}

// TrimResult reports what happened during a trim.
type TrimResult struct {
	Messages []Message
	Dropped  int // messages removed from the middle/start of the conversation
	Used     int // estimated tokens in the kept prompt
	Limit    int // the prompt budget that applied (0 = unlimited)
	Trimmed  bool
}

// Trim fits a conversation into the budget.
//
// Rules, in order:
//  1. System prompts are always kept — they define the agent's behaviour.
//  2. Whole turns are dropped from the oldest end until the rest fits.
//  3. The final message is always kept: if it alone overflows the budget its
//     text is trimmed from the front (context and attachments are prepended,
//     so the user's actual question lives at the end).
func (b Budget) Trim(msgs []Message) TrimResult {
	res := TrimResult{Messages: msgs, Limit: b.limit()}
	if b.Max <= 0 {
		for _, m := range msgs {
			res.Used += EstimateMessageTokens(m)
		}
		return res
	}
	limit := res.Limit

	var system, rest []Message
	for _, m := range msgs {
		if m.Role == RoleSystem {
			system = append(system, m)
			continue
		}
		rest = append(rest, m)
	}

	used := 0
	for _, m := range system {
		used += EstimateMessageTokens(m)
	}

	// Walk backwards, keeping the newest turns that still fit.
	var kept []Message
	for i := len(rest) - 1; i >= 0; i-- {
		cost := EstimateMessageTokens(rest[i])
		if used+cost > limit && len(kept) > 0 {
			break
		}
		used += cost
		kept = append(kept, rest[i])
		if used > limit {
			break
		}
	}

	// Reverse into chronological order, keeping the system prompts first.
	out := make([]Message, 0, len(system)+len(kept))
	out = append(out, system...)
	for i := len(kept) - 1; i >= 0; i-- {
		out = append(out, kept[i])
	}

	// The final message must survive even if it is enormous.
	if n := len(out); n > len(system) {
		last := &out[n-1]
		if EstimateMessageTokens(*last) > limit && limit > 0 {
			last.Content = TailTokens(last.Content, limit)
			used = 0
			for _, m := range out {
				used += EstimateMessageTokens(m)
			}
		}
	}

	res.Messages = out
	res.Dropped = len(rest) - len(kept)
	res.Used = used
	res.Trimmed = res.Dropped > 0
	return res
}

// TailTokens keeps the last `tokens` worth of text, marking the cut.
//
// The tail is kept because Vajra appends the user's question after any
// attachments or retrieved context, so the end of the message is the part
// that must reach the model.
func TailTokens(s string, tokens int) string {
	if tokens <= 0 {
		return ""
	}
	max := tokens * charsPerToken
	if len(s) <= max {
		return s
	}
	cut := len(s) - max
	// Do not split a rune.
	for cut < len(s) && !utf8Start(s[cut]) {
		cut++
	}
	return "…[trimmed to fit the context window]\n\n" + s[cut:]
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

// HeadTokens keeps the first `tokens` worth of text, marking the cut. It is
// used for retrieved context, where the beginning of a passage matters most.
func HeadTokens(s string, tokens int) string {
	if tokens <= 0 {
		return ""
	}
	max := tokens * charsPerToken
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8Start(s[cut]) {
		cut--
	}
	return strings.TrimRight(s[:cut], " \n\t") + "\n…[truncated]"
}
