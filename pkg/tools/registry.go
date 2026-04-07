package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
	pkgsandbox "github.com/easyspace-ai/minote/pkg/sandbox"
)

type Sandbox = pkgsandbox.Sandbox

type contextKey string

const (
	sandboxContextKey  contextKey = "tool_sandbox"
	threadIDContextKey contextKey = "tool_thread_id"
	runtimeContextKey  contextKey = "tool_runtime_context"
)

var toolCallSeq uint64

const (
	// 重试配置
	defaultMaxRetries    = 2   // 最大重试次数
	defaultInitialDelay  = 100 * time.Millisecond // 初始重试延迟
	defaultMaxDelay      = 2 * time.Second // 最大重试延迟
)

// 可重试错误列表
var retryableErrors = []string{
	"timeout",
	"deadline exceeded",
	"connection refused",
	"connection reset",
	"network is unreachable",
	"no such host",
	"temporary error",
	"throttle",
	"rate limit",
	"too many requests",
	"500",
	"502",
	"503",
	"504",
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]models.Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]models.Tool)}
}

func (r *Registry) Register(tool models.Tool) error {
	if err := tool.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %q already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	return nil
}

func (r *Registry) Unregister(name string) bool {
	if r == nil {
		return false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; !exists {
		return false
	}
	delete(r.tools, name)
	return true
}

func (r *Registry) Get(name string) *models.Tool {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[strings.TrimSpace(name)]
	if !ok {
		return nil
	}
	copy := tool
	return &copy
}

func (r *Registry) List() []models.Tool {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]models.Tool, 0, len(names))
	for _, name := range names {
		out = append(out, r.tools[name])
	}
	return out
}

func (r *Registry) ListByGroup(group string) []models.Tool {
	if r == nil {
		return nil
	}
	group = strings.TrimSpace(group)
	if group == "" {
		return r.List()
	}

	all := r.List()
	filtered := make([]models.Tool, 0, len(all))
	for _, tool := range all {
		for _, candidate := range tool.Groups {
			if candidate == group {
				filtered = append(filtered, tool)
				break
			}
		}
	}
	return filtered
}

func (r *Registry) Descriptions() string {
	tools := r.List()
	if len(tools) == 0 {
		return ""
	}
	var lines []string
	for _, tool := range tools {
		line := fmt.Sprintf("- %s: %s", tool.Name, strings.TrimSpace(tool.Description))
		if len(tool.InputSchema) > 0 {
			if raw, err := json.MarshalIndent(tool.InputSchema, "", "  "); err == nil {
				line += "\n  schema: " + strings.ReplaceAll(string(raw), "\n", "\n  ")
			}
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func WithSandbox(ctx context.Context, sandbox *Sandbox) context.Context {
	if sandbox == nil {
		return ctx
	}
	return context.WithValue(ctx, sandboxContextKey, sandbox)
}

func SandboxFromContext(ctx context.Context) *Sandbox {
	if ctx == nil {
		return nil
	}
	sandbox, _ := ctx.Value(sandboxContextKey).(*Sandbox)
	return sandbox
}

func WithThreadID(ctx context.Context, threadID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return ctx
	}
	return context.WithValue(ctx, threadIDContextKey, threadID)
}

func ThreadIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	threadID, _ := ctx.Value(threadIDContextKey).(string)
	return strings.TrimSpace(threadID)
}

func WithRuntimeContext(ctx context.Context, values map[string]any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(values) == 0 {
		return ctx
	}

	cloned := make(map[string]any, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cloned[key] = value
	}
	if len(cloned) == 0 {
		return ctx
	}
	return context.WithValue(ctx, runtimeContextKey, cloned)
}

func RuntimeContextFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	values, _ := ctx.Value(runtimeContextKey).(map[string]any)
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func (r *Registry) Call(ctx context.Context, name string, args map[string]interface{}, sandbox *Sandbox) (string, error) {
	if r == nil {
		return "", fmt.Errorf("tool registry is nil")
	}
	tool := r.Get(name)
	if tool == nil {
		return "", fmt.Errorf("tool %q not found", strings.TrimSpace(name))
	}
	if err := validateArgs(tool.InputSchema, args); err != nil {
		return "", err
	}

	call := models.ToolCall{
		ID:          newToolCallID(strings.TrimSpace(name)),
		Name:        strings.TrimSpace(name),
		Arguments:   args,
		Status:      models.CallStatusPending,
		RequestedAt: time.Now().UTC(),
	}
	result, err := r.executeWithSandbox(ctx, call, sandbox)
	if err != nil {
		if strings.TrimSpace(result.Error) != "" {
			return result.Content, errors.New(result.Error)
		}
		return result.Content, err
	}
	return result.Content, nil
}

func (r *Registry) Execute(ctx context.Context, call models.ToolCall) (models.ToolResult, error) {
	return r.executeWithSandbox(ctx, call, SandboxFromContext(ctx))
}

// isRetryableError 判断错误是否可以重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	// 检查是否是网络相关错误
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Temporary() {
		return true
	}

	// 检查错误消息是否包含可重试关键词
	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// 指数退避计算延迟
func backoffDelay(attempt int) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * defaultInitialDelay
	if delay > defaultMaxDelay {
		delay = defaultMaxDelay
	}
	return delay
}

