package adkagents

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileWatcher 文件变更监听器
type FileWatcher struct {
	dir      string
	interval time.Duration
	handler  func(FileEvent)

	stopCh chan struct{}
	wg     sync.WaitGroup

	// 记录文件状态（路径 -> 修改时间）
	files map[string]time.Time
	mutex sync.RWMutex
}

// NewFileWatcher 创建文件监听器
//
//	dir: 监听的目录
//	interval: 检查间隔
//	handler: 文件变更处理函数
func NewFileWatcher(dir string, interval time.Duration, handler func(FileEvent)) *FileWatcher {
	return &FileWatcher{
		dir:      dir,
		interval: interval,
		handler:  handler,
		stopCh:   make(chan struct{}),
		files:    make(map[string]time.Time),
	}
}

// Start 启动文件监听
func (w *FileWatcher) Start() error {
	// 初始扫描，记录文件状态
	if err := w.scan(); err != nil {
		return err
	}

	w.wg.Add(1)
	go w.run()

	log.Printf("[FileWatcher] Started watching directory: %s (interval: %v)", w.dir, w.interval)
	return nil
}

// Stop 停止文件监听
func (w *FileWatcher) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	log.Printf("[FileWatcher] Stopped watching directory: %s", w.dir)
}

// run 主循环
func (w *FileWatcher) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.scan(); err != nil {
				log.Printf("[FileWatcher] Scan error: %v", err)
			}
		}
	}
}

// scan 扫描目录，检测变更
func (w *FileWatcher) scan() error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，忽略
		}
		return err
	}

	// 当前扫描到的文件
	currentFiles := make(map[string]time.Time)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 只处理 .yaml 和 .yml 文件
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(w.dir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		currentFiles[path] = info.ModTime()
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	// 检测新增和修改
	for path, modTime := range currentFiles {
		lastModTime, exists := w.files[path]
		if !exists {
			// 新增文件
			w.handler(FileEvent{Type: "create", Path: path})
		} else if modTime.After(lastModTime) {
			// 修改文件（至少间隔 1 秒才认为真正修改，防抖）
			if time.Since(modTime) > time.Second {
				w.handler(FileEvent{Type: "modify", Path: path})
			}
		}
	}

	// 检测删除
	for path := range w.files {
		if _, exists := currentFiles[path]; !exists {
			w.handler(FileEvent{Type: "delete", Path: path})
		}
	}

	// 更新状态
	w.files = currentFiles

	return nil
}
