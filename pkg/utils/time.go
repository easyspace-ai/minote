package utils

import "time"

const (
	// RFC3339Milli 带毫秒的RFC3339格式
	RFC3339Milli = "2006-01-02T15:04:05.000Z07:00"
)

// NowRFC3339 获取当前时间的RFC3339格式字符串
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// NowRFC3339Milli 获取当前时间的带毫秒RFC3339格式字符串
func NowRFC3339Milli() string {
	return time.Now().UTC().Format(RFC3339Milli)
}

// ParseRFC3339 解析RFC3339格式的时间字符串
func ParseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// Days 转换天数为时间 duration
func Days(n int) time.Duration {
	return time.Duration(n) * 24 * time.Hour
}

// Hours 转换小时数为时间 duration
func Hours(n int) time.Duration {
	return time.Duration(n) * time.Hour
}

// Minutes 转换分钟数为时间 duration
func Minutes(n int) time.Duration {
	return time.Duration(n) * time.Minute
}