func (r *Registry) executeWithSandbox(ctx context.Context, call models.ToolCall, sandbox *Sandbox) (models.ToolResult, error) {
	if r == nil {
		return models.ToolResult{}, fmt.Errorf("tool registry is nil")
	}

	toolName := strings.TrimSpace(call.Name)
	if toolName == "" {
		return models.ToolResult{
			CallID:      call.ID,
			ToolName:    "unknown",
			Status:      models.CallStatusFailed,
			Error:       "Error: Tool name is empty. Please specify a valid tool name to call.",
			CompletedAt: time.Now().UTC(),
		}, fmt.Errorf("tool name is empty")
	}

	r.mu.RLock()
	tool, ok := r.tools[toolName]
	r.mu.RUnlock()
	if !ok {
		availableTools := make([]string, 0, len(r.tools))
		for name := range r.tools {
			availableTools = append(availableTools, name)
		}
		sort.Strings(availableTools)
		errorMsg := fmt.Sprintf(
			"Error: Tool %q not found. Available tools: %s. Use tool_search to find more tools, or choose an available tool from the list.",
			toolName,
			strings.Join(availableTools, ", "),
		)
		return models.ToolResult{
			CallID:      call.ID,
			ToolName:    toolName,
			Status:      models.CallStatusFailed,
			Error:       errorMsg,
			CompletedAt: time.Now().UTC(),
		}, fmt.Errorf("tool %q not found", toolName)
	}
	if err := validateArgs(tool.InputSchema, call.Arguments); err != nil {
		return models.ToolResult{
			CallID:      call.ID,
			ToolName:    call.Name,
			Status:      models.CallStatusFailed,
			Error:       FormatToolExecutionError(call.Name, err),
			CompletedAt: time.Now().UTC(),
		}, err
	}

	var (
		result models.ToolResult
		err    error
	)

	// 重试逻辑
	for attempt := 0; attempt <= defaultMaxRetries; attempt++ {
		started := time.Now().UTC()
		call.Status = models.CallStatusRunning
		call.StartedAt = started

		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					err = fmt.Errorf("tool %q panicked: %v", call.Name, recovered)
					result = models.ToolResult{
						CallID:      call.ID,
						ToolName:    call.Name,
						Status:      models.CallStatusFailed,
						Error:       formatToolPanicMessage(call.Name, recovered),
						CompletedAt: time.Now().UTC(),
					}
				}
			}()
			result, err = tool.Handler(WithSandbox(ctx, sandbox), call)
		}()

		// 成功或者不可重试错误，直接返回
		if err == nil || !isRetryableError(err) {
			break
		}

		// 最后一次重试失败，不再等待
		if attempt == defaultMaxRetries {
			break
		}

		// 指数退避等待
		delay := backoffDelay(attempt)
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
			// 继续重试
		}
	}

	if result.CallID == "" {
		result.CallID = call.ID
	}
	if result.ToolName == "" {
		result.ToolName = call.Name
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now().UTC()
	}
	if result.Duration == 0 {
		result.Duration = time.Since(result.CompletedAt)
	}
	if err != nil {
		if result.Status == "" {
			result.Status = models.CallStatusFailed
		}
		if result.Error == "" {
			result.Error = FormatToolExecutionError(call.Name, err)
		}
		return result, err
	}
	if result.Status == "" {
		result.Status = models.CallStatusCompleted
	}
	return result, nil
}

func formatToolPanicMessage(toolName string, recovered any) string {
	detail := strings.TrimSpace(fmt.Sprint(recovered))
	if detail == "" {
		detail = "panic"
	}
	if len(detail) > 500 {
		detail = detail[:497] + "..."
	}
	stack := strings.TrimSpace(string(debug.Stack()))
	if stack != "" {
		return fmt.Sprintf(
			"Error: Tool %q panicked: %s. Continue with available context, or choose an alternative tool.\n\nStack trace:\n%s",
			toolName,
			detail,
			stack,
		)
	}
	return fmt.Sprintf(
		"Error: Tool %q panicked: %s. Continue with available context, or choose an alternative tool.",
		toolName,
		detail,
	)
}

