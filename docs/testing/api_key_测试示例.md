# API Key 功能测试代码示例

## 概述

本文档展示了为 openDeepWiki 项目的 **API Key 管理功能** 编写的完整测试代码示例。这些测试代码遵循 `docs/TESTING_PLAN.md` 中定义的测试方案。

---

## 测试文件结构

```
opendeepwiki-testing/
├── backend/
│   └── internal/
│       ├── model/
│       │   └── api_key_test.go           # 模型层测试
│       ├── repository/
│       │   └── api_key_repo_test.go       # 数据访问层测试
│       ├── service/
│       │   └── api_key_test.go           # 业务逻辑层测试
│       └── handler/
│           └── api_key_test.go           # HTTP处理器测试
└── frontend/
    ├── src/
    │   ├── test/
    │   │   └── setup.ts                 # 测试环境配置
    │   ├── services/
    │   │   └── api.test.ts              # API服务测试
    │   └── pages/
    │       └── APIKeyManager.test.tsx   # 页面组件测试
    └── vitest.config.ts                 # 前端测试配置
```

---

## 后端测试

### 1. 模型层测试 (`model/api_key_test.go`)

**覆盖场景：**
- API Key 脱敏功能（`MaskAPIKey`）
- API Key 可用性检查（`IsAvailable`）
- GORM 钩子（`BeforeUpdate`）
- 表名定义（`TableName`）

**测试函数：**
| 函数名 | 说明 |
|--------|------|
| `TestAPIKey_MaskAPIKey` | 测试 API Key 脱敏，包括正常长度、短key等场景 |
| `TestAPIKey_IsAvailable` | 测试可用性检查，包括不同状态和限速场景 |
| `TestAPIKey_BeforeUpdate` | 测试 GORM 钩子是否正确更新 UpdatedAt |
| `TestAPIKey_TableName` | 测试表名是否正确 |

**代码示例：**
```go
// 测试 API Key 脱敏功能
func TestAPIKey_MaskAPIKey(t *testing.T) {
    tests := []struct {
        name     string
        apiKey   string
        expected string
    }{
        {
            name:     "正常长度key",
            apiKey:   "sk-1234567890abcdef",
            expected: "sk-***cdef",
        },
        // ... 更多测试用例
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            key := &APIKey{APIKey: tt.apiKey}
            assert.Equal(t, tt.expected, key.MaskAPIKey())
        })
    }
}
```

### 2. 数据访问层测试 (`repository/api_key_repo_test.go`)

**覆盖场景：**
- CRUD 操作（创建、读取、更新、删除）
- 查询操作（按ID、名称、提供商、名称列表）
- 优先级获取
- 统计信息
- 状态更新
- 统计信息更新
- 速率限制管理

**测试函数：**
| 函数名 | 说明 |
|--------|------|
| `TestAPIKeyRepository_Create` | 测试创建 API Key |
| `TestAPIKeyRepository_Create_Duplicate` | 测试创建重复名称的 API Key |
| `TestAPIKeyRepository_GetByID` | 测试根据 ID 获取 |
| `TestAPIKeyRepository_GetByName` | 测试根据名称获取 |
| `TestAPIKeyRepository_Update` | 测试更新 API Key |
| `TestAPIKeyRepository_Delete` | 测试删除（软删除） |
| `TestAPIKeyRepository_List` | 测试列出所有（按优先级排序） |
| `TestAPIKeyRepository_ListByProvider` | 测试按提供商列出 |
| `TestAPIKeyRepository_GetHighestPriority` | 测试获取优先级最高的 |
| `TestAPIKeyRepository_UpdateStatus` | 测试更新状态 |
| `TestAPIKeyRepository_IncrementStats` | 测试增加统计信息 |
| `TestAPIKeyRepository_SetRateLimitReset` | 测试设置限速重置时间 |
| `TestAPIKeyRepository_GetStats` | 测试获取统计信息 |
| `TestAPIKeyRepository_RateLimit` | 测试速率限制功能 |

**关键特性：**
- 使用内存 SQLite 数据库进行测试
- `setupTestDB` 辅助函数用于快速创建测试环境
- 每个测试独立运行，使用内存数据库隔离
- 验证业务逻辑（如优先级排序、状态过滤）

**代码示例：**
```go
// setupTestDB 创建内存数据库用于测试
func setupTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)

    err = db.AutoMigrate(&model.APIKey{})
    require.NoError(t, err)

    return db
}

// 测试创建 API Key
func TestAPIKeyRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    repo := NewAPIKeyRepository(db)
    ctx := context.Background()

    apiKey := &model.APIKey{
        Name:     "test-key",
        Provider: "openai",
        BaseURL:  "https://api.openai.com/v1",
        APIKey:   "sk-test123456789",
        Model:    "gpt-4",
        Priority: 10,
        Status:   "enabled",
    }

    err := repo.Create(ctx, apiKey)
    require.NoError(t, err)
    assert.NotZero(t, apiKey.ID)
    assert.NotZero(t, apiKey.CreatedAt)
}
```

