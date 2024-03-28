package helper

import (
	"crypto/md5"
	"crypto/sha256"
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

// IsSymlink 判断路径是否符号链接
func IsSymlink(path string) (bool, error) {
	if info, err := os.Lstat(path); err != nil {
		return false, err
	} else if os.ModeSymlink == (info.Mode() & os.ModeSymlink) {
		return true, nil
	}
	return false, nil
}

// IsDirEmpty 判断文件夹是否为空
func IsDirEmpty(dirPath string) (bool, error) {
	if entries, err := os.ReadDir(dirPath); err == nil {
		return len(entries) == 0, nil
	} else {
		return false, err
	}
}

// GetSymlinkTarget 获取符号链接指向的目标
func GetSymlinkTarget(path string) (string, error) {
	if target, err := os.Readlink(path); err != nil {
		return "", err
	} else {
		return target, nil
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

// Copy 拷贝文件
func Copy(src, dst string) (written int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	if written, err = io.Copy(dstFile, srcFile); err != nil {
		return 0, err
	}
	return written, nil
}

// FileMd5 计算文件的MD5
func FileMd5(path string) (string, error) {
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

// FileSha256 计算文件的SHA256
func FileSha256(path string) (string, error) {
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 创建一个新的SHA256哈希计算器
	hash := sha256.New()

	// 读取文件内容并更新哈希计算器
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 计算最终的哈希值
	sum := hash.Sum(nil)

	// 将哈希值转换为十六进制字符串
	return fmt.Sprintf("%x", sum), nil
}

// ByteFormat 将字节为单位的大小转换为易读的字符串格式
func ByteFormat(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b) // 小于1KB直接以B为单位
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
