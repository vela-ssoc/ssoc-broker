package mlink

import "sync"

type container interface {
	Put(id string, conn *connect) bool
	Get(id string) *connect
	Del(id string) *connect
	IDs() []int64
}

type Iter interface {
	Has() bool
	Next() int64
}

func newSafeMap(size int) *safeMap {
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

func newSegmentMap(slot, size int) container {
	if slot <= 0 {
		slot = 8
	}
	sm := &segmentMap{
		size: slot,
		slot: make([]*safeMap, slot),
	}
	for i := 0; i < slot; i++ {
		sm.slot[i] = newSafeMap(size)
	}

	return sm
}

type segmentMap struct {
	size int
	slot []*safeMap
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

func (sm *segmentMap) IDs() []int64 {
	ret := make([]int64, 0, 2000)
	for _, c := range sm.slot {
		cs := c.connections()
		for _, c2 := range cs {
			ret = append(ret, c2.id)
		}

	}

	return ret
}

// getSLOT 根据 key 计算所在的存储桶
func (sm *segmentMap) getSLOT(key string) *safeMap {
	hash := sm.fnv32(key)
	idx := int(hash) % sm.size
	return sm.slot[idx]
}

// fnv32 https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV_hash_parameters
// https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function#FNV-1_hash
func (*segmentMap) fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 uint32 = 16777619
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
