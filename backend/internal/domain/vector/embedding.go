package vector

import (
	"context"
)

// EmbeddingProvider 向量嵌入提供者接口
// 定义了生成文本向量嵌入的标准方法
type EmbeddingProvider interface {
	// Embed 为单个文本生成向量嵌入
	// 参数:
	//   - ctx: 上下文，用于控制请求超时和取消
	//   - text: 要生成向量的文本
	// 返回:
	//   - []float32: 生成的向量数组
	//   - error: 错误信息
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch 批量为多个文本生成向量嵌入
	// 参数:
	//   - ctx: 上下文，用于控制请求超时和取消
	//   - texts: 要生成向量的文本数组
	// 返回:
	//   - [][]float32: 生成的向量数组，每个向量对应一个文本
	//   - error: 错误信息
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension 返回向量的维度
	Dimension() int

	// ModelName 返回使用的模型名称
	ModelName() string

	// HealthCheck 检查提供者是否可用
	HealthCheck(ctx context.Context) error
}