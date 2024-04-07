package kv

import (
	"sync"
	"time"
)

var once sync.Once
var globalKV *KV

// Item 集合中的元素，包含值和过期时间
type Item struct {
	Value    interface{}
	ExpireAt time.Time
}

// KV 一个支持元素具有声明周期的Set集合
type KV struct {
	mu    sync.Mutex
	items map[interface{}]*Item
}

// newKV 创建一个TTLSet对象
func newKV() *KV {
	return &KV{
		items: make(map[interface{}]*Item),
	}
}

func init() {
	go GetKV().clean(5 * time.Second)
}

// GetKV 获取KV实例
func GetKV() *KV {
	once.Do(func() {
		globalKV = newKV()
	})
	return globalKV
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
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &Item{
		Value:    value,
		ExpireAt: time.Now().Add(ttl),
	}
	s.items[key] = item
}

// get 获取元素
func (s *KV) get(key interface{}) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok {
		return nil
	}
	if time.Now().After(item.ExpireAt) {
		delete(s.items, key)
		return nil
	}
	return item
}

// delete 删除KV中的指定元素
func (s *KV) delete(key interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[key]
	if !ok {
		return
	}
	delete(s.items, key)
	return
}

// contains 检查一个元素是否存在于集合中，并且未过期
func (s *KV) contains(key interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.items[key]
	if !exists {
		return false
	}
	// 检查元素是否已过期
	if time.Now().After(item.ExpireAt) {
		// 如果元素已过期，则移除
		delete(s.items, key)
		return false
	}

	return true
}

// clean 是一个后台协程，定期清理过期的元素
func (s *KV) clean(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for key, item := range s.items {
				if now.After(item.ExpireAt) {
					delete(s.items, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
