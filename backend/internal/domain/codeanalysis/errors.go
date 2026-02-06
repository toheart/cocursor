package codeanalysis

import "fmt"

const (
	// ErrorCodeAlgorithmFailed 算法执行失败
	ErrorCodeAlgorithmFailed = "ALGORITHM_FAILED"
)

// AlgorithmFailedError 算法失败错误
type AlgorithmFailedError struct {
	// Algorithm 失败的算法
	Algorithm AlgorithmType
	// Reason 失败原因
	Reason string
	// Suggestion 建议操作
	Suggestion string
	// Details 详细信息
	Details string
}

// Error 返回错误描述
func (e *AlgorithmFailedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Reason != "" {
		return fmt.Sprintf("%s 算法分析失败: %s", e.Algorithm, e.Reason)
	}
	return fmt.Sprintf("%s 算法分析失败", e.Algorithm)
}
