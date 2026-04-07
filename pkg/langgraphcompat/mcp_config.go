package langgraphcompat

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// mcpServerYAMLConfig represents a single MCP server from config.yaml.
type mcpServerYAMLConfig struct {
	Type        string            `yaml:"type"`
	Enabled     *bool             `yaml:"enabled"`
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args"`
	Env         map[string]string `yaml:"env"`
	URL         string            `yaml:"url"`
	Headers     map[string]string `yaml:"headers"`
	Description string            `yaml:"description"`
}

// loadMCPServersFromConfig loads MCP server configurations from config.yaml
// and merges them into the server's MCP config. Config file entries are
// treated as defaults; persisted/extensions_config entries take precedence.
func (s *Server) loadMCPServersFromConfig(cfg *gatewayConfigFile) {
	yamlServers := loadMCPServersFromYAML()
	if len(yamlServers) == 0 {
		return
	}

	s.uiStateMu.Lock()
	defer s.uiStateMu.Unlock()

	if s.mcpConfig.MCPServers == nil {
		s.mcpConfig.MCPServers = make(map[string]gatewayMCPServerConfig)
	}

	for name, yamlCfg := range yamlServers {
		// Don't overwrite servers already loaded from persisted state or extensions_config
		if _, exists := s.mcpConfig.MCPServers[name]; exists {
			continue
		}

		enabled := true
		if yamlCfg.Enabled != nil {
			enabled = *yamlCfg.Enabled
		}

		serverType := strings.TrimSpace(yamlCfg.Type)
		if serverType == "" {
			if yamlCfg.URL != "" {
				serverType = "sse"
			} else if yamlCfg.Command != "" {
				serverType = "stdio"
			}
		}

		s.mcpConfig.MCPServers[name] = gatewayMCPServerConfig{
			Type:        serverType,
			Enabled:     enabled,
			Command:     yamlCfg.Command,
			Args:        yamlCfg.Args,
			Env:         yamlCfg.Env,
			URL:         yamlCfg.URL,
			Headers:     yamlCfg.Headers,
			Description: yamlCfg.Description,
		}
	}
}

func loadMCPServersFromYAML() map[string]mcpServerYAMLConfig {
	configPath, ok := resolveGatewayConfigPath()
	if !ok {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	resolved := resolveConfigEnvVars(string(data))

	var raw struct {
		MCPServers map[string]mcpServerYAMLConfig `yaml:"mcp_servers"`
	}
	if err := yaml.Unmarshal([]byte(resolved), &raw); err != nil {
		return nil
	}
	return raw.MCPServers
}
