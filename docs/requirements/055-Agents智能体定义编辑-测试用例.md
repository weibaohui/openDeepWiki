# 055-Agents智能体定义编辑-测试用例.md

## 0. 文件修改记录表

| 修改人 | 修改时间 | 修改内容 |
| ------ | -------- | -------- |
| AI | 2026-02-22 | 初始版本 |

---

## 1. 测试概述

本测试用例用于验证 Agents 智能体定义编辑功能的正确性，包括后端 API 和前端 UI 的功能验证。

---

## 2. 测试范围

| 模块 | 测试类型 | 优先级 |
|------|---------|--------|
| 后端 Model | 单元测试 | P0 |
| 后端 Repository | 单元测试 | P0 |
| 后端 Service | 单元测试 | P0 |
| 后端 Handler | 单元测试 | P0 |
| 前端 AgentList | 组件测试 | P0 |
| 前端 AgentEditor | 组件测试 | P0 |
| 前端 VersionHistory | 组件测试 | P0 |

---

## 3. 后端测试用例

### 3.1 Model 测试

#### AgentVersion Model 测试

```go
package model

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestAgentVersion_TableName(t *testing.T) {
    v := &AgentVersion{}
    assert.Equal(t, "agent_versions", v.TableName())
}

func TestAgentVersion_Fields(t *testing.T) {
    now := time.Now()
    v := &AgentVersion{
        ID:                1,
        FileName:          "test.yaml",
        Content:           "name: test",
        Version:           1,
        SavedAt:           now,
        Source:            "web",
        RestoreFromVersion: nil,
        CreatedAt:         now,
    }

    assert.Equal(t, uint(1), v.ID)
    assert.Equal(t, "test.yaml", v.FileName)
    assert.Equal(t, "name: test", v.Content)
    assert.Equal(t, 1, v.Version)
    assert.Equal(t, "web", v.Source)
    assert.Nil(t, v.RestoreFromVersion)
}
```

---

### 3.2 Repository 测试

#### AgentVersionRepository 测试

```go
package repository

import (
    "context"
    "testing"
    "time"

    "github.com/glebarez/sqlite"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gorm.io/gorm"

    "github.com/weibaohui/opendeepwiki/backend/internal/model"
)

func setupAgentVersionTestDB(t *testing.T) *gorm.DB {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    err = db.AutoMigrate(&model.AgentVersion{})
    require.NoError(t, err)
    return db
}

func TestAgentVersionRepository_Create(t *testing.T) {
    db := setupAgentVersionTestDB(t)
    repo := NewAgentVersionRepository(db)
    ctx := context.Background()

    v := &model.AgentVersion{
        FileName: "test.yaml",
        Content:  "name: test",
        Version:  1,
        SavedAt:  time.Now(),
        Source:   "web",
    }

    err := repo.Create(ctx, v)
    require.NoError(t, err)
    assert.NotZero(t, v.ID)
}

func TestAgentVersionRepository_GetVersionsByFileName(t *testing.T) {
    db := setupAgentVersionTestDB(t)
    repo := NewAgentVersionRepository(db)
    ctx := context.Background()

    // 创建多个版本
    versions := []*model.AgentVersion{
        {FileName: "test.yaml", Content: "v1", Version: 1, SavedAt: time.Now(), Source: "web"},
        {FileName: "test.yaml", Content: "v2", Version: 2, SavedAt: time.Now(), Source: "web"},
        {FileName: "other.yaml", Content: "v1", Version: 1, SavedAt: time.Now(), Source: "web"},
    }

    for _, v := range versions {
        err := repo.Create(ctx, v)
        require.NoError(t, err)
    }

    // 查询指定文件的版本
    results, err := repo.GetVersionsByFileName(ctx, "test.yaml")
    require.NoError(t, err)
    assert.Len(t, results, 2)
    assert.Equal(t, "v2", results[0].Content) // 应该按版本降序排列
    assert.Equal(t, "v1", results[1].Content)
}

func TestAgentVersionRepository_GetLatestVersion(t *testing.T) {
    db := setupAgentVersionTestDB(t)
    repo := NewAgentVersionRepository(db)
    ctx := context.Background()

    versions := []*model.AgentVersion{
        {FileName: "test.yaml", Content: "v1", Version: 1, SavedAt: time.Now(), Source: "web"},
        {FileName: "test.yaml", Content: "v2", Version: 2, SavedAt: time.Now(), Source: "web"},
    }

    for _, v := range versions {
        err := repo.Create(ctx, v)
        require.NoError(t, err)
    }

    latest, err := repo.GetLatestVersion(ctx, "test.yaml")
    require.NoError(t, err)
    assert.Equal(t, 2, latest.Version)
    assert.Equal(t, "v2", latest.Content)
}
```

