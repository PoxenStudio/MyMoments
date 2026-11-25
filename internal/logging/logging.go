// Package logging 提供服务端日志的文件落盘与按大小轮转。
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// FileName 是当前日志文件名。
	FileName = "moments.log"
	// MaxSizeBytes 是单个日志文件的上限（5MB）。
	MaxSizeBytes = 5 << 20
	// MaxBackups 是保留的归档数量，加上当前文件共 7 个。
	MaxBackups = 6
)

// RotatingWriter 实现 io.Writer，按大小切分日志文件并保留固定数量的归档。
// 归档命名为 moments.1.log（最新）到 moments.N.log（最旧）。
type RotatingWriter struct {
	dir      string
	maxSize  int64
	maxFiles int

	mu   sync.Mutex
	file *os.File
	size int64
}

// NewRotatingWriter 创建日志目录并打开当前日志文件。
func NewRotatingWriter(dir string, maxSize int64, maxFiles int) (*RotatingWriter, error) {
	if dir == "" {
		return nil, fmt.Errorf("log dir is empty")
	}
	if maxSize <= 0 {
		return nil, fmt.Errorf("log max size must be positive")
	}
	if maxFiles < 1 {
		maxFiles = 1
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	w := &RotatingWriter{dir: dir, maxSize: maxSize, maxFiles: maxFiles}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingWriter) currentPath() string { return filepath.Join(w.dir, FileName) }

func (w *RotatingWriter) backupPath(index int) string {
	return filepath.Join(w.dir, fmt.Sprintf("moments.%d.log", index))
}

func (w *RotatingWriter) open() error {
	f, err := os.OpenFile(w.currentPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	w.file = f
	w.size = info.Size()
	return nil
}

// Write 写入日志，超限时先轮转。
func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	if w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingWriter) rotateLocked() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log file: %w", err)
		}
		w.file = nil
	}
	oldest := w.maxFiles - 1
	if oldest >= 1 {
		_ = os.Remove(w.backupPath(oldest))
		for i := oldest - 1; i >= 1; i-- {
			_ = os.Rename(w.backupPath(i), w.backupPath(i+1))
		}
		_ = os.Rename(w.currentPath(), w.backupPath(1))
	} else {
		_ = os.Remove(w.currentPath())
	}
	return w.open()
}

// Close 关闭当前日志文件。
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
