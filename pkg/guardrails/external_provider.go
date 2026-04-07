package guardrails

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ExternalProvider delegates guardrail evaluation to an external process.
// The external command receives the request as JSON on stdin and returns
// a JSON decision on stdout.
type ExternalProvider struct {
	command string
	args    []string
	timeout time.Duration
}

// NewExternalProvider creates an external process guardrails provider.
func NewExternalProvider(command string, providerArgs map[string]any) *ExternalProvider {
	p := &ExternalProvider{
		command: command,
		timeout: 10 * time.Second,
	}
	if providerArgs != nil {
		if args, ok := providerArgs["args"]; ok {
			if items, ok := args.([]any); ok {
				for _, item := range items {
					if s := stringValue(item); s != "" {
						p.args = append(p.args, s)
					}
				}
			}
		}
		if timeoutSecs, ok := providerArgs["timeout_seconds"]; ok {
			if v, ok := timeoutSecs.(float64); ok && v > 0 {
				p.timeout = time.Duration(v) * time.Second
			}
		}
	}
	return p
}

func (p *ExternalProvider) Name() string {
	return "external_process_provider"
}

type externalRequest struct {
	ToolName   string         `json:"tool_name"`
	ToolInput  map[string]any `json:"tool_input"`
	AgentID    string         `json:"agent_id"`
	ThreadID   string         `json:"thread_id"`
	IsSubagent bool           `json:"is_subagent"`
}

type externalResponse struct {
	Allow   bool   `json:"allow"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (p *ExternalProvider) Evaluate(req Request) (Decision, error) {
	input := externalRequest{
		ToolName:   req.ToolName,
		ToolInput:  req.ToolInput,
		AgentID:    req.AgentID,
		ThreadID:   req.ThreadID,
		IsSubagent: req.IsSubagent,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return Decision{}, fmt.Errorf("external guardrail: marshal request: %w", err)
	}

	cmd := exec.Command(p.command, p.args...)
	cmd.Stdin = bytes.NewReader(inputJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return Decision{
				Allow:   false,
				Reasons: []Reason{{Code: "process_error", Message: fmt.Sprintf("external guardrail failed: %v: %s", err, stderr.String())}},
			}, nil
		}
	case <-time.After(p.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return Decision{
			Allow:   false,
			Reasons: []Reason{{Code: "timeout", Message: "external guardrail timed out"}},
		}, nil
	}

	var resp externalResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return Decision{
			Allow:   false,
			Reasons: []Reason{{Code: "parse_error", Message: fmt.Sprintf("external guardrail returned invalid JSON: %v", err)}},
		}, nil
	}

	decision := Decision{
		Allow:    resp.Allow,
		PolicyID: "external",
	}
	if !resp.Allow {
		code := resp.Code
		if code == "" {
			code = "denied"
		}
		msg := resp.Message
		if msg == "" {
			msg = "Denied by external guardrail"
		}
		decision.Reasons = []Reason{{Code: code, Message: msg}}
	}
	return decision, nil
}
