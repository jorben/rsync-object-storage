package helper

import "os"

// IsDir 判断路径是否文件夹
func IsDir(path string) (bool, error) {
	if info, err := os.Stat(path); err != nil {
		return false, err
	} else {
		return info.IsDir(), nil
	}
}
