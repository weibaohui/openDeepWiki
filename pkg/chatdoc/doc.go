package chatdoc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/weibaohui/openDeepWiki/internal/dao"
	"github.com/weibaohui/openDeepWiki/pkg/comm/utils"
	"github.com/weibaohui/openDeepWiki/pkg/constants"
	"github.com/weibaohui/openDeepWiki/pkg/models"
	"github.com/weibaohui/openDeepWiki/pkg/service"
	"k8s.io/klog/v2"
)

type docService struct {
	repo     *models.Repo
	analysis *models.DocAnalysis
}

// NewDocService 创建并返回一个基于给定仓库的 docService 实例。
func NewDocService(repo *models.Repo) *docService {

	ds := &docService{
		repo: repo,
	}

	return ds
}

// NewDocServiceWithRepoID 根据仓库ID创建并返回一个 docService 实例。
// 如果找不到对应ID的仓库或发生其他错误，将返回错误。
func NewDocServiceWithRepoID(repoID string) *docService {
	// 将 repoID 转换为 uint 类型
	repoIDInt := utils.ToUInt(repoID)
	if repoIDInt == 0 {
		klog.Errorf("解析仓库ID失败")
		return nil
	}

	// 查询仓库信息
	repo := &models.Repo{}
	if err := dao.DB().First(repo, repoIDInt).Error; err != nil {
		klog.Errorf("查询仓库信息失败: %v", err)
		return nil
	}

	// 创建并返回 docService 实例
	return NewDocService(repo)
}

// NewDocServiceWithAnalysisID 根据分析ID创建并返回一个 docService 实例。
// 如果找不到对应ID的分析记录或发生其他错误，将返回 nil。
func NewDocServiceWithAnalysisID(analysisID string) *docService {
	// 将 analysisID 转换为 uint 类型
	analysisIDInt := utils.ToUInt(analysisID)
	if analysisIDInt == 0 {
		klog.Errorf("解析分析ID失败")
		return nil
	}

	// 查询分析记录
	analysis := &models.DocAnalysis{}
	if err := dao.DB().First(analysis, analysisIDInt).Error; err != nil {
		klog.Errorf("查询分析记录失败: %v", err)
		return nil
	}

	// 查询仓库信息
	repo := &models.Repo{}
	if err := dao.DB().First(repo, analysis.RepoID).Error; err != nil {
		klog.Errorf("查询仓库信息失败: %v", err)
		return nil
	}

	// 创建并返回 docService 实例
	service := NewDocService(repo)
	service.SetAnalysis(analysis)
	return service
}

func (s *docService) MustBeInitialized() error {
	if s.analysis == nil {
		return fmt.Errorf("analysis is nil")
	}
	if s.repo == nil {
		return fmt.Errorf("repo is nil")
	}
	return nil
}

func (s *docService) MustHaveAnalysisInstance() error {
	if s.analysis == nil {
		return fmt.Errorf("analysis is nil")
	}

	return nil
}
func (s *docService) chat(ctx context.Context, systemPrompt, initMessage string) (io.Reader, error) {
	ctx = context.WithValue(ctx, constants.SystemPrompt, systemPrompt)
	ctx = context.WithValue(ctx, constants.RepoName, s.repo.Name)
	// 创建一个带有读写功能的管道
	pr, pw := io.Pipe()

	// 启动一个goroutine来处理AI服务的输出
	go func() {
		defer pw.Close()
		// 调用AI服务处理消息，将输出写入管道
		err := service.ChatService().RunOneRound(ctx, initMessage, pw)
		if err != nil {
			klog.Errorf("AI处理消息失败: %v", err)
			return
		}
	}()

	return pr, nil
}
func (s *docService) agentChat(ctx context.Context, initMessage string) (io.Reader, error) {
	ctx = context.WithValue(ctx, constants.RepoName, s.repo.Name)
	// 创建一个带有读写功能的管道
	pr, pw := io.Pipe()

	// 启动一个goroutine来处理AI服务的输出
	go func() {
		defer pw.Close()
		// 调用AI服务处理消息，将输出写入管道
		err := service.ChatService().RunOneRound(ctx, initMessage, pw)
		if err != nil {
			klog.Errorf("AI处理消息失败: %v", err)
			return
		}
	}()

	return pr, nil
}

