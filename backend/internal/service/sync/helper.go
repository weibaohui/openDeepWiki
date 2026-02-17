package syncservice

import (
	"context"
	"fmt"

	syncdto "github.com/weibaohui/opendeepwiki/backend/internal/dto/sync"
	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
)

// NormalizeDocumentIDs 清理文档ID列表，去除无效值并保持去重
func NormalizeDocumentIDs(documentIDs []uint) []uint {
	if len(documentIDs) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(documentIDs))
	out := make([]uint, 0, len(documentIDs))
	for _, id := range documentIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// CollectTaskIDsByDocuments 根据文档ID列表收集任务ID集合，并校验文档所属仓库
func CollectTaskIDsByDocuments(ctx context.Context, docRepo repository.DocumentRepository, repoID uint, documentIDs []uint) (map[uint]struct{}, error) {
	taskIDs := make(map[uint]struct{}, len(documentIDs))
	for _, docID := range documentIDs {
		doc, err := docRepo.Get(docID)
		if err != nil {
			return nil, fmt.Errorf("文档不存在: docID=%d, error=%w", docID, err)
		}
		if doc.RepositoryID != repoID {
			return nil, fmt.Errorf("文档仓库不匹配: docID=%d, repoID=%d", docID, repoID)
		}
		taskIDs[doc.TaskID] = struct{}{}
	}
	return taskIDs, nil
}

// FilterTasksByID 根据任务ID集合过滤任务列表
func FilterTasksByID(tasks []model.Task, taskIDs map[uint]struct{}) []model.Task {
	if len(taskIDs) == 0 {
		return nil
	}
	filtered := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if _, ok := taskIDs[task.ID]; ok {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterDocumentsByID 根据文档ID集合过滤文档列表
func FilterDocumentsByID(docs []model.Document, documentIDs map[uint]struct{}) []model.Document {
	if len(documentIDs) == 0 {
		return docs
	}
	filtered := make([]model.Document, 0, len(docs))
	for _, doc := range docs {
		if _, ok := documentIDs[doc.ID]; ok {
			filtered = append(filtered, doc)
		}
	}
	return filtered
}

// SelectLatestDocument 选择一组文档中的最新版本
func SelectLatestDocument(docs []model.Document) *model.Document {
	if len(docs) == 0 {
		return nil
	}
	latest := docs[0]
	for _, doc := range docs[1:] {
		if doc.Version > latest.Version || (doc.Version == latest.Version && doc.ID > latest.ID) {
			latest = doc
		}
	}
	return &latest
}

// GroupPullDocumentsByTask 按任务ID分组拉取文档
func GroupPullDocumentsByTask(docs []syncdto.PullDocumentData) map[uint][]syncdto.PullDocumentData {
	grouped := make(map[uint][]syncdto.PullDocumentData)
	for _, doc := range docs {
		grouped[doc.TaskID] = append(grouped[doc.TaskID], doc)
	}
	return grouped
}

// SelectLatestPullDocument 选择拉取文档中的最新版本
func SelectLatestPullDocument(docs []syncdto.PullDocumentData, docIDMap map[uint]uint, localDocs []model.Document) uint {
	if len(docs) == 0 {
		return 0
	}
	var latestSourceID uint
	for _, doc := range docs {
		if doc.IsLatest {
			latestSourceID = doc.DocumentID
			break
		}
	}
	if latestSourceID != 0 {
		return docIDMap[latestSourceID]
	}
	latest := SelectLatestDocument(localDocs)
	if latest == nil {
		return 0
	}
	return latest.ID
}
