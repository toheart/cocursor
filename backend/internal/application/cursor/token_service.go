package cursor

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
)

// TokenService Token 计算服务
type TokenService struct {
	dbReader          *infraCursor.DBReader
	pathResolver      *infraCursor.PathResolver
	tiktokenEstimator *infraCursor.TiktokenEstimator
}

// NewTokenService 创建 Token 计算服务实例
func NewTokenService() *TokenService {
	// 尝试获取 tiktoken 估算器，失败时使用字符估算作为 fallback
	estimator, err := infraCursor.GetTiktokenEstimator()
	if err != nil {
		// 记录警告但不阻止服务启动
		// 后续 countTokens 会使用字符估算作为 fallback
		estimator = nil
	}

	return &TokenService{
		dbReader:          infraCursor.NewDBReader(),
		pathResolver:      infraCursor.NewPathResolver(),
		tiktokenEstimator: estimator,
	}
}

// GetTokenUsage 获取 Token 使用统计
// date: 日期 YYYY-MM-DD，如果为空则使用今天
// projectName: 项目名称（可选），如果提供则只统计该项目，否则统计所有项目
// 返回: TokenUsage 和错误
func (s *TokenService) GetTokenUsage(date string, projectName string) (*domainCursor.TokenUsage, error) {
	// 如果没有提供日期，使用今天
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// 计算昨日日期（用于趋势对比）
	yesterday := targetDate.AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")

	// 获取工作区列表
	var workspaceIDs []string
	if projectName != "" {
		// 如果指定了项目，只查询该项目的工作区
		// TODO: 需要 ProjectManager 支持，暂时先查询所有工作区
		workspaceIDs, err = s.getAllWorkspaceIDs()
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace IDs: %w", err)
		}
	} else {
		// 查询所有工作区
		workspaceIDs, err = s.getAllWorkspaceIDs()
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace IDs: %w", err)
		}
	}

	// 统计今日 Token
	todayUsage := &domainCursor.TokenByType{}
	for _, workspaceID := range workspaceIDs {
		usage, err := s.calculateWorkspaceTokenUsage(workspaceID, date)
		if err != nil {
			// 静默失败，继续处理其他工作区
			continue
		}
		todayUsage.Tab += usage.Tab
		todayUsage.Composer += usage.Composer
		todayUsage.Chat += usage.Chat
	}

	// 统计昨日 Token（用于趋势对比）
	yesterdayUsage := &domainCursor.TokenByType{}
	for _, workspaceID := range workspaceIDs {
		usage, err := s.calculateWorkspaceTokenUsage(workspaceID, yesterdayStr)
		if err != nil {
			continue
		}
		yesterdayUsage.Tab += usage.Tab
		yesterdayUsage.Composer += usage.Composer
		yesterdayUsage.Chat += usage.Chat
	}

	// 计算总 Token
	totalToday := todayUsage.Tab + todayUsage.Composer + todayUsage.Chat
	totalYesterday := yesterdayUsage.Tab + yesterdayUsage.Composer + yesterdayUsage.Chat

	// 计算趋势
	trend := s.calculateTrend(totalToday, totalYesterday)

	// 确定计算方法
	method := "estimate"
	if s.tiktokenEstimator != nil {
		method = s.tiktokenEstimator.GetMethod()
	}

	return &domainCursor.TokenUsage{
		Date:        date,
		TotalTokens: totalToday,
		ByType:      *todayUsage,
		Trend:       trend,
		Method:      method,
	}, nil
}

