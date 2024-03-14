package helper

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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

// IsExist 判断路径是否存在
func IsExist(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
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

// Md5 计算文件的MD5
func Md5(path string) (string, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 创建一个新的 MD5 hash 实例
	hash := md5.New()

	// 将文件内容写入 hash
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 计算 MD5 校验和
	hashInBytes := hash.Sum(nil)
	md5Checksum := hex.EncodeToString(hashInBytes)

	return md5Checksum, nil
}
