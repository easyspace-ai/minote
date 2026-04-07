package memory

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/easyspace-ai/minote/pkg/models"
)

const MemoryUpdateSystemPrompt = `You maintain durable user memory for an AI agent.

## Core Principles
1. **Stability First**: Only capture facts that are unlikely to change quickly
2. **High Value**: Prioritize information that improves future interactions
3. **Privacy Aware**: Avoid storing sensitive personal data (passwords, secrets, IDs)

## When to Update Each Field

### User Context (Rarely Changes)
- **workContext**: User's job, role, company, industry. Update only when career changes.
- **personalContext**: Location, timezone, language preference. Update only when moved.
- **topOfMind**: Current priorities, goals, active concerns. Update when user expresses new priorities.

### History Context (Rolling Window)
- **recentMonths**: Summary of past 1-2 months of interactions. Update monthly.
- **earlierContext**: Older significant context. Compress when recentMonths grows.
- **longTermBackground**: Persistent background (e.g., "long-time user since 2024").

### Facts (Dynamic but Stable)
Add facts when:
- User states a preference ("I prefer dark mode")
- User describes a project ("I'm building a mobile app")
- User sets a constraint ("I'm allergic to nuts" - if relevant)
- User shares a goal ("I want to learn Go programming")

Update confidence when:
- User contradicts a previous fact (lower confidence of old, add new with high confidence)
- Fact is confirmed multiple times (increase confidence)

## Fact Categories
- **work**: Job title, skills, tools used, career goals, company info
- **personal**: Location, timezone, family status (only if work-relevant)
- **preference**: UI themes, output formats, communication style, tools preferred
- **project**: Active projects, deadlines, tech stack, requirements
- **constraint**: Hard limits (budget, time, compliance requirements)
- **knowledge**: Domain expertise, certifications, learning goals

## Examples

Input: "我下周要去上海出差，帮我规划一下行程"
Output: {
  "user": {},
  "history": {},
  "facts": [{
    "id": "trip_shanghai_2024",
    "content": "Has upcoming business trip to Shanghai next week",
    "category": "work",
    "confidence": 0.9,
    "source": "session_abc123"
  }]
}

Input: "我喜欢用深色主题，看久了眼睛不累"
Output: {
  "user": {},
  "history": {},
  "facts": [{
    "id": "pref_dark_mode",
    "content": "Prefers dark mode UI for reduced eye strain",
    "category": "preference",
    "confidence": 0.95,
    "source": "session_def456"
  }]
}

Input: "随便聊聊" (No durable information)
Output: {
  "user": {},
  "history": {},
  "facts": [],
  "source": "session_xyz789"
}

Input: "我是做后端开发的，主要用 Go 和 Python"
Output: {
  "user": {
    "workContext": "Backend developer specializing in Go and Python"
  },
  "history": {},
  "facts": [{
    "id": "skill_go_python",
    "content": "Backend developer, proficient in Go and Python",
    "category": "work",
    "confidence": 0.9,
    "source": "session_ghi012"
  }]
}

## Schema
{
  "user": {
    "workContext": "string (job, role, skills - stable)",
    "personalContext": "string (location, timezone - rarely changes)",
    "topOfMind": "string (current priorities, goals)"
  },
  "history": {
    "recentMonths": "string (past 1-2 months summary)",
    "earlierContext": "string (older compressed context)",
    "longTermBackground": "string (persistent background info)"
  },
  "facts": [
    {
      "id": "unique_stable_id",
      "content": "Clear, specific fact text",
      "category": "work|personal|preference|project|constraint|knowledge",
      "confidence": 0.0-1.0,
      "source": "session_id_where_learned"
    }
  ],
  "source": "current_session_id"
}`

func BuildMemoryUpdatePrompt(messages []models.Message, current Document) string {
	currentJSON, _ := json.MarshalIndent(current, "", "  ")
	return fmt.Sprintf(
		"Current memory:\n%s\n\nRecent conversation:\n%s\n\nProduce the memory update JSON.",
		string(currentJSON),
		renderMessagesForPrompt(messages),
	)
}

func BuildInjection(doc Document) string {
	return BuildInjectionWithContext(doc, "", 0)
}

