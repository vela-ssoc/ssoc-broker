package linkhub

import (
	"hash/fnv"
	"sync"
)

type Huber interface {
	Get(host string) Peer
	Put(Peer) (succeed bool)
	Del(host string) Peer
	All() []Peer

	GetByID(id int64) Peer
	DelByID(id int64) Peer
}

func NewSafeMap(capacity ...int) Huber {
	return newSafeMap(capacity...)
}

func newSafeMap(capacity ...int) *safeMap {
	var size int
	if len(capacity) != 0 && capacity[0] > 0 {
		size = capacity[0]
	}

	return &safeMap{
		peers: make(map[string]Peer, size),
	}
}

type safeMap struct {
	mutex sync.RWMutex
	peers map[string]Peer // key: host
}

func (sm *safeMap) Get(host string) Peer {
	sm.mutex.RLock()
	peer := sm.peers[host]
	sm.mutex.RUnlock()

	return peer
}

func (sm *safeMap) Put(peer Peer) bool {
	if peer == nil {
		return false
	}

	host := peer.Host()
	sm.mutex.Lock()
	_, exists := sm.peers[host]
	if !exists {
		sm.peers[host] = peer
	}
	sm.mutex.Unlock()

	return !exists
}

func (sm *safeMap) Del(host string) Peer {
	if host == "" {
		return nil
	}

	sm.mutex.Lock()
	peer := sm.peers[host]
	if peer != nil {
		delete(sm.peers, host)
	}
	defer sm.mutex.Unlock()

	return peer
}

func (sm *safeMap) All() []Peer {
	sm.mutex.RLock()
	peers := make([]Peer, 0, len(sm.peers))
	for _, p := range sm.peers {
		peers = append(peers, p)
	}
	sm.mutex.RUnlock()

	return peers
}

func (sm *safeMap) GetByID(id int64) Peer {
	host := formatID(id)
	return sm.Get(host)
}

func (sm *safeMap) DelByID(id int64) Peer {
	host := formatID(id)
	return sm.Del(host)
}

func NewShardMap(capacity ...int) Huber {
	var size int
	if len(capacity) != 0 && capacity[0] > 0 {
		size = capacity[0]
	}

	const shards = 16
	num := size / shards

	sm := new(shardMap)
	for i := range sm.shards {
		sm.shards[i] = newSafeMap(num)
	}

	return sm
}

type shardMap struct {
	shards [16]*safeMap
}

func (sm *shardMap) Get(host string) Peer {
	shard := sm.shard(host)
	return shard.Get(host)
}

func (sm *shardMap) Put(peer Peer) bool {
	if peer == nil {
		return false
	}

	host := peer.Host()
	shard := sm.shard(host)

	return shard.Put(peer)
}

func (sm *shardMap) Del(host string) Peer {
	if host == "" {
		return nil
	}
	shard := sm.shard(host)

	return shard.Del(host)
}

func (sm *shardMap) All() []Peer {
	peers := make([]Peer, 0, len(sm.shards)*32)
	for _, shard := range sm.shards {
		ps := shard.All()
		peers = append(peers, ps...)
	}

	return peers
}

func (sm *shardMap) GetByID(id int64) Peer {
	host := formatID(id)
	return sm.Get(host)
}

func (sm *shardMap) DelByID(id int64) Peer {
	host := formatID(id)
	return sm.Del(host)
}

func (sm *shardMap) shard(host string) *safeMap {
	f := fnv.New32a()
	_, _ = f.Write([]byte(host))
	sum := f.Sum32()

	i := int(sum) % len(sm.shards)

	return sm.shards[i]
}
