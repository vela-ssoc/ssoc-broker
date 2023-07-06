package concurrent

import "sync"

// Map 并发安全的 map
type Map[K comparable, V any] struct {
	mutex sync.RWMutex
	elems map[K]V
}

// NewMap 新建并发 map
func NewMap[K comparable, V any](capacity int) *Map[K, V] {
	return &Map[K, V]{elems: make(map[K]V, capacity)}
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	v, ok := m.elems[key]
	return v, ok
}

func (m *Map[K, V]) Store(key K, val V) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	m.elems[key] = val
}

func (m *Map[K, V]) Delete(key K) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.elems, key)
}

func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	v, ok := m.elems[key]
	if ok {
		delete(m.elems, key)
	}
	return v, ok
}

// Range 遍历执行方法
func (m *Map[K, V]) Range(fn func(K, V)) {
	// 要先克隆数据，因为 fn 由调用方传入，并不确定每个 fn 的执行时长，如果不克隆一份数据直接加锁遍历。
	// 遍历时持锁时长会与 fn 的执行时长直接相关
	tmp := m.clone()
	for k, v := range tmp {
		fn(k, v)
	}
}

// clone 将 bucket 的数据克隆
func (m *Map[K, V]) clone() map[K]V {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	ret := make(map[K]V, len(m.elems))
	for k, v := range m.elems {
		ret[k] = v
	}
	return ret
}
