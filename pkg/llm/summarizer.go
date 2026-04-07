package llm

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/easyspace-ai/minote/pkg/models"
)

const (
	defaultSummarizationMaxTokens     = 100000
	defaultSummarizationKeepRounds    = 5
	defaultSummarizationTrimTokens    = 4000
	defaultSummarizationPrompt        = `You are a conversation summarizer. Summarize the following conversation history into a concise summary that preserves all important context, decisions, and action items. Focus on:
1. Key topics discussed
2. Decisions made
3. Important facts and context
4. Current task status and next steps

Provide a clear, structured summary in 2-4 paragraphs.`
)

// SummarizationConfig configures the LLM-powered conversation summarizer.
type SummarizationConfig struct {
	// Enabled turns on LLM-based summarization.
	Enabled bool
	// Model is the LLM model name for summarization.
	Model string
	// MaxTokenThreshold is the total token count above which summarization triggers.
	MaxTokenThreshold int
	// MaxMessageCount triggers summarization if message count exceeds this.
	MaxMessageCount int
	// KeepRounds is how many recent conversation rounds to preserve verbatim.
	KeepRounds int
	// TrimTokensToSummarize is the target token count for the summarized block.
	TrimTokensToSummarize int
	// SummaryPrompt is the system prompt used for the summarization LLM call.
	SummaryPrompt string
}

// ConversationSummarizer summarizes older conversation history using an LLM
// while keeping recent messages intact.
type ConversationSummarizer struct {
	config   SummarizationConfig
	provider LLMProvider
	trimmer  *defaultMessageTrimmer // reuse token counting
}

// NewConversationSummarizer creates a summarizer with the given LLM provider and config.
func NewConversationSummarizer(provider LLMProvider, config SummarizationConfig) *ConversationSummarizer {
	if config.MaxTokenThreshold <= 0 {
		config.MaxTokenThreshold = defaultSummarizationMaxTokens
	}
	if config.KeepRounds <= 0 {
		config.KeepRounds = defaultSummarizationKeepRounds
	}
	if config.TrimTokensToSummarize <= 0 {
		config.TrimTokensToSummarize = defaultSummarizationTrimTokens
	}
	if config.SummaryPrompt == "" {
		config.SummaryPrompt = defaultSummarizationPrompt
	}
	return &ConversationSummarizer{
		config:   config,
		provider: provider,
		trimmer:  &defaultMessageTrimmer{config: TrimmerConfig{MaxTokens: config.MaxTokenThreshold}},
	}
}

// ShouldSummarize returns true if the messages exceed the configured thresholds.
func (s *ConversationSummarizer) ShouldSummarize(messages []models.Message) bool {
	if !s.config.Enabled {
		return false
	}
	if s.config.MaxMessageCount > 0 && len(messages) >= s.config.MaxMessageCount {
		return true
	}
	totalTokens := s.trimmer.countTotalTokens(messages)
	return totalTokens > s.config.MaxTokenThreshold
}

// Summarize replaces older messages with an LLM-generated summary, keeping
// recent rounds and the system prompt intact.
func (s *ConversationSummarizer) Summarize(ctx context.Context, messages []models.Message) ([]models.Message, error) {
	if len(messages) <= 2 {
		return messages, nil
	}

	// Separate system message
	var systemMsg *models.Message
	conversationMsgs := messages
	if len(messages) > 0 && messages[0].Role == models.RoleSystem {
		systemMsg = &messages[0]
		conversationMsgs = messages[1:]
	}

	// Find the split point: keep the last N conversation rounds
	keepFrom := findKeepFromIndex(conversationMsgs, s.config.KeepRounds)
	if keepFrom <= 0 {
		return messages, nil // Nothing old enough to summarize
	}

	oldMessages := conversationMsgs[:keepFrom]
	recentMessages := conversationMsgs[keepFrom:]

	// Build the text to summarize
	var sb strings.Builder
	for _, msg := range oldMessages {
		role := string(msg.Role)
		content := msg.Content
		if content == "" && len(msg.ToolCalls) > 0 {
			var calls []string
			for _, tc := range msg.ToolCalls {
				calls = append(calls, fmt.Sprintf("tool_call(%s)", tc.Name))
			}
			content = strings.Join(calls, ", ")
		}
		if msg.ToolResult != nil {
			content = fmt.Sprintf("[tool_result] %s", msg.ToolResult.Content)
			if msg.ToolResult.Error != "" {
				content += " [error: " + msg.ToolResult.Error + "]"
			}
		}
		if content != "" {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, content))
		}
	}

	conversationText := sb.String()
	if strings.TrimSpace(conversationText) == "" {
		return messages, nil
	}

	// Truncate if needed to stay within summarization token budget
	maxChars := s.config.TrimTokensToSummarize * charsPerToken
	if utf8.RuneCountInString(conversationText) > maxChars {
		runes := []rune(conversationText)
		conversationText = string(runes[:maxChars])
	}

	// Call LLM to generate summary
	summaryResp, err := s.provider.Chat(ctx, ChatRequest{
		Model: s.config.Model,
		Messages: []models.Message{
			{Role: models.RoleSystem, Content: s.config.SummaryPrompt},
			{Role: models.RoleHuman, Content: "Please summarize the following conversation:\n\n" + conversationText},
		},
	})
	if err != nil {
		return messages, fmt.Errorf("summarization LLM call failed: %w", err)
	}

	summaryContent := strings.TrimSpace(summaryResp.Message.Content)
	if summaryContent == "" {
		return messages, nil
	}

	// Reconstruct message list: system + summary + recent
	result := make([]models.Message, 0, 2+len(recentMessages))
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}
	result = append(result, models.Message{
		Role:    models.RoleSystem,
		Content: fmt.Sprintf("[Previous conversation summary]\n\n%s", summaryContent),
	})
	result = append(result, recentMessages...)
	return result, nil
}

// findKeepFromIndex finds the index from which to keep messages,
// preserving the last `keepRounds` human messages and their responses.
func findKeepFromIndex(messages []models.Message, keepRounds int) int {
	rounds := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.RoleHuman {
			rounds++
			if rounds >= keepRounds {
				return i
			}
		}
	}
	return 0
}