### 3. 业务逻辑层测试 (`service/api_key_test.go`)

**覆盖场景：**
- 创建 API Key（含名称唯一性验证）
- 更新 API Key（含名称冲突检查）
- 删除 API Key
- 获取 API Key（单个、列表、按名称）
- 更新状态
- 记录请求统计
- 标记不可用
- 获取统计信息

**测试函数：**
| 函数名 | 说明 |
|--------|------|
| `TestAPIKeyService_Create` | 测试创建 API Key（含成功、名称重复、数据库失败等场景） |
| `TestAPIKeyService_Update` | 测试更新 API Key（含名称冲突验证） |
| `TestAPIKeyService_Delete` | 测试删除 API Key |
| `TestAPIKeyService_GetAPIKey` | 测试获取单个 API Key |
| `TestAPIKeyService_ListAPIKeys` | 测试列出所有 API Key |
| `TestAPIKeyService_UpdateAPIKeyStatus` | 测试更新状态 |
| `TestAPIKeyService_GetStats` | 测试获取统计信息 |
| `TestAPIKeyService_RecordRequest` | 测试记录请求 |
| `TestAPIKeyService_MarkUnavailable` | 测试标记为不可用 |
| `TestAPIKeyService_GetAPIKeyByName` | 测试按名称获取 |
| `TestAPIKeyService_GetAPIKeysByNames` | 测试按名称列表获取 |

**关键特性：**
- 使用 Mock 对象隔离依赖
- 使用 testify/mock 框架
- Table-driven test 模式
- 验证业务规则（如名称唯一性、状态有效性）

**代码示例：**
```go
// MockAPIKeyRepository Mock仓库接口
type MockAPIKeyRepository struct {
    mock.Mock
}

// 测试创建 API Key
func TestAPIKeyService_Create(t *testing.T) {
    tests := []struct {
        name        string
        req         *CreateAPIKeyRequest
        mockSetup   func(*MockAPIKeyRepository)
        expectedErr error
        verify      func(*testing.T, *model.APIKey, error)
    }{
        {
            name: "成功创建 API Key",
            req: &CreateAPIKeyRequest{
                Name:     "test-key",
                Provider: "openai",
                BaseURL:  "https://api.openai.com/v1",
                APIKey:   "sk-test123456789",
                Model:    "gpt-4",
                Priority: 10,
            },
            mockSetup: func(m *MockAPIKeyRepository) {
                m.On("GetByName", mock.Anything, "test-key").Return(nil, repository.ErrAPIKeyNotFound)
                m.On("Create", mock.Anything, mock.AnythingOfType("*model.APIKey")).Return(nil)
            },
            verify: func(t *testing.T, result *model.APIKey, err error) {
                require.NoError(t, err)
                assert.Equal(t, "test-key", result.Name)
                assert.Equal(t, "enabled", result.Status)
            },
        },
        // ... 更多测试用例
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(MockAPIKeyRepository)
            tt.mockSetup(mockRepo)

            service := NewAPIKeyService(mockRepo)
            result, err := service.CreateAPIKey(context.Background(), tt.req)

            if tt.verify != nil {
                tt.verify(t, result, err)
            }

            mockRepo.AssertExpectations(t)
        })
    }
}
```

### 4. HTTP处理器测试 (`handler/api_key_test.go`)

**覆盖场景：**
- HTTP 端点测试（GET、POST、PUT、DELETE、PATCH）
- 请求参数验证
- 响应格式验证
- 错误处理
- API Key 脱敏验证

**测试函数：**
| 函数名 | 说明 |
|--------|------|
| `TestAPIKeyHandler_CreateAPIKey` | 测试创建端点 |
| `TestAPIKeyHandler_GetAPIKey` | 测试获取单个端点 |
| `TestAPIKeyHandler_ListAPIKeys` | 测试列表端点 |
| `TestAPIKeyHandler_UpdateAPIKey` | 测试更新端点 |
| `TestAPIKeyHandler_DeleteAPIKey` | 测试删除端点 |
| `TestAPIKeyHandler_UpdateStatus` | 测试状态更新端点 |
| `TestAPIKeyHandler_GetStats` | 测试统计端点 |
| `TestAPIKeyHandler_toResponse` | 测试响应转换（脱敏） |

