# 011-ToolFactoryConversion-设计

## 1. 问题描述

在 `backend/internal/service/einodoc/tools/factory.go` 文件中，`CreateToolsX` 函数试图调用 `tool.ToToolInfoList(tools)` 将工具列表转换为 `ToolInfo` 列表。然而，经过编译检查，发现 `tool.ToToolInfoList` 方法在 `github.com/cloudwego/eino/components/tool` 包中未定义。

## 2. 目标

修复 `factory.go` 中的编译错误，手动实现从 `[]tool.BaseTool` 到 `[]*schema.ToolInfo` 的转换逻辑，以满足 `CreateToolsX` 函数的返回值要求。

## 3. 设计方案

由于 `tool.BaseTool` 接口定义了 `Info(context.Context) (*schema.ToolInfo, error)` 方法，我们可以通过遍历工具列表并调用该方法来获取工具信息。

### 3.1 实现逻辑

1.  初始化一个空的 `[]*schema.ToolInfo` 切片，容量与输入 `tools` 切片相同。
2.  遍历 `tools` 切片中的每一个 `tool`。
3.  对每个 `tool` 调用 `Info(context.Background())` 方法。
    *   使用 `context.Background()` 是因为 `CreateToolsX` 是一个工厂方法，通常在初始化阶段调用，且未传入 Context。
4.  处理 `Info` 方法的返回值：
    *   如果调用成功，将返回的 `*schema.ToolInfo` 添加到结果切片中。
    *   如果调用失败（返回 error），记录错误日志（使用 `klog.Errorf`），并跳过该工具。这可以防止因为单个工具的元数据获取失败导致整个列表创建失败，同时也暴露了问题。
5.  返回最终的 `[]*schema.ToolInfo` 切片。

### 3.2 代码变更示例

```go
func CreateToolsX(basePath string) []*schema.ToolInfo {
	tools := []tool.BaseTool{
		NewGitCloneTool(basePath),
		NewListDirTool(basePath),
		NewReadFileTool(basePath),
		NewSearchFilesTool(basePath),
	}

	toolInfos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			klog.Errorf("[CreateToolsX] 获取工具信息失败: %v", err)
			continue
		}
		toolInfos = append(toolInfos, info)
	}
	return toolInfos
}
```

## 4. 依赖项

*   `context` 包：用于创建 `context.Background()`。
*   `klog` 包：用于日志记录（已在文件中导入）。

## 5. 测试计划

*   运行 `go build ./backend/internal/service/einodoc/tools/...` 验证编译是否通过。
*   由于是简单的逻辑转换，编译通过且无逻辑错误即可认为修复完成。
