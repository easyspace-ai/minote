package types

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Version 语义化版本
type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

// String 返回版本字符串
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseVersion 解析版本字符串
func ParseVersion(s string) (Version, error) {
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(s))
	if matches == nil {
		return Version{}, fmt.Errorf("invalid version format: %s", s)
	}

	var v Version
	fmt.Sscanf(matches[1], "%d", &v.Major)
	fmt.Sscanf(matches[2], "%d", &v.Minor)
	fmt.Sscanf(matches[3], "%d", &v.Patch)
	return v, nil
}

// Compare 比较两个版本
// 返回 -1 如果 v < other, 0 如果相等, 1 如果 v > other
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}
	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}
	return 0
}

// IsCompatibleWith 检查是否与目标版本兼容
func (v Version) IsCompatibleWith(target Version) bool {
	// 主版本号必须相同才兼容
	if v.Major != target.Major {
		return false
	}
	// 当前版本不能低于目标版本
	return v.Compare(target) >= 0
}

// SkillVersionInfo Skill 版本信息
type SkillVersionInfo struct {
	Current    Version   `json:"current"`
	MinServer  Version   `json:"min_server_version"`
	Deprecated []string  `json:"deprecated_features,omitempty"`
	Removed    []string  `json:"removed_features,omitempty"`
	Added      []string  `json:"added_features,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MigrationStep 迁移步骤
type MigrationStep struct {
	FromVersion Version  `json:"from_version"`
	ToVersion   Version  `json:"to_version"`
	Actions     []string `json:"actions"`
	Breaking    bool     `json:"breaking"`
}

// SkillCompatibility Skill 兼容性配置
type SkillCompatibility struct {
	SkillName       string            `json:"skill_name"`
	SkillVersion    Version           `json:"skill_version"`
	MinServerVer    Version           `json:"min_server_version"`
	MaxServerVer    *Version          `json:"max_server_version,omitempty"`
	Deprecated      []DeprecationInfo `json:"deprecated,omitempty"`
	BreakingChanges []BreakingChange  `json:"breaking_changes,omitempty"`
}

// DeprecationInfo 弃用信息
type DeprecationInfo struct {
	Feature     string    `json:"feature"`
	Since       Version   `json:"since"`
	Replacement string    `json:"replacement,omitempty"`
	RemoveAt    *Version  `json:"remove_at,omitempty"`
}

// BreakingChange 破坏性变更
type BreakingChange struct {
	Version Version  `json:"version"`
	Change  string   `json:"change"`
	Impact  string   `json:"impact"`
	Migrate string   `json:"migration_guide"`
}

// VersionChecker 版本检查器
type VersionChecker struct {
	serverVersion Version
}

// NewVersionChecker 创建版本检查器
func NewVersionChecker(serverVersion string) (*VersionChecker, error) {
	v, err := ParseVersion(serverVersion)
	if err != nil {
		return nil, fmt.Errorf("invalid server version: %w", err)
	}
	return &VersionChecker{serverVersion: v}, nil
}

// CheckCompatibility 检查 Skill 兼容性
func (vc *VersionChecker) CheckCompatibility(compat *SkillCompatibility) (*CompatibilityResult, error) {
	if compat == nil {
		return nil, fmt.Errorf("compatibility info is nil")
	}

	result := &CompatibilityResult{
		SkillName:    compat.SkillName,
		SkillVersion: compat.SkillVersion,
		Compatible:   true,
		Issues:       []string{},
		Warnings:     []string{},
	}

	// 检查最低版本要求
	if vc.serverVersion.Compare(compat.MinServerVer) < 0 {
		result.Compatible = false
		result.Issues = append(result.Issues,
			fmt.Sprintf("Server version %s is below minimum required %s",
				vc.serverVersion, compat.MinServerVer))
	}

	// 检查最高版本限制
	if compat.MaxServerVer != nil && vc.serverVersion.Compare(*compat.MaxServerVer) > 0 {
		result.Compatible = false
		result.Issues = append(result.Issues,
			fmt.Sprintf("Server version %s exceeds maximum supported %s",
				vc.serverVersion, *compat.MaxServerVer))
	}

	// 检查弃用功能
	for _, dep := range compat.Deprecated {
		if vc.serverVersion.Compare(dep.Since) >= 0 {
			msg := fmt.Sprintf("Feature '%s' is deprecated since %s", dep.Feature, dep.Since)
			if dep.Replacement != "" {
				msg += fmt.Sprintf(". Use '%s' instead", dep.Replacement)
			}
			if dep.RemoveAt != nil {
				msg += fmt.Sprintf(". Will be removed in %s", *dep.RemoveAt)
			}
			result.Warnings = append(result.Warnings, msg)
		}
	}

	return result, nil
}

// CompatibilityResult 兼容性检查结果
type CompatibilityResult struct {
	SkillName    string   `json:"skill_name"`
	SkillVersion Version  `json:"skill_version"`
	Compatible   bool     `json:"compatible"`
	Issues       []string `json:"issues,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// CanLoad 是否可以加载
func (r *CompatibilityResult) CanLoad() bool {
	return r.Compatible && len(r.Issues) == 0
}

// SkillManifest Skill 清单（扩展 YAML 结构）
type SkillManifest struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Description string            `json:"description" yaml:"description"`
	Compatibility SkillCompatYAML `json:"compatibility,omitempty" yaml:"compatibility,omitempty"`
	Migration   *MigrationYAML    `json:"migration,omitempty" yaml:"migration,omitempty"`
}

// SkillCompatYAML YAML 兼容性配置
type SkillCompatYAML struct {
	MinServerVersion string   `json:"min_server_version" yaml:"min_server_version"`
	MaxServerVersion *string  `json:"max_server_version,omitempty" yaml:"max_server_version,omitempty"`
	Deprecated       []string `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}

// MigrationYAML YAML 迁移配置
type MigrationYAML struct {
	Notes   string            `json:"notes,omitempty" yaml:"notes,omitempty"`
	Guides  map[string]string `json:"guides,omitempty" yaml:"guides,omitempty"`
	AutoMigrate bool          `json:"auto_migrate,omitempty" yaml:"auto_migrate,omitempty"`
}

// ValidateManifest 验证 Skill 清单
func ValidateManifest(manifest *SkillManifest) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}

	if manifest.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	if manifest.Version == "" {
		return fmt.Errorf("skill version is required")
	}

	_, err := ParseVersion(manifest.Version)
	if err != nil {
		return fmt.Errorf("invalid skill version: %w", err)
	}

	if manifest.Compatibility.MinServerVersion != "" {
		_, err = ParseVersion(manifest.Compatibility.MinServerVersion)
		if err != nil {
			return fmt.Errorf("invalid min_server_version: %w", err)
		}
	}

	return nil
}

// ServerVersion 当前服务器版本
// 在构建时通过 -ldflags 注入
var ServerVersion = "2.0.0"

// GetServerVersion 获取服务器版本
func GetServerVersion() Version {
	v, _ := ParseVersion(ServerVersion)
	return v
}
