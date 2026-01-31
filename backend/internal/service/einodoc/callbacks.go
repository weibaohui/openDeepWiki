package einodoc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"k8s.io/klog/v2"
)

// EinoCallbacks Eino 回调处理器
// 用于观察和记录 Workflow 执行过程中的各种事件，包括：
// - LLM 调用的 prompt、tools、tokens 等
// - 工具调用的参数和响应
// - 各节点的执行时间和状态
type EinoCallbacks struct {
	enabled      bool                 // 是否启用回调
	logLevel     klog.Level           // 日志级别 (0=关闭, 4=错误, 6=信息, 8=调试)
	startTimes   map[string]time.Time // 记录各节点开始时间
	callSequence int                  // 调用序列号
}

// NewEinoCallbacks 创建新的 Eino 回调处理器
// enabled: 是否启用回调
// logLevel: 日志级别 (建议使用 klog 的级别: 0=关闭, 4=错误, 6=信息, 8=调试)
func NewEinoCallbacks(enabled bool, logLevel klog.Level) *EinoCallbacks {
	return &EinoCallbacks{
		enabled:    enabled,
		logLevel:   logLevel,
		startTimes: make(map[string]time.Time),
	}
}

// Handler 获取 Eino 的 Handler 接口实现
// 返回配置好的 callbacks.Handler，可用于 Chain 或 Graph 的回调配置
func (ec *EinoCallbacks) Handler() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(ec.onStart).
		OnEndFn(ec.onEnd).
		OnErrorFn(ec.onError).
		OnStartWithStreamInputFn(ec.onStartWithStreamInput).
		OnEndWithStreamOutputFn(ec.onEndWithStreamOutput).
		Build()
}

// onStart 处理组件开始执行的回调
func (ec *EinoCallbacks) onStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if !ec.enabled {
		return ctx
	}

	ec.callSequence++
	nodeKey := ec.nodeKey(info)
	ec.startTimes[nodeKey] = time.Now()

	klog.V(6).InfoS("[EinoCallback] 节点开始执行",
		"sequence", ec.callSequence,
		"component", info.Component,
		"type", info.Type,
		"name", info.Name,
	)

	// 根据组件类型记录详细信息
	switch info.Component {
	case "ChatModel", "Model":
		ec.logModelInput(input, info)
	case "Tool":
		ec.logToolInput(input, info)
	default:
		ec.logGenericInput(input, info)
	}

	return ctx
}

// onEnd 处理组件执行完成的回调
func (ec *EinoCallbacks) onEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if !ec.enabled {
		return ctx
	}

	nodeKey := ec.nodeKey(info)
	startTime, exists := ec.startTimes[nodeKey]
	duration := time.Since(startTime)
	if !exists {
		duration = 0
	}

	klog.V(6).InfoS("[EinoCallback] 节点执行完成",
		"component", info.Component,
		"type", info.Type,
		"name", info.Name,
		"duration_ms", duration.Milliseconds(),
	)

	// 根据组件类型记录详细信息
	switch info.Component {
	case "ChatModel", "Model":
		ec.logModelOutput(output, info)
	case "Tool":
		ec.logToolOutput(output, info)
	default:
		ec.logGenericOutput(output, info)
	}

	delete(ec.startTimes, nodeKey)
	return ctx
}

// onError 处理组件执行出错的回调
func (ec *EinoCallbacks) onError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if !ec.enabled {
		return ctx
	}

	nodeKey := ec.nodeKey(info)
	startTime, exists := ec.startTimes[nodeKey]
	duration := time.Since(startTime)
	if !exists {
		duration = 0
	}

	klog.ErrorS(err, "[EinoCallback] 节点执行出错",
		"component", info.Component,
		"type", info.Type,
		"name", info.Name,
		"duration_ms", duration.Milliseconds(),
	)

	delete(ec.startTimes, nodeKey)
	return ctx
}

// onStartWithStreamInput 处理流式输入开始的回调
func (ec *EinoCallbacks) onStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if !ec.enabled {
		return ctx
	}

	klog.V(6).InfoS("[EinoCallback] 流式输入开始",
		"component", info.Component,
		"type", info.Type,
		"name", info.Name,
	)

	// 流式输入时，我们无法直接读取内容，只能记录开始事件
	return ctx
}

