package rag

// TurnSummary 对话总结结构
type TurnSummary struct {
	MainTopic    string   `json:"main_topic"`    // 主要主题
	Problem      string   `json:"problem"`       // 问题/需求
	Solution     string   `json:"solution"`      // 解决方案
	TechStack    []string `json:"tech_stack"`    // 技术栈
	CodeSnippets []string `json:"code_snippets"` // 代码片段
	KeyPoints    []string `json:"key_points"`    // 关键知识点
	Lessons      []string `json:"lessons"`       // 经验教训
	Tags         []string `json:"tags"`          // 标签
	Summary      string   `json:"summary"`       // 一句话总结
	Context      string   `json:"context"`       // 精华上下文
}