func BuildInjectionWithContext(doc Document, currentContext string, maxTokens int) string {
	if maxTokens <= 0 {
		maxTokens = 2000
	}

	lines := []string{"## User Memory"}

	if v := strings.TrimSpace(doc.User.WorkContext); v != "" {
		lines = append(lines, "Work Context: "+v)
	}
	if v := strings.TrimSpace(doc.User.PersonalContext); v != "" {
		lines = append(lines, "Personal Context: "+v)
	}
	if v := strings.TrimSpace(doc.User.TopOfMind); v != "" {
		lines = append(lines, "Top Of Mind: "+v)
	}
	if v := strings.TrimSpace(doc.History.RecentMonths); v != "" {
		lines = append(lines, "Recent Months: "+v)
	}
	if v := strings.TrimSpace(doc.History.EarlierContext); v != "" {
		lines = append(lines, "Earlier Context: "+v)
	}
	if v := strings.TrimSpace(doc.History.LongTermBackground); v != "" {
		lines = append(lines, "Long Term Background: "+v)
	}
	base := strings.Join(lines, "\n")
	selectedFacts := selectRelevantFacts(doc.Facts, currentContext, maxTokens-approximateTokenCount(base))
	if len(selectedFacts) > 0 {
		lines = append(lines, "Known Facts:")
		for _, fact := range selectedFacts {
			if strings.TrimSpace(fact.Category) != "" {
				lines = append(lines, fmt.Sprintf("- [%s] %s", fact.Category, fact.Content))
				continue
			}
			lines = append(lines, "- "+fact.Content)
		}
	}
	if len(lines) == 1 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n\n## Current Session\n"
}

func renderMessagesForPrompt(messages []models.Message) string {
	if len(messages) == 0 {
		return "(no messages)"
	}

	var b strings.Builder
	for _, msg := range messages {
		role := string(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if content == "" && msg.ToolResult != nil {
			content = strings.TrimSpace(msg.ToolResult.Content)
			if content == "" {
				content = strings.TrimSpace(msg.ToolResult.Error)
			}
		}
		if content == "" {
			continue
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "(no textual messages)"
	}
	return out
}

func selectRelevantFacts(facts []Fact, currentContext string, remainingTokens int) []Fact {
	if len(facts) == 0 || remainingTokens <= 0 {
		return nil
	}

	type scoredFact struct {
		fact       Fact
		score      float64
		similarity float64
		confidence float64
	}

	contextTerms := extractTerms(currentContext)
	useContext := len(contextTerms) > 0
	scored := make([]scoredFact, 0, len(facts))
	hasContextMatch := false
	for _, fact := range facts {
		fact.Content = strings.TrimSpace(fact.Content)
		if fact.Content == "" {
			continue
		}
		confidence := clamp01(fact.Confidence)
		score := confidence
		similarity := 0.0
		if useContext {
			similarity = cosineSimilarity(contextTerms, extractTerms(fact.Content))
			score = (similarity * 0.6) + (confidence * 0.4)
			if similarity > 0 {
				hasContextMatch = true
			}
		}
		scored = append(scored, scoredFact{
			fact:       fact,
			score:      score,
			similarity: similarity,
			confidence: confidence,
		})
	}
	if useContext && hasContextMatch {
		filtered := scored[:0]
		for _, item := range scored {
			if item.similarity > 0 {
				filtered = append(filtered, item)
			}
		}
		scored = filtered
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].confidence != scored[j].confidence {
			return scored[i].confidence > scored[j].confidence
		}
		if !scored[i].fact.UpdatedAt.Equal(scored[j].fact.UpdatedAt) {
			return scored[i].fact.UpdatedAt.After(scored[j].fact.UpdatedAt)
		}
		if !scored[i].fact.CreatedAt.Equal(scored[j].fact.CreatedAt) {
			return scored[i].fact.CreatedAt.After(scored[j].fact.CreatedAt)
		}
		return scored[i].fact.ID < scored[j].fact.ID
	})

	selected := make([]Fact, 0, len(scored))
	usedTokens := 0
	for _, item := range scored {
		line := item.fact.Content
		if strings.TrimSpace(item.fact.Category) != "" {
			line = fmt.Sprintf("- [%s] %s", item.fact.Category, item.fact.Content)
		} else {
			line = "- " + item.fact.Content
		}
		cost := approximateTokenCount(line)
		if usedTokens > 0 && usedTokens+cost > remainingTokens {
			break
		}
		selected = append(selected, item.fact)
		usedTokens += cost
	}
	return selected
}

func approximateTokenCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	runes := len([]rune(text))
	if runes <= 4 {
		return 1
	}
	return (runes + 3) / 4
}

func extractTerms(text string) map[string]float64 {
	terms := make(map[string]float64)
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		token := strings.ToLower(strings.TrimSpace(string(current)))
		if token != "" {
			terms[token]++
		}
		current = current[:0]
	}

	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.In(r, unicode.Han):
			flush()
			terms[string(r)]++
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			current = append(current, r)
		default:
			flush()
		}
	}
	flush()
	return terms
}

func cosineSimilarity(left, right map[string]float64) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	var dot float64
	var leftNorm float64
	var rightNorm float64
	for token, value := range left {
		leftNorm += value * value
		dot += value * right[token]
	}
	for _, value := range right {
		rightNorm += value * value
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
