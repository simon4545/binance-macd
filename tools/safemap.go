package tools

import "sync"

type SafeMap[K string, V float64] struct {
	m  map[K]V
	mu sync.RWMutex
}

func NewSafeMap[K string, V float64]() *SafeMap[K, V] {
	return &SafeMap[K, V]{
		m: make(map[K]V),
	}
}

func (sm *SafeMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

func (sm *SafeMap[K, V]) Get(key K) (value V, exists bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	value, exists = sm.m[key]
	return
}
