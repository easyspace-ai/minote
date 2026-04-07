package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// DynamicAgentConfigLoader 动态 Agent 配置加载器
type DynamicAgentConfigLoader struct {
	mu       sync.RWMutex
	configs  map[AgentType]AgentTypeConfig
	sources  map[AgentType]string // 记录配置来源（文件路径）
	loaded   bool
}

// NewDynamicAgentConfigLoader 创建新的加载器
func NewDynamicAgentConfigLoader() *DynamicAgentConfigLoader {
	return &DynamicAgentConfigLoader{
		configs: make(map[AgentType]AgentTypeConfig),
	sources: make(map[AgentType]string),
	}
}

// GlobalDynamicLoader 全局动态加载器实例
var GlobalDynamicLoader = NewDynamicAgentConfigLoader()

// AgentConfigFile 配置文件结构
type AgentConfigFile struct {
	Agents []AgentTypeConfigYAML `json:"agents" yaml:"agents"`
}

// AgentTypeConfigYAML YAML 格式的 Agent 配置
type AgentTypeConfigYAML struct {
	Type         string   `json:"type" yaml:"type"`
	Name         string   `json:"name" yaml:"name"`
	Description  string   `json:"description" yaml:"description"`
	SystemPrompt string   `json:"system_prompt" yaml:"system_prompt"`
	DefaultTools []string `json:"default_tools,omitempty" yaml:"default_tools,omitempty"`
	MaxTurns     int      `json:"max_turns" yaml:"max_turns"`
	Temperature  float64  `json:"temperature" yaml:"temperature"`
	Extends      string   `json:"extends,omitempty" yaml:"extends,omitempty"` // 继承的基础类型
}

// LoadFromDirectory 从目录加载所有配置文件
func (l *DynamicAgentConfigLoader) LoadFromDirectory(dir string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 首先加载内置配置作为基础
	for k, v := range BuiltinAgentTypes {
		l.configs[k] = v
		l.sources[k] = "builtin"
	}

	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 目录不存在，只使用内置配置
		l.loaded = true
		return nil
	}

	// 遍历目录中的所有配置文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read config directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".json") {
			continue
		}

		path := filepath.Join(dir, name)
		if err := l.loadFile(path); err != nil {
			// 记录错误但继续加载其他文件
			fmt.Fprintf(os.Stderr, "Warning: failed to load agent config from %s: %v\n", path, err)
		}
	}

	l.loaded = true
	return nil
}

// LoadFromFile 从单个文件加载配置
func (l *DynamicAgentConfigLoader) LoadFromFile(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.loadFile(path)
}

// loadFile 加载单个配置文件（内部方法，需要在外部持有锁）
func (l *DynamicAgentConfigLoader) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var configFile AgentConfigFile
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &configFile)
	case ".json":
		err = json.Unmarshal(data, &configFile)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 处理每个 Agent 配置
	for _, yamlConfig := range configFile.Agents {
		config := l.convertFromYAML(yamlConfig)
		agentType := AgentType(strings.TrimSpace(string(config.Type)))

		// 如果指定了继承，合并基础配置
		if yamlConfig.Extends != "" {
			baseType := AgentType(yamlConfig.Extends)
			baseConfig, exists := l.configs[baseType]
			if !exists {
				// 尝试从内置配置查找
				baseConfig, exists = BuiltinAgentTypes[baseType]
			}
			if exists {
				config = l.mergeWithBase(baseConfig, config)
			}
		}

		l.configs[agentType] = config
		l.sources[agentType] = path
	}

	return nil
}

// convertFromYAML 将 YAML 配置转换为标准配置
func (l *DynamicAgentConfigLoader) convertFromYAML(yaml AgentTypeConfigYAML) AgentTypeConfig {
	config := AgentTypeConfig{
		Type:         AgentType(yaml.Type),
		Name:         yaml.Name,
		Description:  yaml.Description,
		SystemPrompt: yaml.SystemPrompt,
		DefaultTools: yaml.DefaultTools,
		MaxTurns:     yaml.MaxTurns,
		Temperature:  yaml.Temperature,
	}

	// 设置默认值
	if config.Name == "" {
		config.Name = string(config.Type)
	}
	if config.MaxTurns <= 0 {
		config.MaxTurns = defaultMaxTurns
	}
	if config.Temperature < 0 || config.Temperature > 2 {
		config.Temperature = 0.2
	}

	return config
}

