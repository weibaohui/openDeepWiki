package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/weibaohui/opendeepwiki/backend/internal/model"
	"github.com/weibaohui/opendeepwiki/backend/internal/repository"
	vectordomain "github.com/weibaohui/opendeepwiki/backend/internal/domain/vector"
	"k8s.io/klog/v2"
)

// OpenAIConfig OpenAI 配置
type OpenAIConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	Timeout   time.Duration
	Dimension int
}

// openaiEmbeddingResponse OpenAI 兼容 API 响应结构
type openaiEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Index    int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// openaiEmbeddingRequest OpenAI 兼容 API 请求结构
type openaiEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// OpenAIEmbeddingProvider 嵌入提供者实现
// 支持任何兼容 OpenAI API 格式的嵌入服务（包括 Qwen3-Embedding-4B、OpenAI 等）
// 支持从数据库动态加载配置，优先级顺序：使用指定配置 → 按优先级选择
type OpenAIEmbeddingProvider struct {
	configRepo    repository.EmbeddingKeyRepository
	embeddingKeyID uint
	config       OpenAIConfig
	httpClient  *http.Client
}

// NewOpenAIEmbeddingProvider 创建嵌入提供者
// 支持任何兼容 OpenAI API 格式的嵌入服务
// 如果 embeddingKeyID 为 0，则使用默认配置（环境变量或硬编码）
func NewOpenAIEmbeddingProvider(configRepo repository.EmbeddingKeyRepository, embeddingKeyID uint) (vectordomain.EmbeddingProvider, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	provider := &OpenAIEmbeddingProvider{
		configRepo:    configRepo,
		embeddingKeyID: embeddingKeyID,
		httpClient:  client,
	}

	// 加载初始配置
	if err := provider.loadConfig(context.Background()); err != nil {
		klog.Warningf("OpenAIEmbeddingProvider: 加载配置失败: %v", err)
		// 设置默认配置（Qwen3-Embedding-4B）
		provider.config = OpenAIConfig{
			BaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Model:     "text-embedding-v3",
			Dimension: 2560,
			Timeout:   30 * time.Second,
		}
	}

	return provider, nil
}

// loadConfig 从数据库加载配置
func (p *OpenAIEmbeddingProvider) loadConfig(ctx context.Context) error {
	var config *model.EmbeddingKey
	var err error

	if p.embeddingKeyID > 0 {
		// 使用指定的配置
		config, err = p.configRepo.GetByID(ctx, p.embeddingKeyID)
		if err != nil {
			return fmt.Errorf("failed to get embedding config by id %d: %w", p.embeddingKeyID, err)
		}
	} else {
		// 按优先级选择第一个可用配置
		configs, err := p.configRepo.GetAvailable(ctx)
		if err != nil {
			return fmt.Errorf("failed to get available embedding configs: %w", err)
		}
		if len(configs) == 0 {
			return fmt.Errorf("no available embedding config found")
		}
		config = &configs[0]
		klog.V(6).Infof("OpenAIEmbeddingProvider: 使用配置 %s (ID: %d)", config.Name, config.ID)
		p.embeddingKeyID = config.ID
	}

	p.config = OpenAIConfig{
		APIKey:    config.APIKey,
		BaseURL:   config.BaseURL,
		Model:     config.Model,
		Dimension: config.Dimension,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

	p.httpClient.Timeout = p.config.Timeout

	return nil
}

// Embed 为单个文本生成向量嵌入
func (p *OpenAIEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	klog.V(6).Infof("OpenAIEmbeddingProvider: 开始为文本生成向量，长度: %d", len(text))

	vectors, err := p.EmbedBatch(ctx, []string{text})
	if err != nil {
		klog.V(6).Infof("OpenAIEmbeddingProvider: 向量生成失败: %v", err)
		return nil, err
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("no vectors returned")
	}

	// 更新使用统计
	if p.embeddingKeyID > 0 {
		_ = p.configRepo.IncrementRequestCount(ctx, p.embeddingKeyID)
		_ = p.configRepo.UpdateLastUsedAt(ctx, p.embeddingKeyID)
	}

	klog.V(6).Infof("OpenAIEmbeddingProvider: 向量生成成功，维度: %d", len(vectors[0]))
	return vectors[0], nil
}

// EmbedBatch 批量为多个文本生成向量嵌入
func (p *OpenAIEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	klog.V(6).Infof("OpenAIEmbeddingProvider: 批量生成向量，文本数量: %d", len(texts))

	// 尝试加载配置（如果当前配置不可用）
	if p.config.APIKey == "" {
		if err := p.loadConfig(ctx); err != nil {
			klog.Warningf("OpenAIEmbeddingProvider: 重新加载配置失败: %v", err)
			return nil, fmt.Errorf("no valid config available: %w", err)
		}
	}

	requestBody := openaiEmbeddingRequest{
		Input: texts,
		Model: p.config.Model,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		klog.Warningf("OpenAIEmbeddingProvider: 序列化请求失败: %v", err)
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", p.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		klog.Warningf("OpenAIEmbeddingProvider: 创建请求失败: %v", err)
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		klog.Warningf("OpenAIEmbeddingProvider: 发送请求失败: %v", err)
		// 更新错误计数
		if p.embeddingKeyID > 0 {
			_ = p.configRepo.IncrementErrorCount(ctx, p.embeddingKeyID)
		}
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		klog.Warningf("OpenAIEmbeddingProvider: 读取响应失败: %v", err)
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		klog.Warningf("OpenAIEmbeddingProvider: API 返回错误，状态码: %d, 响应: %s", resp.StatusCode, string(body))
		// 更新错误计数
		if p.embeddingKeyID > 0 {
			_ = p.configRepo.IncrementErrorCount(ctx, p.embeddingKeyID)
		}
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var response openaiEmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		klog.Warningf("OpenAIEmbeddingProvider: 解析响应失败: %v", err)
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 提取向量
	vectors := make([][]float32, len(response.Data))
	for i, item := range response.Data {
		vectors[i] = item.Embedding
	}

	klog.V(6).Infof("OpenAIEmbeddingProvider: 批量向量生成成功，Token 用量: prompt=%d, total=%d",
		response.Usage.PromptTokens, response.Usage.TotalTokens)

	return vectors, nil
}

// Dimension 返回向量的维度
func (p *OpenAIEmbeddingProvider) Dimension() int {
	return p.config.Dimension
}

// ModelName 返回使用的模型名称
func (p *OpenAIEmbeddingProvider) ModelName() string {
	return p.config.Model
}

// HealthCheck 检查提供者是否可用
func (p *OpenAIEmbeddingProvider) HealthCheck(ctx context.Context) error {
	if p.config.APIKey == "" {
		// 尝试加载配置
		if err := p.loadConfig(ctx); err != nil {
			return err
		}
	}

	// 简单的健康检查：尝试为一个短文本生成向量
	testText := "health check"
	_, err := p.Embed(ctx, testText)
	return err
}