---

### 3.3 Service 测试

#### AgentService 测试

```go
package service

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// MockAgentVersionRepository Mock 接口
type MockAgentVersionRepository struct {
    mock.Mock
}

func (m *MockAgentVersionRepository) Create(ctx context.Context, v *model.AgentVersion) error {
    args := m.Called(ctx, v)
    return args.Error(0)
}

func (m *MockAgentVersionRepository) GetVersionsByFileName(ctx context.Context, fileName string) ([]*model.AgentVersion, error) {
    args := m.Called(ctx, fileName)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]*model.AgentVersion), args.Error(1)
}

func (m *MockAgentVersionRepository) GetLatestVersion(ctx context.Context, fileName string) (*model.AgentVersion, error) {
    args := m.Called(ctx, fileName)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*model.AgentVersion), args.Error(1)
}

func TestAgentService_ListAgents(t *testing.T) {
    // 准备测试数据
    testAgents := []string{
        "markdown_checker.yaml",
        "toc_checker.yaml",
        "api_explorer.yaml",
    }

    // 这里需要 mock 文件系统读取，或者使用测试目录
    // 实际实现中可能需要注入文件系统接口
}

func TestAgentService_GetAgent(t *testing.T) {
    ctx := context.Background()

    mockRepo := new(MockAgentVersionRepository)
    service := NewAgentService(mockRepo, "./test_agents")

    content := "name: test\nmodel: gpt-4"
    mockVersion := &model.AgentVersion{
        FileName: "test.yaml",
        Content:  content,
        Version:  1,
        SavedAt:  time.Now(),
        Source:   "web",
    }

    mockRepo.On("GetVersionsByFileName", ctx, "test.yaml").Return([]*model.AgentVersion{mockVersion}, nil)

    result, err := service.GetAgent(ctx, "test.yaml")
    require.NoError(t, err)
    assert.Equal(t, "test.yaml", result.FileName)
    assert.Equal(t, content, result.Content)
    assert.Equal(t, 1, result.Version)
}

func TestAgentService_SaveAgent(t *testing.T) {
    ctx := context.Background()
    now := time.Now()

    mockRepo := new(MockAgentVersionRepository)
    service := NewAgentService(mockRepo, "./test_agents")

    savedVersion := &model.AgentVersion{
        ID:       1,
        FileName:  "test.yaml",
        Content:   "name: test\nmodel: gpt-4",
        Version:   1,
        SavedAt:   now,
        Source:    "web",
        CreatedAt: now,
    }

    mockRepo.On("Create", ctx, mock.AnythingOfType("*model.AgentVersion")).Return(nil)

    result, err := service.SaveAgent(ctx, "test.yaml", "name: test\nmodel: gpt-4", "web", nil)
    require.NoError(t, err)
    assert.Equal(t, "test.yaml", result.FileName)
    assert.Equal(t, 1, result.Version)

    mockRepo.AssertExpectations(t)
}

func TestAgentService_RestoreVersion(t *testing.T) {
    ctx := context.Background()
    now := time.Now()

    mockRepo := new(MockAgentVersionRepository)
    service := NewAgentService(mockRepo, "./test_agents")

    // Mock 获取历史版本
    oldVersion := &model.AgentVersion{
        ID:       1,
        FileName:  "test.yaml",
        Content:   "name: old\nmodel: gpt-3",
        Version:   1,
        SavedAt:   now,
        Source:    "web",
    }

    mockRepo.On("GetVersionsByFileName", ctx, "test.yaml").Return([]*model.AgentVersion{oldVersion}, nil)
    mockRepo.On("Create", ctx, mock.AnythingOfType("*model.AgentVersion")).Return(nil)

    newVersion, err := service.RestoreVersion(ctx, "test.yaml", 1)
    require.NoError(t, err)
    assert.Equal(t, "test.yaml", newVersion.FileName)
    assert.Equal(t, "name: old\nmodel: gpt-3", newVersion.Content)
    assert.Equal(t, 2, newVersion.Version) // 新版本号应该是 2
}
```

