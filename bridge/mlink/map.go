package mlink

import "sync"

type container interface {
	Put(id string, conn *connect) bool
	Get(id string) *connect
	Del(id string) *connect
	Iter() Iter
}

type Iter interface {
	Has() bool
	Next() int64
}

func newSafeMap(size int) container {
	if size <= 0 {
		size = 16
	}

	return &safeMap{
		mutex: sync.RWMutex{},
		elems: make(map[string]*connect, size),
	}
}

type safeMap struct {
	mutex sync.RWMutex
	elems map[string]*connect
}

func (sm *safeMap) Put(id string, conn *connect) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	if _, ok := sm.elems[id]; ok {
		return false
	}
	sm.elems[id] = conn
	return true
}

func (sm *safeMap) Get(id string) *connect {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.elems[id]
}

func (sm *safeMap) Del(id string) *connect {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	c, ok := sm.elems[id]
	if ok {
		delete(sm.elems, id)
	}
	return c
}

func (sm *safeMap) Iter() Iter {
	connects := sm.connections()
	size := len(connects)

	return &safeMapIter{
		size:  size,
		elems: connects,
	}
}

func (sm *safeMap) connections() []*connect {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	size := len(sm.elems)
	elems := make([]*connect, 0, size)
	for _, c := range sm.elems {
		elems = append(elems, c)
	}

	return elems
}

type safeMapIter struct {
	cursor int
	size   int
	elems  []*connect
}

func (it *safeMapIter) Has() bool {
	return it.cursor < it.size
}

func (it *safeMapIter) Next() int64 {
	if it.cursor >= it.size {
		return 0
	}

	c := it.elems[it.cursor]
	it.cursor++

	return c.id
}

func newSegmentMap(slot, size int) container {
	if slot <= 0 {
		slot = 8
	}
	sm := &segmentMap{
		size: slot,
		slot: make([]container, slot),
	}
	for i := 0; i < slot; i++ {
		sm.slot[i] = newSafeMap(size)
	}

	return sm
}

type segmentMap struct {
	size int
	slot []container
}

func (sm *segmentMap) Put(id string, conn *connect) bool {
	return sm.getSLOT(id).Put(id, conn)
}

func (sm *segmentMap) Get(id string) *connect {
	return sm.getSLOT(id).Get(id)
}

func (sm *segmentMap) Del(id string) *connect {
	return sm.getSLOT(id).Del(id)
}

func (sm *segmentMap) Iter() Iter {
	return &segmentMapIter{
		slot: sm.slot,
	}
}

// getSLOT 根据 key 计算所在的存储桶
func (sm *segmentMap) getSLOT(key string) container {
	hash := sm.fnv32(key)
	idx := int(hash) % sm.size
	return sm.slot[idx]
}

// fnv32 https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV_hash_parameters
// https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV-1_hash
func (*segmentMap) fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

type segmentMapIter struct {
	iter   Iter
	cursor int
	slot   []container
}

func (sm *segmentMapIter) Has() bool {
	iter := sm.currentIter()
	if iter == nil {
		return false
	}
	return iter.Has()
}

func (sm *segmentMapIter) Next() int64 {
	iter := sm.currentIter()
	if iter == nil {
		return 0
	}
	return iter.Next()
}

func (sm *segmentMapIter) currentIter() Iter {
	iter := sm.iter
	if iter == nil || !iter.Has() {
		size := len(sm.slot)
		if size <= sm.cursor {
			return nil
		}
		next := sm.slot[sm.cursor].Iter()
		sm.iter = next
		sm.cursor++

		return next
	}
	return iter
}
