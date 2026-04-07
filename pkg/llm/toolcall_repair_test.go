package llm

import (
	"testing"

	"github.com/easyspace-ai/minote/pkg/models"
)

func TestRepairToolCallIDsForAPI_pairsToolResults(t *testing.T) {
	msgs := []models.Message{
		{ID: "1", SessionID: "s", Role: models.RoleAI, ToolCalls: []models.ToolCall{
			{ID: "", Name: "bash", Arguments: map[string]any{}, Status: models.CallStatusPending},
		}},
		{ID: "2", SessionID: "s", Role: models.RoleTool, ToolResult: &models.ToolResult{
			CallID: "", ToolName: "bash", Content: "ok", Status: models.CallStatusCompleted,
		}},
	}
	repairToolCallIDsForAPI(msgs)
	if got := msgs[0].ToolCalls[0].ID; got != "call_0" {
		t.Fatalf("assistant tool id=%q", got)
	}
	if got := msgs[1].ToolResult.CallID; got != "call_0" {
		t.Fatalf("tool call_id=%q", got)
	}
}

func TestRepairToolCallIDsForAPI_multiToolOrder(t *testing.T) {
	msgs := []models.Message{
		{ID: "1", SessionID: "s", Role: models.RoleAI, ToolCalls: []models.ToolCall{
			{ID: "", Name: "a", Arguments: nil, Status: models.CallStatusPending},
			{ID: "", Name: "b", Arguments: nil, Status: models.CallStatusPending},
		}},
		{ID: "2", SessionID: "s", Role: models.RoleTool, ToolResult: &models.ToolResult{
			CallID: "", ToolName: "a", Content: "1", Status: models.CallStatusCompleted,
		}},
		{ID: "3", SessionID: "s", Role: models.RoleTool, ToolResult: &models.ToolResult{
			CallID: "", ToolName: "b", Content: "2", Status: models.CallStatusCompleted,
		}},
	}
	repairToolCallIDsForAPI(msgs)
	if msgs[0].ToolCalls[0].ID != "call_0" || msgs[0].ToolCalls[1].ID != "call_1" {
		t.Fatalf("assistant ids: %#v", msgs[0].ToolCalls)
	}
	if msgs[1].ToolResult.CallID != "call_0" || msgs[2].ToolResult.CallID != "call_1" {
		t.Fatalf("tool ids: %s %s", msgs[1].ToolResult.CallID, msgs[2].ToolResult.CallID)
	}
}

func TestRepairToolCallIDsForAPI_preservesExistingIDs(t *testing.T) {
	msgs := []models.Message{
		{ID: "1", SessionID: "s", Role: models.RoleAI, ToolCalls: []models.ToolCall{
			{ID: "tc_1", Name: "bash", Arguments: map[string]any{}, Status: models.CallStatusPending},
		}},
		{ID: "2", SessionID: "s", Role: models.RoleTool, ToolResult: &models.ToolResult{
			CallID: "tc_1", ToolName: "bash", Content: "ok", Status: models.CallStatusCompleted,
		}},
	}
	repairToolCallIDsForAPI(msgs)
	if msgs[1].ToolResult.CallID != "tc_1" {
		t.Fatalf("call_id=%q", msgs[1].ToolResult.CallID)
	}
}