func FormatToolExecutionError(toolName string, err error) string {
	detail := ""
	if err != nil {
		detail = strings.TrimSpace(err.Error())
	}
	if detail == "" {
		detail = "tool execution failed"
	}
	if len(detail) > 500 {
		detail = detail[:497] + "..."
	}
	errType := "error"
	if err != nil {
		errType = errTypeName(err)
	}

	// 根据错误类型给出不同的建议
	suggestion := "Continue with available context, or choose an alternative tool."
	if strings.Contains(strings.ToLower(detail), "timeout") || strings.Contains(strings.ToLower(detail), "deadline exceeded") {
		suggestion = "You can retry the call with a longer timeout, or try a different tool."
	} else if strings.Contains(strings.ToLower(detail), "permission denied") || strings.Contains(strings.ToLower(detail), "access denied") {
		suggestion = "Check if you have permission to access this resource, or try a different approach."
	} else if strings.Contains(strings.ToLower(detail), "not found") || strings.Contains(strings.ToLower(detail), "no such file") {
		suggestion = "Verify the path/resource exists and is correct, then try again."
	}

	return fmt.Sprintf(
		"Error: Tool %q failed with %s: %s. %s",
		toolName,
		errType,
		detail,
		suggestion,
	)
}

func errTypeName(err error) string {
	if err == nil {
		return "error"
	}
	typeName := fmt.Sprintf("%T", err)
	if idx := strings.LastIndex(typeName, "."); idx >= 0 && idx+1 < len(typeName) {
		typeName = typeName[idx+1:]
	}
	typeName = strings.TrimPrefix(typeName, "*")
	if strings.TrimSpace(typeName) == "" {
		return "error"
	}
	return typeName
}

func (r *Registry) Restrict(allowed []string) *Registry {
	if r == nil {
		return NewRegistry()
	}
	if len(allowed) == 0 {
		return r
	}

	allow := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		name = strings.TrimSpace(name)
		if name != "" {
			allow[name] = struct{}{}
		}
	}

	restricted := NewRegistry()
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, tool := range r.tools {
		if _, ok := allow[name]; ok {
			restricted.tools[name] = tool
		}
	}
	return restricted
}

func newToolCallID(name string) string {
	seq := atomic.AddUint64(&toolCallSeq, 1)
	return fmt.Sprintf("%s_%d_%d", name, time.Now().UTC().UnixNano(), seq)
}

// normalizeArgsForSchema coerces LLM-frequent type mismatches so validation and handlers see
// canonical JSON types (e.g. "true"/"false" strings → bool for boolean properties).
func normalizeArgsForSchema(schema map[string]any, args map[string]any) {
	if len(schema) == 0 || len(args) == 0 {
		return
	}
	properties, _ := schema["properties"].(map[string]any)
	for name, rawSpec := range properties {
		spec, ok := rawSpec.(map[string]any)
		if !ok {
			continue
		}
		kind, _ := spec["type"].(string)
		if kind != "boolean" {
			continue
		}
		v, ok := args[name]
		if !ok || v == nil {
			continue
		}
		if b, ok := coerceToBool(v); ok {
			args[name] = b
		}
	}
}

func coerceToBool(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		switch s {
		case "true", "1", "yes", "y", "on":
			return true, true
		case "false", "0", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func validateArgs(schema map[string]any, args map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	if args == nil {
		args = map[string]any{}
	}

	normalizeArgsForSchema(schema, args)

	required, _ := schema["required"].([]any)
	if len(required) == 0 {
		if typed, ok := schema["required"].([]string); ok {
			required = make([]any, 0, len(typed))
			for _, item := range typed {
				required = append(required, item)
			}
		}
	}
	for _, raw := range required {
		name, _ := raw.(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		value, ok := args[name]
		if !ok || value == nil {
			return fmt.Errorf("missing required argument '%s'. Please provide a value for this parameter.", name)
		}
	}

	properties, _ := schema["properties"].(map[string]any)
	for name, rawSpec := range properties {
		value, ok := args[name]
		if !ok || value == nil {
			continue
		}
		spec, _ := rawSpec.(map[string]any)
		if err := validateType(name, spec["type"], value); err != nil {
			return fmt.Errorf("%s Please check the parameter type and try again.", err.Error())
		}
	}
	return nil
}

func validateType(name string, expected any, value any) error {
	kind, _ := expected.(string)
	actualType := fmt.Sprintf("%T", value)
	switch kind {
	case "", "any":
		return nil
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("argument '%s' must be a string, but got %s", name, actualType)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("argument '%s' must be a boolean (true/false), but got %s", name, actualType)
		}
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		case float32:
			if float32(int64(value.(float32))) == value.(float32) {
				return nil
			}
		case float64:
			if float64(int64(value.(float64))) == value.(float64) {
				return nil
			}
		}
		return fmt.Errorf("argument '%s' must be an integer, but got %s", name, actualType)
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return nil
		default:
			return fmt.Errorf("argument '%s' must be a number, but got %s", name, actualType)
		}
	case "array":
		switch value.(type) {
		case []any, []string:
			return nil
		default:
			return fmt.Errorf("argument '%s' must be an array, but got %s", name, actualType)
		}
	case "object":
		switch value.(type) {
		case map[string]any:
			return nil
		default:
			return fmt.Errorf("argument '%s' must be an object (JSON map), but got %s", name, actualType)
		}
	default:
		return nil
	}
	return nil
}
