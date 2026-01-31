更合适的 ADK 组合方式（建议架构，不改代码）

- 全局层：Supervisor 或 Orchestrator 作为“总控 Agent”
  - 使用 Supervisor 模式或 ChatModelAgent + SubAgents 进行动态调度。
  - 适配需求中的 Orchestrator，负责章节依赖、并行控制、整体状态维护。
  - 适用协作方式： ToolCall(AgentAsTool) 触发下游能力，获取输出做下一步决策。
- 主线流程：SequentialAgent 保留，但只作为“骨架”
  - RepoInitializer → Architect 仍适合顺序执行。
  - 但后续应进入“章节级并行处理”，而不是线性 Writer。
- 章节级处理：ParallelAgent + SequentialAgent 组合
  - 对每个章节启动并行子流程（Parallel）。
  - 每个章节内部按顺序执行：Explorer → Planner → Writer → Reviewer → Editor（Sequential）。
  - 这种“并行包裹顺序”的组合能对齐需求的“按章节并行 + 章节内流程化”。
- 质量闭环：LoopAgent 在 Writer/Reviewer 之间形成迭代
  - Reviewer 反馈驱动 Writer 修改，直到 Reviewer 明确通过。
  - 这是需求中“质量控制层”的直接映射。
- 上下文策略：两段式上下文输入
  - Explorer → Planner → Writer：使用“New Task Description”摘要替代全历史，降低上下文噪音。
  - Writer/Reviewer：使用“Upstream Agent Full Dialogue”保留细节。
- 恢复与断点续跑：用 Resume 替代临时兜底
  - 对长流程在关键节点设置可恢复点，出现迭代上限时走 Runner.Resume，而非新建临时 Agent。
- AgentRunOption 定向控制
  - 例如对 Writer 选择更长 max iteration、对 Reviewer 设置更严格输出格式，使用 DesignateAgent 来指定生效范围。
推荐的目标组合（摘要）

- 总控层 ：Supervisor / Orchestrator（动态决策、全局状态）
- 主线骨架 ：SequentialAgent（RepoInitializer → Architect）
- 章节并行 ：ParallelAgent（多章节并发）
- 章节内部 ：SequentialAgent（Explorer → Planner → Writer → Reviewer → Editor）
- 质量闭环 ：LoopAgent（Writer ↔ Reviewer）
- 协作方式 ：ToolCall(AgentAsTool) 为主，Transfer 用于任务转交
- 上下文策略 ：摘要上下文 + 全量上下文混合
- 断点恢复 ：Runner.Resume + ResumableAgent 的 Checkpoint 机制
与当前实现可对齐的节点

- 你们已经用了 SequentialAgent 、 Runner 、 ChatModelAgent 这类基础原语，可以直接把“并行/循环/监督”扩展为更贴合流程的组合，而不是推翻重做。
- 现有处理函数（如 processArchitectOutput ）在未来可作为“章节内结果归档”的钩子继续用。 workflow.go