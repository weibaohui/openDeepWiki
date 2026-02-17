package syncservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"k8s.io/klog/v2"
)

// RemoteClient 处理远程 HTTP 通信
type RemoteClient struct {
	client *http.Client
}

// NewRemoteClient 创建新的远程客户端
func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CheckTarget 检查目标服务器连通性
func (c *RemoteClient) CheckTarget(ctx context.Context, targetServer string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetServer+"/ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("目标服务器响应异常: status=%d, body=%s", resp.StatusCode, string(body))
	}
	return nil
}

// PostJSON 发送 JSON 请求
func (c *RemoteClient) PostJSON(ctx context.Context, url string, reqBody interface{}, respBody interface{}) error {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败: status=%d, body=%s", resp.StatusCode, string(body))
	}
	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return err
		}
	}
	return nil
}

// FetchPullExportData 从远端获取拉取导出数据
func (c *RemoteClient) FetchPullExportData(ctx context.Context, targetServer string, repoID uint, documentIDs []uint) (syncdto.PullExportData, error) {
	reqBody := syncdto.PullExportRequest{
		RepositoryID: repoID,
		DocumentIDs:  documentIDs,
	}
	var respBody syncdto.PullExportResponse
	if err := c.PostJSON(ctx, targetServer+"/pull-export", reqBody, &respBody); err != nil {
		return syncdto.PullExportData{}, err
	}
	return respBody.Data, nil
}

// CreateRemoteRepository 向目标服务创建或更新仓库信息
func (c *RemoteClient) CreateRemoteRepository(ctx context.Context, targetServer string, repo RepositoryData) error {
	reqBody := syncdto.RepositoryUpsertRequest{
		RepositoryID: repo.ID,
		Name:         repo.Name,
		URL:          repo.URL,
		Description:  repo.Description,
		CloneBranch:  repo.CloneBranch,
		CloneCommit:  repo.CloneCommit,
		SizeMB:       repo.SizeMB,
		Status:       repo.Status,
		ErrorMsg:     repo.ErrorMsg,
		CreatedAt:    repo.CreatedAt,
		UpdatedAt:    repo.UpdatedAt,
	}
	var respBody syncdto.RepositoryUpsertResponse
	if err := c.PostJSON(ctx, targetServer+"/repository-upsert", reqBody, &respBody); err != nil {
		return err
	}
	klog.V(6).Infof("远端仓库同步完成: repoID=%d", repo.ID)
	return nil
}

// ClearRemoteRepository 清空远端仓库数据
func (c *RemoteClient) ClearRemoteRepository(ctx context.Context, targetServer string, repoID uint) error {
	reqBody := syncdto.RepositoryClearRequest{
		RepositoryID: repoID,
	}
	var respBody syncdto.RepositoryClearResponse
	if err := c.PostJSON(ctx, targetServer+"/repository-clear", reqBody, &respBody); err != nil {
		return err
	}
	klog.V(6).Infof("远端仓库数据已清空: repoID=%d", repoID)
	return nil
}

// CreateRemoteTask 创建远端任务
func (c *RemoteClient) CreateRemoteTask(ctx context.Context, targetServer string, task TaskData) (uint, error) {
	reqBody := syncdto.TaskCreateRequest{
		TaskID:       task.ID,
		RepositoryID: task.RepositoryID,
		DocID:        task.DocID,
		WriterName:   string(task.WriterName),
		TaskType:     string(task.TaskType),
		Title:        task.Title,
		Outline:      task.Outline,
		Status:       task.Status,
		RunAfter:     task.RunAfter,
		ErrorMsg:     task.ErrorMsg,
		SortOrder:    task.SortOrder,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
	var respBody syncdto.TaskCreateResponse
	if err := c.PostJSON(ctx, targetServer+"/task-create", reqBody, &respBody); err != nil {
		return 0, err
	}
	return respBody.Data.TaskID, nil
}

// CreateRemoteDocument 创建远端文档
func (c *RemoteClient) CreateRemoteDocument(ctx context.Context, targetServer string, doc DocumentData, remoteTaskID uint) (uint, error) {
	reqBody := syncdto.DocumentCreateRequest{
		DocumentID:   doc.ID,
		RepositoryID: doc.RepositoryID,
		TaskID:       remoteTaskID,
		Title:        doc.Title,
		Filename:     doc.Filename,
		Content:      doc.Content,
		SortOrder:    doc.SortOrder,
		Version:      doc.Version,
		IsLatest:     doc.IsLatest,
		ReplacedBy:   doc.ReplacedBy,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
	var respBody syncdto.DocumentCreateResponse
	if err := c.PostJSON(ctx, targetServer+"/document-create", reqBody, &respBody); err != nil {
		return 0, err
	}
	return respBody.Data.DocumentID, nil
}

// UpdateRemoteTaskDocID 更新远端任务的文档ID
func (c *RemoteClient) UpdateRemoteTaskDocID(ctx context.Context, targetServer string, taskID uint, docID uint) error {
	reqBody := syncdto.TaskUpdateDocIDRequest{
		TaskID:     taskID,
		DocumentID: docID,
	}
	var respBody syncdto.TaskUpdateDocIDResponse
	return c.PostJSON(ctx, targetServer+"/task-update-docid", reqBody, &respBody)
}

// CreateRemoteTaskUsages 同步任务用量列表到对端
func (c *RemoteClient) CreateRemoteTaskUsages(ctx context.Context, targetServer string, remoteTaskID uint, usages []TaskUsageData) error {
	if len(usages) == 0 {
		return nil
	}
	items := make([]syncdto.TaskUsageCreateItem, 0, len(usages))
	for _, usage := range usages {
		items = append(items, syncdto.TaskUsageCreateItem{
			ID:               0,
			TaskID:           remoteTaskID,
			APIKeyName:       usage.APIKeyName,
			PromptTokens:     usage.PromptTokens,
			CompletionTokens: usage.CompletionTokens,
			TotalTokens:      usage.TotalTokens,
			CachedTokens:     usage.CachedTokens,
			ReasoningTokens:  usage.ReasoningTokens,
			CreatedAt:        usage.CreatedAt.Format(time.RFC3339Nano),
		})
	}
	reqBody := syncdto.TaskUsageCreateRequest{
		TaskID:     remoteTaskID,
		TaskUsages: items,
	}
	var respBody syncdto.TaskUsageCreateResponse
	if err := c.PostJSON(ctx, targetServer+"/task-usage-create", reqBody, &respBody); err != nil {
		return err
	}
	klog.V(6).Infof("远端任务用量同步完成: remoteTaskID=%d, count=%d", remoteTaskID, len(usages))
	return nil
}

// RepositoryData 仓库数据传输对象
type RepositoryData struct {
	ID           uint
	Name         string
	URL          string
	Description  string
	CloneBranch  string
	CloneCommit  string
	SizeMB       float64
	Status       string
	ErrorMsg     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TaskData 任务数据传输对象
type TaskData struct {
	ID           uint
	RepositoryID uint
	DocID        uint
	WriterName   string
	TaskType     string
	Title        string
	Outline      string
	Status       string
	RunAfter     uint
	ErrorMsg     string
	SortOrder    int
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DocumentData 文档数据传输对象
type DocumentData struct {
	ID           uint
	RepositoryID uint
	TaskID       uint
	Title        string
	Filename     string
	Content      string
	SortOrder    int
	Version      int
	IsLatest     bool
	ReplacedBy   uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TaskUsageData 任务用量数据传输对象
type TaskUsageData struct {
	TaskID           uint
	APIKeyName       string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CachedTokens     int
	ReasoningTokens  int
	CreatedAt        time.Time
}
