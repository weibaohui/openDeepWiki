# Skill Middleware 问题诊断

## 问题现象

从日志分析：

```
[EinoCallback] Model 输入 Tools: tool_names=["search_files","list_dir","read_file","skill"]
```
✅ Skill 工具已正确添加

```
[EinoCallback] 调用工具 ToolCall: function_name="repo-detection" function_arguments="{\"path\": \"/tmp/...\"}"
```
❌ LLM 直接调用了 "repo-detection" 而不是 "skill"

```
[EinoCallback] 节点执行出错: tool repo-detection not found in toolsNode indexes
```
❌ ToolNode 找不到 "repo-detection" 工具

## 根本原因

### Eino Skill Middleware 的工作机制

根据 Eino 源码 (`github.com/cloudwego/eino@v0.7.28/adk/middlewares/skill/skill.go`)：

1. **Skill 工具的参数定义**：
```go
ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
    "skill": {
        Type:     schema.String,
        Desc:     "技能名称（无需其他参数）。例如：\"pdf\" 或 \"xlsx\"",
        Required: true,
    },
}),
```

2. **期望的调用方式**：
- Tool name: `"skill"`
- Arguments: `{"skill": "repo-detection"}`

3. **Skill 工具的描述**：
```
<available_skills>
<skill>
<name>repo-detection</name>
<description>识别技术栈和项目类型...</description>
</skill>
...
</available_skills>
```

### 问题所在

**LLM 误解了 skill 工具的描述**：
- ✅ LLM 看到了 "skill" 工具
- ✅ LLM 看到了工具描述中列出的所有 skills（repo-detection, doc-generation 等）
- ❌ LLM 误以为这些 skills 是独立的工具，可以直接调用
- ❌ LLM 没有理解应该调用 "skill" 工具并传递 skill 名称作为参数

## 可能的原因

### 1. LLM 模型理解问题

某些 LLM 模型可能无法正确理解 Eino skill middleware 的 prompt 格式。

**验证方法**：
- 检查使用的 LLM 模型
- 尝试使用不同的模型（如 GPT-4, Claude 等）

### 2. Prompt 冲突

可能有其他地方的 prompt 或配置告诉 LLM 可以直接调用 skills。

**检查点**：
- Agent 的 instruction 中是否提到了 skills
- 是否有其他 middleware 注入了相关信息
- System prompt 中是否有冲突的指令

### 3. Tool Schema 格式问题

LLM 可能无法正确解析 skill 工具的 schema。

**验证方法**：
- 打印完整的 tool schema 发送给 LLM
- 检查 schema 格式是否符合 LLM 的期望

### 4. Eino 版本问题

当前使用的 Eino 版本 (v0.7.28) 可能存在 bug。

**验证方法**：
- 检查 Eino 的 GitHub issues
- 尝试升级到最新版本

## 解决方案

### 方案 1：修改 Agent Instruction（推荐）

在 agent 的 instruction 中明确说明如何使用 skills：

```yaml
instruction: |
  你的任务是分析仓库并生成文档大纲。

  **重要：如何使用 Skills**
  - 当需要使用某个 skill 时，必须调用 "skill" 工具
  - 参数格式：{"skill": "skill-name"}
  - 例如：要使用 repo-detection skill，调用 {"skill": "repo-detection"}
  - 不要直接调用 skill 名称作为工具名

  可用的 skills 会在 "skill" 工具的描述中列出。

  你的具体任务：
  1. 分析仓库的目录结构
  2. 识别仓库类型（go/java/python/frontend/mixed）
  ...
```

### 方案 2：使用自定义 Skill Tool Name

尝试使用更明确的工具名称，避免混淆：

```go
skillToolName := "use_skill"  // 或 "invoke_skill", "execute_skill"
sm, err := skill.New(context.Background(), &skill.Config{
    Backend:       skillBackend,
    UseChinese:    true,
    SkillToolName: &skillToolName,
})
```

### 方案 3：添加调试日志

在 agent 执行前后添加日志，查看完整的 prompt 和 tool schema：

```go
// 在 manager.go 的 createADKAgent 中添加
klog.V(4).Infof("[Manager] Agent tools: %+v", tools)
klog.V(4).Infof("[Manager] Agent middlewares: %+v", config.Middlewares)
```

### 方案 4：检查 LLM 模型配置

确保使用的 LLM 模型支持复杂的 tool calling：

```go
// 检查模型配置
klog.V(6).Infof("[Manager] Using model: %s", def.Model)
```

