# Eino Callbacks 机制设计

## 1. 概述

本文档描述 openDeepWiki 项目中 Eino 框架的 Callbacks 机制设计，用于观察和记录 Workflow 执行过程中的各种事件，包括 LLM 调用、工具执行等。

## 2. 设计目标

1. **可观测性**: 能够详细观察 LLM 的 prompt、tools、tokens 使用情况
2. **调试支持**: 提供完整的调用链路信息，便于问题排查
3. **性能监控**: 记录各节点的执行时间和状态
4. **灵活配置**: 支持不同级别的日志记录，适应开发和生产环境

## 3. 架构设计

### 3.1 组件关系

```
┌─────────────────────────────────────────────────────────────────┐
│                     RepoDocService                              │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    RepoDocChain                           │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐      │  │
│  │  │ Step 1  │→ │ Step 2  │→ │ Step 3  │→ │ Step 4  │      │  │
│  │  │ Clone   │  │ Analyze │  │ Outline │  │ Write   │      │  │
│  │  └─────────┘  └────┬────┘  └─────────┘  └─────────┘      │  │
│  │                    │                                     │  │
│  │              ┌─────┴─────┐                               │  │
│  │              │   LLM     │                               │  │
│  │              │ Generate  │                               │  │
│  │              └───────────┘                               │  │
│  └────────────────────┬──────────────────────────────────────┘  │
│                       │                                         │
│              ┌────────┴────────┐                                │
│              │  EinoCallbacks  │                                │
│              │  - onStart      │                                │
│              │  - onEnd        │                                │
│              │  - onError      │                                │
│              │  - TokenUsage   │                                │
│              └────────┬────────┘                                │
│                       │                                         │
│                       ▼                                         │
│              ┌────────────────┐                                 │
│              │  klog 日志输出  │                                 │
│              └────────────────┘                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心组件

#### EinoCallbacks

`EinoCallbacks` 是回调处理器的核心结构，实现了 Eino 框架的 `callbacks.Handler` 接口：

```go
type EinoCallbacks struct {
    enabled      bool                 // 是否启用回调
    logLevel     int                  // 日志级别
    startTimes   map[string]time.Time // 记录各节点开始时间
    callSequence int                  // 调用序列号
}
```

#### 回调时机

Eino 框架支持以下回调时机：

| 回调时机 | 说明 |
|---------|------|
| `TimingOnStart` | 组件开始执行时 |
| `TimingOnEnd` | 组件执行完成时 |
| `TimingOnError` | 组件执行出错时 |
| `TimingOnStartWithStreamInput` | 流式输入开始时 |
| `TimingOnEndWithStreamOutput` | 流式输出结束时 |

## 4. 功能特性

### 4.1 LLM 调用观察

当 LLM (ChatModel) 被调用时，Callbacks 会记录以下信息：

#### 输入信息 (onStart)
- **Messages**: 完整的对话历史，包括 System、User、Assistant 消息
- **Tools**: 可用的工具列表及其参数定义
- **ToolChoice**: 工具选择策略
- **Config**: 模型配置（模型名称、max_tokens、temperature、top_p 等）

#### 输出信息 (onEnd)
- **Message**: LLM 生成的响应内容
- **ToolCalls**: 工具调用请求（包括函数名和参数）
- **TokenUsage**: Token 使用情况统计
  - `PromptTokens`: 输入 token 数
  - `CompletionTokens`: 输出 token 数
  - `TotalTokens`: 总 token 数
  - `ReasoningTokens`: 推理 token 数（如支持）
  - `CachedTokens`: 缓存 token 数

#### 示例日志输出

```
[EinoCallback] 节点开始执行 component=ChatModel type=model name=generate_outline
[EinoCallback] Model 输入 Messages name=generate_outline message_count=2
[EinoCallback]   Message index=0 role=system content_length=256
[EinoCallback]   Message index=1 role=user content_length=1024
[EinoCallback] Model 输入 Tools name=generate_outline tool_count=3
[EinoCallback]   Tool index=0 name=read_file description=读取文件内容
[EinoCallback] Model 输入 Config name=generate_outline model=gpt-4o max_tokens=4096 temperature=0.7

[EinoCallback] 节点执行完成 component=ChatModel type=model name=generate_outline duration_ms=1250
[EinoCallback] Model 输出 Message name=generate_outline role=assistant content_length=512
[EinoCallback] Model Token 使用情况 name=generate_outline prompt_tokens=1280 completion_tokens=256 total_tokens=1536
```

### 4.2 工具调用观察

当 Tool 被调用时，Callbacks 会记录：

#### 输入信息
- **ArgumentsInJSON**: 工具调用的 JSON 格式参数

#### 输出信息
- **Response**: 工具的响应内容

#### 示例日志输出

```
[EinoCallback] 节点开始执行 component=Tool type=tool name=read_file
[EinoCallback] Tool 输入参数 name=read_file arguments={"file_path": "/path/to/file.go"}

[EinoCallback] 节点执行完成 component=Tool type=tool name=read_file duration_ms=50
[EinoCallback] Tool 输出响应 name=read_file response_length=2048
```

### 4.3 执行时间统计

Callbacks 自动记录每个节点的执行时间：

```
[EinoCallback] 节点执行完成 component=ChatModel type=model name=analyze_repo duration_ms=2100
```

### 4.4 错误处理

当节点执行出错时，Callbacks 会记录详细的错误信息：

```
[EinoCallback] 节点执行出错 component=ChatModel type=model name=generate_content error="rate limit exceeded" duration_ms=5000
```

## 5. 使用方式

### 5.1 基础使用

```go
// 创建带回调的服务
callbacks := einodoc.NewDebugCallbacks() // 调试级别
service, err := einodoc.NewRepoDocServiceWithCallbacks(basePath, llmCfg, callbacks)
if err != nil {
    log.Fatal(err)
}

