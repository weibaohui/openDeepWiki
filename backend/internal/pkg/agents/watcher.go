package agents

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileWatcher 文件监听器
type FileWatcher struct {
	dir      string
	interval time.Duration
	callback func(FileEvent)
	stop     chan bool
	states   map[string]os.FileInfo
}

// NewFileWatcher 创建文件监听器
func NewFileWatcher(dir string, interval time.Duration, callback func(FileEvent)) *FileWatcher {
	return &FileWatcher{
		dir:      dir,
		interval: interval,
		callback: callback,
		stop:     make(chan bool),
		states:   make(map[string]os.FileInfo),
	}
}

// Start 启动监听
func (w *FileWatcher) Start() error {
	go w.watch()
	return nil
}

// Stop 停止监听
func (w *FileWatcher) Stop() {
	select {
	case <-w.stop:
		// 已经关闭
		return
	default:
		close(w.stop)
	}
}

// watch 监听循环
func (w *FileWatcher) watch() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 初始状态
	w.scan()

	for {
		select {
		case <-ticker.C:
			w.scan()
		case <-w.stop:
			return
		}
	}
}

// scan 扫描目录变化
func (w *FileWatcher) scan() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}

	current := make(map[string]os.FileInfo)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// 只监控 .yaml, .yml, .json 文件
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		path := filepath.Join(w.dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		current[path] = info

		// 检查变化
		if old, exists := w.states[path]; !exists {
			// 新建
			w.callback(FileEvent{Type: "create", Path: path})
		} else if info.ModTime() != old.ModTime() || info.Size() != old.Size() {
			// 修改
			w.callback(FileEvent{Type: "modify", Path: path})
		}
	}

	// 检查删除
	for path := range w.states {
		if _, exists := current[path]; !exists {
			w.callback(FileEvent{Type: "delete", Path: path})
		}
	}

	w.states = current
}
