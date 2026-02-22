package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"
)

// AgentServiceAgentService Agent 服务接口
type AgentServiceAgentService interface {
	// ListAgents 列出所有 Agent
	ListAgents(ctx context.Context) ([]*AgentInfo, error)

	// GetAgent 获取指定 Agent 的内容
	GetAgent(ctx context.Context, fileName string) (*AgentDTO, error)

	// SaveAgent 保存 Agent 定义
	SaveAgent(ctx context.Context, fileName, content, source string, restoreFrom *int) (*SaveResultDTO, error)

	// GetVersions 获取 Agent 的版本历史
	GetVersions(ctx context.Context, fileName string) ([]*Version, error)

	// GetVersionContent 获取指定版本的完整内容
	GetVersionContent(ctx context.Context, fileName string, version int) (*VersionContentDTO, error)

	// RestoreVersion 从历史版本恢复 Agent
	RestoreVersion(ctx context.Context, fileName string, version int) (*SaveResultDTO, error)

	// DeleteVersion 删除指定历史版本
	DeleteVersion(ctx context.Context, fileName string, version int) error

	// DeleteVersions 批量删除历史版本
	DeleteVersions(ctx context.Context, fileName string, versions []int) error

	// RecordFileChange 记录文件变更（由文件监听器调用）
	RecordFileChange(ctx context.Context, fileName, content string) error
}

// AgentService Agent 服务实现
type AgentService struct {
	versionRepo repository.AgentVersionRepository
	agentsDir   string
}

// NewAgentService 创建 Agent 服务
func NewAgentService(versionRepo repository.AgentVersionRepository, agentsDir string) AgentServiceAgentService {
	return &AgentService{
		versionRepo: versionRepo,
		agentsDir:   agentsDir,
	}
}

// AgentInfo Agent 信息（用于列表展示）
type AgentInfo struct {
	FileName   string `json:"file_name"`
	Name       string `json:"name"`
	Description string `json:"description"`
}

// AgentDTO Agent 数据传输对象
type AgentDTO struct {
	FileName       string `json:"file_name"`
	Content        string `json:"content"`
	CurrentVersion int    `json:"current_version"`
}

// SaveResultDTO 保存结果
type SaveResultDTO struct {
	FileName    string    `json:"file_name"`
	Version     int       `json:"version"`
	SavedAt     string    `json:"saved_at"`
	RestoredFrom *int      `json:"restored_from,omitempty"`
}

// Version 版本信息
type Version struct {
	ID                int    `json:"id"`
	Version           int    `json:"version"`
	SavedAt           string `json:"saved_at"`
	Source            string `json:"source"`
	RestoreFromVersion *int   `json:"restore_from_version,omitempty"`
}

// VersionContentDTO 版本内容（用于 diff 展示）
type VersionContentDTO struct {
	FileName string `json:"file_name"`
	Version  int    `json:"version"`
	Content  string `json:"content"`
}

// ListAgents 列出所有 Agent
func (s *AgentService) ListAgents(ctx context.Context) ([]*AgentInfo, error) {
	// 读取 agents 目录
	entries, err := os.ReadDir(s.agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	var agents []*AgentInfo
	for _, entry := range entries {
		// 只处理 .yaml 文件
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// 解析 YAML 文件获取 Agent 信息
		filePath := filepath.Join(s.agentsDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			klog.V(6).Infof("[AgentService] Failed to read file %s: %v", entry.Name(), err)
			continue
		}

		agentInfo, err := s.parseAgentInfo(entry.Name(), content)
		if err != nil {
			klog.V(6).Infof("[AgentService] Failed to parse agent %s: %v", entry.Name(), err)
			continue
		}

		agents = append(agents, agentInfo)
	}

	return agents, nil
}

// GetAgent 获取指定 Agent 的内容
func (s *AgentService) GetAgent(ctx context.Context, fileName string) (*AgentDTO, error) {
	filePath := filepath.Join(s.agentsDir, fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent file: %w", err)
	}

	// 获取最新版本号
	latest, err := s.versionRepo.GetLatestVersion(ctx, fileName)
	if err != nil {
		if err == repository.ErrAgentVersionNotFound {
			// 没有版本记录，返回版本 0
			return &AgentDTO{
				FileName:       fileName,
				Content:        string(content),
				CurrentVersion: 0,
			}, nil
		}
		return nil, err
	}

	return &AgentDTO{
		FileName:       fileName,
		Content:        string(content),
		CurrentVersion: latest.Version,
	}, nil
}

// SaveAgent 保存 Agent 定义
func (s *AgentService) SaveAgent(ctx context.Context, fileName, content, source string, restoreFrom *int) (*SaveResultDTO, error) {
	// 获取下一个版本号
	nextVersion, err := s.versionRepo.GetNextVersion(ctx, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version: %w", err)
	}

	// 先写入临时文件（保证原子性）
	filePath := filepath.Join(s.agentsDir, fileName)
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// 原子性重命名
	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath) // 清理临时文件
		return nil, fmt.Errorf("failed to rename file: %w", err)
	}

	// 创建版本记录
	now := time.Now()
	version := &model.AgentVersion{
		FileName:          fileName,
		Content:           content,
		Version:           nextVersion,
		SavedAt:           now,
		Source:            source,
		RestoreFromVersion: restoreFrom,
		CreatedAt:         now,
	}

	if err := s.versionRepo.Create(ctx, version); err != nil {
		klog.Errorf("[AgentService] Failed to create version record: %v", err)
		// 文件已保存成功，版本记录失败不影响文件内容
	}

	klog.V(6).Infof("[AgentService] Saved agent %s, version: %d, source: %s", fileName, nextVersion, source)

	return &SaveResultDTO{
		FileName:    fileName,
		Version:     nextVersion,
		SavedAt:     now.Format("2006-01-02T15:04:05Z"),
		RestoredFrom: restoreFrom,
	}, nil
}

