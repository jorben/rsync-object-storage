package helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsDir 测试目录判断
func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("是目录", func(t *testing.T) {
		result, err := IsDir(tmpDir)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("是文件", func(t *testing.T) {
		file := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(file, []byte("content"), 0644)
		assert.NoError(t, err)

		result, err := IsDir(file)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("路径不存在", func(t *testing.T) {
		_, err := IsDir(filepath.Join(tmpDir, "nonexistent"))
		assert.Error(t, err)
	})
}

// TestIsExist 测试路径存在判断
func TestIsExist(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("路径存在", func(t *testing.T) {
		result, err := IsExist(tmpDir)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("路径不存在", func(t *testing.T) {
		result, err := IsExist(filepath.Join(tmpDir, "nonexistent"))
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

// TestIsSymlink 测试符号链接判断
func TestIsSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("是符号链接", func(t *testing.T) {
		target := filepath.Join(tmpDir, "target.txt")
		err := os.WriteFile(target, []byte("content"), 0644)
		assert.NoError(t, err)

		link := filepath.Join(tmpDir, "link.txt")
		err = os.Symlink(target, link)
		assert.NoError(t, err)

		result, err := IsSymlink(link)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("不是符号链接", func(t *testing.T) {
		file := filepath.Join(tmpDir, "regular.txt")
		err := os.WriteFile(file, []byte("content"), 0644)
		assert.NoError(t, err)

		result, err := IsSymlink(file)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("目录不是符号链接", func(t *testing.T) {
		result, err := IsSymlink(tmpDir)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

// TestIsDirEmpty 测试目录是否为空
func TestIsDirEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("空目录", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		err := os.Mkdir(emptyDir, 0755)
		assert.NoError(t, err)

		result, err := IsDirEmpty(emptyDir)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("非空目录", func(t *testing.T) {
		nonEmptyDir := filepath.Join(tmpDir, "nonempty")
		err := os.Mkdir(nonEmptyDir, 0755)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"), 0644)
		assert.NoError(t, err)

		result, err := IsDirEmpty(nonEmptyDir)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("路径不存在", func(t *testing.T) {
		_, err := IsDirEmpty(filepath.Join(tmpDir, "nonexistent"))
		assert.Error(t, err)
	})
}

// TestGetSymlinkTarget 测试获取符号链接目标
func TestGetSymlinkTarget(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("获取符号链接目标", func(t *testing.T) {
		target := filepath.Join(tmpDir, "target.txt")
		err := os.WriteFile(target, []byte("content"), 0644)
		assert.NoError(t, err)

		link := filepath.Join(tmpDir, "link.txt")
		err = os.Symlink(target, link)
		assert.NoError(t, err)

		result, err := GetSymlinkTarget(link)
		assert.NoError(t, err)
		assert.Equal(t, target, result)
	})

	t.Run("非符号链接返回错误", func(t *testing.T) {
		file := filepath.Join(tmpDir, "regular.txt")
		err := os.WriteFile(file, []byte("content"), 0644)
		assert.NoError(t, err)

		_, err = GetSymlinkTarget(file)
		assert.Error(t, err)
	})
}

// TestIsIgnore 测试忽略规则
func TestIsIgnore(t *testing.T) {
	ignoreList := []string{".git", "node_modules", "*.log", ".DS_Store"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"匹配目录名", "/project/.git/config", true},
		{"匹配目录结尾", "/project/.git", true},
		{"匹配 node_modules", "/project/node_modules/package.json", true},
		{"匹配通配符", "/project/debug.log", true},
		{"匹配精确文件名", "/project/.DS_Store", true},
		{"不匹配", "/project/src/main.go", false},
		{"部分名称不匹配", "/project/git/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIgnore(tt.path, ignoreList)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCopy 测试文件拷贝
func TestCopy(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("成功拷贝文件", func(t *testing.T) {
		src := filepath.Join(tmpDir, "source.txt")
		content := []byte("hello world")
		err := os.WriteFile(src, content, 0644)
		assert.NoError(t, err)

		dst := filepath.Join(tmpDir, "dest.txt")
		written, err := Copy(src, dst)

		assert.NoError(t, err)
		assert.Equal(t, int64(len(content)), written)

		// 验证内容一致
		dstContent, err := os.ReadFile(dst)
		assert.NoError(t, err)
		assert.Equal(t, content, dstContent)
	})

	t.Run("源文件不存在", func(t *testing.T) {
		src := filepath.Join(tmpDir, "nonexistent.txt")
		dst := filepath.Join(tmpDir, "dest2.txt")
		_, err := Copy(src, dst)
		assert.Error(t, err)
	})
}

// TestFileMd5 测试文件 MD5 计算
func TestFileMd5(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("计算文件 MD5", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(file, []byte("hello world"), 0644)
		assert.NoError(t, err)

		md5, err := FileMd5(file)
		assert.NoError(t, err)
		// "hello world" 的 MD5
		assert.Equal(t, "5eb63bbbe01eeed093cb22bb8f5acdc3", md5)
	})

	t.Run("空文件 MD5", func(t *testing.T) {
		file := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(file, []byte{}, 0644)
		assert.NoError(t, err)

		md5, err := FileMd5(file)
		assert.NoError(t, err)
		// 空文件的 MD5
		assert.Equal(t, "d41d8cd98f00b204e9800998ecf8427e", md5)
	})

	t.Run("文件不存在", func(t *testing.T) {
		_, err := FileMd5(filepath.Join(tmpDir, "nonexistent.txt"))
		assert.Error(t, err)
	})
}

// TestFileSha256 测试文件 SHA256 计算
func TestFileSha256(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("计算文件 SHA256", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(file, []byte("hello world"), 0644)
		assert.NoError(t, err)

		sha256, err := FileSha256(file)
		assert.NoError(t, err)
		// "hello world" 的 SHA256
		assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", sha256)
	})

	t.Run("文件不存在", func(t *testing.T) {
		_, err := FileSha256(filepath.Join(tmpDir, "nonexistent.txt"))
		assert.Error(t, err)
	})
}

// TestByteFormat 测试字节格式化
func TestByteFormat(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"字节", 500, "500 B"},
		{"KB", 1024, "1.0 kB"},
		{"KB 带小数", 1536, "1.5 kB"},
		{"MB", 1048576, "1.0 MB"},
		{"GB", 1073741824, "1.0 GB"},
		{"TB", 1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByteFormat(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
