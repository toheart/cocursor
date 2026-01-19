package rag

import (
	"time"
)

// ParseScanInterval 解析扫描间隔字符串
func ParseScanInterval(interval string) time.Duration {
	switch interval {
	case "30m":
		return 30 * time.Minute
	case "1h":
		return 1 * time.Hour
	case "2h":
		return 2 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "manual":
		return 0 // 手动模式，不自动扫描
	default:
		return 1 * time.Hour // 默认 1 小时
	}
}
