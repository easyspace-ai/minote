package llm

import (
	"encoding/json"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/easyspace-ai/minote/pkg/models"
)

var (
	// 匹配工具调用的各种格式
	toolCallRegex = regexp.MustCompile(`(?is)<\|tool_call\|>\s*(.*?)\s*<\|/tool_call\|>`)
	jsonRegex     = regexp.MustCompile(`(?is)\{.*\}|\[.*\]`)

	// 常见的JSON格式错误修复
	jsonFixes = []struct{
		from string
		to   string
	}{
		{`\n`, ``},        // 移除换行
		{`\t`, ``},        // 移除制表符
		{`\s+`, ` `},      // 多个空格合并为一个
		{`,\s*}`, `}`},    // 移除最后一个属性后的逗号
		{`,\s*]`, `]`},    // 移除最后一个数组元素后的逗号
		{`'`, `"`},        // 单引号转双引号
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ParsedOutput 解析后的输出结果
type ParsedOutput struct {
	Content   string             `json:"content"`
	ToolCalls []models.ToolCall  `json:"tool_calls"`
	Errors    []string           `json:"errors"`
}

// OutputParser 输出解析器，处理LLM返回的内容，分离文本和工具调用，修复格式错误
type OutputParser interface {
	Parse(content string) ParsedOutput
}

// NewOutputParser 创建默认的输出解析器
func NewOutputParser() OutputParser {
	return &defaultOutputParser{}
}

type defaultOutputParser struct{}

func (p *defaultOutputParser) Parse(content string) ParsedOutput {
	result := ParsedOutput{
		Content:   content,
		ToolCalls: make([]models.ToolCall, 0),
		Errors:    make([]string, 0),
	}

	// 1. 提取工具调用块
	toolCallMatches := toolCallRegex.FindAllStringSubmatch(content, -1)
	if len(toolCallMatches) == 0 {
		// 没有工具调用，直接返回
		return result
	}

	// 2. 移除内容中的工具调用块，保留纯文本
	result.Content = toolCallRegex.ReplaceAllString(content, "")
	result.Content = strings.TrimSpace(result.Content)

	// 3. 处理每个工具调用
	for _, match := range toolCallMatches {
		if len(match) < 2 {
			continue
		}
		toolCallJSON := strings.TrimSpace(match[1])
		if toolCallJSON == "" {
			continue
		}

		// 4. 尝试修复JSON格式
		fixedJSON, ok := p.fixJSON(toolCallJSON)
		if !ok {
			result.Errors = append(result.Errors, "invalid tool call JSON format: "+toolCallJSON)
			continue
		}

		// 5. 先尝试解析为数组
		var rawCalls []map[string]any
		if err := json.Unmarshal([]byte(fixedJSON), &rawCalls); err == nil {
			for _, raw := range rawCalls {
				call := p.convertRawToToolCall(raw)
				if p.validateToolCall(&call) {
					result.ToolCalls = append(result.ToolCalls, call)
				} else {
					result.Errors = append(result.Errors, "invalid tool call: "+call.Name)
				}
			}
			continue
		}

		// 6. 解析为单个对象
		var rawCall map[string]any
		if err := json.Unmarshal([]byte(fixedJSON), &rawCall); err != nil {
			result.Errors = append(result.Errors, "failed to parse tool call: "+err.Error())
			continue
		}

		call := p.convertRawToToolCall(rawCall)
		if p.validateToolCall(&call) {
			result.ToolCalls = append(result.ToolCalls, call)
		} else {
			result.Errors = append(result.Errors, "invalid tool call: "+call.Name)
		}
	}

	return result
}

// convertRawToToolCall 把raw map转换为ToolCall，支持parameters和arguments两种格式
func (p *defaultOutputParser) convertRawToToolCall(raw map[string]any) models.ToolCall {
	call := models.ToolCall{
		Arguments: make(map[string]any),
	}

	// 处理ID
	if id, ok := raw["id"].(string); ok {
		call.ID = id
	}

	// 处理名称
	if name, ok := raw["name"].(string); ok {
		call.Name = name
	}

	// 处理参数，支持parameters和arguments两种key
	if params, ok := raw["parameters"].(map[string]any); ok {
		call.Arguments = params
	} else if args, ok := raw["arguments"].(map[string]any); ok {
		call.Arguments = args
	}

	return call
}

// fixJSON 尝试修复常见的JSON格式错误
func (p *defaultOutputParser) fixJSON(input string) (string, bool) {
	if input == "" {
		return "", false
	}

	// 提取JSON部分
	jsonMatch := jsonRegex.FindString(input)
	if jsonMatch == "" {
		return "", false
	}
	input = jsonMatch

	// 应用常见修复
	for _, fix := range jsonFixes {
		input = regexp.MustCompile(fix.from).ReplaceAllString(input, fix.to)
	}

	// 验证是否是合法JSON
	var temp any
	if err := json.Unmarshal([]byte(input), &temp); err == nil {
		return input, true
	}

	// 尝试修复未闭合的引号
	if strings.Count(input, `"`)%2 != 0 {
		input += `"`
		if err := json.Unmarshal([]byte(input), &temp); err == nil {
			return input, true
		}
	}

	// 尝试修复未闭合的括号
	if strings.Count(input, "{") > strings.Count(input, "}") {
		input += strings.Repeat("}", strings.Count(input, "{")-strings.Count(input, "}"))
		if err := json.Unmarshal([]byte(input), &temp); err == nil {
			return input, true
		}
	}
	if strings.Count(input, "[") > strings.Count(input, "]") {
		input += strings.Repeat("]", strings.Count(input, "[")-strings.Count(input, "]"))
		if err := json.Unmarshal([]byte(input), &temp); err == nil {
			return input, true
		}
	}

	return "", false
}

// validateToolCall 验证工具调用是否有效
func (p *defaultOutputParser) validateToolCall(call *models.ToolCall) bool {
	// 工具名称不能为空
	if strings.TrimSpace(call.Name) == "" {
		return false
	}

	// 参数如果为空，初始化为空map
	if len(call.Arguments) == 0 {
		call.Arguments = make(map[string]any)
	}

	// 自动生成ID（如果为空）
	if strings.TrimSpace(call.ID) == "" {
		call.ID = "call_" + randomString(8)
	}

	return true
}

// randomString 生成随机字符串
func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
