package cursor

import (
	"runtime"
	"strings"
)

// PathMatcher 路径匹配器
type PathMatcher struct{}

// NewPathMatcher 创建路径匹配器实例
func NewPathMatcher() *PathMatcher {
	return &PathMatcher{}
}

// CalculatePathSimilarity 计算路径相似度
// 使用最长公共子序列算法
// 返回: 相似度（0.0-1.0）
func (m *PathMatcher) CalculatePathSimilarity(path1, path2 string) float64 {
	// 规范化路径
	norm1 := m.simplifyPath(path1)
	norm2 := m.simplifyPath(path2)

	// 计算最长公共子序列
	lcs := m.longestCommonSubsequence(norm1, norm2)

	// 计算相似度：LCS 长度 / 最大路径长度
	maxLength := max(len(norm1), len(norm2))
	if maxLength == 0 {
		return 0.0
	}

	similarity := float64(len(lcs)) / float64(maxLength)
	return similarity
}

// simplifyPath 简化路径（统一分隔符、移除尾部斜杠、转小写）
func (m *PathMatcher) simplifyPath(path string) string {
	// 1. 统一分隔符为 /
	path = strings.ReplaceAll(path, "\\", "/")

	// 2. 移除尾部斜杠
	path = strings.TrimRight(path, "/")

	// 3. Windows 转小写（大小写不敏感），Unix 保持原样
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}

	return path
}

// longestCommonSubsequence 计算最长公共子序列
// 使用动态规划算法
func (m *PathMatcher) longestCommonSubsequence(s1, s2 string) string {
	n1, n2 := len(s1), len(s2)
	if n1 == 0 || n2 == 0 {
		return ""
	}

	// 创建 DP 表
	dp := make([][]int, n1+1)
	for i := range dp {
		dp[i] = make([]int, n2+1)
	}

	// 填充 DP 表
	for i := 1; i <= n1; i++ {
		for j := 1; j <= n2; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// 回溯构建 LCS
	lcs := make([]byte, 0, dp[n1][n2])
	i, j := n1, n2
	for i > 0 && j > 0 {
		if s1[i-1] == s2[j-1] {
			lcs = append([]byte{s1[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return string(lcs)
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
