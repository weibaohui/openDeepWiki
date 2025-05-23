package ai

import (
	"sync"

	"github.com/sashabaranov/go-openai"
)

// MemoryService 用于按仓库隔离存储和获取对话历史
// 线程安全，适合多仓库在线服务场景
// 历史数据以仓库名称为 key 进行隔离

type memoryService struct {
	mu      sync.RWMutex
	storage map[string][]openai.ChatCompletionMessage // repoName -> 对话历史
}

// NewMemoryService 创建 MemoryService 实例
func NewMemoryService() *memoryService {
	return &memoryService{
		storage: make(map[string][]openai.ChatCompletionMessage),
	}
}

// GetRepoHistory 获取指定仓库的对话历史
func (m *memoryService) GetRepoHistory(repoName string) []openai.ChatCompletionMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history := m.storage[repoName]
	// 返回副本，避免外部修改
	copied := make([]openai.ChatCompletionMessage, len(history))
	copy(copied, history)
	return copied
}

// AppendRepoHistory 向指定仓库追加一条历史记录
func (m *memoryService) AppendRepoHistory(repoName string, msg openai.ChatCompletionMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage[repoName] = append(m.storage[repoName], msg)
}

// ClearRepoHistory 清空指定仓库的历史记录
func (m *memoryService) ClearRepoHistory(repoName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.storage, repoName)
}

// SetRepoHistory 设置指定仓库的对话历史
func (m *memoryService) SetRepoHistory(repoName string, history []openai.ChatCompletionMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storage[repoName] = history
}
