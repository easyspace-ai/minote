package llm

import (
	"testing"

	einoSchema "github.com/cloudwego/eino/schema"
	"github.com/easyspace-ai/minote/pkg/models"
)

func TestToEinoMessageUsesUserInputMultiContent(t *testing.T) {
	msg := models.Message{
		ID:        "m1",
		SessionID: "s1",
		Role:      models.RoleHuman,
		Content:   "fallback",
		Metadata: map[string]string{
			"multi_content": `[{"type":"text","text":"look"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc"}}]`,
		},
	}

	out := toEinoMessage(msg)
	if out.Role != einoSchema.User {
		t.Fatalf("role=%s", out.Role)
	}
	if out.Content != "" {
		t.Fatalf("content=%q want empty", out.Content)
	}
	if len(out.UserInputMultiContent) != 2 {
		t.Fatalf("parts=%d want 2", len(out.UserInputMultiContent))
	}
	if out.UserInputMultiContent[1].Type != einoSchema.ChatMessagePartTypeImageURL {
		t.Fatalf("part type=%s", out.UserInputMultiContent[1].Type)
	}
}
