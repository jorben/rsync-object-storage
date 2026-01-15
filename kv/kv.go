package kv

import (
	"sync"
	"time"
)

var once sync.Once

// globalKV 是热点文件缓存的全局单例实例
// 设计说明：采用单例模式简化热点文件检测，用于判断文件是否为频繁修改的热点文件
// 优化：使用 sync.Map 替代 sync.Mutex，减少锁竞争，提升高并发场景性能
//
// 测试时Mock方法：
// 1. 使用 ResetForTest() 重置全局实例
// 2. 在测试中直接操作全局函数 Set/Get/Exists/Delete
// 3. 如需完全隔离，可在测试前调用 ResetForTest()，测试后再调用一次清理
var globalKV *KV

// stopChan 用于停止清理协程
var stopChan chan struct{}
var stopOnce sync.Once

// Item 集合中的元素，包含值和过期时间
type Item struct {
	Value    interface{}
	ExpireAt time.Time
}

// KV 一个支持元素具有生命周期的Set集合
// 使用 sync.Map 替代 map + Mutex，在读多写少场景下性能更好
type KV struct {
	items sync.Map // map[interface{}]*Item
}

// newKV 创建一个TTLSet对象
func newKV() *KV {
	return &KV{}
}

func init() {
	stopChan = make(chan struct{})
	go GetKV().clean(5*time.Second, stopChan)
}

// GetKV 获取KV实例
func GetKV() *KV {
	once.Do(func() {
		globalKV = newKV()
	})
	return globalKV
}

// Stop 停止KV清理协程，用于优雅退出
func Stop() {
	stopOnce.Do(func() {
		close(stopChan)
	})
}

// ResetForTest 重置全局实例，仅用于测试
// 注意：此方法仅应在测试中使用，生产环境中不应调用
func ResetForTest() {
	once = sync.Once{}
	stopOnce = sync.Once{}
	globalKV = nil
	stopChan = make(chan struct{})
}

// Set 往集合中添加元素
func Set(key interface{}, value interface{}, ttl time.Duration) {
	GetKV().set(key, value, ttl)
}

// Exists 判断元素是否在集合中且未过期
func Exists(key interface{}) bool {
	return GetKV().contains(key)
}

// Get 获取KV中的指定元素
func Get(key interface{}) interface{} {
	return GetKV().get(key)
}

// Delete 删除KV中指定的元素
func Delete(key interface{}) {
	GetKV().delete(key)
}

// set 往Set中增加元素
func (s *KV) set(key interface{}, value interface{}, ttl time.Duration) {
	item := &Item{
		Value:    value,
		ExpireAt: time.Now().Add(ttl),
	}
	s.items.Store(key, item)
}

// get 获取元素
func (s *KV) get(key interface{}) interface{} {
	value, ok := s.items.Load(key)
	if !ok {
		return nil
	}
	item := value.(*Item)
	if time.Now().After(item.ExpireAt) {
		s.items.Delete(key)
		return nil
	}
	return item
}

// delete 删除KV中的指定元素
func (s *KV) delete(key interface{}) {
	s.items.Delete(key)
}

// contains 检查一个元素是否存在于集合中，并且未过期
func (s *KV) contains(key interface{}) bool {
	value, exists := s.items.Load(key)
	if !exists {
		return false
	}
	item := value.(*Item)
	// 检查元素是否已过期
	if time.Now().After(item.ExpireAt) {
		// 如果元素已过期，则移除
		s.items.Delete(key)
		return false
	}
	return true
}

// clean 是一个后台协程，定期清理过期的元素
// 支持通过stopCh通道停止，实现优雅退出
func (s *KV) clean(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			s.items.Range(func(key, value interface{}) bool {
				item := value.(*Item)
				if now.After(item.ExpireAt) {
					s.items.Delete(key)
				}
				return true
			})
		}
	}
}
