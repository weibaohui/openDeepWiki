package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/klog/v2"

	"github.com/weibaohui/opendeepwiki/backend/config"
	"github.com/weibaohui/opendeepwiki/backend/internal/domain"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/pkg/git"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/orchestrator"
	"github.com/weibaohui/opendeepwiki/backend/internal/service/statemachine"
)

type RepositoryService struct {
	cfg          *config.Config
	repoRepo     repository.RepoRepository
	taskRepo     repository.TaskRepository
	docRepo      repository.DocumentRepository
	taskHintRepo repository.HintRepository
	docService   *DocumentService
	taskService  *TaskService

	// 状态机
	repoStateMachine *statemachine.RepositoryStateMachine
	taskStateMachine *statemachine.TaskStateMachine

	// 编排器
	orchestrator *orchestrator.Orchestrator

	// 目录分析服务
	dirMakerService DirMakerService
	dbModelWriter   domain.Writer
	apiWriter       domain.Writer
	problemAnalyzer domain.Writer
	titleRewriter   domain.Writer
}

// DirMakerService 目录分析服务接口。
type DirMakerService interface {
	CreateDirs(ctx context.Context, repo *model.Repository) (*model.DirMakerGenerationResult, error)
}

type APIAnalyzer interface {
	Generate(ctx context.Context, localPath string, title string, repoID uint, taskID uint) (string, error)
}

type TitleRewriter interface {
	RewriteTitle(ctx context.Context, docID uint) (string, string, bool, error)
}

// NewRepositoryService 创建仓库服务实例。
func NewRepositoryService(cfg *config.Config, repoRepo repository.RepoRepository, taskRepo repository.TaskRepository, docRepo repository.DocumentRepository, taskHintRepo repository.HintRepository, taskService *TaskService, dirMakerService DirMakerService, docService *DocumentService, dbModelParser domain.Writer, apiWriter domain.Writer, problemAnalyzer domain.Writer, titleRewriter domain.Writer) *RepositoryService {
	return &RepositoryService{
		cfg:              cfg,
		repoRepo:         repoRepo,
		taskRepo:         taskRepo,
		docRepo:          docRepo,
		taskHintRepo:     taskHintRepo,
		docService:       docService,
		taskService:      taskService,
		repoStateMachine: statemachine.NewRepositoryStateMachine(),
		taskStateMachine: statemachine.NewTaskStateMachine(),
		orchestrator:     orchestrator.GetGlobalOrchestrator(),
		dirMakerService:  dirMakerService,
		dbModelWriter:    dbModelParser,
		apiWriter:        apiWriter,
		problemAnalyzer:  problemAnalyzer,
		titleRewriter:    titleRewriter,
	}
}

type CreateRepoRequest struct {
	URL string `json:"url" binding:"required"`
}

var (
	ErrInvalidRepositoryURL          = errors.New("invalid repository url")
	ErrRepositoryAlreadyExists       = errors.New("repository already exists")
	ErrCannotDeleteRepoInvalidStatus = errors.New("无法删除仓库：已完成或正在分析中的仓库不能删除")
)

// Create 创建仓库并初始化任务
func (s *RepositoryService) Create(req CreateRepoRequest) (*model.Repository, error) {
	normalizedURL, repoKey, err := git.NormalizeRepoURL(req.URL)
	if err != nil {
		klog.V(6).Infof("仓库URL校验失败: url=%s, error=%v", req.URL, err)
		return nil, ErrInvalidRepositoryURL
	}

	existingRepos, err := s.repoRepo.List()
	if err != nil {
		return nil, fmt.Errorf("获取仓库列表失败: %w", err)
	}
	for _, existing := range existingRepos {
		_, existingKey, parseErr := git.NormalizeRepoURL(existing.URL)
		if parseErr != nil {
			klog.V(6).Infof("已有仓库URL无法解析，跳过去重: repoID=%d, url=%s, error=%v", existing.ID, existing.URL, parseErr)
			continue
		}
		if existingKey == repoKey {
			klog.V(6).Infof("仓库已存在，拒绝重复添加: repoID=%d, url=%s", existing.ID, normalizedURL)
			return nil, ErrRepositoryAlreadyExists
		}
	}

	// 生成仓库名称和本地路径
	repoName := git.ParseRepoName(normalizedURL)
	localPath := filepath.Join(s.cfg.Data.RepoDir, repoName+"-"+fmt.Sprintf("%d", time.Now().Unix()))

	// 创建仓库（初始状态为pending）
	repo := &model.Repository{
		Name:      repoName,
		URL:       normalizedURL,
		LocalPath: localPath,
		Status:    string(statemachine.RepoStatusPending),
	}

	if err := s.repoRepo.Create(repo); err != nil {
		return nil, fmt.Errorf("创建仓库失败: %w", err)
	}

	klog.V(6).Infof("仓库创建成功: repoID=%d, name=%s, url=%s", repo.ID, repo.Name, repo.URL)

	// 异步克隆仓库
	go s.cloneRepository(repo.ID)

	return repo, nil
}

