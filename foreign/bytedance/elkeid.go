package bytedance

import (
	"context"
	"io"
	"io/fs"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vela-ssoc/vela-common-mba/netutil"
)

func ElkeidFS(dir string, cli netutil.HTTPClient) fs.FS {
	_ = os.MkdirAll(dir, os.ModePerm)
	addrs := []string{
		"https://lf3-elkeid.bytetos.com/obj/elkeid-download/ko/",
		"https://lf6-elkeid.bytetos.com/obj/elkeid-download/ko/",
		"https://lf9-elkeid.bytetos.com/obj/elkeid-download/ko/",
		"https://lf26-elkeid.bytetos.com/obj/elkeid-download/ko/",
	}
	return &elkeidFS{
		dir:   dir,
		cli:   cli,
		downs: make(map[string]struct{}, 8),
		addrs: addrs,
	}
}

// elkeidFS 文件下载
//
// https://github.com/bytedance/Elkeid/blob/a540bb8a225ebd071148fffe51ff66adea49f755/driver/README-zh_CN.md
type elkeidFS struct {
	dir   string
	cli   netutil.HTTPClient
	mutex sync.Mutex
	downs map[string]struct{}
	addrs []string
}

func (e *elkeidFS) Open(name string) (fs.File, error) {
	fname := filepath.Join(e.dir, name)
	if file, err := os.Open(fname); err == nil {
		return file, nil
	}

	e.mutex.Lock()
	_, exist := e.downs[name]
	if !exist {
		e.downs[name] = struct{}{}
	}
	e.mutex.Unlock()

	// 临时文件名字
	temp := fname + ".downloading"
	addr := e.randAddr() + name

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		cancel()
		return nil, err
	}

	res, err := e.cli.Do(req)
	if err != nil {
		cancel()
		return nil, err
	}

	info := &bodyFileInfo{
		name:  name,
		size:  res.ContentLength,
		mtime: time.Now(),
	}

	if exist {
		return &bodyFile{info: info, rc: res.Body, cancel: cancel}, nil
	}

	file, err := os.Create(temp)
	if err != nil {
		e.mutex.Lock()
		delete(e.downs, name)
		e.mutex.Unlock()
		_ = res.Body.Close()
		cancel()
		return nil, err
	}

	cf := &copyFile{
		info:   info,
		temp:   temp,
		name:   name,
		fname:  fname,
		file:   file,
		body:   res.Body,
		read:   io.TeeReader(res.Body, file),
		fs:     e,
		cancel: cancel,
	}

	return cf, nil
}

func (e *elkeidFS) randAddr() string {
	size := len(e.addrs)
	idx := rand.Intn(size)
	return e.addrs[idx]
}

type bodyFile struct {
	info   fs.FileInfo
	rc     io.ReadCloser
	cancel context.CancelFunc
}

func (b *bodyFile) Stat() (fs.FileInfo, error) {
	return b.info, nil
}

func (b *bodyFile) Read(p []byte) (int, error) {
	return b.rc.Read(p)
}

func (b *bodyFile) Close() error {
	b.cancel()
	return b.rc.Close()
}

type copyFile struct {
	info   fs.FileInfo
	temp   string
	name   string
	fname  string
	file   io.WriteCloser
	body   io.Closer
	read   io.Reader
	fs     *elkeidFS
	err    error
	cancel context.CancelFunc
	closed atomic.Bool
}

func (c *copyFile) Stat() (fs.FileInfo, error) {
	return c.info, nil
}

func (c *copyFile) Read(p []byte) (int, error) {
	n, err := c.read.Read(p)
	if err != nil {
		c.err = err
	}
	return n, err
}

func (c *copyFile) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	c.cancel()
	_ = c.body.Close()
	_ = c.file.Close()
	if err := c.err; err == io.EOF {
		_ = os.Rename(c.temp, c.fname)
	}

	c.fs.mutex.Lock()
	delete(c.fs.downs, c.name)
	c.fs.mutex.Unlock()

	return nil
}

type bodyFileInfo struct {
	name  string
	size  int64
	mtime time.Time
}

func (b *bodyFileInfo) Name() string {
	return b.name
}

func (b *bodyFileInfo) Size() int64 {
	return b.size
}

func (b *bodyFileInfo) Mode() fs.FileMode {
	return os.ModePerm
}

func (b *bodyFileInfo) ModTime() time.Time {
	return b.mtime
}

func (b *bodyFileInfo) IsDir() bool {
	return false
}

func (b *bodyFileInfo) Sys() any {
	return nil
}
