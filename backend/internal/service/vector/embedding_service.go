package vector

import (
	"context"
	"fmt"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	vector_domain "github.com/weibaohui/opendeepwiki/backend/internal/domain/vector"
	"k8s.io/klog/v2"
)

// VectorEmbeddingService 向量生成服务
type VectorEmbeddingService struct {
	provider       vector_domain.EmbeddingProvider
	vectorRepo     repository.VectorRepository
	vectorTaskRepo repository.VectorTaskRepository
	docRepo        repository.DocumentRepository
	taskQueue      chan uint
	workers        int
}

// NewVectorEmbeddingService 创建向量生成服务
func NewVectorEmbeddingService(
	provider vector_domain.EmbeddingProvider,
	vectorRepo repository.VectorRepository,
	vectorTaskRepo repository.VectorTaskRepository,
	docRepo repository.DocumentRepository,
	workers int,
) *VectorEmbeddingService {
	if workers <= 0 {
		workers = 2
	}

	return &VectorEmbeddingService{
		provider:       provider,
		vectorRepo:     vectorRepo,
		vectorTaskRepo: vectorTaskRepo,
		docRepo:        docRepo,
		taskQueue:      make(chan uint, 100), // 缓冲队列
		workers:        workers,
	}
}

// GenerateForDocument 为文档生成向量
func (s *VectorEmbeddingService) GenerateForDocument(ctx context.Context, docID uint) error {
	klog.V(6).Infof("VectorEmbeddingService: 开始为文档 %d 生成向量", docID)

	// 创建任务
	task := &model.VectorTask{
		DocumentID: docID,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	if err := s.vectorTaskRepo.Create(ctx, task); err != nil {
		klog.Warningf("VectorEmbeddingService: 创建任务失败: %v", err)
		return fmt.Errorf("create task: %w", err)
	}

	// 将任务放入队列
	s.taskQueue <- docID

	klog.V(6).Infof("VectorEmbeddingService: 文档 %d 已加入生成队列", docID)
	return nil
}

// GenerateForRepository 批量为仓库的所有文档生成向量
func (s *VectorEmbeddingService) GenerateForRepository(ctx context.Context, repoID uint) error {
	klog.V(6).Infof("VectorEmbeddingService: 开始为仓库 %d 批量生成向量", repoID)

	// 获取仓库的所有最新文档
	docs, err := s.docRepo.GetByRepository(repoID)
	if err != nil {
		klog.Warningf("VectorEmbeddingService: 获取文档列表失败: %v", err)
		return fmt.Errorf("get documents: %w", err)
	}

	// 为每个文档创建任务
	for _, doc := range docs {
		// 检查是否已经有向量
		_, err := s.vectorRepo.GetByDocumentID(ctx, doc.ID)
		if err == nil {
			klog.V(6).Infof("VectorEmbeddingService: 文档 %d 已有向量，跳过", doc.ID)
			continue
		}

		// 检查是否有待处理任务
		tasks, err := s.vectorTaskRepo.GetByDocumentID(ctx, doc.ID)
		if err == nil && len(tasks) > 0 {
			// 检查是否有 pending 或 processing 状态的任务
			for _, task := range tasks {
				if task.Status == "pending" || task.Status == "processing" {
					klog.V(6).Infof("VectorEmbeddingService: 文档 %d 已有待处理任务，跳过", doc.ID)
					continue
				}
			}
		}

		// 创建新任务
		if err := s.GenerateForDocument(ctx, doc.ID); err != nil {
			klog.Warningf("VectorEmbeddingService: 为文档 %d 创建任务失败: %v", doc.ID, err)
		}
	}

	klog.V(6).Infof("VectorEmbeddingService: 仓库 %d 的批量生成任务已加入队列", repoID)
	return nil
}

// Start 启动异步任务处理
func (s *VectorEmbeddingService) Start(ctx context.Context) {
	klog.V(6).Infof("VectorEmbeddingService: 启动 %d 个工作协程", s.workers)

	for i := 0; i < s.workers; i++ {
		go s.worker(ctx)
	}
}

// worker 处理任务的协程
func (s *VectorEmbeddingService) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.V(6).Infof("VectorEmbeddingService: 工作协程收到退出信号")
			return

		case docID := <-s.taskQueue:
			s.processTask(ctx, docID)
		}
	}
}