// GetVersions 获取 Agent 的版本历史
func (s *AgentService) GetVersions(ctx context.Context, fileName string) ([]*Version, error) {
	versions, err := s.versionRepo.GetVersionsByFileName(ctx, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	// 转换为 DTO 格式
	result := make([]*Version, 0, len(versions))
	for _, v := range versions {
		result = append(result, &Version{
			ID:                int(v.ID),
			Version:           v.Version,
			SavedAt:           v.SavedAt.Format("2006-01-02T15:04:05Z"),
			Source:            v.Source,
			RestoreFromVersion: v.RestoreFromVersion,
		})
	}
	return result, nil
}

// GetVersionContent 获取指定版本的完整内容
func (s *AgentService) GetVersionContent(ctx context.Context, fileName string, version int) (*VersionContentDTO, error) {
	agentVersion, err := s.versionRepo.GetVersion(ctx, fileName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return &VersionContentDTO{
		FileName: fileName,
		Version:  version,
		Content:  agentVersion.Content,
	}, nil
}

// RestoreVersion 从历史版本恢复 Agent
func (s *AgentService) RestoreVersion(ctx context.Context, fileName string, version int) (*SaveResultDTO, error) {
	// 获取指定版本的记录
	agentVersion, err := s.versionRepo.GetVersion(ctx, fileName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// 保存为当前版本（source = web, restoreFrom = version）
	return s.SaveAgent(ctx, fileName, agentVersion.Content, "web", &version)
}

// DeleteVersion 删除指定历史版本
func (s *AgentService) DeleteVersion(ctx context.Context, fileName string, version int) error {
	err := s.versionRepo.DeleteVersion(ctx, fileName, version)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}
	klog.V(6).Infof("[AgentService] Deleted version %d of file %s", version, fileName)
	return nil
}

// DeleteVersions 批量删除历史版本
func (s *AgentService) DeleteVersions(ctx context.Context, fileName string, versions []int) error {
	if len(versions) == 0 {
		return nil
	}
	err := s.versionRepo.DeleteVersions(ctx, fileName, versions)
	if err != nil {
		return fmt.Errorf("failed to delete versions: %w", err)
	}
	klog.V(6).Infof("[AgentService] Deleted %d versions of file %s", len(versions), fileName)
	return nil
}

// RecordFileChange 记录文件变更（由文件监听器调用）
func (s *AgentService) RecordFileChange(ctx context.Context, fileName, content string) error {
	// 检查文件内容是否与最新版本相同
	latest, err := s.versionRepo.GetLatestVersion(ctx, fileName)
	if err == nil && latest.Content == content {
		// 内容相同，不需要创建新版本
		klog.V(6).Infof("[AgentService] File %s content unchanged, skip version recording", fileName)
		return nil
	}

	// 创建新版本记录（source = file_change）
	_, err = s.SaveAgent(ctx, fileName, content, "file_change", nil)
	return err
}

// parseAgentInfo 从 YAML 内容解析 Agent 信息
func (s *AgentService) parseAgentInfo(fileName string, content []byte) (*AgentInfo, error) {
	var agent struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	if err := yaml.Unmarshal(content, &agent); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &AgentInfo{
		FileName:   fileName,
		Name:       agent.Name,
		Description: agent.Description,
	}, nil
}
