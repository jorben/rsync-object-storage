package helper

import (
	"math"
	"strings"
)

// HideSecret 隐藏字符串的中间字符
func HideSecret(secret string, count uint32) string {
	length := len(secret)
	if 0 == length {
		return ""
	}
	mask := strings.Repeat("*", int(count))
	if length <= int(count) {
		return mask[0:length]
	}

	prefix := math.Ceil(float64(length-int(count)) / 2)
	suffix := math.Floor(float64(length-int(count)) / 2)

	return secret[0:int(prefix)] + mask + secret[length-int(suffix):]
}