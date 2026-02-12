# 024-ADKAgents模型文件重构优化-设计.md

## 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| Claude | 2026-02-12 | 初始版本 |

---

## 1. 背景

`backend/internal/pkg/adkagents/` 目录下的 `llm.go`、`proxy_model.go`、`model_provider.go` 三个文件存在以下问题：

1. **代码重复严重**：`proxy_model.go` 中 `Generate` 和 `Stream` 方法有约 70% 的重复代码
2. **职责划分不清**：两个类承担过多职责
3. **工具绑定逻辑分散**：使用类型断言方式不够优雅
4. **错误定义位置不合理**：错误变量分散，不利于维护

---

## 2. 目标

- [ ] 消除 `proxy_model.go` 中 `Generate` 和 `Stream` 的代码重复
- [ ] 职责清晰分离：模型管理、代理调用、Rate Limit 处理、工具绑定各司其职
- [ ] 统一错误定义
- [ ] 提高代码可测试性和可扩展性

---

## 3. 非目标

- [ ] 不改变现有对外接口和功能行为
- [ ] 不修改数据库结构或模型定义
- [ ] 不改变 API 行为

---

## 4. 现状分析

### 4.1 文件概览

| 文件 | 行数 | 主要职责 |
|------|------|----------|
| llm.go | 30 | 创建默认 LLM ChatModel |
| proxy_model.go | 271 | 动态代理模型，实现 ChatModel 接口 |
| model_provider.go | 310 | 模型提供者，管理模型池和缓存 |

### 4.2 主要问题

#### 4.2.1 代码重复严重 ⚠️
`proxy_model.go` 中 `Generate` 和 `Stream` 方法有约 70% 的重复代码：
- 获取模型池
- 默认模型兜底处理
- 工具绑定
- Rate Limit 检测和处理
- 请求记录

#### 4.2.2 职责划分不清 🔧

| 组件 | 当前职责 | 问题 |
|------|----------|------|
| ProxyChatModel | 代理调用 + 工具绑定 + Rate Limit 处理 + 用量记录 | 职责过多 |
| EnhancedModelProviderImpl | 模型管理 + 缓存 + Rate Limit 判断 + 模型切换 | 职责过多 |

#### 4.2.3 工具绑定逻辑分散
工具绑定代码在 `Generate` 和 `Stream` 中各出现一次，且使用类型断言的方式不够优雅。

#### 4.2.4 错误定义位置不合理
错误变量定义在 `model_provider.go` 末尾，但被 `proxy_model.go` 使用，应该独立出来。

---

## 5. 优化方案（方案一：提取公共逻辑 + 组件化）⭐

### 5.1 重构后的文件结构

```
backend/internal/pkg/adkagents/
├── errors.go           # 统一错误定义（新增）
├── llm.go              # 保持不变（简单工厂函数）
├── model_provider.go   # 精简：只负责模型管理
├── proxy_model.go      # 精简：只负责代理调用
├── rate_limiter.go     # 新增：Rate Limit 处理逻辑
└── tool_binder.go      # 新增：工具绑定逻辑
```

### 5.2 errors.go（新增）

```go
package adkagents

import (
    "fmt"
    "time"
    "github.com/cloudwego/eino-ext/components/model/openai"
)

// ErrAPIKeyNotFound API Key 不存在错误
var ErrAPIKeyNotFound = fmt.Errorf("api key not found")

// ErrModelUnavailable 模型不可用错误
var ErrModelUnavailable = fmt.Errorf("model unavailable")

// ErrNoAvailableModel 没有可用模型错误
var ErrNoAvailableModel = fmt.Errorf("no available model")

// ErrRateLimitExceeded 速率限制超出错误
var ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")

// FallbackToDefault 兜底到默认模型的错误类型
type FallbackToDefault struct {
    Model *openai.ChatModel
}

func (e *FallbackToDefault) Error() string {
    return "fallback to default model"
}

// ModelUnavailableDetail 模型不可用详细信息
type ModelUnavailableDetail struct {
    ModelName string
    ResetAt   time.Time
    Reason    string
}

func (e *ModelUnavailableDetail) Error() string {
    return fmt.Sprintf("model %s unavailable: %s, reset at %v", e.ModelName, e.Reason, e.ResetAt)
}
```

### 5.3 rate_limiter.go（新增）

