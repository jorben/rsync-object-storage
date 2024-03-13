package helper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IsDir 判断路径是否文件夹
func IsDir(path string) (bool, error) {
	if info, err := os.Stat(path); err != nil {
		return false, err
	} else {
		return info.IsDir(), nil
	}
}

// IsIgnore 判断路径或文件是否在忽略配置中
func IsIgnore(path string, ignoreList []string) bool {
	for _, ignore := range ignoreList {
		// 假定忽略配置对象是指文件夹
		if strings.Contains(path, fmt.Sprintf("/%s/", strings.Trim(ignore, "/"))) ||
			strings.HasSuffix(path, fmt.Sprintf("/%s", strings.Trim(ignore, "/"))) {
			return true
		}

		// 假定忽略配置对象是具体的文件名
		base := filepath.Base(path)
		if ignore == base {
			return true
		}

		// 假定忽略配置对象是文件名模式
		if match, _ := filepath.Match(ignore, base); match {
			return true
		}
	}
	return false
}
