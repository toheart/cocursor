# Testing Specification

This specification defines unit testing and integration testing standards for the project, following TDD (Test-Driven Development) principles.

## TDD Development Process

### Red-Green-Refactor Cycle

1. **Red**: Write failing tests first, define expected behavior
2. **Green**: Write minimal code to make tests pass
3. **Refactor**: Refactor code while keeping tests passing

### Test-First Principle

- Write test cases before developing new features
- Write tests to reproduce bugs before fixing them
- Ensure existing test coverage when refactoring

## Go Testing Standards

### File Organization

- Test files in the same directory as source files, named `*_test.go`
- Each public function/method should have corresponding tests
- Test helper functions use `setup*` or `helper*` prefix

```
domain/example/
├── entity.go
├── entity_test.go
├── service.go
└── service_test.go
```

### Test Function Naming

Use `Test<FunctionName>_<Scenario>` format:

```go
func TestService_GetExample(t *testing.T) { ... }
func TestService_GetExample_EmptyID(t *testing.T) { ... }
func TestService_GetExample_NotFound(t *testing.T) { ... }
```

### Using testify

Use `testify` library for assertions:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/mock"
)

func TestExample(t *testing.T) {
    // require: fail immediately
    require.NoError(t, err)
    require.NotNil(t, result)

    // assert: continue on failure
    assert.Equal(t, expected, actual)
    assert.Len(t, items, 3)
    assert.Contains(t, list, item)
    assert.ErrorIs(t, err, ErrNotFound)
}
```

### Table-Driven Tests

Use table-driven pattern for multiple scenarios:

```go
func TestValidateRequest(t *testing.T) {
    tests := []struct {
        name       string
        req        Request
        wantErrors int
    }{
        {
            name: "Complete valid request",
            req:  Request{Type: "feature", Summary: "Test"},
            wantErrors: 0,
        },
        {
            name: "Empty request",
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

### Subtests

Use `t.Run()` to organize related tests:

```go
func TestReportWorkProgress(t *testing.T) {
    handler := NewCommandHandler()

    t.Run("Report work progress success", func(t *testing.T) {
        // Normal flow test
    })

    t.Run("Missing required parameters", func(t *testing.T) {
        // Parameter validation test
    })

    t.Run("Invalid work type", func(t *testing.T) {
        // Boundary condition test
    })
}
```

### Mock Objects

Use `testify/mock` to create mock objects:

```go
// MockRepository Mock repository
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

### Test Helper Functions

Encapsulate repeated setup logic:

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

    // Test logic
}
```

### Test Coverage Scenarios

Each function test should cover:

1. **Happy path**: Expected input, expected output
2. **Boundary conditions**: Empty values, zero values, extreme values
3. **Error paths**: Invalid input, external dependency failures
4. **State changes**: Verify side effects

```go
func TestArchiveRepository_SaveAndGet(t *testing.T) {
    repo, cleanup := setupTestRepo(t)
    defer cleanup()

    ctx := context.Background()

    // Happy path: Save and get
    session := &ArchivedSession{ID: "test-1", Name: "Test"}
    err := repo.Save(ctx, session)
    require.NoError(t, err)

    retrieved, err := repo.GetByID(ctx, "test-1")
    require.NoError(t, err)
    assert.Equal(t, session.Name, retrieved.Name)

    // Boundary condition: Non-existent ID
    _, err = repo.GetByID(ctx, "non-existent")
    assert.ErrorIs(t, err, ErrNotFound)

    // State change: Verify after update
    session.Name = "Updated"
    err = repo.Save(ctx, session)
    require.NoError(t, err)

    updated, _ := repo.GetByID(ctx, "test-1")
    assert.Equal(t, "Updated", updated.Name)
}
```

## TypeScript Testing Standards

### Testing Framework

Use Jest or Vitest for React applications:

```typescript
import { describe, it, expect } from 'vitest';

describe('ApiService', () => {
  it('should fetch data successfully', async () => {
    const result = await api.get('/examples');
    expect(result).toBeDefined();
  });
});
```

### Async Tests

```typescript
it('Async operation', async () => {
  const result = await someAsyncFunction();
  expect(result.status).toBe('success');
});
```

### Mock Dependencies

```typescript
import { vi } from 'vitest';

it('API call', async () => {
  const mockGet = vi.spyOn(api, 'get').mockResolvedValue({ data: [] });

  const result = await service.getData();

  expect(mockGet).toHaveBeenCalledOnce();
  mockGet.mockRestore();
});
```

## Layered Testing Strategy

### Domain Layer - Unit Tests

- Pure logic tests, no external dependencies
- Mock all repository interfaces
- High coverage requirement (>80%)

```go
// domain/example/service_test.go
func TestService_ListExamples(t *testing.T) {
    mockRepo := new(MockRepository)
    service := NewService(mockRepo)

    mockRepo.On("ListExamples", mock.Anything).Return([]Example{}, nil)

    result, err := service.ListExamples(context.Background())

    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Application Layer - Integration Tests

- Test use case orchestration logic
- Mock infrastructure layer
- Verify DTO conversion

### Infrastructure Layer - Integration Tests

- Use real dependencies (temporary database, files)
- Clean up resources after tests
- Can be skipped (CI environment)

```go
func TestArchiveRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    repo, cleanup := setupTestRepo(t)
    defer cleanup()

    // Use real SQLite for testing
}
```

### Interfaces Layer - E2E Tests

- Test HTTP endpoints
- Use `httptest` package
- Verify request/response format

```go
func TestExampleHandler_ListExamples(t *testing.T) {
    router := setupTestRouter()

    req := httptest.NewRequest("GET", "/api/v1/examples", nil)
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var resp Response
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, 0, resp.Code)
}
```

## Running Tests

### Go Test Commands

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/domain/example/...

# Run specific test
go test -run TestService_GetExample ./internal/domain/example/

# Show coverage
make test-coverage

# Skip integration tests
go test -short ./...
```

### TypeScript Test Commands

```bash
# Run tests
npm test

# Run with coverage
npm run test:coverage
```

## Testing Best Practices

### Do

- Test names describe expected behavior
- Use English comments to explain test intent
- One test verifies one behavior
- Keep tests independent, no execution order dependency
- Use `t.Helper()` to mark helper functions

### Don't

- Don't test private functions (test through public interfaces)
- Don't use `time.Sleep` in tests (use channels or mocks)
- Don't ignore error return values
- Don't share test state
- Don't test third-party library behavior
