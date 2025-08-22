package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DailyLogWriter 是一个自定义的 io.Writer，它会根据日期创建和切换日志文件
type DailyLogWriter struct {
	logDir      string
	filename    string
	currentFile *os.File
	mu          sync.Mutex
	currentDate string
}

// NewDailyLogWriter 创建一个新的 DailyLogWriter 实例
func NewDailyLogWriter(logDir, baseFilename string) (*DailyLogWriter, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	w := &DailyLogWriter{
		logDir:   logDir,
		filename: baseFilename,
	}
	// 初始化文件
	if err := w.rotateFileIfNeeded(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write 实现 io.Writer 接口
func (w *DailyLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateFileIfNeeded(); err != nil {
		return 0, err
	}
	return w.currentFile.Write(p)
}

// rotateFileIfNeeded 检查日期是否改变，如果改变则关闭旧文件并创建新文件
func (w *DailyLogWriter) rotateFileIfNeeded() error {
	today := time.Now().Format("2006-01-02")
	if today != w.currentDate {
		if w.currentFile != nil {
			w.currentFile.Close() // 关闭旧文件
		}
		logFilename := fmt.Sprintf("%s-%s.log", w.filename, today)
		filePath := filepath.Join(w.logDir, logFilename)

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %w", filePath, err)
		}
		w.currentFile = file
		w.currentDate = today
	}
	return nil
}

// Close 关闭当前的日志文件
func (w *DailyLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}