**关键特性：**
- 使用 httptest 进行 HTTP 测试
- 使用 Gin 测试模式
- Mock Service 层依赖
- 验证 HTTP 状态码、响应体、响应头

**代码示例：**
```go
// setupRouter 设置测试路由
func setupRouter(service *MockAPIKeyService) *gin.Engine {
    gin.SetMode(gin.TestMode)
    router := gin.New()

    handler := NewAPIKeyService(service)
    h := &APIKeyHandler{service: handler}

    api := router.Group("/api/v1")
    h.RegisterRoutes(api)

    return router
}

// 测试创建 API Key
func TestAPIKeyHandler_CreateAPIKey(t *testing.T) {
    tests := []struct {
        name           string
        requestBody    interface{}
        mockSetup      func(*MockAPIKeyService)
        expectedStatus int
        verifyResponse func(*testing.T, *httptest.ResponseRecorder)
    }{
        {
            name: "成功创建 API Key",
            requestBody: map[string]interface{}{
                "name":     "test-key",
                "provider": "openai",
                "base_url": "https://api.openai.com/v1",
                "api_key":  "sk-test123456789",
                "model":    "gpt-4",
                "priority": 10,
            },
            mockSetup: func(m *MockAPIKeyService) {
                apiKey := &model.APIKey{
                    ID:        1,
                    Name:      "test-key",
                    Provider:  "openai",
                    BaseURL:   "https://api.openai.com/v1",
                    APIKey:    "sk-test123456789",
                    Model:     "gpt-4",
                    Priority:  10,
                    Status:    "enabled",
                }
                m.On("CreateAPIKey", mock.Anything, mock.Anything).Return(apiKey, nil)
            },
            expectedStatus: http.StatusCreated,
            verifyResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
                var response map[string]interface{}
                err := json.Unmarshal(w.Body.Bytes(), &response)
                require.NoError(t, err)
                assert.Equal(t, uint(1), uint(response["id"].(float64)))
                // API Key 应该被脱敏
                assert.Equal(t, "sk-***6789", response["api_key"])
            },
        },
        // ... 更多测试用例
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockService := new(MockAPIKeyService)
            tt.mockSetup(mockService)

            router := setupRouter(mockService)

            body, _ := json.Marshal(tt.requestBody)
            req, _ := http.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")

            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)

            assert.Equal(t, tt.expectedStatus, w.Code)
            tt.verifyResponse(t, w)
            mockService.AssertExpectations(t)
        })
    }
}
```

---

## 前端测试

### 1. 测试配置 (`vitest.config.ts`, `src/test/setup.ts`)

**配置内容：**
- 使用 jsdom 环境
- 启用全局测试函数
- 配置覆盖率报告
- 设置路径别名（@ -> src）
- 配置测试清理

### 2. API服务测试 (`src/services/api.test.ts`)

**覆盖场景：**
- 列出 API Key
- 获取单个 API Key
- 创建 API Key
- 更新 API Key
- 删除 API Key
- 更新状态
- 获取统计信息

**测试函数：**
| 测试组 | 说明 |
|--------|------|
| `list` | 测试获取列表、错误处理 |
| `get` | 测试获取单个、404处理 |
| `create` | 测试创建、参数验证 |
| `update` | 测试更新、404处理 |
| `delete` | 测试删除、404处理 |
| `updateStatus` | 测试状态更新、无效状态 |
| `getStats` | 测试获取统计信息 |

**关键特性：**
- 使用 MSW (Mock Service Worker) Mock API
- 隔离网络请求
- 验证请求和响应

**代码示例：**
```typescript
describe('apiKeyApi', () => {
  describe('list', () => {
    it('应该成功获取 API Key 列表', async () => {
      server.use(
        rest.get('/api-keys', (req, res, ctx) => {
          return res(
            ctx.status(200),
            ctx.json({
              data: [mockApiKey],
              total: 1
            })
          )
        })
      )

      const response = await apiKeyApi.list()

      expect(response.status).toBe(200)
      expect(response.data.data).toHaveLength(1)
      expect(response.data.data[0].name).toBe('test-key')
    })

    it('当 API 返回错误时应该抛出异常', async () => {
      server.use(
        rest.get('/api-keys', (req, res, ctx) => {
          return res(
            ctx.status(500),
            ctx.json({ error: 'Server error' })
          )
        })
      )

      await expect(apiKeyApi.list()).rejects.toThrow()
    })
  })
})
```

### 3. 页面组件测试 (`src/pages/APIKeyManager.test.tsx`)