```go
package adkagents

import (
    "context"
    "fmt"
    "regexp"
    "strings"
    "time"

    "k8s.io/klog/v2"
)

// RateLimiter 速率限制处理器
type RateLimiter struct {
    provider ModelProvider
}

// ModelProvider 模型提供者接口
type ModelProvider interface {
    MarkModelUnavailable(ctx context.Context, modelName string, resetTime time.Time) error
}

// NewRateLimiter 创建速率限制处理器
func NewRateLimiter(provider ModelProvider) *RateLimiter {
    return &RateLimiter{
        provider: provider,
    }
}

// IsRateLimitError 判断错误是否为 Rate Limit 错误
func (r *RateLimiter) IsRateLimitError(err error) bool {
    if err == nil {
        return false
    }

    errMsg := strings.ToLower(err.Error())

    // 检查 HTTP 状态码
    if strings.Contains(errMsg, "429") {
        return true
    }

    // 检查错误消息关键词
    rateLimitKeywords := []string{
        "rate limit",
        "quota exceeded",
        "too many requests",
        "rate-limited",
        "request rate exceeded",
        "请求次数超过限制",
        "超过限制",
        "每分钟请求次数",
    }

    for _, keyword := range rateLimitKeywords {
        if strings.Contains(errMsg, keyword) {
            return true
        }
    }

    return false
}

// ParseResetTime 从错误中解析重置时间
func (r *RateLimiter) ParseResetTime(err error) time.Time {
    if err == nil {
        return time.Time{}
    }

    errMsg := err.Error()

    // 解析持续时间模式：Try again in 60s, Retry after 1m, Try again in 2h
    durationPatterns := []struct {
        pattern  string
        duration time.Duration
    }{
        {`Try again in (\d+)s`, time.Second},
        {`Retry after (\d+)s`, time.Second},
        {`Try again in (\d+)m`, time.Minute},
        {`Retry after (\d+)m`, time.Minute},
        {`Try again in (\d+)h`, time.Hour},
        {`Retry after (\d+)h`, time.Hour},
    }

    for _, pt := range durationPatterns {
        re := regexp.MustCompile(pt.pattern)
        matches := re.FindStringSubmatch(errMsg)
        if len(matches) >= 2 {
            var duration int
            if _, err := fmt.Sscanf(matches[1], "%d", &duration); err == nil {
                return time.Now().Add(time.Duration(duration) * pt.duration)
            }
        }
    }

    // 解析具体时间模式：Reset at 2026-02-04 12:00:00
    timePatterns := []string{
        `Reset at (\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`,
        `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})`,
    }

    for _, pattern := range timePatterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(errMsg)
        if len(matches) >= 2 {
            var resetTime time.Time
            if err := resetTime.UnmarshalText([]byte(matches[1])); err == nil {
                return resetTime
            }
        }
    }

    return time.Time{}
}

// HandleRateLimit 处理 Rate Limit 错误
func (r *RateLimiter) HandleRateLimit(ctx context.Context, modelName string, err error, defaultResetDuration time.Duration) error {
    klog.Warningf("RateLimiter: rate limit hit for model %s: %v", modelName, err)

    resetTime := r.ParseResetTime(err)
    if resetTime.IsZero() {
        resetTime = time.Now().Add(defaultResetDuration)
    }

    if markErr := r.provider.MarkModelUnavailable(ctx, modelName, resetTime); markErr != nil {
        klog.Errorf("RateLimiter: failed to mark model unavailable: %v", markErr)
    }

    return err
}
```

### 5.4 tool_binder.go（新增）

```go
package adkagents

import (
    "sync"

    "github.com/cloudwego/eino/schema"
    "k8s.io/klog/v2"
)

// ToolBinder 工具绑定器
type ToolBinder struct {
    tools []*schema.ToolInfo
    mu    sync.RWMutex
}

// NewToolBinder 创建工具绑定器
func NewToolBinder() *ToolBinder {
    return &ToolBinder{
        tools: make([]*schema.ToolInfo, 0),
    }
}

// BindTools 设置要绑定的工具列表
func (b *ToolBinder) BindTools(tools []*schema.ToolInfo) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.tools = tools
    return nil
}

// BindToModel 将工具绑定到指定模型
func (b *ToolBinder) BindToModel(model interface{}) error {
    b.mu.RLock()
    tools := b.tools
    b.mu.RUnlock()

    if len(tools) == 0 {
        return nil
    }

    // 使用类型断言检查模型是否支持 BindTools
    type ToolBindable interface {
        BindTools(tools []*schema.ToolInfo) error
    }

    if binder, ok := model.(ToolBindable); ok {
        if err := binder.BindTools(tools); err != nil {
            klog.Warningf("ToolBinder: failed to bind tools to model: %v", err)
        }
    } else {
        klog.V(6).Infof("ToolBinder: model does not support BindTools")
    }

    return nil
}

// GetTools 获取当前工具列表
func (b *ToolBinder) GetTools() []*schema.ToolInfo {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return b.tools
}
```