// onEndWithStreamOutput 处理流式输出结束的回调
func (ec *EinoCallbacks) onEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if !ec.enabled {
		return ctx
	}

	klog.V(6).InfoS("[EinoCallback] 流式输出结束",
		"component", info.Component,
		"type", info.Type,
		"name", info.Name,
	)

	// 流式输出时，我们无法直接读取内容，只能记录结束事件
	return ctx
}

// ========== Model (LLM) 回调详情 ==========

// logModelInput 记录模型输入详情
func (ec *EinoCallbacks) logModelInput(input callbacks.CallbackInput, info *callbacks.RunInfo) {
	modelInput := model.ConvCallbackInput(input)
	if modelInput == nil {
		klog.V(6).InfoS("[EinoCallback] Model 输入转换失败",
			"name", info.Name,
			"input_type", fmt.Sprintf("%T", input),
		)
		return
	}

	// 收集 Tools 信息
	toolCount := len(modelInput.Tools)
	toolNames := make([]string, 0, toolCount)
	for _, t := range modelInput.Tools {
		if t != nil {
			toolNames = append(toolNames, t.Name)
		}
	}

	// 在第一条日志中显示关键信息：Messages 和 Tools 摘要
	klog.V(6).InfoS("[EinoCallback] Model 输入",
		"name", info.Name,
		"message_count", len(modelInput.Messages),
		"tool_count", toolCount,
		"tool_names", toolNames,
	)

	// 记录 Prompt/Messages 详情
	if len(modelInput.Messages) > 0 {
		for i, msg := range modelInput.Messages {
			if msg != nil {
				klog.V(6).InfoS("[EinoCallback]   Message",
					"index", i,
					"role", msg.Role,
					"content_length", len(msg.Content),
				)
				// 在调试级别下记录完整内容
				klog.V(8).InfoS("[EinoCallback]   Message Content",
					"index", i,
					"content", msg.Content,
				)
			}
		}
	}

	// 记录 Tools 详情
	if toolCount > 0 {
		for i, t := range modelInput.Tools {
			if t != nil {
				klog.V(6).InfoS("[EinoCallback]   Tool",
					"index", i,
					"name", t.Name,
					"description", t.Desc,
				)
				// 在调试级别下记录完整参数定义
				klog.V(8).InfoS("[EinoCallback]   Tool Params",
					"index", i,
					"params", t.ParamsOneOf,
				)
			}
		}
	}

	// 记录 ToolChoice
	if modelInput.ToolChoice != nil {
		toolChoiceJSON, _ := json.Marshal(modelInput.ToolChoice)
		klog.V(6).InfoS("[EinoCallback] Model 输入 ToolChoice",
			"name", info.Name,
			"tool_choice", string(toolChoiceJSON),
		)
	}

	// 记录 Config
	if modelInput.Config != nil {
		klog.V(6).InfoS("[EinoCallback] Model 输入 Config",
			"name", info.Name,
			"model", modelInput.Config.Model,
			"max_tokens", modelInput.Config.MaxTokens,
			"temperature", modelInput.Config.Temperature,
			"top_p", modelInput.Config.TopP,
		)
	}

	// 记录 Extra
	if len(modelInput.Extra) > 0 {
		extraJSON, _ := json.Marshal(modelInput.Extra)
		klog.V(6).InfoS("[EinoCallback] Model 输入 Extra",
			"name", info.Name,
			"extra", string(extraJSON),
		)
	}
}

