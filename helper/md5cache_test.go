package helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetCachedFileMd5 测试带缓存的 MD5 计算
func TestGetCachedFileMd5(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("首次计算", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test1.txt")
		err := os.WriteFile(file, []byte("hello"), 0644)
		assert.NoError(t, err)

		md5, err := GetCachedFileMd5(file)
		assert.NoError(t, err)
		// "hello" 的 MD5
		assert.Equal(t, "5d41402abc4b2a76b9719d911017c592", md5)
	})

	t.Run("缓存命中", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test2.txt")
		err := os.WriteFile(file, []byte("world"), 0644)
		assert.NoError(t, err)

		// 第一次计算
		md5_1, err := GetCachedFileMd5(file)
		assert.NoError(t, err)

		// 第二次应该从缓存获取
		md5_2, err := GetCachedFileMd5(file)
		assert.NoError(t, err)

		assert.Equal(t, md5_1, md5_2)
	})

	t.Run("文件修改后重新计算", func(t *testing.T) {
		file := filepath.Join(tmpDir, "test3.txt")
		err := os.WriteFile(file, []byte("original"), 0644)
		assert.NoError(t, err)

		// 第一次计算
		md5_1, err := GetCachedFileMd5(file)
		assert.NoError(t, err)

		// 等待一小段时间确保时间戳变化
		time.Sleep(10 * time.Millisecond)

		// 修改文件
		err = os.WriteFile(file, []byte("modified"), 0644)
		assert.NoError(t, err)

		// 应该重新计算
		md5_2, err := GetCachedFileMd5(file)
		assert.NoError(t, err)

		assert.NotEqual(t, md5_1, md5_2)
	})

	t.Run("文件不存在", func(t *testing.T) {
		_, err := GetCachedFileMd5(filepath.Join(tmpDir, "nonexistent.txt"))
		assert.Error(t, err)
	})
}

// TestInvalidateMd5Cache 测试缓存失效
func TestInvalidateMd5Cache(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(file, []byte("content"), 0644)
	assert.NoError(t, err)

	// 首次计算
	_, err = GetCachedFileMd5(file)
	assert.NoError(t, err)

	// 使缓存失效
	InvalidateMd5Cache(file)

	// 再次获取应该重新计算
	md5, err := GetCachedFileMd5(file)
	assert.NoError(t, err)
	// "content" 的 MD5
	assert.Equal(t, "9a0364b9e99bb480dd25e1f0284c8555", md5)
}

// TestMD5Cache_GetMd5 测试 MD5Cache 的 GetMd5 方法
func TestMD5Cache_GetMd5(t *testing.T) {
	cache := &MD5Cache{}
	tmpDir := t.TempDir()

	t.Run("计算并缓存", func(t *testing.T) {
		file := filepath.Join(tmpDir, "file1.txt")
		err := os.WriteFile(file, []byte("test"), 0644)
		assert.NoError(t, err)

		md5, err := cache.GetMd5(file)
		assert.NoError(t, err)
		// "test" 的 MD5
		assert.Equal(t, "098f6bcd4621d373cade4e832627b4f6", md5)
	})

	t.Run("缓存一致性", func(t *testing.T) {
		file := filepath.Join(tmpDir, "file2.txt")
		err := os.WriteFile(file, []byte("data"), 0644)
		assert.NoError(t, err)

		// 多次获取应该返回相同结果
		for i := 0; i < 5; i++ {
			md5, err := cache.GetMd5(file)
			assert.NoError(t, err)
			assert.Equal(t, "8d777f385d3dfec8815d20f7496026dc", md5)
		}
	})
}

// TestMD5Cache_Invalidate 测试单个缓存项失效
func TestMD5Cache_Invalidate(t *testing.T) {
	cache := &MD5Cache{}
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(file, []byte("hello"), 0644)
	assert.NoError(t, err)

	// 缓存
	_, err = cache.GetMd5(file)
	assert.NoError(t, err)

	// 失效
	cache.Invalidate(file)

	// 失效不存在的key不应该报错
	cache.Invalidate("nonexistent")
}

// TestMD5Cache_Clear 测试清空所有缓存
func TestMD5Cache_Clear(t *testing.T) {
	cache := &MD5Cache{}
	tmpDir := t.TempDir()

	// 创建多个文件并缓存
	for i := 0; i < 5; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune('a'+i))+".txt")
		err := os.WriteFile(file, []byte("content"), 0644)
		assert.NoError(t, err)
		_, err = cache.GetMd5(file)
		assert.NoError(t, err)
	}

	// 清空缓存
	cache.Clear()

	// 清空后再次清空不应该报错
	cache.Clear()
}

// TestMD5Cache_ConcurrentAccess 测试并发访问
func TestMD5Cache_ConcurrentAccess(t *testing.T) {
	cache := &MD5Cache{}
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "concurrent.txt")
	err := os.WriteFile(file, []byte("concurrent test"), 0644)
	assert.NoError(t, err)

	done := make(chan bool, 10)

	// 并发读写
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_, _ = cache.GetMd5(file)
			}
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}
