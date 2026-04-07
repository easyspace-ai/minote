package llm

import (
	"fmt"
	"unicode/utf8"

	"github.com/easyspace-ai/minote/pkg/models"
)

const (
	// 默认token阈值，按平均1个token=4个字符估算
	defaultMaxTokens = 8000
	// 默认保留最近的对话轮数
	defaultKeepRounds = 10
	// 每个字符平均占0.25个token
	charsPerToken = 4
)

// MessageTrimmer 消息历史修剪器，避免context overflow
type MessageTrimmer interface {
	Trim(messages []models.Message) []models.Message
}

// TrimmerConfig 修剪配置
type TrimmerConfig struct {
	MaxTokens  int // 最大token数
	KeepRounds int // 保留最近的轮数
}

// NewMessageTrimmer 创建默认的消息修剪器
func NewMessageTrimmer(config ...TrimmerConfig) MessageTrimmer {
	cfg := TrimmerConfig{
		MaxTokens:  defaultMaxTokens,
		KeepRounds: defaultKeepRounds,
	}
	if len(config) > 0 {
		if config[0].MaxTokens > 0 {
			cfg.MaxTokens = config[0].MaxTokens
		}
		if config[0].KeepRounds > 0 {
			cfg.KeepRounds = config[0].KeepRounds
		}
	}
	return &defaultMessageTrimmer{
		config: cfg,
	}
}

type defaultMessageTrimmer struct {
	config TrimmerConfig
}

func (t *defaultMessageTrimmer) Trim(messages []models.Message) []models.Message {
	if len(messages) == 0 {
		return messages
	}

	// 1. 先按轮数修剪，保留最近N轮
	trimmed := t.trimByRounds(messages)
	if len(trimmed) <= 2 { // 至少保留系统提示和最新一轮
		return trimmed
	}

	// 2. 再按token数修剪，如果还是超过阈值
	totalTokens := t.countTotalTokens(trimmed)
	if totalTokens <= t.config.MaxTokens {
		return trimmed
	}

	// 3. 循环删除最早的非系统消息，直到token数低于阈值
	for totalTokens > t.config.MaxTokens && len(trimmed) > 2 {
		// 找到第一个非系统消息
		for i := 1; i < len(trimmed); i++ {
			if trimmed[i].Role != models.RoleSystem {
				// 删除这条消息和对应的回复（如果是用户消息，下一条通常是AI回复）
				removeCount := 1
				if i+1 < len(trimmed) && trimmed[i+1].Role == models.RoleAI {
					removeCount = 2
				}
				trimmed = append(trimmed[:i], trimmed[i+removeCount:]...)
				break
			}
		}
		totalTokens = t.countTotalTokens(trimmed)
	}

	return trimmed
}

// trimByRounds 按轮数修剪，保留最近N轮对话
func (t *defaultMessageTrimmer) trimByRounds(messages []models.Message) []models.Message {
	if len(messages) == 0 {
		return messages
	}

	// 系统消息总是保留
	var systemMsg *models.Message
	if messages[0].Role == models.RoleSystem {
		systemMsg = &messages[0]
		messages = messages[1:]
	}

	// 计算对话轮数（用户消息+AI回复算一轮）
	rounds := 0
	startIdx := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.RoleHuman {
			rounds++
			if rounds >= t.config.KeepRounds {
				startIdx = i
				break
			}
		}
	}

	// 保留最近的轮数
	trimmed := messages[startIdx:]

	// 加回系统消息
	if systemMsg != nil {
		trimmed = append([]models.Message{*systemMsg}, trimmed...)
	}

	return trimmed
}

// countTotalTokens 估算所有消息的总token数
func (t *defaultMessageTrimmer) countTotalTokens(messages []models.Message) int {
	total := 0
	for _, msg := range messages {
		total += t.countMessageTokens(msg)
	}
	return total
}

// countMessageTokens 估算单条消息的token数
func (t *defaultMessageTrimmer) countMessageTokens(msg models.Message) int {
	chars := 0
	// 统计内容字符数
	chars += utf8.RuneCountInString(msg.Content)
	// 统计工具调用字符数
	for _, call := range msg.ToolCalls {
		chars += utf8.RuneCountInString(call.Name)
		for k, v := range call.Arguments {
			chars += utf8.RuneCountInString(k)
			chars += utf8.RuneCountInString(stringAnyToString(v))
		}
	}
	// 统计工具结果字符数
	if msg.ToolResult != nil {
		chars += utf8.RuneCountInString(msg.ToolResult.Content)
		chars += utf8.RuneCountInString(msg.ToolResult.Error)
	}
	// 转换为token数
	return chars / charsPerToken
}

// stringAnyToString 把任意类型转换为字符串用于长度统计
func stringAnyToString(v any) string {
	return fmt.Sprintf("%v", v)
}
