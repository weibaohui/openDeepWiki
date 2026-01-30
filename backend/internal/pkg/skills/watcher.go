package skills

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileEvent 文件事件
type FileEvent struct {
	Type string // create, modify, delete
	Path string
	Info os.FileInfo
}

// FileWatcher 文件监听器
type FileWatcher struct {
	dir      string
	interval time.Duration
	stop     chan struct{}
	callback func(event FileEvent)
	files    map[string]os.FileInfo
}

// NewFileWatcher 创建文件监听器
func NewFileWatcher(dir string, interval time.Duration, callback func(event FileEvent)) *FileWatcher {
	return &FileWatcher{
		dir:      dir,
		interval: interval,
		stop:     make(chan struct{}),
		callback: callback,
		files:    make(map[string]os.FileInfo),
	}
}

// Start 启动监听
func (w *FileWatcher) Start() error {
	// 初始扫描
	if err := w.scan(); err != nil {
		return err
	}

	// 定时扫描
	ticker := time.NewTicker(w.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := w.scan(); err != nil {
					log.Printf("Failed to scan skills directory: %v", err)
				}
			case <-w.stop:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// Stop 停止监听
func (w *FileWatcher) Stop() {
	close(w.stop)
}

// scan 扫描目录变化
func (w *FileWatcher) scan() error {
	// 检查目录是否存在
	if _, err := os.Stat(w.dir); os.IsNotExist(err) {
		// 目录不存在，清空记录
		for path, info := range w.files {
			w.callback(FileEvent{Type: "delete", Path: path, Info: info})
		}
		w.files = make(map[string]os.FileInfo)
		return nil
	}

	currentFiles := make(map[string]os.FileInfo)

	// 读取目录
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 跳过隐藏目录
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		path := filepath.Join(w.dir, entry.Name())

		// 检查是否是 Skill 目录
		skillMDPath := filepath.Join(path, "SKILL.md")
		info, err := os.Stat(skillMDPath)
		if err != nil {
			continue // 不是 Skill 目录
		}

		currentFiles[path] = info
	}

	// 检测新增和修改
	for path, info := range currentFiles {
		oldInfo, exists := w.files[path]
		if !exists {
			w.callback(FileEvent{Type: "create", Path: path, Info: info})
		} else if info.ModTime() != oldInfo.ModTime() || info.Size() != oldInfo.Size() {
			w.callback(FileEvent{Type: "modify", Path: path, Info: info})
		}
	}

	// 检测删除
	for path, info := range w.files {
		if _, exists := currentFiles[path]; !exists {
			w.callback(FileEvent{Type: "delete", Path: path, Info: info})
		}
	}

	w.files = currentFiles
	return nil
}

// WatchSkillMD 监听特定 Skill 的 SKILL.md 文件变化
func (w *FileWatcher) WatchSkillMD(skillPath string, onChange func()) {
	skillMDPath := filepath.Join(skillPath, "SKILL.md")

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		var lastModTime time.Time

		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(skillMDPath)
				if err != nil {
					continue
				}

				if !lastModTime.IsZero() && info.ModTime() != lastModTime {
					onChange()
				}
				lastModTime = info.ModTime()

			case <-w.stop:
				return
			}
		}
	}()
}
