package utils

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

var (
	// 邮箱正则
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	// 手机号正则（中国大陆）
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// URL正则
	urlRegex = regexp.MustCompile(`^https?://`)
)

// IsValidEmail 验证邮箱格式
func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsValidPhone 验证手机号格式（中国大陆）
func IsValidPhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false
	}
	return phoneRegex.MatchString(phone)
}

// IsValidURL 验证URL格式（HTTP/HTTPS）
func IsValidURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// IsPublicIP 验证是否是公网IP
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return false
		}
	}
	return true
}

// IsHostAllowed 验证主机是否允许访问（禁止内网和本地地址）
func IsHostAllowed(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = strings.ToLower(h)
	}
	if host == "localhost" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return IsPublicIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if !IsPublicIP(ip) {
			return false
		}
	}
	return true
}