// mergeWithBase 将自定义配置与基础配置合并
func (l *DynamicAgentConfigLoader) mergeWithBase(base, override AgentTypeConfig) AgentTypeConfig {
	result := base

	// 覆盖类型（必须匹配）
	if override.Type != "" {
		result.Type = override.Type
	}

	// 覆盖名称
	if override.Name != "" {
		result.Name = override.Name
	}

	// 覆盖描述
	if override.Description != "" {
		result.Description = override.Description
	}

	// 覆盖系统提示词（如果提供了新的，添加到基础提示词后）
	if override.SystemPrompt != "" {
		if base.SystemPrompt != "" {
			result.SystemPrompt = base.SystemPrompt + "\n\nAdditional instructions:\n" + override.SystemPrompt
		} else {
			result.SystemPrompt = override.SystemPrompt
		}
	}

	// 覆盖默认工具（完全替换）
	if len(override.DefaultTools) > 0 {
		result.DefaultTools = override.DefaultTools
	}

	// 覆盖最大轮数
	if override.MaxTurns > 0 {
		result.MaxTurns = override.MaxTurns
	}

	// 覆盖温度
	if override.Temperature >= 0 && override.Temperature <= 2 {
		result.Temperature = override.Temperature
	}

	return result
}

// GetConfig 获取指定类型的配置
func (l *DynamicAgentConfigLoader) GetConfig(t AgentType) (AgentTypeConfig, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	config, exists := l.configs[t]
	return config, exists
}

// GetAllConfigs 获取所有配置
func (l *DynamicAgentConfigLoader) GetAllConfigs() map[AgentType]AgentTypeConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回副本
	result := make(map[AgentType]AgentTypeConfig, len(l.configs))
	for k, v := range l.configs {
		result[k] = v
	}
	return result
}

// GetSource 获取配置来源
func (l *DynamicAgentConfigLoader) GetSource(t AgentType) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.sources[t]
}

// RegisterConfig 动态注册配置
func (l *DynamicAgentConfigLoader) RegisterConfig(config AgentTypeConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if config.Type == "" {
		return fmt.Errorf("agent type is required")
	}

	l.configs[config.Type] = config
	l.sources[config.Type] = "dynamic"
	return nil
}

// RemoveConfig 移除配置（不能移除内置配置）
func (l *DynamicAgentConfigLoader) RemoveConfig(t AgentType) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	source, exists := l.sources[t]
	if !exists {
		return fmt.Errorf("agent type %s not found", t)
	}

	if source == "builtin" {
		return fmt.Errorf("cannot remove builtin agent type %s", t)
	}

	delete(l.configs, t)
	delete(l.sources, t)
	return nil
}

// IsLoaded 检查是否已加载
func (l *DynamicAgentConfigLoader) IsLoaded() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded
}

// ListCustomTypes 列出所有自定义类型
func (l *DynamicAgentConfigLoader) ListCustomTypes() []AgentType {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var result []AgentType
	for t, source := range l.sources {
		if source != "builtin" {
			result = append(result, t)
		}
	}
	return result
}

// InitializeAgentTypes 初始化 Agent 类型系统
func InitializeAgentTypes(configDir string) error {
	if GlobalDynamicLoader.IsLoaded() {
		return nil // 已初始化
	}

	return GlobalDynamicLoader.LoadFromDirectory(configDir)
}

// GetAgentTypeConfig 获取 Agent 类型配置（优先使用动态加载器）
func GetAgentTypeConfigDynamic(t AgentType) AgentTypeConfig {
	// 首先尝试从动态加载器获取
	if GlobalDynamicLoader.IsLoaded() {
		if config, exists := GlobalDynamicLoader.GetConfig(t); exists {
			return config
		}
	}

	// 回退到内置配置
	return GetAgentTypeConfig(t)
}

// BuiltinAgentTypesList 返回所有内置 Agent 类型列表
func BuiltinAgentTypesList() []AgentType {
	return []AgentType{
		AgentTypeGeneral,
		AgentTypeResearch,
		AgentTypeCoder,
		AgentTypeAnalyst,
	}
}