---

### 3.4 Handler 测试

#### AgentHandler 测试

```go
package handler

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "github.com/weibaohui/opendeepwiki/backend/internal/model"
)

// MockAgentService Mock 接口
type MockAgentService struct {
    mock.Mock
}

func (m *MockAgentService) ListAgents(ctx context.Context) ([]*AgentInfo, error) {
    args := m.Called(ctx)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]*AgentInfo), args.Error(1)
}

func (m *MockAgentService) GetAgent(ctx context.Context, fileName string) (*AgentDTO, error) {
    args := m.Called(ctx, fileName)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*AgentDTO), args.Error(1)
}

func (m *MockAgentService) SaveAgent(ctx context.Context, fileName, content, source string, restoreFrom *int) (*SaveResultDTO, error) {
    args := m.Called(ctx, fileName, content, source, restoreFrom)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*SaveResultDTO), args.Error(1)
}

func TestAgentHandler_ListAgents(t *testing.T) {
    mockService := new(MockAgentService)
    handler := NewAgentHandler(mockService)

    agents := []*AgentInfo{
        {FileName: "test1.yaml", Name: "test1", Description: "Test 1"},
        {FileName: "test2.yaml", Name: "test2", Description: "Test 2"},
    }

    mockService.On("ListAgents", mock.Anything).Return(agents, nil)

    // 设置路由
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/api/agents", handler.ListAgents)

    req, _ := http.NewRequest("GET", "/api/agents", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err)

    data := response["data"].([]interface{})
    assert.Len(t, data, 2)
}

func TestAgentHandler_GetAgent(t *testing.T) {
    mockService := new(MockAgentService)
    handler := NewAgentHandler(mockService)

    agent := &AgentDTO{
        FileName:       "test.yaml",
        Content:        "name: test",
        CurrentVersion: 1,
    }

    mockService.On("GetAgent", mock.Anything, "test.yaml").Return(agent, nil)

    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/api/agents/:filename", handler.GetAgent)

    req, _ := http.NewRequest("GET", "/api/agents/test.yaml", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err)

    assert.Equal(t, "test.yaml", response["file_name"])
    assert.Equal(t, "name: test", response["content"])
}

func TestAgentHandler_SaveAgent(t *testing.T) {
    mockService := new(MockAgentService)
    handler := NewAgentHandler(mockService)

    result := &SaveResultDTO{
        FileName: "test.yaml",
        Version:  2,
        SavedAt:  time.Now(),
    }

    mockService.On("SaveAgent", mock.Anything, "test.yaml", "name: test", "web", (*int)(nil)).Return(result, nil)

    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.PUT("/api/agents/:filename", handler.SaveAgent)

    requestBody := map[string]string{
        "content": "name: test",
    }
    body, _ := json.Marshal(requestBody)

    req, _ := http.NewRequest("PUT", "/api/agents/test.yaml", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err)

    assert.Equal(t, "test.yaml", response["file_name"])
    assert.Equal(t, float64(2), response["version"])
}

func TestAgentHandler_GetVersions(t *testing.T) {
    mockService := new(MockAgentService)
    handler := NewAgentHandler(mockService)

    versions := []*model.AgentVersion{
        {
            ID:      1,
            FileName: "test.yaml",
            Version:  1,
            SavedAt:  time.Now(),
            Source:   "web",
        },
        {
            ID:      2,
            FileName: "test.yaml",
            Version:  2,
            SavedAt:  time.Now(),
            Source:   "web",
        },
    }

    mockService.On("GetVersions", mock.Anything, "test.yaml").Return(versions, nil)

    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/api/agents/:filename/versions", handler.GetVersions)

    req, _ := http.NewRequest("GET", "/api/agents/test.yaml/versions", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err)

    data := response["versions"].([]interface{})
    assert.Len(t, data, 2)
}

func TestAgentHandler_RestoreVersion(t *testing.T) {
    mockService := new(MockAgentService)
    handler := NewAgentHandler(mockService)

    result := &SaveResultDTO{
        FileName:    "test.yaml",
        RestoredFrom: 1,
        Version:     3,
        SavedAt:     time.Now(),
    }

    mockService.On("SaveAgent", mock.Anything, "test.yaml", "name: old", "web", mock.AnythingOfType("*int")).Return(result, nil)

    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.POST("/api/agents/:filename/versions/:version/restore", handler.RestoreVersion)

    req, _ := http.NewRequest("POST", "/api/agents/test.yaml/versions/1/restore", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    require.NoError(t, err)

    assert.Equal(t, "test.yaml", response["file_name"])
    assert.Equal(t, float64(1), response["restored_from"])
}
```