// processTask 处理单个向量生成任务
func (s *VectorEmbeddingService) processTask(ctx context.Context, docID uint) {
	klog.V(6).Infof("VectorEmbeddingService: 开始处理文档 %d 的向量生成", docID)

	// 获取待处理任务
	tasks, err := s.vectorTaskRepo.GetByDocumentID(ctx, docID)
	if err != nil || len(tasks) == 0 {
		klog.Warningf("VectorEmbeddingService: 获取文档 %d 的任务失败", docID)
		return
	}

	// 找到第一个 pending 任务
	var task *model.VectorTask
	for i := range tasks {
		if tasks[i].Status == "pending" {
			task = &tasks[i]
			break
		}
	}

	if task == nil {
		klog.V(6).Infof("VectorEmbeddingService: 文档 %d 没有 pending 任务", docID)
		return
	}

	// 更新任务状态为 processing
	if err := s.vectorTaskRepo.UpdateStatus(ctx, task.ID, "processing", ""); err != nil {
		klog.Warningf("VectorEmbeddingService: 更新任务状态失败: %v", err)
		return
	}

	// 获取文档内容
	doc, err := s.docRepo.Get(docID)
	if err != nil {
		klog.Warningf("VectorEmbeddingService: 获取文档失败: %v", err)
		s.vectorTaskRepo.UpdateStatus(ctx, task.ID, "failed", "document not found")
		return
	}

	// 准备文本：标题 + 内容
	text := doc.Title + "\n" + doc.Content

	// 生成向量
	vectorData, err := s.provider.Embed(ctx, text)
	if err != nil {
		klog.Warningf("VectorEmbeddingService: 生成向量失败: %v", err)
		s.vectorTaskRepo.UpdateStatus(ctx, task.ID, "failed", err.Error())
		return
	}

	// 保存向量到数据库
	docVector := &model.DocumentVector{
		DocumentID:  docID,
		ModelName:   s.provider.ModelName(),
		Vector:      vectorData,
		Dimension:   s.provider.Dimension(),
		GeneratedAt: time.Now(),
	}

	if err := s.vectorRepo.Create(ctx, docVector); err != nil {
		klog.Warningf("VectorEmbeddingService: 保存向量失败: %v", err)
		s.vectorTaskRepo.UpdateStatus(ctx, task.ID, "failed", "failed to save vector")
		return
	}

	// 更新任务状态为 completed
	if err := s.vectorTaskRepo.UpdateStatus(ctx, task.ID, "completed", ""); err != nil {
		klog.Warningf("VectorEmbeddingService: 更新任务状态失败: %v", err)
	}

	klog.V(6).Infof("VectorEmbeddingService: 文档 %d 的向量生成完成", docID)
}

// RegenerateForDocument 重新生成文档的向量
func (s *VectorEmbeddingService) RegenerateForDocument(ctx context.Context, docID uint) error {
	klog.V(6).Infof("VectorEmbeddingService: 开始重新生成文档 %d 的向量", docID)

	// 删除现有向量
	if err := s.vectorRepo.DeleteByDocumentID(ctx, docID); err != nil {
		klog.Warningf("VectorEmbeddingService: 删除现有向量失败: %v", err)
	}

	// 创建新任务
	return s.GenerateForDocument(ctx, docID)
}

// GetStatus 获取向量生成状态
func (s *VectorEmbeddingService) GetStatus(ctx context.Context) (*repository.VectorStatusDTO, error) {
	return s.vectorRepo.GetStatus(ctx)
}
