package langgraphcompat

import "testing"

func TestEffectiveChatModelID(t *testing.T) {
	t.Parallel()
	if got := effectiveChatModelID("", "m1"); got != "m1" {
		t.Fatalf("empty requested: %q", got)
	}
	if got := effectiveChatModelID("openai", "doubao-seed-2.0-pro"); got != "doubao-seed-2.0-pro" {
		t.Fatalf("provider slug: %q", got)
	}
	if got := effectiveChatModelID("OpenAI", "m"); got != "m" {
		t.Fatalf("case: %q", got)
	}
	if got := effectiveChatModelID("doubao-seed-2.0-pro", "other"); got != "doubao-seed-2.0-pro" {
		t.Fatalf("real id: %q", got)
	}
}
