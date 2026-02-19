# API Key 测试代码快速入门

## 已创建的测试文件

```
opendeepwiki-testing/
├── backend/
│   └── internal/
│       ├── model/
│       │   └── api_key_test.go              ✅ 新增
│       ├── repository/
│       │   └── api_key_repo_test.go          ✅ 增强（原文件基础上扩展）
│       ├── service/
│       │   └── api_key_test.go              ✅ 新增
│       └── handler/
│           └── api_key_test.go              ✅ 新增
└── frontend/
    ├── src/
    │   ├── test/
    │   │   └── setup.ts                    ✅ 新增
    │   ├── services/
    │   │   └── api.test.ts                 ✅ 新增
    │   └── pages/
    │       └── APIKeyManager.test.tsx       ✅ 新增
    ├── vitest.config.ts                      ✅ 新增
    └── package.json                         ✅ 更新（添加测试脚本）
```

---

## 后端测试运行指南

### 1. 运行所有测试

```bash
cd /Users/weibh/projects/go/opendeepwiki-testing/backend
go test -v ./...
```

### 2. 运行特定层级的测试

```bash
# 模型层测试
go test -v ./internal/model

# Repository层测试
go test -v ./internal/repository

# Service层测试
go test -v ./internal/service

# Handler层测试
go test -v ./internal/handler
```

### 3. 运行特定测试函数

```bash
# 运行单个测试
go test -v ./internal/model -run TestAPIKey_MaskAPIKey

# 运行相关测试
go test -v ./internal/model -run TestAPIKey
```

### 4. 生成测试覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./internal/...

# 查看覆盖率
go tool cover -func=coverage.out

# 生成HTML覆盖率报告
go tool cover -html=coverage.out -o coverage.html
```

### 5. 并发/竞态检测

```bash
# 使用 race detector 运行测试
go test -race ./...
```

---

## 前端测试运行指南

### 1. 安装依赖

```bash
cd /Users/weibh/projects/go/opendeepwiki-testing/frontend
pnpm install
```

### 2. 运行测试

```bash
# 运行所有测试
pnpm test

# 运行测试（watch 模式）
pnpm test -- --watch

# 运行测试并显示 UI
pnpm test:ui
```

### 3. 生成覆盖率

```bash
# 生成覆盖率报告
pnpm test:coverage
```

---

## 测试统计

### 后端测试

| 层级 | 测试文件 | 测试函数数 | 预估覆盖率 |
|------|---------|----------|----------|
| model | api_key_test.go | 4 | 100% |
| repository | api_key_repo_test.go | 20+ | ~85% |
| service | api_key_test.go | 10+ | ~80% |
| handler | api_key_test.go | 8 | ~75% |

### 前端测试

| 类型 | 测试文件 | 测试组数 | 预估覆盖率 |
|------|---------|----------|----------|
| API Service | api.test.ts | 7 | ~90% |
| 组件 | APIKeyManager.test.tsx | 5 | ~70% |

---

## AI 生成测试模板

### 后端测试模板

```go
package {package_name}

import (
    "testing"
    "context"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)

// Mock{RepositoryName} Mock仓库接口
type Mock{RepositoryName} struct {
    mock.Mock
}

// Test{ServiceName}_{Action} 测试说明
func Test{ServiceName}_{Action}(t *testing.T) {
    tests := []struct {
        name        string
        input       interface{}
        mockSetup   func(*Mock{RepositoryName})
        expected    interface{}
        expectedErr error
    }{
        {
            name: "正常场景",
            input: {input_value},
            mockSetup: func(m *Mock{RepositoryName}) {
                m.On("MethodName", mock.Anything, {args}).
                    Return({result}, nil)
            },
            expected: {expected_result},
        },
        {
            name: "错误场景",
            input: {error_input},
            mockSetup: func(m *Mock{RepositoryName}) {
                m.On("MethodName", mock.Anything, {args}).
                    Return(nil, {error})
            },
            expectedErr: {error},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(Mock{RepositoryName})
            tt.mockSetup(mockRepo)

            service := New{ServiceName}(mockRepo)
            result, err := service.Method(context.Background(), tt.input)

            if tt.expectedErr != nil {
                assert.Error(t, err)
                assert.Equal(t, tt.expectedErr, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### 前端测试模板

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { rest } from 'msw'
import { setupServer } from 'msw/node'
import userEvent from '@testing-library/user-event'
import {ComponentName} from './{ComponentName}'

const server = setupServer()

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('{ComponentName}', () => {
    beforeEach(() => {
        server.use(
            rest.get('/api/endpoint', (req, res, ctx) => {
                return res(ctx.status(200), ctx.json({ data: 'mock' }))
            })
        )
    })

    describe('渲染测试', () => {
        it('应该正确渲染组件', async () => {
            render(<ComponentName />)

            await waitFor(() => {
                expect(screen.getByText('Expected Text')).toBeInTheDocument()
            })
        })
    })

    describe('交互测试', () => {
        it('应该响应用户操作', async () => {
            const user = userEvent.setup()
            render(<ComponentName />)

            await user.click(screen.getByRole('button', { name: /action/i }))

            await waitFor(() => {
                expect(screen.getByText('Result')).toBeInTheDocument()
            })
        })
    })
})
```

---

## 常见问题

### Q1: 后端测试报错 "undefined: testify"

**A**: 确保 go.mod 中包含 testify 依赖：
```bash
go get github.com/stretchr/testify
```

### Q2: 前端测试报错 "MSW is not configured"

**A**: 确保在测试文件顶部有 MSW server 的 setup：
```typescript
const server = setupServer()
beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
```

### Q3: 测试运行很慢

**A**: 后端可以使用 `-short` flag 跳过慢速测试：
```bash
go test -v -short ./...
```

前端可以运行特定测试：
```bash
pnpm test {test_file}
```

---

## 下一步

1. 运行测试，验证测试代码的正确性
2. 根据测试结果调整测试代码
3. 将测试代码合并回主分支
4. 为其他功能模块创建类似的测试

---

## 相关文档

- 完整测试方案：`docs/TESTING_PLAN.md`
- 测试示例文档：`docs/API_KEY_TESTING_DEMO.md`
- 测试模板：上述代码模板
