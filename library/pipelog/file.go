package pipelog

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"os"
	"sync"
	"time"
)

type File interface {
	io.Writer

	// Subscriber 订阅写入的数据，输出最后 n 行（类似于 tail 命令）。
	// n < 0 代表输出所有数据。
	Subscriber(w io.Writer, n int) bool
	Unsubscriber(w io.Writer)
	Resize(maxsize int64)
	Maxsize() int64

	Stat() (fs.FileInfo, error)
}

func newLimitedFile(root, name string, maxsize int64, idle time.Duration) *limitedFile {
	return &limitedFile{
		root:    root,
		name:    name,
		maxsize: maxsize,
		idle:    idle,
	}
}

type limitedFile struct {
	root     string                 // 文件目录
	name     string                 // 文件名
	maxsize  int64                  // 文件最大大小
	idle     time.Duration          // 空闲时间，长时间没有写入数据时，自动关闭文件。
	fmtx     sync.Mutex             // 文件锁
	timer    *time.Timer            // 关闭文件定时器
	file     *os.File               // 文件（如果已打开）
	filesize int64                  // 文件实时大小
	submtx   sync.RWMutex           // 订阅者锁
	submap   map[io.Writer]struct{} // 订阅者列表
}

// Subscriber 订阅文件的写入数据。
func (lf *limitedFile) Subscriber(w io.Writer, n int) bool {
	if w == nil {
		return false
	}

	lf.submtx.Lock()
	if lf.submap == nil {
		lf.submap = make(map[io.Writer]struct{}, 4)
	}
	lf.submap[w] = struct{}{}
	lf.submtx.Unlock()

	if n == 0 {
		return true
	}

	of, err := os.OpenInRoot(lf.root, lf.name)
	if err != nil {
		return true
	}
	defer of.Close()

	if n > 0 {
		_ = lf.tailN(w, of, n)
	} else {
		_ = lf.tailAll(w, of)
	}

	return true
}

// Unsubscriber 取消订阅。
func (lf *limitedFile) Unsubscriber(w io.Writer) {
	if w == nil {
		return
	}

	lf.submtx.Lock()
	defer lf.submtx.Unlock()

	delete(lf.submap, w)
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
		lf.notifyAll(p) // 通知订阅者
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

func (lf *limitedFile) notifyAll(b []byte) {
	lf.submtx.RLock()
	defer lf.submtx.RUnlock()

	for w := range lf.submap {
		v := make([]byte, len(b))
		copy(v, b)
		_, _ = w.Write(v)
	}
}

func (lf *limitedFile) autoclose() {
	_ = lf.Close()
}

func (lf *limitedFile) Close() error {
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
	_ = lf.Close()

	root, err := os.OpenRoot(lf.root)
	if err != nil {
		return err
	}
	defer root.Close()

	return root.Remove(lf.name)
}

func (lf *limitedFile) tailAll(w io.Writer, r io.Reader) error {
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

func (lf *limitedFile) tailN(w io.Writer, f *os.File, n int) error {
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
