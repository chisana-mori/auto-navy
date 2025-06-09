package es

import (
	"strings"
)

// splitAndTrim 分割字符串并去除空白
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// safePercentage 安全计算百分比，避免除零错误
func safePercentage(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return (numerator / denominator) * 100
}
