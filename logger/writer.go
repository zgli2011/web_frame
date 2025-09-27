package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type RotatingFileWriter struct {
	filePath   string
	fileName   string
	maxSize    int64 // bytes
	maxBackups int
	currentFile *os.File
	currentSize int64
	mu          sync.Mutex
}

func NewRotatingFileWriter(filePath, fileName string, maxSizeMB, maxBackups int) (*RotatingFileWriter, error) {
	if err := os.MkdirAll(filePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	writer := &RotatingFileWriter{
		filePath:   filePath,
		fileName:   fileName,
		maxSize:    int64(maxSizeMB) * 1024 * 1024,
		maxBackups: maxBackups,
	}

	if err := writer.openFile(); err != nil {
		return nil, err
	}

	return writer, nil
}

func (w *RotatingFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	if w.currentSize+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = w.currentFile.Write(p)
	if err != nil {
		return n, err
	}

	w.currentSize += int64(n)
	return n, nil
}

func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

func (w *RotatingFileWriter) openFile() error {
	fullPath := filepath.Join(w.filePath, w.fileName)

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to get file info: %v", err)
	}

	w.currentFile = file
	w.currentSize = info.Size()

	return nil
}

func (w *RotatingFileWriter) rotate() error {
	if w.currentFile != nil {
		w.currentFile.Close()
		w.currentFile = nil
	}

	currentPath := filepath.Join(w.filePath, w.fileName)

	backupPath := fmt.Sprintf("%s.1", currentPath)
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %v", err)
	}

	w.cleanupOldBackups()

	return w.openFile()
}

func (w *RotatingFileWriter) cleanupOldBackups() {
	pattern := filepath.Join(w.filePath, w.fileName+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	var backups []string
	for _, match := range matches {
		base := filepath.Base(match)
		if strings.HasPrefix(base, w.fileName+".") {
			suffix := strings.TrimPrefix(base, w.fileName+".")
			if _, err := strconv.Atoi(suffix); err == nil {
				backups = append(backups, match)
			}
		}
	}

	if len(backups) <= w.maxBackups {
		return
	}

	sort.Slice(backups, func(i, j int) bool {
		iSuffix := strings.TrimPrefix(filepath.Base(backups[i]), w.fileName+".")
		jSuffix := strings.TrimPrefix(filepath.Base(backups[j]), w.fileName+".")
		iNum, _ := strconv.Atoi(iSuffix)
		jNum, _ := strconv.Atoi(jSuffix)
		return iNum > jNum
	})

	for i := w.maxBackups; i < len(backups); i++ {
		os.Remove(backups[i])
	}

	for i := w.maxBackups - 1; i >= 1; i-- {
		oldPath := filepath.Join(w.filePath, fmt.Sprintf("%s.%d", w.fileName, i))
		newPath := filepath.Join(w.filePath, fmt.Sprintf("%s.%d", w.fileName, i+1))
		os.Rename(oldPath, newPath)
	}
}

type MultiWriter struct {
	writers []io.Writer
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, writer := range mw.writers {
		n, err = writer.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}