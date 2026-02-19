# 测试规范

本目录包含 openDeepWiki 项目的自动化测试规范文档。

## 文档列表

| 文档 | 说明 |
|------|------|
| [前端自动化测试规范.md](./测试规范/前端自动化测试规范.md) | 前端测试规范，包括 Vitest、React Testing Library、Playwright |
| [后端自动化测试规范.md](./测试规范/后端自动化测试规范.md) | 后端测试规范，包括 Go testing、testify、httptest |

## 测试示例

| 示例 | 说明 |
|------|------|
| [api_key_测试示例.md](./testing/api_key_测试示例.md) | API Key 功能的完整测试代码示例 |

## 快速开始

### 前端测试

```bash
cd frontend
pnpm install
pnpm test              # 运行所有测试
pnpm test:unit        # 运行单元测试
pnpm test:e2e          # 运行 E2E 测试
pnpm test:coverage   # 生成覆盖率报告
```

### 后端测试

```bash
cd backend
go test -v ./...             # 运行所有测试
make test-coverage           # 生成覆盖率报告
make test-coverage-html       # 生成 HTML 覆盖率报告
```

## 测试覆盖目标

| 类型 | 目标覆盖率 |
|------|----------|
| 前端 | 75% |
| 后端 | 75% |

## AI 生成测试

使用提供的测试规范中的 AI Prompt 模板，可以快速为新功能生成测试代码。

详见：
- 前端 AI Prompt：[前端自动化测试规范.md#八ai-生成测试指南](./测试规范/前端自动化测试规范.md#八ai-生成测试指南)
- 后端 AI Prompt：[后端自动化测试规范.md#八ai-生成测试指南](./测试规范/后端自动化测试规范.md#八ai-生成测试指南)
