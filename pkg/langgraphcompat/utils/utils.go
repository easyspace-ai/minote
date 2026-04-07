package utils

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)
// Constants for skill categories.
const (
	SkillCategoryPublic = "public"
	SkillCategoryCustom = "custom"
)


var (
	// AgentNameRE validates agent names.
	AgentNameRE = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	// ThreadIDRE validates thread IDs.
	ThreadIDRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	// SkillFrontmatterNameRE validates skill names in frontmatter.
	SkillFrontmatterNameRE = regexp.MustCompile(`^[a-z0-9-]+$`)
	// WindowsAbsolutePathRE detects Windows absolute paths.
	WindowsAbsolutePathRE = regexp.MustCompile(`^[A-Za-z]:[\\/].*`)
	// ArtifactVirtualPathRE extracts virtual paths from content.
	ArtifactVirtualPathRE = regexp.MustCompile(`(?i)/mnt/user-data/(?:uploads|outputs|workspace)/[^<>"')\]\r\n\t]+`)
	// SuggestionBulletRE matches bullet point prefixes.
	SuggestionBulletRE = regexp.MustCompile(`^(?:[-*•]|\d+[.)])\s+`)
)

// ActiveContentMIMETypes lists MIME types considered as active content.
var ActiveContentMIMETypes = map[string]struct{}{
	"text/html":             {},
	"application/xhtml+xml": {},
	"image/svg+xml":         {},
}

// NowUnix returns the current Unix timestamp.
func NowUnix() int64 {
	return time.Now().UTC().Unix()
}

// ToInt64 converts a value to int64.
func ToInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	default:
		return 0
	}
}

// AsString converts a value to string.
func AsString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return ""
	}
}

// SanitizePathFilename sanitizes a filename for path use.
func SanitizePathFilename(name string) string {
	// Remove path separators and null bytes
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\x00", "")

	// Trim spaces and dots
	name = strings.TrimSpace(name)
	name = strings.Trim(name, ".")

	// Limit length
	if len(name) > 255 {
		name = name[:255]
	}

	if name == "" {
		return "unnamed"
	}
	return name
}

// ValidateUploadedFilename validates and sanitizes an uploaded filename.
func ValidateUploadedFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrInvalidFilename
	}

	// Check for path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "//") {
		return "", ErrPathTraversal
	}

	// Sanitize the filename
	name = SanitizePathFilename(name)

	return name, nil
}

// CompactSubject compacts text into a subject line.
func CompactSubject(text string) string {
	// Replace newlines with spaces
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	// Collapse multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Trim and limit length
	text = strings.TrimSpace(text)
	if len(text) > 100 {
		// Try to break at word boundary
		idx := strings.LastIndex(text[:100], " ")
		if idx > 80 {
			text = text[:idx] + "..."
		} else {
			text = text[:100] + "..."
		}
	}

	return text
}

// NormalizeSuggestionText normalizes suggestion text.
func NormalizeSuggestionText(text string) string {
	text = strings.TrimSpace(text)

	// Remove bullet prefix if present
	text = SuggestionBulletRE.ReplaceAllString(text, "")

	// Capitalize first letter
	if len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		if r != utf8.RuneError {
			text = string(unicode.ToUpper(r)) + text[size:]
		}
	}

	return text
}

// NormalizeBulletSuggestion normalizes a bullet point suggestion.
func NormalizeBulletSuggestion(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	// Remove bullet prefix
	line = SuggestionBulletRE.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	return line
}

// SanitizeSkillName sanitizes a skill name.
func SanitizeSkillName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))

	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Remove non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	name = result.String()

	// Collapse multiple hyphens
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	// Trim hyphens
	name = strings.Trim(name, "-")

	if name == "" {
		return "skill"
	}
	return name
}

// NormalizeAgentName normalizes an agent name.
func NormalizeAgentName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}

	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")

	// Check valid characters
	if !AgentNameRE.MatchString(name) {
		return "", false
	}

	return name, true
}

// NormalizeSkillCategory normalizes a skill category.
func NormalizeSkillCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	switch category {
	case SkillCategoryPublic, SkillCategoryCustom:
		return category
	default:
		return SkillCategoryCustom
	}
}

// ResolveSkillCategory resolves the skill category with fallback.
func ResolveSkillCategory(category, fallback string) string {
	category = NormalizeSkillCategory(category)
	if category != "" {
		return category
	}
	return NormalizeSkillCategory(fallback)
}

// InferSkillCategory infers category from skill name.
func InferSkillCategory(name string) string {
	name = strings.ToLower(name)
	if strings.HasPrefix(name, "builtin-") || strings.HasPrefix(name, "system-") {
		return SkillCategoryPublic
	}
	return SkillCategoryCustom
}

// SkillStorageKey creates a storage key from category and name.
func SkillStorageKey(category, name string) string {
	category = NormalizeSkillCategory(category)
	name = SanitizeSkillName(name)
	return category + "/" + name
}

// SplitSkillStorageKey splits a storage key into category and name.
func SplitSkillStorageKey(key string) (string, string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return SkillCategoryCustom, key
}

// ParseJSONStringList parses a JSON string into a string list.
func ParseJSONStringList(raw string) []string {
	var result []string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}