// logModelOutput 记录模型输出详情
func (ec *EinoCallbacks) logModelOutput(output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	modelOutput := model.ConvCallbackOutput(output)
	if modelOutput == nil {
		klog.V(6).InfoS("[EinoCallback] Model 输出转换失败",
			"name", info.Name,
			"output_type", fmt.Sprintf("%T", output),
		)
		return
	}

	// 记录生成的 Message
	if modelOutput.Message != nil {
		klog.V(6).InfoS("[EinoCallback] Model 输出 Message",
			"name", info.Name,
			"role", modelOutput.Message.Role,
			"content_length", len(modelOutput.Message.Content),
		)
		// 在调试级别下记录完整内容
		klog.V(8).InfoS("[EinoCallback] Model 输出 Content",
			"name", info.Name,
			"content", modelOutput.Message.Content,
		)

		// 记录工具调用
		if len(modelOutput.Message.ToolCalls) > 0 {
			klog.V(6).InfoS("[EinoCallback] Model 输出 ToolCalls",
				"name", info.Name,
				"tool_call_count", len(modelOutput.Message.ToolCalls),
			)
			for i, tc := range modelOutput.Message.ToolCalls {
				klog.V(6).InfoS("[EinoCallback]   ToolCall",
					"index", i,
					"id", tc.ID,
					"type", tc.Type,
					"function_name", tc.Function.Name,
					"function_arguments", tc.Function.Arguments,
				)
			}
		}
	}

	// 记录 Token 使用情况 (重点!)
	if modelOutput.TokenUsage != nil {
		klog.V(6).InfoS("[EinoCallback] Model Token 使用情况",
			"name", info.Name,
			"prompt_tokens", modelOutput.TokenUsage.PromptTokens,
			"completion_tokens", modelOutput.TokenUsage.CompletionTokens,
			"total_tokens", modelOutput.TokenUsage.TotalTokens,
			"reasoning_tokens", modelOutput.TokenUsage.CompletionTokensDetails.ReasoningTokens,
			"cached_tokens", modelOutput.TokenUsage.PromptTokenDetails.CachedTokens,
		)
	} else {
		klog.V(6).InfoS("[EinoCallback] Model Token 使用情况",
			"name", info.Name,
			"token_usage", "未返回",
		)
	}

	// 记录 Extra
	if len(modelOutput.Extra) > 0 {
		extraJSON, _ := json.Marshal(modelOutput.Extra)
		klog.V(6).InfoS("[EinoCallback] Model 输出 Extra",
			"name", info.Name,
			"extra", string(extraJSON),
		)
	}
}

// ========== Tool 回调详情 ==========

// logToolInput 记录工具输入详情
func (ec *EinoCallbacks) logToolInput(input callbacks.CallbackInput, info *callbacks.RunInfo) {
	toolInput := tool.ConvCallbackInput(input)
	if toolInput == nil {
		klog.V(6).InfoS("[EinoCallback] Tool 输入转换失败",
			"name", info.Name,
			"input_type", fmt.Sprintf("%T", input),
		)
		return
	}

	klog.V(6).InfoS("[EinoCallback] Tool 输入参数",
		"name", info.Name,
		"arguments", toolInput.ArgumentsInJSON,
	)

	if len(toolInput.Extra) > 0 {
		extraJSON, _ := json.Marshal(toolInput.Extra)
		klog.V(6).InfoS("[EinoCallback] Tool 输入 Extra",
			"name", info.Name,
			"extra", string(extraJSON),
		)
	}
}

// logToolOutput 记录工具输出详情
func (ec *EinoCallbacks) logToolOutput(output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	toolOutput := tool.ConvCallbackOutput(output)
	if toolOutput == nil {
		klog.V(6).InfoS("[EinoCallback] Tool 输出转换失败",
			"name", info.Name,
			"output_type", fmt.Sprintf("%T", output),
		)
		return
	}

	klog.V(6).InfoS("[EinoCallback] Tool 输出响应",
		"name", info.Name,
		"response_length", len(toolOutput.Response),
	)

	// 在调试级别下记录完整响应
	klog.V(8).InfoS("[EinoCallback] Tool 输出响应详情",
		"name", info.Name,
		"response", toolOutput.Response,
	)

	if len(toolOutput.Extra) > 0 {
		extraJSON, _ := json.Marshal(toolOutput.Extra)
		klog.V(6).InfoS("[EinoCallback] Tool 输出 Extra",
			"name", info.Name,
			"extra", string(extraJSON),
		)
	}
}

// ========== 通用回调详情 ==========

