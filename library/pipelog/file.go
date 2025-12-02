package pipelog

import (
	"io"
	"io/fs"
	"os"
	"sync"
	"time"
)

type File interface {
	io.Writer

	Resize(maxsize int64)
	Maxsize() int64

	Stat() (fs.FileInfo, error)
}

type limitedFile struct {
	pfs      *pipeFS                // 文件管理器。
	root     string                 // 文件目录。
	name     string                 // 文件名。
	maxsize  int64                  // 文件最大大小。
	idle     time.Duration          // 空闲时间（长时间没有写入数据时，自动关闭文件）。
	fmtx     sync.Mutex             // 文件锁（读、写、关闭文件时用到）。
	timer    *time.Timer            // 空闲定时器（定时器到时自动关闭文件）。
	file     *os.File               // 文件指针（如果已打开）。
	filesize int64                  // 文件实时大小。
	submtx   sync.RWMutex           // 订阅者锁。
	submap   map[io.Writer]struct{} // 订阅者列表。
}

func (lf *limitedFile) Resize(maxsize int64) {
	if maxsize <= 0 {
		return
	}

	lf.fmtx.Lock()
	defer lf.fmtx.Unlock()

	lf.maxsize = maxsize
	lf.truncate()
}

func (lf *limitedFile) Maxsize() int64 {
	lf.fmtx.Lock()
	defer lf.fmtx.Unlock()

	return lf.maxsize
}

func (lf *limitedFile) Stat() (fs.FileInfo, error) {
	f, err := os.OpenInRoot(lf.root, lf.name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Stat()
}

func (lf *limitedFile) Write(p []byte) (int, error) {
	n, err := lf.write(p)
	if err == nil && n > 0 {
		lf.pfs.notifyAll(lf.name, p)
	}

	return n, err
}

func (lf *limitedFile) write(p []byte) (int, error) {
	lf.fmtx.Lock()
	defer lf.fmtx.Unlock()

	f, err := lf.open()
	if err != nil {
		return 0, err
	}

	// 每写入一次，自动延期关闭文件。
	n, err := f.Write(p)
	if err == nil && lf.timer != nil {
		lf.timer.Reset(lf.idle)
	}
	lf.filesize += int64(n)
	lf.truncate()

	return n, err
}

func (lf *limitedFile) open() (*os.File, error) {
	if f := lf.file; f != nil {
		return f, nil
	}

	// 如果文件已经关闭或者不存在，就打开/创建。
	root, err := os.OpenRoot(lf.root)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	f, err := root.OpenFile(lf.name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
	if err != nil {
		return nil, err
	}
	if stat, _ := f.Stat(); stat != nil {
		lf.filesize = stat.Size()
	}

	lf.file = f
	if du := lf.idle; du > 0 {
		lf.timer = time.AfterFunc(lf.idle, lf.autoclose)
	}

	return f, nil
}

func (lf *limitedFile) truncate() {
	f := lf.file
	if f == nil || lf.filesize < lf.maxsize {
		return
	}

	err := f.Truncate(0)
	if err == nil {
		_, err = f.Seek(0, io.SeekStart)
	}
	if err == nil {
		lf.filesize = 0
	}
}

func (lf *limitedFile) autoclose() {
	_ = lf.close()
}

func (lf *limitedFile) close() error {
	lf.fmtx.Lock()
	defer lf.fmtx.Unlock()

	if t := lf.timer; t != nil {
		t.Stop()
	}

	if f := lf.file; f != nil {
		lf.file = nil
		lf.filesize = 0
		return f.Close()
	}

	return nil
}

func (lf *limitedFile) remove() error {
	_ = lf.close()

	root, err := os.OpenRoot(lf.root)
	if err != nil {
		return err
	}
	defer root.Close()

	return root.Remove(lf.name)
}
