package helper

import (
	"path/filepath"
	"strings"
)

// IgnoreMatcher 预编译的忽略规则匹配器
// 使用 Map 存储实现 O(1) 查找，提升大量规则场景下的匹配效率
type IgnoreMatcher struct {
	dirPatterns  map[string]struct{} // 目录模式集合（去除首尾斜杠）
	filePatterns map[string]struct{} // 文件名精确匹配
	globPatterns []string            // 通配符模式（需要遍历匹配）
}

// NewIgnoreMatcher 创建预编译的忽略规则匹配器
func NewIgnoreMatcher(ignoreList []string) *IgnoreMatcher {
	m := &IgnoreMatcher{
		dirPatterns:  make(map[string]struct{}),
		filePatterns: make(map[string]struct{}),
		globPatterns: make([]string, 0),
	}

	for _, ignore := range ignoreList {
		ignore = strings.TrimSpace(ignore)
		if ignore == "" {
			continue
		}

		// 检查是否包含通配符
		if strings.ContainsAny(ignore, "*?[]") {
			m.globPatterns = append(m.globPatterns, ignore)
		} else {
			// 同时作为目录模式和文件名精确匹配
			trimmed := strings.Trim(ignore, "/")
			m.dirPatterns[trimmed] = struct{}{}
			m.filePatterns[ignore] = struct{}{}
		}
	}

	return m
}

// Match 检查路径是否匹配忽略规则
// 返回 true 表示应该忽略该路径
func (m *IgnoreMatcher) Match(path string) bool {
	if m == nil {
		return false
	}

	// 1. 检查目录模式：路径中包含 /pattern/ 或以 /pattern 结尾
	for pattern := range m.dirPatterns {
		dirPattern := "/" + pattern + "/"
		if strings.Contains(path, dirPattern) {
			return true
		}
		suffixPattern := "/" + pattern
		if strings.HasSuffix(path, suffixPattern) {
			return true
		}
	}

	// 2. 检查文件名精确匹配
	base := filepath.Base(path)
	if _, ok := m.filePatterns[base]; ok {
		return true
	}

	// 3. 检查通配符模式
	for _, pattern := range m.globPatterns {
		if match, _ := filepath.Match(pattern, base); match {
			return true
		}
	}

	return false
}

// IsEmpty 检查匹配器是否为空（无任何规则）
func (m *IgnoreMatcher) IsEmpty() bool {
	if m == nil {
		return true
	}
	return len(m.dirPatterns) == 0 && len(m.filePatterns) == 0 && len(m.globPatterns) == 0
}