---

## 4. 前端测试用例

### 4.1 AgentList 组件测试

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AgentList from '@/components/agents/AgentList';

// Mock API
jest.mock('@/api/agents', () => ({
  listAgents: jest.fn(),
}));

import { listAgents } from '@/api/agents';

describe('AgentList', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该显示所有 Agent 列表', async () => {
    (listAgents as jest.Mock).mockResolvedValue([
      {
        file_name: 'markdown_checker.yaml',
        name: 'markdown_checker',
        description: 'Markdown 校验者',
      },
      {
        file_name: 'toc_checker.yaml',
        name: 'toc_checker',
        description: '目录校验者',
      },
    ]);

    render(<AgentList />);

    await waitFor(() => {
      expect(screen.getByText('markdown_checker')).toBeInTheDocument();
      expect(screen.getByText('Markdown 校验者')).toBeInTheDocument();
      expect(screen.getByText('toc_checker')).toBeInTheDocument();
      expect(screen.getByText('目录校验者')).toBeInTheDocument();
    });
  });

  it('应该点击 Agent 时打开编辑器', async () => {
    (listAgents as jest.Mock).mockResolvedValue([
      {
        file_name: 'markdown_checker.yaml',
        name: 'markdown_checker',
        description: 'Markdown 校验者',
      },
    ]);

    const onSelectAgent = jest.fn();
    render(<AgentList onSelectAgent={onSelectAgent} />);

    await waitFor(() => {
      fireEvent.click(screen.getByText('markdown_checker'));
    });

    expect(onSelectAgent).toHaveBeenCalledWith('markdown_checker.yaml');
  });
});
```

### 4.2 AgentEditor 组件测试

```tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AgentEditor from '@/components/agents/AgentEditor';

jest.mock('@/api/agents', () => ({
  getAgent: jest.fn(),
  saveAgent: jest.fn(),
  getVersions: jest.fn(),
  restoreVersion: jest.fn(),
}));

import { getAgent, saveAgent, getVersions, restoreVersion } from '@/api/agents';