某些模型可能对 tool schema 的理解有限，建议使用：
- OpenAI GPT-4 或更新版本
- Anthropic Claude 3 或更新版本
- 其他支持 function calling 的高级模型

### 方案 5：临时禁用 Skill Middleware（用于测试）

暂时禁用 skill middleware，验证其他功能是否正常：

```go
// 在 manager.go 中注释掉 skill middleware
// skillMiddleware, err := m.GetOrCreateSkillMiddleware()
// if err != nil {
//     return nil, fmt.Errorf("failed to create skill middleware: %w", err)
// }

config := &adk.ChatModelAgentConfig{
    Name:          def.Name,
    Description:   def.Description,
    Instruction:   def.Instruction,
    Model:         chatModel,
    MaxIterations: def.MaxIterations,
    // Middlewares:   []adk.AgentMiddleware{skillMiddleware},  // 注释掉
}
```

如果禁用后问题消失，说明确实是 skill middleware 的问题。

## 调试步骤

### 1. 启用详细日志

```bash
# 设置环境变量
export KLOG_V=6

# 或在代码中设置
klog.InitFlags(nil)
flag.Set("v", "6")
```

### 2. 打印 Tool Schema

在 `manager.go` 中添加：

```go
for _, t := range tools {
    if infoTool, ok := t.(interface{ Info(context.Context) (*schema.ToolInfo, error) }); ok {
        info, _ := infoTool.Info(context.Background())
        klog.V(4).Infof("[Manager] Tool: %s, Schema: %+v", info.Name, info)
    }
}
```

### 3. 检查 LLM 请求

在发送给 LLM 的请求中添加日志：

```go
// 在 callbacks.go 或相关文件中
klog.V(4).Infof("[LLM] Request tools: %+v", request.Tools)
klog.V(4).Infof("[LLM] Request messages: %+v", request.Messages)
```

### 4. 验证 Skill Backend

```go
// 在 skills.go 中添加
skills, err := skillBackend.List(context.Background())
klog.V(6).Infof("[Manager] Loaded skills: %+v", skills)
for _, s := range skills {
    fullSkill, _ := skillBackend.Get(context.Background(), s.Name)
    klog.V(6).Infof("[Manager] Skill %s content length: %d", s.Name, len(fullSkill.Content))
}
```

## 预期行为 vs 实际行为

### 预期行为

```
1. LLM 看到 tools: ["search_files", "list_dir", "read_file", "skill"]
2. LLM 看到 "skill" 工具的描述，其中列出了可用的 skills
3. LLM 决定使用 repo-detection skill
4. LLM 调用: function_name="skill", arguments={"skill": "repo-detection"}
5. Skill middleware 拦截调用
6. Middleware 加载 repo-detection skill 的内容
7. Middleware 将 skill 内容注入到对话中
8. LLM 根据 skill 内容继续执行
```

### 实际行为

```
1. LLM 看到 tools: ["search_files", "list_dir", "read_file", "skill"]
2. LLM 看到 "skill" 工具的描述，其中列出了可用的 skills
3. LLM 决定使用 repo-detection skill
4. ❌ LLM 调用: function_name="repo-detection", arguments={"path": "..."}
5. ❌ ToolNode 尝试查找 "repo-detection" 工具
6. ❌ 报错: tool repo-detection not found in toolsNode indexes
```

## 下一步行动

1. **立即尝试**：方案 1 - 修改 Agent Instruction
2. **如果无效**：方案 3 - 添加调试日志，查看完整的 tool schema
3. **如果仍无效**：方案 4 - 检查并更换 LLM 模型
4. **最后手段**：联系 Eino 团队或提交 issue

## 参考资料

- Eino Skill Middleware 源码：`github.com/cloudwego/eino@v0.7.28/adk/middlewares/skill/`
- Eino 官方文档：https://www.cloudwego.io/docs/eino/
- Eino GitHub：https://github.com/cloudwego/eino

## 总结

问题的核心是：**LLM 误解了 skill 工具的使用方式**。Skill middleware 期望 LLM 调用 "skill" 工具并传递 skill 名称作为参数，但 LLM 却直接调用了 skill 名称作为工具名。

这可能是：
1. LLM 模型的理解能力问题
2. Prompt 格式不够清晰
3. Tool schema 格式问题
4. Eino 版本 bug

建议先尝试在 agent instruction 中明确说明如何使用 skills，如果无效再进行深入调试。
