package kv

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupTest 测试前重置全局状态
func setupTest(t *testing.T) {
	t.Helper()
	ResetForTest()
	// 重新启动清理协程
	go GetKV().clean(100*time.Millisecond, stopChan)
}

// TestSet 测试 Set 功能
func TestSet(t *testing.T) {
	setupTest(t)
	defer Stop()

	t.Run("设置键值对", func(t *testing.T) {
		Set("key1", "value1", time.Minute)
		assert.True(t, Exists("key1"))
	})

	t.Run("覆盖已存在的键", func(t *testing.T) {
		Set("key2", "value2a", time.Minute)
		Set("key2", "value2b", time.Minute)
		assert.True(t, Exists("key2"))
	})
}

// TestExists 测试 Exists 功能
func TestExists(t *testing.T) {
	setupTest(t)
	defer Stop()

	t.Run("存在的键", func(t *testing.T) {
		Set("exists", "value", time.Minute)
		assert.True(t, Exists("exists"))
	})

	t.Run("不存在的键", func(t *testing.T) {
		assert.False(t, Exists("nonexistent"))
	})

	t.Run("过期的键", func(t *testing.T) {
		Set("expired", "value", 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		assert.False(t, Exists("expired"))
	})
}

// TestGet 测试 Get 功能
func TestGet(t *testing.T) {
	setupTest(t)
	defer Stop()

	t.Run("获取存在的键", func(t *testing.T) {
		Set("getkey", "getvalue", time.Minute)
		result := Get("getkey")
		assert.NotNil(t, result)
		item, ok := result.(*Item)
		assert.True(t, ok)
		assert.Equal(t, "getvalue", item.Value)
	})

	t.Run("获取不存在的键", func(t *testing.T) {
		result := Get("nonexistent")
		assert.Nil(t, result)
	})

	t.Run("获取过期的键", func(t *testing.T) {
		Set("expiredget", "value", 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		result := Get("expiredget")
		assert.Nil(t, result)
	})
}

// TestDelete 测试 Delete 功能
func TestDelete(t *testing.T) {
	setupTest(t)
	defer Stop()

	t.Run("删除存在的键", func(t *testing.T) {
		Set("deletekey", "value", time.Minute)
		assert.True(t, Exists("deletekey"))
		Delete("deletekey")
		assert.False(t, Exists("deletekey"))
	})

	t.Run("删除不存在的键不报错", func(t *testing.T) {
		Delete("nonexistent")
		assert.False(t, Exists("nonexistent"))
	})
}

// TestTTL 测试 TTL 过期功能
func TestTTL(t *testing.T) {
	setupTest(t)
	defer Stop()

	t.Run("TTL 过期", func(t *testing.T) {
		Set("ttlkey", "value", 50*time.Millisecond)
		assert.True(t, Exists("ttlkey"))

		time.Sleep(100 * time.Millisecond)
		assert.False(t, Exists("ttlkey"))
	})

	t.Run("不同 TTL", func(t *testing.T) {
		Set("short", "value", 20*time.Millisecond)
		Set("long", "value", time.Minute)

		time.Sleep(50 * time.Millisecond)

		assert.False(t, Exists("short"))
		assert.True(t, Exists("long"))
	})
}

// TestClean 测试后台清理功能
func TestClean(t *testing.T) {
	setupTest(t)
	defer Stop()

	// 设置多个短 TTL 的键
	for i := 0; i < 10; i++ {
		Set(i, "value", 30*time.Millisecond)
	}

	// 等待清理
	time.Sleep(200 * time.Millisecond)

	// 所有键应该被清理
	for i := 0; i < 10; i++ {
		assert.False(t, Exists(i))
	}
}

// TestConcurrency 测试并发安全性
func TestConcurrency(t *testing.T) {
	setupTest(t)
	defer Stop()

	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 100

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				Set(key, "value", time.Minute)
			}
		}(i)
	}

	wg.Wait()

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				_ = Exists(key)
				_ = Get(key)
			}
		}(i)
	}

	wg.Wait()

	// 并发删除
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := id*numOperations + j
				Delete(key)
			}
		}(i)
	}

	wg.Wait()
}

// TestGetKV 测试单例获取
func TestGetKV(t *testing.T) {
	setupTest(t)
	defer Stop()

	kv1 := GetKV()
	kv2 := GetKV()

	assert.Same(t, kv1, kv2)
}

// TestStop 测试停止功能
func TestStop(t *testing.T) {
	setupTest(t)

	// 停止应该不会 panic
	Stop()

	// 多次停止不会 panic
	Stop()
}

// TestItem 测试 Item 结构体
func TestItem(t *testing.T) {
	item := &Item{
		Value:    "test value",
		ExpireAt: time.Now().Add(time.Hour),
	}

	assert.Equal(t, "test value", item.Value)
	assert.False(t, time.Now().After(item.ExpireAt))
}

// TestDifferentKeyTypes 测试不同类型的键
func TestDifferentKeyTypes(t *testing.T) {
	setupTest(t)
	defer Stop()

	t.Run("字符串键", func(t *testing.T) {
		Set("string_key", "value", time.Minute)
		assert.True(t, Exists("string_key"))
	})

	t.Run("整数键", func(t *testing.T) {
		Set(123, "value", time.Minute)
		assert.True(t, Exists(123))
	})

	t.Run("结构体键", func(t *testing.T) {
		type Key struct {
			ID   int
			Name string
		}
		key := Key{ID: 1, Name: "test"}
		Set(key, "value", time.Minute)
		assert.True(t, Exists(key))
	})
}
