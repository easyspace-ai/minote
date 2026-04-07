package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
	"github.com/easyspace-ai/minote/pkg/sandbox"
	"github.com/easyspace-ai/minote/pkg/tools"
)

// BashOutputMaxChars limits the stdout/stderr length returned to the LLM.
// 0 means no limit. Can be configured via config.yaml sandbox.bash_output_max_chars.
var BashOutputMaxChars int

// 允许执行的命令白名单
var allowedCommands = map[string]bool{
	"ls": true, "cat": true, "grep": true, "echo": true, "mkdir": true, "rm": true, "cp": true, "mv": true,
	"find": true, "head": true, "tail": true, "wc": true, "sort": true, "uniq": true, "cut": true, "awk": true,
	"sed": true, "diff": true, "patch": true, "tar": true, "zip": true, "unzip": true, "curl": true, "wget": true,
	"git": true, "go": true, "python": true, "python3": true, "node": true, "npm": true, "pnpm": true, "yarn": true,
}

func BashHandler(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
	args := call.Arguments

	cmd, ok := args["command"].(string)
	if !ok || strings.TrimSpace(cmd) == "" {
		return models.ToolResult{CallID: call.ID, ToolName: call.Name}, fmt.Errorf("command is required")
	}

	// 检查命令白名单
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		baseCmd := strings.ToLower(parts[0])
		// 去除路径，只保留命令名
		if idx := strings.LastIndex(baseCmd, "/"); idx >= 0 {
			baseCmd = baseCmd[idx+1:]
		}
		if !allowedCommands[baseCmd] {
			return models.ToolResult{CallID: call.ID, ToolName: call.Name}, fmt.Errorf("command '%s' is not allowed for security reasons", baseCmd)
		}
	}

	timeout := 60 * time.Second
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Second
	}

	cmd = tools.ResolveVirtualCommand(ctx, cmd)
	workdir := tools.ResolveWorkingDirectory(ctx)
	if strings.TrimSpace(workdir) != "" {
		if err := os.MkdirAll(workdir, 0o755); err != nil {
			return models.ToolResult{CallID: call.ID, ToolName: call.Name}, fmt.Errorf("prepare workspace failed: %w", err)
		}
	}
	result, err := sandbox.ExecDirectInDir(ctx, cmd, workdir, timeout)
	if err != nil {
		return models.ToolResult{CallID: call.ID, ToolName: call.Name}, fmt.Errorf("bash failed: %w", err)
	}

	output := &BashOutput{
		Stdout:   truncateOutput(tools.MaskLocalPaths(ctx, result.Stdout())),
		Stderr:   truncateOutput(tools.MaskLocalPaths(ctx, result.Stderr())),
		ExitCode: result.ExitCode(),
	}
	data, _ := json.Marshal(output)
	return models.ToolResult{CallID: call.ID, ToolName: call.Name, Content: string(data)}, nil
}

type BashOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func BashTool() models.Tool {
	return models.Tool{
		Name:        "bash",
		Description: "Execute shell commands. Returns stdout, stderr, and exit code as JSON.",
		Groups:      []string{"builtin"},
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string", "description": "Shell command to execute"},
				"timeout": map[string]any{"type": "number", "description": "Timeout in seconds (default 60)"},
			},
			"required": []any{"command"},
		},
		Handler: BashHandler,
	}
}

func truncateOutput(s string) string {
	if BashOutputMaxChars <= 0 || len(s) <= BashOutputMaxChars {
		return s
	}
	return s[:BashOutputMaxChars] + "\n... [output truncated]"
}