### 5.5 model_provider.go（精简）

```go
// 移除 IsRateLimitError 方法（迁移到 RateLimiter）
// MarkModelUnavailable 方法接收 ctx 参数
// 提取缓存辅助方法

// getOrCacheModel 从缓存获取或创建并缓存模型
func (p *EnhancedModelProviderImpl) getOrCacheModel(name string, createFunc func() (*ModelWithMetadata, error)) (*ModelWithMetadata, error) {
    // 检查缓存
    p.modelCacheMutex.RLock()
    if cachedModel, exists := p.modelCache[name]; exists {
        p.modelCacheMutex.RUnlock()
        klog.V(6).Infof("EnhancedModelProvider: using cached model %s", name)
        return cachedModel, nil
    }
    p.modelCacheMutex.RUnlock()

    // 创建模型
    model, err := createFunc()
    if err != nil {
        return nil, err
    }

    // 缓存模型
    p.modelCacheMutex.Lock()
    p.modelCache[name] = model
    p.modelCacheMutex.Unlock()

    klog.V(6).Infof("EnhancedModelProvider: created and cached model %s", name)
    return model, nil
}

// MarkModelUnavailable 标记模型为不可用（修改签名）
func (p *EnhancedModelProviderImpl) MarkModelUnavailable(ctx context.Context, modelName string, resetTime time.Time) error {
    // 获取 API Key 配置
    apiKey, err := p.apiKeyRepo.GetByName(ctx, modelName)
    if err != nil {
        return err
    }

    // 标记为不可用
    err = p.apiKeyService.MarkUnavailable(ctx, apiKey.ID, resetTime)
    if err != nil {
        return err
    }

    // 清除缓存
    p.modelCacheMutex.Lock()
    delete(p.modelCache, modelName)
    p.modelCacheMutex.Unlock()

    klog.Warningf("EnhancedModelProvider: marked model %s as unavailable, reset at %v", modelName, resetTime)
    return nil
}
```

### 5.6 proxy_model.go（精简）

