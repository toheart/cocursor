package cursor

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePathSimilarity_Identical(t *testing.T) {
	matcher := NewPathMatcher()

	similarity := matcher.CalculatePathSimilarity(
		"/path/to/project",
		"/path/to/project",
	)
	assert.Equal(t, 1.0, similarity)
}

func TestCalculatePathSimilarity_DifferentSeparators(t *testing.T) {
	matcher := NewPathMatcher()

	similarity := matcher.CalculatePathSimilarity(
		"/path/to/project",
		"\\path\\to\\project",
	)
	assert.Equal(t, 1.0, similarity)
}

func TestCalculatePathSimilarity_Similar(t *testing.T) {
	matcher := NewPathMatcher()

	similarity := matcher.CalculatePathSimilarity(
		"/path/to/project",
		"/path/to/project-backup",
	)
	// 相似度应该较高（> 0.5）
	assert.Greater(t, similarity, 0.5)
}

func TestCalculatePathSimilarity_Different(t *testing.T) {
	matcher := NewPathMatcher()

	similarity := matcher.CalculatePathSimilarity(
		"/completely/different/path",
		"/another/unrelated/location",
	)
	// 完全不同的路径相似度应该较低
	assert.Less(t, similarity, 0.5)
}

func TestCalculatePathSimilarity_Empty(t *testing.T) {
	matcher := NewPathMatcher()

	similarity := matcher.CalculatePathSimilarity("", "")
	assert.Equal(t, 0.0, similarity)
}

func TestSimplifyPath(t *testing.T) {
	matcher := NewPathMatcher()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Unix path",
			input:    "/path/to/project",
			expected: "/path/to/project",
		},
		{
			name:  "Windows path",
			input: "C:\\path\\to\\project",
			expected: func() string {
				if runtime.GOOS == "windows" {
					// Windows 上会转小写
					return "c:/path/to/project"
				}
				// 非 Windows 上保持原大小写，只转换分隔符
				return "C:/path/to/project"
			}(),
		},
		{
			name:     "Trailing slash",
			input:    "/path/to/project/",
			expected: "/path/to/project",
		},
		{
			name:     "Mixed separators",
			input:    "/path\\to/project",
			expected: "/path/to/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.simplifyPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLongestCommonSubsequence(t *testing.T) {
	matcher := NewPathMatcher()

	tests := []struct {
		name     string
		s1       string
		s2       string
		expected string
	}{
		{
			name:     "Identical",
			s1:       "abc",
			s2:       "abc",
			expected: "abc",
		},
		{
			name:     "Common prefix",
			s1:       "abcde",
			s2:       "abcfg",
			expected: "abc",
		},
		{
			name:     "No common",
			s1:       "abc",
			s2:       "def",
			expected: "",
		},
		{
			name:     "Empty",
			s1:       "",
			s2:       "abc",
			expected: "",
		},
		{
			name:     "Subsequence",
			s1:       "abcde",
			s2:       "ace",
			expected: "ace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.longestCommonSubsequence(tt.s1, tt.s2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMax(t *testing.T) {
	assert.Equal(t, 5, max(5, 3))
	assert.Equal(t, 5, max(3, 5))
	assert.Equal(t, 0, max(0, 0))
	assert.Equal(t, -1, max(-5, -1))
}
