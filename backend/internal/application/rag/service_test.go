package rag

import (
	"testing"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	domainRAG "github.com/cocursor/backend/internal/domain/rag"
	"github.com/stretchr/testify/mock"
)

// MockSessionService 模拟 SessionService
type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) GetSessionTextContent(sessionID string) ([]*domainCursor.Message, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domainCursor.Message), args.Error(1)
}

// MockEmbeddingClient 模拟 EmbeddingClient
type MockEmbeddingClient struct {
	mock.Mock
}

func (m *MockEmbeddingClient) EmbedTexts(texts []string) ([][]float32, error) {
	args := m.Called(texts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]float32), args.Error(1)
}

func (m *MockEmbeddingClient) GetVectorDimension() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockEmbeddingClient) TestConnection() error {
	args := m.Called()
	return args.Error(0)
}

// MockRAGRepository 模拟 RAGRepository
type MockRAGRepository struct {
	mock.Mock
}

func (m *MockRAGRepository) SaveMessageMetadata(metadata *domainRAG.MessageMetadata) error {
	args := m.Called(metadata)
	return args.Error(0)
}

func (m *MockRAGRepository) GetMessageMetadata(sessionID, messageID string) (*domainRAG.MessageMetadata, error) {
	args := m.Called(sessionID, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domainRAG.MessageMetadata), args.Error(1)
}

func (m *MockRAGRepository) GetMessageMetadataBySession(sessionID string) ([]*domainRAG.MessageMetadata, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domainRAG.MessageMetadata), args.Error(1)
}

func (m *MockRAGRepository) GetMessageMetadataByProject(projectID string) ([]*domainRAG.MessageMetadata, error) {
	args := m.Called(projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domainRAG.MessageMetadata), args.Error(1)
}

func (m *MockRAGRepository) DeleteMessageMetadata(sessionID, messageID string) error {
	args := m.Called(sessionID, messageID)
	return args.Error(0)
}

func (m *MockRAGRepository) SaveTurnMetadata(metadata *domainRAG.TurnMetadata) error {
	args := m.Called(metadata)
	return args.Error(0)
}

func (m *MockRAGRepository) GetTurnMetadata(sessionID string, turnIndex int) (*domainRAG.TurnMetadata, error) {
	args := m.Called(sessionID, turnIndex)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domainRAG.TurnMetadata), args.Error(1)
}

func (m *MockRAGRepository) GetTurnMetadataBySession(sessionID string) ([]*domainRAG.TurnMetadata, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domainRAG.TurnMetadata), args.Error(1)
}

func (m *MockRAGRepository) GetTurnMetadataByProject(projectID string) ([]*domainRAG.TurnMetadata, error) {
	args := m.Called(projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domainRAG.TurnMetadata), args.Error(1)
}

func (m *MockRAGRepository) GetIncompleteTurns(sessionID string) ([]*domainRAG.TurnMetadata, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domainRAG.TurnMetadata), args.Error(1)
}

func (m *MockRAGRepository) DeleteTurnMetadata(sessionID string, turnIndex int) error {
	args := m.Called(sessionID, turnIndex)
	return args.Error(0)
}

func (m *MockRAGRepository) GetFileMetadata(filePath string) (*domainRAG.FileMetadata, error) {
	args := m.Called(filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domainRAG.FileMetadata), args.Error(1)
}

func (m *MockRAGRepository) UpdateFileMtime(filePath string, mtime int64) error {
	args := m.Called(filePath, mtime)
	return args.Error(0)
}

// MockProjectManager 模拟 ProjectManager
type MockProjectManager struct {
	mock.Mock
}

func (m *MockProjectManager) GetProjectBySessionID(sessionID string) (*domainCursor.ProjectInfo, error) {
	args := m.Called(sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domainCursor.ProjectInfo), args.Error(1)
}

// TestIndexSession_Success 测试成功索引会话
// 注意：由于 RAGService 使用具体类型而非接口，这个测试需要真实的依赖
// 在实际集成测试中运行，这里提供测试框架
func TestIndexSession_Success(t *testing.T) {
	t.Skip("需要真实的 SessionService 和 QdrantManager，在集成测试中运行")

	// 测试步骤：
	// 1. 创建真实的 SessionService（需要数据库）
	// 2. 创建真实的 EmbeddingClient（需要 API）
	// 3. 创建真实的 QdrantManager（需要 Qdrant 实例）
	// 4. 创建测试会话数据
	// 5. 执行索引
	// 6. 验证结果
}

// TestIndexSession_EmptyMessages 测试空消息列表
func TestIndexSession_EmptyMessages(t *testing.T) {
	t.Skip("需要真实的依赖，在集成测试中运行")
}

// TestIndexSession_EmbeddingError 测试向量化错误
func TestIndexSession_EmbeddingError(t *testing.T) {
	t.Skip("需要真实的依赖，在集成测试中运行")
}
