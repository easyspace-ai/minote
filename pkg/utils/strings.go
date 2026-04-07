package utils

import (
	"regexp"
	"strings"
)

// TruncateString 截断字符串到指定长度，超过的部分添加省略号
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "..."
}

// SanitizeBody 过滤敏感信息（密码、token等）
func SanitizeBody(body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	patterns := []struct {
		re          *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`"password"\s*:\s*"[^"]*"`), `"password":"***"`},
		{regexp.MustCompile(`"token"\s*:\s*"[^"]*"`), `"token":"***"`},
		{regexp.MustCompile(`"access_token"\s*:\s*"[^"]*"`), `"access_token":"***"`},
		{regexp.MustCompile(`"refresh_token"\s*:\s*"[^"]*"`), `"refresh_token":"***"`},
		{regexp.MustCompile(`"authorization"\s*:\s*"[^"]*"`), `"authorization":"***"`},
		{regexp.MustCompile(`"api_key"\s*:\s*"[^"]*"`), `"api_key":"***"`},
		{regexp.MustCompile(`"secret"\s*:\s*"[^"]*"`), `"secret":"***"`},
	}
	out := body
	for _, p := range patterns {
		out = p.re.ReplaceAllString(out, p.replacement)
	}
	return out
}

// SanitizeHTML 清理HTML，防止XSS攻击
func SanitizeHTML(s string) string {
	// 基础的HTML清理，可以根据需要扩展
	replacer := strings.NewReplacer(
		"<script>", "&lt;script&gt;",
		"</script>", "&lt;/script&gt;",
		"<iframe>", "&lt;iframe&gt;",
		"</iframe>", "&lt;/iframe&gt;",
		"javascript:", "javascript&#58;",
		"onload=", "onload&#61;",
		"onerror=", "onerror&#61;",
		"onclick=", "onclick&#61;",
	)
	return replacer.Replace(s)
}

// IsEmptyString 判断字符串是否为空（包括只包含空白字符）
func IsEmptyString(s string) bool {
	return strings.TrimSpace(s) == ""
}

// PointerToString 安全获取指针字符串的值
func PointerToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