describe('AgentEditor', () => {
  const mockAgent = {
    file_name: 'markdown_checker.yaml',
    content: 'name: markdown_checker\nmodel: gpt-4',
    current_version: 2,
  };

  const mockVersions = [
    {
      id: 1,
      version: 1,
      saved_at: '2026-02-20T10:00:00Z',
      source: 'web',
    },
    {
      id: 2,
      version: 2,
      saved_at: '2026-02-21T15:30:00Z',
      source: 'web',
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该加载并显示 Agent 内容', async () => {
    (getAgent as jest.Mock).mockResolvedValue(mockAgent);

    render(<AgentEditor fileName="markdown_checker.yaml" />);

    await waitFor(() => {
      expect(screen.getByText('markdown_checker')).toBeInTheDocument();
    });
  });

  it('应该保存 Agent 内容', async () => {
    (getAgent as jest.Mock).mockResolvedValue(mockAgent);
    (saveAgent as jest.Mock).mockResolvedValue({
      file_name: 'markdown_checker.yaml',
      version: 3,
      saved_at: '2026-02-22T10:00:00Z',
    });

    render(<AgentEditor fileName="markdown_checker.yaml" />);

    await waitFor(() => {
      // 模拟编辑内容
      fireEvent.change(screen.getByRole('textbox'), {
        target: { value: 'name: markdown_checker\nmodel: gpt-4o' },
      });
    });

    const saveButton = screen.getByText('保存');
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(saveAgent).toHaveBeenCalledWith(
        'markdown_checker.yaml',
        'name: markdown_checker\nmodel: gpt-4o'
      );
    });
  });

  it('应该显示版本历史', async () => {
    (getAgent as jest.Mock).mockResolvedValue(mockAgent);
    (getVersions as jest.Mock).mockResolvedValue({
      file_name: 'markdown_checker.yaml',
      versions: mockVersions,
    });

    render(<AgentEditor fileName="markdown_checker.yaml" />);

    await waitFor(() => {
      fireEvent.click(screen.getByText('版本历史'));
    });

    await waitFor(() => {
      expect(screen.getByText('版本 1')).toBeInTheDocument();
      expect(screen.getByText('版本 2')).toBeInTheDocument();
    });
  });

  it('应该从历史版本恢复', async () => {
    (getAgent as jest.Mock).mockResolvedValue(mockAgent);
    (getVersions as jest.Mock).mockResolvedValue({
      file_name: 'markdown_checker.yaml',
      versions: mockVersions,
    });
    (restoreVersion as jest.Mock).mockResolvedValue({
      file_name: 'markdown_checker.yaml',
      restored_from: 1,
      new_version: 3,
    });

    render(<AgentEditor fileName="markdown_checker.yaml" />);

    await waitFor(() => {
      fireEvent.click(screen.getByText('版本历史'));
    });

    await waitFor(() => {
      // 点击版本 1 的恢复按钮
      const restoreButtons = screen.getAllByText('恢复');
      fireEvent.click(restoreButtons[0]);
    });

    await waitFor(() => {
      expect(restoreVersion).toHaveBeenCalledWith('markdown_checker.yaml', 1);
    });
  });
});
```

---

## 5. 集成测试用例

### 5.1 完整编辑保存流程测试

```go
package integration

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAgentEditFlow(t *testing.T) {
    // 创建临时测试目录
    tempDir := t.TempDir()
    agentsDir := filepath.Join(tempDir, "agents")
    err := os.MkdirAll(agentsDir, 0755)
    require.NoError(t, err)

    // 创建测试 Agent 文件
    testContent := "name: test_agent\ndescription: Test Agent"
    testFile := filepath.Join(agentsDir, "test_agent.yaml")
    err = os.WriteFile(testFile, []byte(testContent), 0644)
    require.NoError(t, err)

    // 创建服务实例（需要注入测试目录）
    // service := service.NewAgentService(repo, agentsDir)

    // 测试流程：
    // 1. 列出所有 Agent
    // 2. 获取 Agent 内容
    // 3. 修改并保存
    // 4. 获取版本历史
    // 5. 从历史版本恢复
}
```

---

## 6. 测试覆盖目标

| 模块 | 目标覆盖率 | 优先级 |
|------|----------|--------|
| AgentVersion Model | 100% | P0 |
| AgentVersion Repository | 90% | P0 |
| AgentService | 80% | P0 |
| AgentHandler | 75% | P0 |
| AgentList 组件 | 80% | P0 |
| AgentEditor 组件 | 75% | P0 |
| VersionHistory 组件 | 80% | P0 |

---

## 7. 验收标准

### 7.1 功能验收

- [ ] 所有单元测试通过
- [ ] 可以通过 API 列出所有 Agent
- [ ] 可以通过 API 获取 Agent 内容
- [ ] 可以通过 API 保存 Agent（创建新版本）
- [ ] 可以通过 API 获取版本历史
- [ ] 可以通过 API 恢复到历史版本
- [ ] 前端组件可以正常渲染和交互

### 7.2 性能验收

- [ ] API 响应时间符合性能要求
- [ ] 前端组件渲染无卡顿

### 7.3 安全验收

- [ ] 文件路径验证正确
- [ ] 无目录遍历漏洞

---

## 8. 执行方式

```bash
# 后端测试
cd backend
go test ./internal/model/...
go test ./internal/repository/...
go test ./internal/service/...
go test ./internal/handler/...

# 前端测试
cd frontend
pnpm test
```