// 执行解析
result, err := service.ParseRepo(ctx, repoURL)
```

### 5.2 使用便捷函数

```go
// 调试级别 - 记录所有详细信息
callbacks := einodoc.NewDebugCallbacks()

// 详细级别 - 记录完整内容
callbacks := einodoc.NewVerboseCallbacks()

// 简化级别 - 仅记录关键信息
callbacks := einodoc.NewSimpleCallbacks()

// 禁用回调
callbacks := einodoc.DisabledCallbacks()
```

### 5.3 自定义配置

```go
// 自定义日志级别
callbacks := einodoc.NewEinoCallbacks(true, 6) // 启用，日志级别 6

// 动态启用/禁用
callbacks.SetEnabled(false)
callbacks.SetEnabled(true)

// 获取统计信息
stats := callbacks.GetStats()
// {"enabled": true, "running_nodes": 2, "total_calls": 15}
```

### 5.4 全局回调注册

```go
// 在程序初始化时注册全局回调
callbacks := einodoc.NewDebugCallbacks()
einodoc.RegisterGlobalCallbacks(callbacks)

// 之后创建的所有 Chain/Graph 都会自动应用此回调
```

### 5.5 高级服务使用

```go
// 创建高级服务
service, err := einodoc.NewEinoRepoDocServiceWithCallbacks(basePath, llmCfg, callbacks)
if err != nil {
    log.Fatal(err)
}

// 动态切换回调
service.SetCallbacks(einodoc.NewVerboseCallbacks())

// 获取当前回调
if cb := service.GetCallbacks(); cb != nil {
    stats := cb.GetStats()
    fmt.Printf("Total calls: %d\n", stats["total_calls"])
}
```

## 6. 日志级别说明

Callbacks 使用 klog 进行日志输出，建议的日志级别：

| 级别 | 值 | 说明 |
|-----|----|------|
| 0 | 关闭 | 不输出任何日志 |
| 4 | 警告 | 仅输出警告和错误 |
| 6 | 信息 | 输出关键信息（推荐生产环境） |
| 8 | 调试 | 输出所有信息，包括完整内容 |

### 启动参数设置

```bash
# 启用详细日志（开发调试）
go run main.go -v=8

# 启用信息级别日志（生产环境）
go run main.go -v=6

# 仅查看 Eino Callbacks 日志
go run main.go -v=6 2>&1 | grep "EinoCallback"
```

## 7. 实现细节

### 7.1 类型转换

Callbacks 使用 Eino 提供的类型转换函数获取具体信息：

```go
// Model 回调输入转换
modelInput := model.ConvCallbackInput(input)
if modelInput != nil {
    messages := modelInput.Messages
    tools := modelInput.Tools
    config := modelInput.Config
}

// Model 回调输出转换
modelOutput := model.ConvCallbackOutput(output)
if modelOutput != nil {
    message := modelOutput.Message
    tokenUsage := modelOutput.TokenUsage
}

// Tool 回调输入转换
toolInput := tool.ConvCallbackInput(input)
if toolInput != nil {
    args := toolInput.ArgumentsInJSON
}

// Tool 回调输出转换
toolOutput := tool.ConvCallbackOutput(output)
if toolOutput != nil {
    response := toolOutput.Response
}
```

### 7.2 Chain 集成

Callbacks 在 Chain 编译时通过 `compose.Option` 注入：

```go
func (c *RepoDocChain) Run(ctx context.Context, input WorkflowInput) (*RepoDocResult, error) {
    var compileOpts []compose.Option
    if c.callbacks != nil && c.callbacks.IsEnabled() {
        compileOpts = append(compileOpts, WithCallbacks(c.callbacks.Handler()))
    }
    
    runnable, err := c.chain.Compile(ctx, compileOpts...)
    // ...
}
```

## 8. 性能考虑

1. **回调开销**: Callbacks 仅在启用时才会产生开销，禁用后几乎无性能影响
2. **日志级别**: 使用适当的日志级别避免过多的日志输出
3. **内存使用**: Callbacks 会短暂保存节点开始时间，节点完成后立即释放

## 9. 扩展建议

### 9.1 自定义指标收集

可以通过扩展 `EinoCallbacks` 实现自定义指标：

```go
func (ec *EinoCallbacks) onEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    if info.Component == "ChatModel" {
        modelOutput := model.ConvCallbackOutput(output)
        if modelOutput != nil && modelOutput.TokenUsage != nil {
            // 上报到 Prometheus/StatsD
            metrics.RecordTokenUsage(
                info.Name,
                modelOutput.TokenUsage.PromptTokens,
                modelOutput.TokenUsage.CompletionTokens,
            )
        }
    }
    return ctx
}
```

### 9.2 调用链追踪

可以结合 OpenTelemetry 实现分布式追踪：

```go
func (ec *EinoCallbacks) onStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    ctx, span := tracer.Start(ctx, info.Name)
    span.SetAttributes(attribute.String("component", info.Component))
    return ctx
}
```

## 10. 总结

Eino Callbacks 机制为 openDeepWiki 提供了完整的 Workflow 执行可观测性，能够：

1. 详细观察 LLM 的 prompt、tools、tokens 使用情况
2. 监控工具调用的参数和响应
3. 记录各节点的执行时间和状态
4. 支持灵活的日志级别配置
5. 便于调试和问题排查

通过合理使用 Callbacks，开发者可以深入了解 Workflow 的执行过程，优化提示词，监控 Token 消耗，并快速定位问题。
