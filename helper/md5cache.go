package helper

import (
	"os"
	"sync"
	"time"
)

// md5CacheEntry MD5 缓存条目
type md5CacheEntry struct {
	md5     string
	modTime time.Time
	size    int64
}

// MD5Cache 基于文件路径、修改时间和大小的 MD5 缓存
// 使用 sync.Map 实现无锁并发访问
type MD5Cache struct {
	cache sync.Map // map[string]*md5CacheEntry
}

// 全局 MD5 缓存实例
var globalMD5Cache = &MD5Cache{}

// GetCachedFileMd5 获取带缓存的文件 MD5
// 如果文件的修改时间和大小与缓存一致，则直接返回缓存的 MD5
// 否则重新计算并更新缓存
func GetCachedFileMd5(path string) (string, error) {
	return globalMD5Cache.GetMd5(path)
}

// InvalidateMd5Cache 使指定路径的 MD5 缓存失效
func InvalidateMd5Cache(path string) {
	globalMD5Cache.Invalidate(path)
}

// GetMd5 获取文件的 MD5，优先使用缓存
func (c *MD5Cache) GetMd5(path string) (string, error) {
	// 获取文件信息
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	modTime := fileInfo.ModTime()
	size := fileInfo.Size()

	// 检查缓存
	if value, ok := c.cache.Load(path); ok {
		entry := value.(*md5CacheEntry)
		// 如果修改时间和大小都一致，直接返回缓存的 MD5
		if entry.modTime.Equal(modTime) && entry.size == size {
			return entry.md5, nil
		}
	}

	// 缓存未命中或已过期，重新计算 MD5
	md5, err := FileMd5(path)
	if err != nil {
		return "", err
	}

	// 更新缓存
	c.cache.Store(path, &md5CacheEntry{
		md5:     md5,
		modTime: modTime,
		size:    size,
	})

	return md5, nil
}

// Invalidate 使指定路径的缓存失效
func (c *MD5Cache) Invalidate(path string) {
	c.cache.Delete(path)
}

// Clear 清空所有缓存
func (c *MD5Cache) Clear() {
	c.cache.Range(func(key, _ interface{}) bool {
		c.cache.Delete(key)
		return true
	})
}
