package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputParser_Parse(t *testing.T) {
	parser := NewOutputParser()

	t.Run("纯文本内容，无工具调用", func(t *testing.T) {
		content := "这是一个纯文本回答，不需要调用工具。"
		result := parser.Parse(content)
		assert.Equal(t, content, result.Content)
		assert.Empty(t, result.ToolCalls)
		assert.Empty(t, result.Errors)
	})

	t.Run("正常格式工具调用", func(t *testing.T) {
		content := `我需要搜索一下这个问题。
<|tool_call|>
{"name": "web_search", "parameters": {"query": "Go语言最佳实践"}}
<|/tool_call|>`
		result := parser.Parse(content)
		assert.Equal(t, "我需要搜索一下这个问题。", result.Content)
		assert.Len(t, result.ToolCalls, 1)
		assert.Equal(t, "web_search", result.ToolCalls[0].Name)
		assert.Equal(t, "Go语言最佳实践", result.ToolCalls[0].Arguments["query"])
		assert.Empty(t, result.Errors)
	})

	t.Run("JSON格式错误，自动修复", func(t *testing.T) {
		content := `<|tool_call|>
{"name": "web_search", "parameters": {"query": "Go语言最佳实践",}
<|/tool_call|>` // 注意最后多了一个逗号
		result := parser.Parse(content)
		assert.Empty(t, result.Content)
		assert.Len(t, result.ToolCalls, 1)
		assert.Equal(t, "web_search", result.ToolCalls[0].Name)
		assert.Equal(t, "Go语言最佳实践", result.ToolCalls[0].Arguments["query"])
		assert.Empty(t, result.Errors)
	})

	t.Run("单引号JSON，自动修复", func(t *testing.T) {
		content := `<|tool_call|>
{'name': 'web_search', 'parameters': {'query': 'Go语言最佳实践'}}
<|/tool_call|>`
		result := parser.Parse(content)
		assert.Len(t, result.ToolCalls, 1)
		assert.Equal(t, "web_search", result.ToolCalls[0].Name)
		assert.Equal(t, "Go语言最佳实践", result.ToolCalls[0].Arguments["query"])
		assert.Empty(t, result.Errors)
	})

	t.Run("多个工具调用", func(t *testing.T) {
		content := `<|tool_call|>
[{"name": "web_search", "parameters": {"query": "Go语言"}}, {"name": "read_file", "parameters": {"path": "test.go"}}]
<|/tool_call|>`
		result := parser.Parse(content)
		assert.Len(t, result.ToolCalls, 2)
		assert.Equal(t, "web_search", result.ToolCalls[0].Name)
		assert.Equal(t, "read_file", result.ToolCalls[1].Name)
		assert.Empty(t, result.Errors)
	})

	t.Run("工具名称为空，自动过滤", func(t *testing.T) {
		content := `<|tool_call|>
{"name": "", "parameters": {"query": "test"}}
<|/tool_call|>`
		result := parser.Parse(content)
		assert.Empty(t, result.ToolCalls)
		assert.Len(t, result.Errors, 1)
	})

	t.Run("JSON未闭合，自动修复", func(t *testing.T) {
		content := `<|tool_call|>
{"name": "web_search", "parameters": {"query": "Go语言最佳实践"
<|/tool_call|>` // 缺少闭合括号
		result := parser.Parse(content)
		assert.Len(t, result.ToolCalls, 1)
		assert.Equal(t, "web_search", result.ToolCalls[0].Name)
		assert.Equal(t, "Go语言最佳实践", result.ToolCalls[0].Arguments["query"])
		assert.Empty(t, result.Errors)
	})

	t.Run("混合格式，文本和多个工具调用", func(t *testing.T) {
		content := `我需要先搜索资料，再读取文件。
<|tool_call|>{"name": "web_search", "parameters": {"query": "Go语言"}}<|/tool_call|>
然后再做下一步。
<|tool_call|>{"name": "read_file", "parameters": {"path": "test.go"}}<|/tool_call|>`
		result := parser.Parse(content)
		assert.Equal(t, "我需要先搜索资料，再读取文件。\n然后再做下一步。", result.Content)
		assert.Len(t, result.ToolCalls, 2)
		assert.Equal(t, "web_search", result.ToolCalls[0].Name)
		assert.Equal(t, "read_file", result.ToolCalls[1].Name)
		assert.Empty(t, result.Errors)
	})
}

func TestOutputParser_fixJSON(t *testing.T) {
	parser := NewOutputParser().(*defaultOutputParser)

	testCases := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			name:     "正常JSON",
			input:    `{"name": "test", "value": 123}`,
			expected: `{"name": "test", "value": 123}`,
			ok:       true,
		},
		{
			name:     "多余逗号",
			input:    `{"name": "test", "value": 123,}`,
			expected: `{"name": "test", "value": 123}`,
			ok:       true,
		},
		{
			name:     "单引号",
			input:    `{'name': 'test', 'value': 123}`,
			expected: `{"name": "test", "value": 123}`,
			ok:       true,
		},
		{
			name:     "缺少闭合括号",
			input:    `{"name": "test", "value": 123`,
			expected: `{"name": "test", "value": 123}`,
			ok:       true,
		},
		{
			name:     "缺少引号",
			input:    `{"name": "test, "value": 123}`,
			expected: `{"name": "test", "value": 123}`,
			ok:       true,
		},
		{
			name:     "包含换行和制表符",
			input:    "{\n\t\"name\": \"test\",\n\t\"value\": 123\n}",
			expected: `{"name": "test", "value": 123}`,
			ok:       true,
		},
		{
			name:     "完全无效的JSON",
			input:    `not a json`,
			expected: "",
			ok:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, ok := parser.fixJSON(tc.input)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.JSONEq(t, tc.expected, result)
			}
		})
	}
}