// calculateWorkspaceTokenUsage 计算单个工作区的 Token 使用
func (s *TokenService) calculateWorkspaceTokenUsage(workspaceID string, date string) (*domainCursor.TokenByType, error) {
	// 获取工作区数据库路径
	workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace DB path: %w", err)
	}

	usage := &domainCursor.TokenByType{}

	// 读取 prompts 和 generations
	promptsValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.prompts")
	if err != nil {
		// 如果没有数据，返回空统计
		return usage, nil
	}

	generationsValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.generations")
	if err != nil {
		// 如果没有数据，返回空统计
		return usage, nil
	}

	// 解析 prompts
	var prompts []map[string]interface{}
	if err := json.Unmarshal(promptsValue, &prompts); err != nil {
		return usage, nil
	}

	// 解析 generations
	generations, err := domainCursor.ParseGenerationsData(string(generationsValue))
	if err != nil {
		return usage, nil
	}

	// 过滤指定日期的数据
	targetDate, _ := time.Parse("2006-01-02", date)
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// 统计 prompts（用户输入）
	for _, prompt := range prompts {
		// prompts 没有时间戳，暂时统计所有
		// TODO: 需要探索 prompts 数据结构，看是否有时间戳字段
		text, ok := prompt["text"].(string)
		if !ok {
			continue
		}
		tokens := s.countTokens(text)
		// prompts 默认归类为 Chat（需要根据 commandType 进一步判断）
		usage.Chat += tokens
	}

	// 统计 generations（AI 回复）
	for _, gen := range generations {
		// 检查是否在目标日期范围内
		genTime := time.Unix(0, gen.UnixMs*int64(time.Millisecond))
		if genTime.Before(startOfDay) || genTime.After(endOfDay) {
			continue
		}

		tokens := s.countTokens(gen.TextDescription)
		switch gen.Type {
		case "tab":
			usage.Tab += tokens
		case "composer":
			usage.Composer += tokens
		case "chat":
			usage.Chat += tokens
		default:
			// 默认归类为 Chat
			usage.Chat += tokens
		}
	}

	return usage, nil
}

// countTokens 计算 Token 数量
// 优先使用 tiktoken 精确计算，如果不可用则使用字符估算作为 fallback
func (s *TokenService) countTokens(text string) int {
	if text == "" {
		return 0
	}

	// 优先使用 tiktoken 精确计算
	if s.tiktokenEstimator != nil {
		return s.tiktokenEstimator.CountTokens(text)
	}

	// Fallback: 使用字符估算
	return s.estimateTokensFallback(text)
}

// estimateTokensFallback 字符估算 Token 数量（fallback 方法）
// 使用粗略估算：1 token ≈ 4 字符（英文）或 1.5 字符（中文）
func (s *TokenService) estimateTokensFallback(text string) int {
	if text == "" {
		return 0
	}

	// 简单估算：中文字符按 1.5 字符/token，其他按 4 字符/token
	chineseChars := 0
	otherChars := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			// 中文字符范围
			chineseChars++
		} else {
			otherChars++
		}
	}

	// 估算：中文 1.5 字符/token，其他 4 字符/token
	tokens := int(float64(chineseChars)/1.5) + int(float64(otherChars)/4)
	if tokens < 1 {
		tokens = 1 // 至少 1 个 token
	}

	return tokens
}

// calculateTrend 计算趋势（与昨日对比）
func (s *TokenService) calculateTrend(today, yesterday int) string {
	if yesterday == 0 {
		if today > 0 {
			return "+100%"
		}
		return "0%"
	}

	change := float64(today-yesterday) / float64(yesterday) * 100
	if change > 0 {
		return fmt.Sprintf("+%.1f%%", change)
	} else if change < 0 {
		return fmt.Sprintf("%.1f%%", change)
	}
	return "0%"
}

// getAllWorkspaceIDs 获取所有工作区 ID
func (s *TokenService) getAllWorkspaceIDs() ([]string, error) {
	workspaceDir, err := s.pathResolver.GetWorkspaceStorageDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace storage directory: %w", err)
	}

	// 读取所有子目录
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace storage directory: %w", err)
	}

	var workspaceIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			workspaceIDs = append(workspaceIDs, entry.Name())
		}
	}

	return workspaceIDs, nil
}
