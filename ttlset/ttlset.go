package ttlset

import (
	"sync"
	"time"
)

var seter *TTLSet

func init() {
	seter = NewTTLSet()
	go seter.Clean(5 * time.Second)
}

// Add 往集合中添加元素
func Add(value interface{}, ttl time.Duration) {
	seter.Add(value, ttl)
}

// Exists 判断元素是否在集合中且未过期
func Exists(value interface{}) bool {
	return seter.Contains(value)
}

// Item 集合中的元素，包含值和过期时间
type Item struct {
	Value    interface{}
	ExpireAt time.Time
}

// TTLSet 一个支持元素具有声明周期的Set集合
type TTLSet struct {
	mu    sync.Mutex
	items map[interface{}]*Item
}

// NewTTLSet 创建一个TTLSet对象
func NewTTLSet() *TTLSet {
	return &TTLSet{
		items: make(map[interface{}]*Item),
	}
}

// Add 往Set中增加元素
func (s *TTLSet) Add(value interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item := &Item{
		Value:    value,
		ExpireAt: time.Now().Add(ttl),
	}
	s.items[value] = item
}

// Contains 检查一个元素是否存在于集合中，并且未过期
func (s *TTLSet) Contains(value interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.items[value]
	if !exists {
		return false
	}
	// 检查元素是否已过期
	if time.Now().After(item.ExpireAt) {
		// 如果元素已过期，则移除
		delete(s.items, value)
		return false
	}

	return true
}

// Clean 是一个后台协程，定期清理过期的元素
func (s *TTLSet) Clean(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for value, item := range s.items {
				if now.After(item.ExpireAt) {
					delete(s.items, value)
				}
			}
			s.mu.Unlock()
		}
	}
}
