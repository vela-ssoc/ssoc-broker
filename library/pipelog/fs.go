package pipelog

import (
	"io"
	"os"
	"sync"
	"time"
)

type FS interface {
	io.Closer
	Open(name string) (File, error)
	Remove(name string) error
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
}

func (f *pipeFS) Open(name string) (File, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if fl := f.files[name]; fl != nil {
		return fl, nil
	}
	if f.files == nil {
		f.files = make(map[string]*limitedFile, 32)
	}
	lf := newLimitedFile(f.root, name, f.maxsize, f.idle)
	f.files[name] = lf

	return lf, nil
}

func (f *pipeFS) Close() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, lf := range f.files {
		_ = lf.Close()
	}
	f.files = nil

	return nil
}

func (f *pipeFS) Remove(name string) (err error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if fl := f.files[name]; fl != nil {
		delete(f.files, name)
		return fl.remove()
	}

	return f.remove(name)
}

func (f *pipeFS) remove(name string) error {
	root, err := os.OpenRoot(f.root)
	if err != nil {
		return err
	}
	defer root.Close()

	return root.Remove(name)
}
