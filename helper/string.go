package helper

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"math"
	"math/big"
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

// RandomString 生成指定长度的随机字符串
func RandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// StringMd5 计算字符串的MD5值
func StringMd5(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	// 计算 MD5 校验和
	return hex.EncodeToString(hash.Sum(nil))
}
