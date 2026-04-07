package llm

import "testing"

func TestNormalizeReasoningEffortForProvider(t *testing.T) {
	t.Parallel()
	if got := normalizeReasoningEffortForProvider("ollama", "minimal"); got != "low" {
		t.Fatalf("ollama minimal: got %q want low", got)
	}
	if got := normalizeReasoningEffortForProvider("ollama", "high"); got != "high" {
		t.Fatalf("ollama high: got %q want high", got)
	}
	if got := normalizeReasoningEffortForProvider("ollama", "bogus"); got != "" {
		t.Fatalf("ollama bogus: got %q want empty", got)
	}
	if got := normalizeReasoningEffortForProvider("openai", "minimal"); got != "minimal" {
		t.Fatalf("openai minimal: got %q want minimal", got)
	}
}
