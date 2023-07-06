package concurrent

import "strconv"

type BucketMap[K ~string | ~int64, V any] struct {
	size     int
	capacity int
	buckets  []*Map[K, V]
}

func NewBucketMap[K ~string | ~int64, V any](size, capacity int) BucketMap[K, V] {
	buckets := make([]*Map[K, V], size)
	for i := 0; i < size; i++ {
		buckets[i] = NewMap[K, V](capacity)
	}
	return BucketMap[K, V]{size: size, capacity: capacity, buckets: buckets}
}

func (m BucketMap[K, V]) Load(key K) (V, bool) {
	return m.bucket(key).Load(key)
}

func (m BucketMap[K, V]) Store(key K, val V) {
	m.bucket(key).Store(key, val)
}

func (m BucketMap[K, V]) Delete(key K) {
	m.bucket(key).Delete(key)
}

func (m BucketMap[K, V]) LoadAndDelete(key K) (V, bool) {
	return m.bucket(key).LoadAndDelete(key)
}

func (m BucketMap[K, V]) Exist(key K) bool {
	_, ok := m.bucket(key).Load(key)
	return ok
}

func (m BucketMap[K, V]) Range(fn func(K, V)) {
	for i := 0; i < m.size; i++ {
		m.buckets[i].Range(fn)
	}
}

// bucket 获取id落在的桶位
func (m BucketMap[K, V]) bucket(key K) *Map[K, V] {
	sum := m.fnv32(key)
	idx := int(sum) % m.size
	return m.buckets[idx]
}

// fnv32 https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV_hash_parameters
// https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV-1_hash
func (BucketMap[K, V]) fnv32(k K) uint32 {
	var key string
	switch kt := any(k).(type) {
	case string:
		key = kt
	case int64:
		key = strconv.FormatInt(kt, 10)
	default:
		return 0
	}

	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}

	return hash
}
