# 测试规范

本规范定义 cocursor 项目的单元测试和集成测试编写标准，遵循 TDD（测试驱动开发）原则。

## TDD 开发流程

### Red-Green-Refactor 循环

1. **Red**: 先写失败的测试，定义预期行为
2. **Green**: 写最少代码使测试通过
3. **Refactor**: 重构代码，保持测试通过

### 测试先行原则

- 新功能开发前先写测试用例
- Bug 修复前先写复现 bug 的测试
- 重构时确保已有测试覆盖

## Go 测试规范

### 文件组织

- 测试文件与源文件同目录，命名 `*_test.go`
- 每个公开函数/方法至少有对应测试
- 测试辅助函数使用 `setup*` 或 `helper*` 前缀

```
domain/chat/
├── entity.go
├── entity_test.go
├── service.go
└── service_test.go
```

### 测试函数命名

使用 `Test<函数名>_<场景>` 格式：

```go
func TestService_GetConversation(t *testing.T) { ... }
func TestService_GetConversation_EmptyID(t *testing.T) { ... }
func TestService_GetConversation_NotFound(t *testing.T) { ... }
```

### 使用 testify

统一使用 `testify` 库进行断言：

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)

func TestExample(t *testing.T) {
    // require: 失败立即终止
    require.NoError(t, err)
    require.NotNil(t, result)

    // assert: 失败继续执行
    assert.Equal(t, expected, actual)
    assert.Len(t, items, 3)
    assert.Contains(t, list, item)
    assert.ErrorIs(t, err, ErrNotFound)
}
```

### 表驱动测试

多场景测试使用表驱动模式：

```go
func TestValidateRequest(t *testing.T) {
    tests := []struct {
        name       string
        req        Request
        wantErrors int
    }{
        {
            name: "完整有效请求",
            req:  Request{Type: "feature", Summary: "测试"},
            wantErrors: 0,
        },
        {
            name: "空请求",
            req:  Request{},
            wantErrors: 4,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            errors := tt.req.Validate()
            assert.Len(t, errors, tt.wantErrors)
        })
    }
}
```

### 子测试

使用 `t.Run()` 组织相关测试：

```go
func TestReportWorkProgress(t *testing.T) {
    handler := NewCommandHandler()

    t.Run("报告工作进展成功", func(t *testing.T) {
        // 正常流程测试
    })

    t.Run("缺少必填参数", func(t *testing.T) {
        // 参数验证测试
    })

    t.Run("无效的工作类型", func(t *testing.T) {
        // 边界条件测试
    })
}
```

### Mock 对象

使用 `testify/mock` 创建模拟对象：

```go
// MockRepository 模拟仓储
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*Entity, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Entity), args.Error(1)
}

func TestService_GetByID(t *testing.T) {
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)

    expected := &Entity{ID: "test-1"}
    mockRepo.On("GetByID", mock.Anything, "test-1").Return(expected, nil)

    result, err := service.GetByID(context.Background(), "test-1")

    assert.NoError(t, err)
    assert.Equal(t, expected.ID, result.ID)
    mockRepo.AssertExpectations(t)
}
```

### 测试辅助函数

封装重复的设置逻辑：

```go
func setupTestRepo(t *testing.T) (*Repository, func()) {
    t.Helper()

    tmpDir, err := os.MkdirTemp("", "test")
    require.NoError(t, err)

    repo, err := NewRepository(filepath.Join(tmpDir, "test.db"))
    require.NoError(t, err)

    cleanup := func() {
        repo.Close()
        os.RemoveAll(tmpDir)
    }

    return repo, cleanup
}

