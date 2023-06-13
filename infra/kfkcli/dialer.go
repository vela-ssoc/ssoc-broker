package kfkcli

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"strings"
	"sync"

	"github.com/segmentio/kafka-go"
)

type Dialer interface {
	Dial(addr []string) Producer
	release(id string)
}

func ReuseDialer() Dialer {
	return &reuseDialer{produces: make(map[string]Producer, 16)}
}

type reuseDialer struct {
	mutex    sync.RWMutex
	produces map[string]Producer
}

func (d *reuseDialer) Dial(addrs []string) Producer {
	id, addr := d.preformat(addrs)
	d.mutex.RLock()
	p, ok := d.produces[id]
	p.acquire()
	d.mutex.RUnlock()
	if ok {
		return p
	}

	return d.slowDial(id, addr)
}

func (d *reuseDialer) slowDial(id string, addr []string) Producer {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if p, ok := d.produces[id]; ok {
		p.acquire()
		return p
	}

	writer := &kafka.Writer{
		Addr:  kafka.TCP(addr...),
		Async: true,
	}
	p := &shadowProducer{
		id:     id,
		dialer: d,
		write:  writer,
	}
	d.produces[id] = p
	p.acquire()

	return p
}

func (d *reuseDialer) preformat(addrs []string) (string, []string) {
	// addrs 去重并排序，计算 md5 作为 key
	size := len(addrs)
	result := make([]string, 0, size)
	hashset := make(map[string]struct{}, size)
	for _, addr := range addrs {
		if _, ok := hashset[addr]; !ok {
			hashset[addr] = struct{}{}
			result = append(result, addr)
		}
	}
	sort.Strings(result)
	str := strings.Join(result, ",")
	sum := md5.Sum([]byte(str))
	id := hex.EncodeToString(sum[:])

	return id, result
}

func (d *reuseDialer) release(id string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	delete(d.produces, id)
}
