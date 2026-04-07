package llm

import (
	"fmt"
	"strings"

	"github.com/easyspace-ai/minote/pkg/models"
)

// cloneMessagesForAPIRequest returns a deep-ish copy of messages with tool_call_id repair applied.
// Some gateways (e.g. Volcengine Ark) reject requests when tool-role messages omit tool_call_id
// (JSON omitempty strips empty strings).
func cloneMessagesForAPIRequest(msgs []models.Message) []models.Message {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]models.Message, len(msgs))
	for i := range msgs {
		out[i] = cloneMessageForAPI(msgs[i])
	}
	repairToolCallIDsForAPI(out)
	return out
}

func cloneMessageForAPI(m models.Message) models.Message {
	out := m
	if m.ToolResult != nil {
		tr := *m.ToolResult
		out.ToolResult = &tr
	}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = append([]models.ToolCall(nil), m.ToolCalls...)
	}
	if len(m.Metadata) > 0 {
		out.Metadata = make(map[string]string, len(m.Metadata))
		for k, v := range m.Metadata {
			out.Metadata[k] = v
		}
	}
	return out
}

// repairToolCallIDsForAPI fills empty ToolResult.CallID from the preceding assistant message's
// tool_calls (by order), and fills empty assistant tool call IDs so outbound JSON stays consistent.
func repairToolCallIDsForAPI(msgs []models.Message) {
	var pending *models.Message
	var toolIdx int

	for i := range msgs {
		m := &msgs[i]
		switch m.Role {
		case models.RoleHuman:
			pending = nil
		case models.RoleAI:
			if len(m.ToolCalls) > 0 {
				pending = m
				toolIdx = 0
			} else {
				pending = nil
			}
		case models.RoleTool:
			if m.ToolResult == nil {
				continue
			}
			if strings.TrimSpace(m.ToolResult.CallID) != "" {
				if pending != nil && toolIdx < len(pending.ToolCalls) {
					toolIdx++
				}
				continue
			}
			if pending != nil && toolIdx < len(pending.ToolCalls) {
				id := strings.TrimSpace(pending.ToolCalls[toolIdx].ID)
				if id == "" {
					id = fmt.Sprintf("call_%d", toolIdx)
					pending.ToolCalls[toolIdx].ID = id
				}
				m.ToolResult.CallID = id
				toolIdx++
				continue
			}
			m.ToolResult.CallID = "tool_call_id_missing"
		default:
			pending = nil
		}
	}
}