// GetLatestLogs 获取最新的日志内容
func (s *docService) GetLatestLogs(ctx context.Context) (string, error) {
	if s.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	runtimeDir, err := utils.EnsureRuntimeDir(s.repo.Name)
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

// readAndWrite 从reader读取数据并同时写入文件
func (s *docService) readAndWrite(ctx context.Context, reader io.Reader) (string, error) {
	if s.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	// 获取运行时文件路径
	filePath, err := s.GetRuntimeFilePath()
	if err != nil {
		return "", fmt.Errorf("failed to get runtime file path: %v", err)
	}

	// 创建文件
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	var all string
	// 创建一个缓冲区用于临时存储读取的数据
	buf := make([]byte, 1024)

	// 持续读取输出并写入文件
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			content := string(buf[:n])
			all += content

			// 写入文件
			if _, err := f.WriteString(content); err != nil {
				klog.Errorf("写入文件失败: %v", err)
				break
			}
			klog.V(8).Infof("成功写入文件 %s %d 长度字符串", filePath, len(content))

			// 输出到控制台
			// klog.V(6).Infof("AI响应: %s", content)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			klog.Errorf("读取AI响应失败: %v", err)
			break
		}
	}

	klog.Infof("会话结束，对话结果已写入文件 %s", filePath)
	return all, nil
}

// TailFile 持续读取文件新增内容，并将每一行通过 channel 返回
func (s *docService) TailFile(ctx context.Context) (<-chan string, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("repository not initialized")
	}

	filePath, err := s.GetRuntimeFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime file path: %v", err)
	}

	updates := make(chan string)

	go func() {
		defer close(updates)
		file, err := os.Open(filePath)
		if err != nil {
			klog.Errorf("打开文件失败: %v", err)
			return
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		cache := "" // 用于缓存未遇到换行符的数据
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						if line != "" {
							cache += line
						}
						time.Sleep(1000 * time.Millisecond)
						continue
					}
					klog.Errorf("读取文件失败: %v", err)
					return
				}
				cache += line
				if len(cache) > 0 && cache[len(cache)-1] == '\n' {
					updates <- cache
					cache = ""
				}
			}
		}
	}()

	return updates, nil
}

// GetRuntimeFilePath 获取运行时文件的完整路径
// 格式：AnalysisID/Chat_2023-10-01_12-00-00.log
func (s *docService) GetRuntimeFilePath() (string, error) {

	if err := s.MustBeInitialized(); err != nil {
		return "", err
	}

	runtimeDir, err := utils.EnsureRuntimeDir(s.repo.Name)
	if err != nil {
		return "", err
	}

	// 创建分析ID子目录
	analysisDir := filepath.Join(runtimeDir, fmt.Sprintf("%d", s.analysis.ID))
	if err := os.MkdirAll(analysisDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create analysis directory: %v", err)
	}

	// 基于分析任务的开始时间生成文件名
	filename := fmt.Sprintf("chat_%s.log", s.analysis.StartTime.Format("20060102_150405"))
	return filepath.Join(analysisDir, filename), nil
}
func (s *docService) GetRuntimeFolder() (string, error) {

	if err := s.MustBeInitialized(); err != nil {
		return "", err
	}

	runtimeDir, err := utils.EnsureRuntimeDir(s.repo.Name)
	if err != nil {
		return "", err
	}

	// 创建分析ID子目录
	analysisDir := filepath.Join(runtimeDir, fmt.Sprintf("%d", s.analysis.ID))
	if err := os.MkdirAll(analysisDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create analysis directory: %v", err)
	}
	return analysisDir, nil
}

// GetLatestLogFile 获取最新的日志文件名
func (s *docService) GetLatestLogFile(ctx context.Context) (string, error) {
	if s.repo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	runtimeDir, err := utils.EnsureRuntimeDir(s.repo.Name)
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

func (s *docService) SetAnalysis(analysis *models.DocAnalysis) {
	s.analysis = analysis
}