// logGenericInput 记录通用输入详情
func (ec *EinoCallbacks) logGenericInput(input callbacks.CallbackInput, info *callbacks.RunInfo) {
	klog.V(8).InfoS("[EinoCallback] 通用输入",
		"component", info.Component,
		"name", info.Name,
		"input_type", fmt.Sprintf("%T", input),
		"input", fmt.Sprintf("%+v", input),
	)
}

// logGenericOutput 记录通用输出详情
func (ec *EinoCallbacks) logGenericOutput(output callbacks.CallbackOutput, info *callbacks.RunInfo) {
	klog.V(8).InfoS("[EinoCallback] 通用输出",
		"component", info.Component,
		"name", info.Name,
		"output_type", fmt.Sprintf("%T", output),
		"output", fmt.Sprintf("%+v", output),
	)
}

// nodeKey 生成节点唯一键
func (ec *EinoCallbacks) nodeKey(info *callbacks.RunInfo) string {
	return fmt.Sprintf("%s:%s:%s", info.Component, info.Type, info.Name)
}

// ========== 全局回调注册 ==========

// RegisterGlobalCallbacks 注册全局回调处理器
// 全局回调会在所有 Chain/Graph 执行前被调用
// 注意：此函数不是线程安全的，应在程序初始化时调用
func RegisterGlobalCallbacks(callbacks *EinoCallbacks) {
	if callbacks != nil && callbacks.enabled {
		callbacks.AppendGlobalHandlers(callbacks.Handler())
		klog.V(4).InfoS("[EinoCallback] 全局回调处理器已注册")
	}
}

// AppendGlobalHandlers 将处理器添加到全局回调列表
// 这是 interface 的封装，用于调用 callbacks.AppendGlobalHandlers
func (ec *EinoCallbacks) AppendGlobalHandlers(handler callbacks.Handler) {
	if ec.enabled {
		callbacks.AppendGlobalHandlers(handler)
	}
}

// ========== Chain/Graph 配置选项 ==========

// WithCallbacks 返回 Chain 或 Graph 的回调配置选项
// 使用方式: chain.Compile(ctx, WithCallbacks(callbacks.Handler()))
func WithCallbacks(handler callbacks.Handler) compose.Option {
	return compose.WithCallbacks(handler)
}

// WithGlobalCallbacks 使用全局回调编译选项
// 如果已注册全局回调，此选项会自动应用
func WithGlobalCallbacks() compose.Option {
	return compose.WithCallbacks(nil) // nil 会使用全局回调
}

// ========== 便捷函数 ==========

// NewDebugCallbacks 创建调试级别的回调处理器
// 记录所有详细信息，包括完整的 prompt、response、tools 等
func NewDebugCallbacks() *EinoCallbacks {
	return NewEinoCallbacks(true, 6) // 使用 klog 级别 6 (信息级别)
}

// NewVerboseCallbacks 创建详细级别的回调处理器
// 记录所有信息，适合深入排查问题
func NewVerboseCallbacks() *EinoCallbacks {
	return NewEinoCallbacks(true, 8) // 使用 klog 级别 8 (调试级别)
}

// NewSimpleCallbacks 创建简化级别的回调处理器
// 仅记录关键信息，适合生产环境
func NewSimpleCallbacks() *EinoCallbacks {
	return NewEinoCallbacks(true, 4) // 使用 klog 级别 4 (警告级别，实际会记录关键信息)
}

// DisabledCallbacks 返回禁用的回调处理器
// 用于生产环境或不需要观察的场景
func DisabledCallbacks() *EinoCallbacks {
	return NewEinoCallbacks(false, 0)
}

// IsEnabled 检查回调是否启用
func (ec *EinoCallbacks) IsEnabled() bool {
	return ec.enabled
}

// SetEnabled 设置回调启用状态
func (ec *EinoCallbacks) SetEnabled(enabled bool) {
	ec.enabled = enabled
}

// GetStats 获取回调统计信息
// 返回当前正在执行的节点数和已完成的调用次数
func (ec *EinoCallbacks) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":       ec.enabled,
		"running_nodes": len(ec.startTimes),
		"total_calls":   ec.callSequence,
	}
}