**覆盖场景：**
- 页面渲染
- 统计数据显示
- API Key 列表显示
- 用户交互（添加、编辑、删除、刷新）
- 表单验证
- 状态切换
- 边界情况（空列表、错误处理）

**测试函数：**
| 测试组 | 说明 |
|--------|------|
| `渲染测试` | 测试页面标题、统计卡片、表格、按钮显示 |
| `数据展示测试` | 测试 API Key 列表、Provider 标签 |
| `用户交互测试` | 测试打开模态框、关闭模态框、刷新、状态切换 |
| `表单测试` | 测试表单验证、创建流程 |
| `边界情况测试` | 测试空列表、API错误 |

**关键特性：**
- 使用 React Testing Library
- 使用 userEvent 模拟真实用户操作
- 使用 MSW Mock API 响应
- 验证用户可见行为而非实现细节

**代码示例：**
```typescript
describe('APIKeyManager', () => {
  describe('渲染测试', () => {
    it('应该正确渲染页面标题', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('API Key Management')).toBeInTheDocument()
      })
    })

    it('应该显示统计卡片', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('Total')).toBeInTheDocument()
        expect(screen.getByText('Enabled')).toBeInTheDocument()
      })
    })
  })

  describe('用户交互测试', () => {
    it('应该能够打开创建模态框', async () => {
      const user = userEvent.setup()
      renderWithRouter(<APIKeyManager />)

      await user.click(screen.getByRole('button', { name: /add new/i }))

      expect(screen.getByText('Add API Key')).toBeInTheDocument()
      expect(screen.getByLabelText(/name/i)).toBeInTheDocument()
    })
  })
})
```

---

## 测试覆盖率目标

| 层级 | 目标覆盖率 | 说明 |
|------|----------|------|
| model | 100% | 简单的数据模型方法 |
| repository | 85% | 数据库操作层 |
| service | 80% | 业务逻辑层 |
| handler | 75% | HTTP 端点层 |
| frontend API services | 90% | API 调用服务 |
| frontend components | 70% | 组件交互 |

---

## 运行测试

### 后端测试

```bash
# 运行所有测试
cd backend
go test -v ./...

# 运行特定包的测试
go test -v ./internal/model
go test -v ./internal/repository
go test -v ./internal/service
go test -v ./internal/handler

# 运行特定测试函数
go test -v ./internal/model -run TestAPIKey_MaskAPIKey

# 运行测试并生成覆盖率
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html
```

### 前端测试

```bash
# 安装依赖
cd frontend
pnpm install

# 安装测试依赖（如果需要）
pnpm add -D vitest @vitest/ui @vitest/coverage-v8 @testing-library/react @testing-library/jest-dom @testing-library/user-event msw

# 运行测试
pnpm test

# 运行测试（UI模式）
pnpm test:ui

# 生成覆盖率报告
pnpm test:coverage
```

---

## AI 生成测试提示词

根据这些示例，可以使用以下提示词让 AI 生成类似测试：

### 后端测试生成提示词

```
你是一个专业的 Go 后端测试工程师。请为以下代码编写单元测试：

文件路径：{filePath}
代码结构：{codeStructure}

测试要求：
1. 使用 testify 和 mock 框架
2. 创建 Mock 接口实现
3. 使用 table-driven test 模式
4. 至少包含以下场景：
   - 正常场景
   - 错误场景
   - 边界情况
5. 使用 require.NoError/require.NotNil 进行前置断言
6. 使用 assert.Equal/assert.Error 进行结果验证
7. 确保调用 Mock.AssertExpectations(t)

输出格式：单个测试文件，与被测文件在同一目录
```

### 前端测试生成提示词

```
你是一个专业的前端测试工程师。请为以下 React 组件编写测试：

组件路径：{componentPath}
组件功能：{componentDescription}

测试要求：
1. 使用 Vitest + React Testing Library
2. 使用 userEvent 模拟用户操作
3. 使用 MSW Mock API 调用
4. 至少包含以下场景：
   - 组件渲染测试
   - 用户交互测试
   - 表单验证测试
   - 错误处理测试
5. 遵循 AAA 模式（Arrange-Act-Assert）
6. 使用 screen 查询元素（优先使用可访问性查询）

输出格式：单个测试文件，文件名为 {ComponentName}.test.tsx
```

---

## 总结

本测试代码示例展示了：

1. **完整的测试分层**：从 model 到 handler 的完整测试覆盖
2. **一致的测试风格**：使用相同的工具和模式
3. **AI 友好的测试模板**：便于 AI 理解和生成
4. **全面的场景覆盖**：正常、异常、边界情况
5. **清晰的代码结构**：每个测试都有明确的描述和验证

这些测试代码可以作为项目中其他功能的测试模板和参考。
