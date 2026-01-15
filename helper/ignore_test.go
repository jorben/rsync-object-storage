package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewIgnoreMatcher 测试创建忽略匹配器
func TestNewIgnoreMatcher(t *testing.T) {
	t.Run("空规则列表", func(t *testing.T) {
		m := NewIgnoreMatcher(nil)
		assert.NotNil(t, m)
		assert.True(t, m.IsEmpty())
	})

	t.Run("混合规则", func(t *testing.T) {
		rules := []string{".git", "*.log", "node_modules", "  ", ""}
		m := NewIgnoreMatcher(rules)
		assert.NotNil(t, m)
		assert.False(t, m.IsEmpty())
	})
}

// TestIgnoreMatcher_Match 测试忽略匹配器的匹配功能
func TestIgnoreMatcher_Match(t *testing.T) {
	rules := []string{".git", "node_modules", "*.log", ".DS_Store", "temp"}
	m := NewIgnoreMatcher(rules)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// 目录模式匹配
		{"匹配 .git 目录中的文件", "/project/.git/config", true},
		{"匹配 .git 目录结尾", "/project/.git", true},
		{"匹配 node_modules 中的文件", "/project/node_modules/package/index.js", true},
		{"匹配嵌套 node_modules", "/project/packages/a/node_modules/pkg", true},

		// 文件名精确匹配
		{"匹配 .DS_Store", "/project/.DS_Store", true},
		{"匹配 temp 文件", "/project/temp", true},
		{"匹配嵌套 temp 目录", "/project/subdir/temp/file.txt", true},

		// 通配符匹配
		{"匹配 .log 文件", "/project/app.log", true},
		{"匹配嵌套 .log 文件", "/project/logs/debug.log", true},

		// 不匹配的情况
		{"不匹配普通文件", "/project/src/main.go", false},
		{"不匹配部分名称 git", "/project/git/file.txt", false},
		{"不匹配 .gitignore", "/project/.gitignore", false},
		{"不匹配 .log 结尾但不是扩展名", "/project/catalog", false},
		{"不匹配包含 temp 的文件名", "/project/temperature.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Match(tt.path)
			assert.Equal(t, tt.expected, result, "path: %s", tt.path)
		})
	}
}

// TestIgnoreMatcher_Match_NilMatcher 测试 nil 匹配器
func TestIgnoreMatcher_Match_NilMatcher(t *testing.T) {
	var m *IgnoreMatcher
	result := m.Match("/any/path")
	assert.False(t, result)
}

// TestIgnoreMatcher_IsEmpty 测试 IsEmpty 方法
func TestIgnoreMatcher_IsEmpty(t *testing.T) {
	t.Run("nil 匹配器", func(t *testing.T) {
		var m *IgnoreMatcher
		assert.True(t, m.IsEmpty())
	})

	t.Run("空规则匹配器", func(t *testing.T) {
		m := NewIgnoreMatcher([]string{})
		assert.True(t, m.IsEmpty())
	})

	t.Run("只有空白规则", func(t *testing.T) {
		m := NewIgnoreMatcher([]string{"", "  ", "\t"})
		assert.True(t, m.IsEmpty())
	})

	t.Run("有效规则", func(t *testing.T) {
		m := NewIgnoreMatcher([]string{".git"})
		assert.False(t, m.IsEmpty())
	})
}

// TestIgnoreMatcher_GlobPatterns 测试通配符规则
func TestIgnoreMatcher_GlobPatterns(t *testing.T) {
	rules := []string{"*.tmp", "test?.txt", "[abc].log"}
	m := NewIgnoreMatcher(rules)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"匹配 *.tmp", "/project/cache.tmp", true},
		{"匹配 test?.txt", "/project/test1.txt", true},
		{"匹配 [abc].log", "/project/a.log", true},
		{"匹配 [abc].log - b", "/project/b.log", true},
		{"不匹配 [abc].log - d", "/project/d.log", false},
		{"不匹配 test?.txt - 两个字符", "/project/test12.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Match(tt.path)
			assert.Equal(t, tt.expected, result, "path: %s", tt.path)
		})
	}
}

// TestIgnoreMatcher_EdgeCases 测试边界情况
func TestIgnoreMatcher_EdgeCases(t *testing.T) {
	t.Run("规则带有斜杠", func(t *testing.T) {
		m := NewIgnoreMatcher([]string{"/build/", "dist/"})
		assert.True(t, m.Match("/project/build/output.js"))
		assert.True(t, m.Match("/project/dist/bundle.js"))
	})

	t.Run("路径没有斜杠", func(t *testing.T) {
		m := NewIgnoreMatcher([]string{".git"})
		// 没有斜杠的路径，文件名匹配
		assert.True(t, m.Match(".git"))
	})

	t.Run("根目录", func(t *testing.T) {
		m := NewIgnoreMatcher([]string{"root"})
		assert.True(t, m.Match("/root/file.txt"))
	})
}
