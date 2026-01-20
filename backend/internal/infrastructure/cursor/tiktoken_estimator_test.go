package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTiktokenEstimator(t *testing.T) {
	// 测试单例模式
	estimator1, err := GetTiktokenEstimator()
	require.NoError(t, err, "should create estimator without error")
	require.NotNil(t, estimator1, "estimator should not be nil")

	estimator2, err := GetTiktokenEstimator()
	require.NoError(t, err, "should get estimator without error")
	require.NotNil(t, estimator2, "estimator should not be nil")

	// 确保是同一个实例
	assert.Same(t, estimator1, estimator2, "should return the same instance")
}

func TestTiktokenEstimator_CountTokens(t *testing.T) {
	estimator, err := GetTiktokenEstimator()
	require.NoError(t, err)

	tests := []struct {
		name     string
		text     string
		minCount int // 最小预期 token 数
		maxCount int // 最大预期 token 数
	}{
		{
			name:     "空字符串",
			text:     "",
			minCount: 0,
			maxCount: 0,
		},
		{
			name:     "简单英文",
			text:     "Hello, world!",
			minCount: 3,
			maxCount: 5,
		},
		{
			name:     "简单中文",
			text:     "你好世界",
			minCount: 2,
			maxCount: 8,
		},
		{
			name:     "代码片段",
			text:     "func main() {\n\tfmt.Println(\"Hello\")\n}",
			minCount: 10,
			maxCount: 20,
		},
		{
			name:     "长文本",
			text:     "The quick brown fox jumps over the lazy dog. This is a test sentence that should produce a reasonable number of tokens.",
			minCount: 20,
			maxCount: 30,
		},
		{
			name:     "混合中英文",
			text:     "Hello 你好，这是一个测试 test",
			minCount: 5,
			maxCount: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := estimator.CountTokens(tt.text)
			assert.GreaterOrEqual(t, count, tt.minCount, "token count should be >= minCount")
			assert.LessOrEqual(t, count, tt.maxCount, "token count should be <= maxCount")
		})
	}
}

func TestTiktokenEstimator_CountTokensBatch(t *testing.T) {
	estimator, err := GetTiktokenEstimator()
	require.NoError(t, err)

	texts := []string{
		"Hello, world!",
		"你好世界",
		"func main() {}",
	}

	// 批量计数应该等于单独计数之和
	batchCount := estimator.CountTokensBatch(texts)
	
	var singleSum int
	for _, text := range texts {
		singleSum += estimator.CountTokens(text)
	}

	assert.Equal(t, singleSum, batchCount, "batch count should equal sum of individual counts")
}

func TestTiktokenEstimator_GetMethod(t *testing.T) {
	estimator, err := GetTiktokenEstimator()
	require.NoError(t, err)

	method := estimator.GetMethod()
	assert.Equal(t, "tiktoken", method, "method should be 'tiktoken'")
}

func TestTiktokenEstimator_Consistency(t *testing.T) {
	estimator, err := GetTiktokenEstimator()
	require.NoError(t, err)

	// 相同文本应该返回相同的 token 数
	text := "This is a test for consistency."
	count1 := estimator.CountTokens(text)
	count2 := estimator.CountTokens(text)
	count3 := estimator.CountTokens(text)

	assert.Equal(t, count1, count2, "token count should be consistent")
	assert.Equal(t, count2, count3, "token count should be consistent")
}
