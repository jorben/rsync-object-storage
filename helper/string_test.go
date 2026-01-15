package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHideSecret 测试隐藏敏感字符串
func TestHideSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		count    uint32
		expected string
	}{
		{"空字符串", "", 6, ""},
		{"短于掩码长度", "abc", 6, "***"},
		{"等于掩码长度", "abcdef", 6, "******"},
		{"长于掩码长度 - 偶数", "abcdefghij", 6, "ab******ij"},
		{"长于掩码长度 - 奇数", "abcdefghi", 6, "ab******i"},
		{"长密钥", "AKID1234567890ABCDEF", 12, "AKID************CDEF"},
		{"单字符", "a", 6, "*"},
		{"两字符", "ab", 6, "**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HideSecret(tt.secret, tt.count)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRandomString 测试随机字符串生成
func TestRandomString(t *testing.T) {
	t.Run("生成指定长度", func(t *testing.T) {
		lengths := []int{0, 1, 10, 32, 64}
		for _, length := range lengths {
			result, err := RandomString(length)
			assert.NoError(t, err)
			assert.Equal(t, length, len(result))
		}
	})

	t.Run("只包含合法字符", func(t *testing.T) {
		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result, err := RandomString(100)
		assert.NoError(t, err)

		for _, char := range result {
			found := false
			for _, validChar := range charset {
				if char == validChar {
					found = true
					break
				}
			}
			assert.True(t, found, "字符 %c 不在合法字符集中", char)
		}
	})

	t.Run("每次生成不同", func(t *testing.T) {
		s1, _ := RandomString(32)
		s2, _ := RandomString(32)
		assert.NotEqual(t, s1, s2)
	})
}

// TestStringMd5 测试字符串 MD5 计算
func TestStringMd5(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"空字符串", "", "d41d8cd98f00b204e9800998ecf8427e"},
		{"hello world", "hello world", "5eb63bbbe01eeed093cb22bb8f5acdc3"},
		{"特殊字符", "!@#$%^&*()", "05b28d17a7b6e7024b6e5d8cc43a8bf7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringMd5(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	// 测试中文字符串的 MD5 计算（只验证非空和长度）
	t.Run("中文", func(t *testing.T) {
		result := StringMd5("你好世界")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 32)
	})
}