func TestRepository_CRUD(t *testing.T) {
    repo, cleanup := setupTestRepo(t)
    defer cleanup()

    // 测试逻辑
}
```

### 测试覆盖场景

每个函数测试应覆盖：

1. **正常路径**: 预期输入，预期输出
2. **边界条件**: 空值、零值、极限值
3. **错误路径**: 无效输入、外部依赖失败
4. **状态变化**: 验证副作用

```go
func TestArchiveRepository_SaveAndGet(t *testing.T) {
    repo, cleanup := setupTestRepo(t)
    defer cleanup()

    ctx := context.Background()

    // 正常路径：保存并获取
    session := &ArchivedSession{ID: "test-1", Name: "测试"}
    err := repo.Save(ctx, session)
    require.NoError(t, err)

    retrieved, err := repo.GetByID(ctx, "test-1")
    require.NoError(t, err)
    assert.Equal(t, session.Name, retrieved.Name)

    // 边界条件：不存在的 ID
    _, err = repo.GetByID(ctx, "non-existent")
    assert.ErrorIs(t, err, ErrNotFound)

    // 状态变化：更新后验证
    session.Name = "更新后"
    err = repo.Save(ctx, session)
    require.NoError(t, err)

    updated, _ := repo.GetByID(ctx, "test-1")
    assert.Equal(t, "更新后", updated.Name)
}
```

## TypeScript 测试规范

### 测试框架

使用 VS Code Extension Test 框架：

```typescript
import * as assert from "assert";
import * as vscode from "vscode";

suite("Extension Test Suite", () => {
  vscode.window.showInformationMessage("Start all tests.");

  test("Sample test", () => {
    assert.strictEqual(-1, [1, 2, 3].indexOf(5));
    assert.strictEqual(-1, [1, 2, 3].indexOf(0));
  });
});
```

### 异步测试

```typescript
test("Async operation", async () => {
  const result = await someAsyncFunction();
  assert.strictEqual(result.status, "success");
});
```

### Mock 依赖

```typescript
// 使用 sinon 或 jest mock
import * as sinon from "sinon";

test("API call", async () => {
  const stub = sinon.stub(api, "fetchData").resolves({ data: [] });

  const result = await service.getData();

  assert.ok(stub.calledOnce);
  stub.restore();
});
```

## 分层测试策略

### Domain 层 - 单元测试

- 纯逻辑测试，无外部依赖
- Mock 所有仓储接口
- 高覆盖率要求（>80%）

```go
// domain/chat/service_test.go
func TestService_ListConversations(t *testing.T) {
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)

    mockRepo.On("ListConversations", mock.Anything).Return([]Conversation{}, nil)

    result, err := service.ListConversations(context.Background())

    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Application 层 - 集成测试

- 测试用例编排逻辑
- Mock 基础设施层
- 验证 DTO 转换

### Infrastructure 层 - 集成测试

- 使用真实依赖（临时数据库、文件）
- 测试后清理资源
- 可选择跳过（CI 环境）

```go
func TestArchiveRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    repo, cleanup := setupTestRepo(t)
    defer cleanup()

    // 使用真实 SQLite 测试
}
```

### Interfaces 层 - E2E 测试

- 测试 HTTP 端点
- 使用 `httptest` 包
- 验证请求/响应格式

```go
func TestChatHandler_ListConversations(t *testing.T) {
    router := setupTestRouter()

    req := httptest.NewRequest("GET", "/api/v1/chats", nil)
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var resp Response
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, 0, resp.Code)
}
```

## 运行测试

### Go 测试命令

```bash
# 运行所有测试
make test

# 运行特定包测试
go test ./internal/domain/chat/...

# 运行特定测试
go test -run TestService_GetConversation ./internal/domain/chat/

# 显示覆盖率
make test-coverage

# 跳过集成测试
go test -short ./...
```

### TypeScript 测试命令

```bash
# 运行插件测试
npm run test
```

## 测试最佳实践

### Do

- 测试名称描述预期行为
- 使用中文注释说明测试意图
- 一个测试只验证一个行为
- 保持测试独立，无执行顺序依赖
- 使用 `t.Helper()` 标记辅助函数

### Don't

- 不要测试私有函数（通过公开接口测试）
- 不要在测试中使用 `time.Sleep`（使用 channel 或 mock）
- 不要忽略错误返回值
- 不要共享测试状态
- 不要测试第三方库的行为