// List 获取所有仓库
func (s *RepositoryService) List() ([]model.Repository, error) {
	return s.repoRepo.List()
}

// Get 获取单个仓库（包含任务和文档）
func (s *RepositoryService) Get(id uint) (*model.Repository, error) {
	return s.repoRepo.Get(id)
}

// Delete 删除仓库
func (s *RepositoryService) Delete(id uint) error {
	// 获取仓库基本信息
	repo, err := s.repoRepo.GetBasic(id)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 校验仓库状态：已完成或正在分析中的仓库不能删除
	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if currentStatus == statemachine.RepoStatusCompleted || currentStatus == statemachine.RepoStatusAnalyzing {
		klog.V(6).Infof("拒绝删除仓库：状态不允许删除: repoID=%d, status=%s", id, currentStatus)
		return ErrCannotDeleteRepoInvalidStatus
	}

	// 删除本地仓库文件
	if repo.LocalPath != "" {
		klog.V(6).Infof("删除本地仓库: repoID=%d, localPath=%s", id, repo.LocalPath)
		if err := git.RemoveRepo(repo.LocalPath); err != nil {
			klog.Warningf("删除本地仓库失败: repoID=%d, error=%v", id, err)
		}
	}

	// TODO 删除数据库记录（使用事务）
	if err := s.docRepo.DeleteByRepositoryID(id); err != nil {
		return fmt.Errorf("删除文档失败: %w", err)
	}
	if err := s.taskRepo.DeleteByRepositoryID(id); err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	if err := s.repoRepo.Delete(id); err != nil {
		return fmt.Errorf("删除仓库失败: %w", err)
	}

	klog.V(6).Infof("仓库删除成功: repoID=%d", id)
	return nil
}

func (s *RepositoryService) PurgeLocalDir(id uint) error {
	repo, err := s.repoRepo.GetBasic(id)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	currentStatus := statemachine.RepositoryStatus(repo.Status)
	if currentStatus == statemachine.RepoStatusCloning || currentStatus == statemachine.RepoStatusAnalyzing {
		return fmt.Errorf("仓库状态不允许删除本地目录: current=%s", currentStatus)
	}

	if repo.LocalPath != "" {
		klog.V(6).Infof("仅删除本地仓库目录: repoID=%d, localPath=%s", id, repo.LocalPath)
		if err := git.RemoveRepo(repo.LocalPath); err != nil {
			klog.Warningf("删除本地仓库目录失败: repoID=%d, error=%v", id, err)
			return fmt.Errorf("删除本地仓库目录失败: %w", err)
		}
		repo.LocalPath = ""
		if err := s.repoRepo.Save(repo); err != nil {
			klog.Errorf("更新仓库记录失败: repoID=%d, error=%v", id, err)
			return fmt.Errorf("更新仓库记录失败: %w", err)
		}
		klog.V(6).Infof("本地仓库目录已删除并清空记录: repoID=%d", id)
	} else {
		klog.V(6).Infof("本地目录为空，跳过删除: repoID=%d", id)
	}

	return nil
}

// SetReady 将仓库状态设置为就绪（用于调试或特殊场景）
func (s *RepositoryService) SetReady(repoID uint) error {
	klog.V(6).Infof("准备将仓库状态设置为就绪: repoID=%d", repoID)

	// 获取仓库
	repo, err := s.repoRepo.GetBasic(repoID)
	if err != nil {
		return fmt.Errorf("获取仓库失败: %w", err)
	}

	// 直接将状态设置为 ready，不进行状态机验证
	// 注意：此功能仅用于调试或特殊场景，可能导致状态不一致
	repo.Status = string(statemachine.RepoStatusReady)
	repo.ErrorMsg = ""

	if err := s.repoRepo.Save(repo); err != nil {
		klog.Errorf("更新仓库状态失败: repoID=%d, error=%v", repoID, err)
		return fmt.Errorf("更新仓库状态失败: %w", err)
	}

	klog.V(6).Infof("仓库状态已设置为 ready: repoID=%d", repoID)
	return nil
}
