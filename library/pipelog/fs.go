package pipelog

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

type FS interface {
	io.Closer
	Open(name string) (File, error)
	Remove(name string) error

	Subscriber(w io.Writer, name string, n int)
	Unsubscriber(w io.Writer, name string)
}

func NewFS(dir string, maxsize int64, idle time.Duration) FS {
	return &pipeFS{
		maxsize: maxsize,
		root:    dir,
		idle:    idle,
	}
}

type pipeFS struct {
	maxsize int64
	root    string
	idle    time.Duration
	mutex   sync.RWMutex
	files   map[string]*limitedFile
	submtx  sync.RWMutex
	submap  map[string]map[io.Writer]struct{} // KEY: filename
}

func (pf *pipeFS) Open(name string) (File, error) {
	pf.mutex.Lock()
	defer pf.mutex.Unlock()

	if fl := pf.files[name]; fl != nil {
		return fl, nil
	}
	if pf.files == nil {
		pf.files = make(map[string]*limitedFile, 128)
	}
	lf := pf.newFile(name)
	pf.files[name] = lf

	return lf, nil
}

func (pf *pipeFS) Close() error {
	pf.mutex.Lock()
	defer pf.mutex.Unlock()

	for _, lf := range pf.files {
		_ = lf.close()
	}
	pf.files = nil

	return nil
}

func (pf *pipeFS) Subscriber(w io.Writer, name string, n int) {
	if w == nil {
		return
	}

	pf.submtx.Lock()
	if pf.submap == nil {
		pf.submap = make(map[string]map[io.Writer]struct{}, 16)
	}
	subs := pf.submap[name]
	if subs == nil {
		subs = make(map[io.Writer]struct{}, 4)
	}
	subs[w] = struct{}{}
	pf.submap[name] = subs
	pf.submtx.Unlock()

	// n = 0 不读取
	// n > 0 读取 N 行
	// n < 0 读取全部
	_ = pf.readN(name, w, n)
}

func (pf *pipeFS) Unsubscriber(w io.Writer, name string) {
	if w == nil {
		return
	}

	pf.submtx.Lock()
	defer pf.submtx.Unlock()
	subs := pf.submap[name]
	delete(subs, w)
}

func (pf *pipeFS) Remove(name string) (err error) {
	pf.mutex.Lock()
	defer pf.mutex.Unlock()

	if fl := pf.files[name]; fl != nil {
		delete(pf.files, name)
		return fl.remove()
	}

	return pf.remove(name)
}

func (pf *pipeFS) remove(name string) error {
	root, err := os.OpenRoot(pf.root)
	if err != nil {
		return err
	}
	defer root.Close()

	return root.Remove(name)
}

func (pf *pipeFS) newFile(name string) *limitedFile {
	return &limitedFile{
		pfs:     pf,
		root:    pf.root,
		name:    name,
		maxsize: pf.maxsize,
		idle:    pf.idle,
	}
}

func (pf *pipeFS) notifyAll(name string, content []byte) {
	pf.submtx.RLock()
	ents := pf.submap[name]
	subs := make([]io.Writer, 0, len(ents))
	for w := range ents {
		subs = append(subs, w)
	}
	pf.submtx.RUnlock()

	for _, sub := range subs {
		msg := make([]byte, len(content))
		copy(msg, content) // 防止订阅者修改内容，每次复制一次。
		_, _ = sub.Write(msg)
	}
}

func (pf *pipeFS) readN(name string, w io.Writer, n int) error {
	if n == 0 {
		return nil
	}

	of, err := os.OpenInRoot(pf.root, name)
	if err != nil {
		return err
	}
	defer of.Close()

	if n > 0 {
		return pf.tailN(w, of, n)
	} else {
		return pf.tailAll(w, of)
	}
}

func (pf *pipeFS) tailAll(w io.Writer, r io.Reader) error {
	br := bufio.NewReader(r)
	for {
		line, _, err := br.ReadLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if _, err = w.Write(line); err != nil {
			return err
		}
	}
}

func (pf *pipeFS) tailN(w io.Writer, f *os.File, n int) error {
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	fileSize := stat.Size()
	pos := fileSize
	var buffer []byte
	var lines [][]byte

	for pos > 0 && len(lines) <= n {
		// 每次读取一个 block（往前）
		readSize := int64(4096)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		chunk := make([]byte, readSize)
		_, err = f.ReadAt(chunk, pos)
		if err != nil && err != io.EOF {
			return err
		}

		// 新读取的数据放在前面，模拟“从后往前拼接”
		buffer = append(chunk, buffer...)
		lines = lines[:0]
		for line := range bytes.Lines(buffer) {
			lines = append(lines, line)
		}
	}

	// 返回最后 n 行
	if n >= 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, line := range lines {
		if _, err = w.Write(line); err != nil {
			return err
		}
	}

	return nil
}
