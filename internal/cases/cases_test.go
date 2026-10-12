package cases

import (
	"strings"
	"testing"
)

// FillerPrompt backs --input-tokens: the same size must always produce the
// same prompt text so repeated runs are comparable.
func TestFillerPromptDeterministic(t *testing.T) {
	if a, b := FillerPrompt(800), FillerPrompt(800); a != b {
		t.Error("FillerPrompt(800) returned different text across calls")
	}
}

// FillerPrompt targets roughly 0.75 words per token (English prose tokenizes
// at about 1.33 tokens per word), so an 800-token request carries ~600 filler
// words plus the trailing instruction.
func TestFillerPromptSize(t *testing.T) {
	words := len(strings.Fields(FillerPrompt(800)))
	if words < 600 || words > 660 {
		t.Errorf("FillerPrompt(800) = %d words, want 600..660", words)
	}
}

func TestFillerPromptNonPositive(t *testing.T) {
	if got := FillerPrompt(0); got != "" {
		t.Errorf("FillerPrompt(0) = %q, want empty", got)
	}
}

// FillerPromptSalt backs --prompt-salt: every call must carry a distinct
// nonce prefix so the provider's prefix cache cannot serve any input.
func TestFillerPromptSaltVariesPrefix(t *testing.T) {
	na, nb := firstLine(FillerPromptSalt(800)), firstLine(FillerPromptSalt(800))
	if !strings.HasPrefix(na, "[nonce ") || !strings.HasSuffix(na, "]") {
		t.Fatalf("nonce line = %q, want a [nonce ...] tag", na)
	}
	if na == nb {
		t.Errorf("nonce identical across calls: %q", na)
	}
	if len(na) != len(nb) {
		t.Errorf("nonce width varies: %d vs %d chars", len(na), len(nb))
	}
}

// The salted prompt keeps the plain filler body, so sizes stay comparable
// with cached-mode runs of the same --input-tokens.
func TestFillerPromptSaltKeepsBody(t *testing.T) {
	if got := FillerPromptSalt(800); !strings.HasSuffix(got, FillerPrompt(800)) {
		t.Error("salted prompt does not end with the plain filler body")
	}
}

func TestFillerPromptSaltNonPositive(t *testing.T) {
	if got := FillerPromptSalt(0); got != "" {
		t.Errorf("FillerPromptSalt(0) = %q, want empty", got)
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