```go
package adkagents

import (
    "context"
    "time"

    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/schema"
    "k8s.io/klog/v2"
)

// ProxyChatModel 动态代理模型，支持自动切换
type ProxyChatModel struct {
    provider   *EnhancedModelProviderImpl
    modelNames []string
    toolBinder *ToolBinder
    rateLimiter *RateLimiter
}

// NewProxyChatModel 创建代理模型
func NewProxyChatModel(provider *EnhancedModelProviderImpl, modelNames []string) *ProxyChatModel {
    return &ProxyChatModel{
        provider:    provider,
        modelNames:  modelNames,
        toolBinder:  NewToolBinder(),
        rateLimiter: NewRateLimiter(provider),
    }
}

// Generate 实现 model.ChatModel 接口
func (p *ProxyChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
    result, err := p.executeWithModel(ctx, input, opts, func(model *ModelWithMetadata) (interface{}, error) {
        return model.ChatModel.Generate(ctx, input, opts...)
    })
    if err != nil {
        return nil, err
    }
    return result.(*schema.Message), nil
}

// Stream 实现 model.ChatModel 接口
func (p *ProxyChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
    result, err := p.executeWithModel(ctx, input, opts, func(model *ModelWithMetadata) (interface{}, error) {
        return model.ChatModel.Stream(ctx, input, opts...)
    })
    if err != nil {
        return nil, err
    }
    return result.(*schema.StreamReader[*schema.Message]), nil
}

// executeWithModel 模板方法：消除 Generate 和 Stream 的重复代码
func (p *ProxyChatModel) executeWithModel(
    ctx context.Context,
    input []*schema.Message,
    opts []model.Option,
    executor func(model *ModelWithMetadata) (interface{}, error),
) (interface{}, error) {
    // 1. 获取模型
    model, err := p.getModel(ctx)
    if err != nil {
        return nil, err
    }

    // 2. 绑定工具
    p.toolBinder.BindToModel(&model.ChatModel)

    // 3. 执行请求
    result, err := executor(model)
    if err != nil {
        // 4. 处理 Rate Limit 错误
        if p.rateLimiter.IsRateLimitError(err) {
            return nil, p.rateLimiter.HandleRateLimit(ctx, model.APIKeyName, err, 2*time.Minute)
        }
        return nil, err
    }

    // 5. 记录用量（仅 Generate）
    if msg, ok := result.(*schema.Message); ok && msg != nil && msg.ResponseMeta != nil && msg.ResponseMeta.Usage != nil {
        p.recordUsage(ctx, model.LLMModel, msg.ResponseMeta.Usage)
    }

    // 6. 记录请求
    if model.APIKeyID > 0 {
        _ = p.provider.apiKeyService.RecordRequest(ctx, model.APIKeyID, true)
    }

    return result, nil
}

// getModel 获取模型，支持兜底逻辑
func (p *ProxyChatModel) getModel(ctx context.Context) (*ModelWithMetadata, error) {
    models, err := p.provider.GetModelPool(ctx, p.modelNames)
    if err != nil {
        return nil, err
    }

    if len(models) == 0 {
        // 如果没有可用模型，且未指定特定模型名称，尝试使用 Env 默认模型
        if len(p.modelNames) == 0 {
            klog.Warningf("ProxyChatModel: no DB models available, falling back to Env default model")
            // 返回特殊标记，让 executeWithModel 处理兜底
            return nil, &FallbackToDefault{Model: p.provider.DefaultModel()}
        }
        return nil, ErrNoAvailableModel
    }

    return models[0], nil
}

// BindTools 实现 model.ChatModel 接口
func (p *ProxyChatModel) BindTools(tools []*schema.ToolInfo) error {
    return p.toolBinder.BindTools(tools)
}

// WithTools 适配 model.ToolCallingChatModel 接口
func (p *ProxyChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
    p.BindTools(tools)
    return p, nil
}

// recordUsage 记录用量
func (p *ProxyChatModel) recordUsage(ctx context.Context, modelName string, usage *schema.Usage) {
    taskID, ok := ctx.Value("taskID").(uint)
    if !ok {
        klog.Infof("任务用量记录失败：未在上下文中获取到 taskID")
        return
    }

    if p.provider.taskUsageService != nil {
        if err := p.provider.taskUsageService.RecordUsage(ctx, taskID, modelName, usage); err != nil {
            klog.Infof("任务用量记录失败：taskID=%d, 模型=%s, err=%v", taskID, modelName, err)
        }
    }

    klog.V(6).Infof("模型返回用量：model=%s, usage=%v", modelName, usage)
}
```

---

## 6. 约束条件

- **技术约束**：保持使用 eino 框架接口
- **架构约束**：不改变现有的模块划分
- **安全约束**：不引入新的安全漏洞
- **性能约束**：优化不应降低性能

---

## 7. 可修改 / 不可修改项

- ❌ 不可修改：
  - 对外 API 接口签名
  - 数据库结构
  - 现有业务逻辑流程

- ✅ 可调整：
  - 内部实现细节
  - 辅助方法命名
  - 日志输出格式（保持 klog）

---

## 8. 验收标准

- [ ] `go build` 编译通过
- [ ] 现有单元测试通过
- [ ] 代码重复消除（proxy_model.go 行数减少约 30-40%）
- [ ] 职责清晰分离（5 个文件各司其职）
- [ ] 无新增安全漏洞
- [ ] 无日志输出格式变化（保持 klog）

---

## 9. 风险与已知不确定点

1. **风险**：模板方法提取可能引入测试失败
   - **处理方式**：运行现有测试，如有失败逐一修复

2. **不确定点**：`GetNextModel` 方法是否被使用
   - **处理方式**：通过 Grep 搜索确认使用情况，如未使用则删除

---

## 10. 预期收益

| 指标 | 优化前 | 优化后 |
|------|--------|--------|
| 代码重复 | ~150 行重复 | < 20 行 |
| 单一职责 | 2个类承担多职责 | 5个类各司其职 |
| 可测试性 | 难以单测 | 各组件可独立测试 |
| 可扩展性 | 修改影响大 | 符合开闭原则 |

---

## 11. 实施计划

### 阶段 1：创建新组件文件
- [ ] 创建 `errors.go`，统一错误定义
- [ ] 创建 `rate_limiter.go`，封装 Rate Limit 逻辑
- [ ] 创建 `tool_binder.go`，封装工具绑定逻辑

### 阶段 2：精简现有文件
- [ ] 重构 `model_provider.go`
- [ ] 重构 `proxy_model.go`

### 阶段 3：测试验证
- [ ] 运行 `go build`
- [ ] 运行单元测试
- [ ] 安全自检
