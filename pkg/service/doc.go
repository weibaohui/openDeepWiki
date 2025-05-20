package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"k8s.io/klog/v2"
)

type docService struct {
	repo *models.Repo
}

// NewDocService 创建并返回一个基于给定仓库的 docService 实例。
func NewDocService(repo *models.Repo) *docService {
	return &docService{
		repo: repo,
	}
}

func (d *docService) chat(ctx context.Context, message string) (io.Reader, error) {
	// 创建一个带有读写功能的管道
	pr, pw := io.Pipe()

	// 启动一个goroutine来处理AI服务的输出
	go func() {
		defer pw.Close()
		// 调用AI服务处理消息，将输出写入管道
		err := ChatService().RunOneRound(ctx, message, pw)
		if err != nil {
			klog.Errorf("AI处理消息失败: %v", err)
			return
		}
	}()

	return pr, nil
}

func (d *docService) writeFile(ctx context.Context, s string) error {
	if d.repo == nil {
		return fmt.Errorf("repository not initialized")
	}

	// 生成文件名（使用时间戳确保唯一性）
	filename := fmt.Sprintf("chat_%s.log", time.Now().Format("20060102_150405"))

	// 获取运行时文件路径
	filePath, err := utils.GetRuntimeFilePath(d.repo.Name, filename)
	if err != nil {
		return fmt.Errorf("failed to get runtime file path: %v", err)
	}

	// 创建文件
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	// 写入内容
	if _, err := f.WriteString(s + "\n"); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	klog.Infof("成功写入文件 %s", filePath)
	return nil
}

// GetLatestLogs 获取最新的日志内容
func (d *docService) GetLatestLogs(ctx context.Context) (string, error) {
	if d.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	runtimeDir, err := utils.EnsureRuntimeDir(d.repo.Name)
	if err != nil {
		return "", err
	}

	// 读取目录下最新的日志文件
	files, err := os.ReadDir(runtimeDir)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	// 获取最新的日志文件
	var latestFile os.DirEntry
	var latestTime time.Time

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".log" {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		if latestFile == nil || fileInfo.ModTime().After(latestTime) {
			latestFile = file
			latestTime = fileInfo.ModTime()
		}
	}

	if latestFile == nil {
		return "", nil
	}

	// 读取文件内容
	content, err := os.ReadFile(filepath.Join(runtimeDir, latestFile.Name()))
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (d *docService) readAll(ctx context.Context, reader io.Reader) (string, error) {

	var all string
	// 创建一个缓冲区用于临时存储读取的数据
	buf := make([]byte, 1024)

	// 启动一个goroutine来持续读取输出
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			all += string(buf[:n])
			// 输出到控制台
			klog.V(6).Infof("AI响应: %s", string(buf[:n]))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			klog.Errorf("读取AI响应失败: %v", err)
			break
		}
	}
	return all, nil
}

// TailFile 持续读取文件新增内容
func (d *docService) TailFile(ctx context.Context, filename string) (<-chan string, error) {
	if d.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	filePath, err := utils.GetRuntimeFilePath(d.repo.Name, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime file path: %v", err)
	}

	// 创建一个用于发送文件更新的通道
	updates := make(chan string)

	go func() {
		defer close(updates)

		var lastSize int64 = 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 检查文件是否存在
				stat, err := os.Stat(filePath)
				if err != nil {
					if !os.IsNotExist(err) {
						klog.Errorf("检查文件状态失败: %v", err)
					}
					time.Sleep(time.Second)
					continue
				}

				currentSize := stat.Size()
				if currentSize > lastSize {
					// 打开文件
					file, err := os.Open(filePath)
					if err != nil {
						klog.Errorf("打开文件失败: %v", err)
						time.Sleep(time.Second)
						continue
					}

					// 从上次读取的位置开始
					_, err = file.Seek(lastSize, 0)
					if err != nil {
						file.Close()
						klog.Errorf("设置文件读取位置失败: %v", err)
						time.Sleep(time.Second)
						continue
					}

					// 读取新增内容
					buffer := make([]byte, currentSize-lastSize)
					n, err := file.Read(buffer)
					file.Close()

					if err != nil && err != io.EOF {
						klog.Errorf("读取文件失败: %v", err)
						time.Sleep(time.Second)
						continue
					}

					if n > 0 {
						updates <- string(buffer[:n])
						lastSize = currentSize
					}
				}

				// 短暂休眠，避免过度消耗CPU
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return updates, nil
}

// GetLatestLogFile 获取最新的日志文件名
func (d *docService) GetLatestLogFile(ctx context.Context) (string, error) {
	if d.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	runtimeDir, err := utils.EnsureRuntimeDir(d.repo.Name)
	if err != nil {
		return "", err
	}

	// 读取目录下的所有文件
	files, err := os.ReadDir(runtimeDir)
	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	// 获取最新的日志文件
	var latestFile os.DirEntry
	var latestTime time.Time

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".log" {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		if latestFile == nil || fileInfo.ModTime().After(latestTime) {
			latestFile = file
			latestTime = fileInfo.ModTime()
		}
	}

	if latestFile == nil {
		return "", nil
	}

	return latestFile.Name(), nil
}
