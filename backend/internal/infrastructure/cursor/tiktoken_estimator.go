package cursor

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

// 在包初始化时设置离线加载器
func init() {
	tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
}

// TiktokenEstimator 使用 tiktoken 精确估算 Token 数量
type TiktokenEstimator struct {
	encoding *tiktoken.Tiktoken
	mu       sync.RWMutex
}

// tiktokenInstance 单例实例
var (
	tiktokenInstance *TiktokenEstimator
	tiktokenOnce     sync.Once
	tiktokenErr      error
)

// GetTiktokenEstimator 获取 TiktokenEstimator 单例
// 使用单例模式避免重复加载编码文件
func GetTiktokenEstimator() (*TiktokenEstimator, error) {
	tiktokenOnce.Do(func() {
		// 使用 cl100k_base 编码（GPT-4、Claude 等模型兼容）
		enc, err := tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			tiktokenErr = err
			return
		}
		tiktokenInstance = &TiktokenEstimator{
			encoding: enc,
		}
	})

	if tiktokenErr != nil {
		return nil, tiktokenErr
	}
	return tiktokenInstance, nil
}

// CountTokens 计算文本的 Token 数量
func (e *TiktokenEstimator) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	tokens := e.encoding.Encode(text, nil, nil)
	return len(tokens)
}

// CountTokensBatch 批量计算多个文本的 Token 数量
func (e *TiktokenEstimator) CountTokensBatch(texts []string) int {
	total := 0
	for _, text := range texts {
		total += e.CountTokens(text)
	}
	return total
}

// GetMethod 返回计算方法标识
func (e *TiktokenEstimator) GetMethod() string {
	return "tiktoken"
}
