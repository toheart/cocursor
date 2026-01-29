package cursor

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
)

// StatsService 统计服务（包含所有统计相关的计算逻辑）
type StatsService struct {
	globalDBReader domainCursor.GlobalDBReader
}

// NewStatsService 创建统计服务实例
func NewStatsService(globalDBReader domainCursor.GlobalDBReader) *StatsService {
	return &StatsService{
		globalDBReader: globalDBReader,
	}
}

// CalculateSessionEntropy 计算会话熵值
// 公式：Score = (ContextUsage * 0.4) + (FileCount * 2.0) + (log(TotalLines) * 1.5)
// 参数：
//   - data: ComposerData 会话数据
//
// 返回：熵值（float64）
func (s *StatsService) CalculateSessionEntropy(data domainCursor.ComposerData) float64 {
	// 上下文使用率（0-100）
	contextUsage := data.ContextUsagePercent

	// 文件数量
	fileCount := float64(data.FilesChangedCount)

	// 总代码行数（添加 + 删除）
	totalLines := float64(data.GetTotalLinesChanged())
	if totalLines < 1 {
		totalLines = 1 // 避免 log(0)
	}

	// 计算对数（使用自然对数）
	logTotalLines := math.Log(totalLines)

	// 应用权重计算熵值
	// ContextUsage * 0.4 + FileCount * 2.0 + log(TotalLines) * 1.5
	entropy := (contextUsage * 0.4) + (fileCount * 2.0) + (logTotalLines * 1.5)

	return entropy
}

// GetHealthStatus 获取会话健康状态
// 参数：
//   - entropy: 熵值
//
// 返回：健康状态和警告信息
func (s *StatsService) GetHealthStatus(entropy float64) (status HealthStatus, warning string) {
	switch {
	case entropy < 40:
		return HealthStatusHealthy, ""
	case entropy < 70:
		return HealthStatusSubHealthy, "建议总结当前会话，熵值较高"
	default:
		return HealthStatusDangerous, "会话熵值过高，AI 极易产生幻觉，建议强制开启新会话"
	}
}

// HealthStatus 健康状态
type HealthStatus string

const (
	// HealthStatusHealthy 健康状态（熵 < 40）
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusSubHealthy 亚健康状态（40 <= 熵 < 70）
	HealthStatusSubHealthy HealthStatus = "sub_healthy"
	// HealthStatusDangerous 危险状态（熵 >= 70）
	HealthStatusDangerous HealthStatus = "dangerous"
)

// HealthInfo 健康信息
type HealthInfo struct {
	Entropy float64      `json:"entropy"` // 熵值
	Status  HealthStatus `json:"status"`  // 健康状态
	Warning string       `json:"warning"` // 警告信息（如果有）
}

// GetAcceptanceRateStats 获取接受率统计（支持日期范围）
// startDate: 开始日期 YYYY-MM-DD
// endDate: 结束日期 YYYY-MM-DD
// 返回: DailyAcceptanceStats 切片和错误
func (s *StatsService) GetAcceptanceRateStats(startDate, endDate string) ([]*domainCursor.DailyAcceptanceStats, error) {
	// 解析日期
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}

	// 验证日期范围（最大 90 天）
	days := int(end.Sub(start).Hours() / 24)
	if days > 90 {
		return nil, fmt.Errorf("date range exceeds maximum 90 days")
	}
	if days < 0 {
		return nil, fmt.Errorf("start_date must be before end_date")
	}

	var results []*domainCursor.DailyAcceptanceStats

	// 遍历日期范围
	current := start
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")
		key := fmt.Sprintf("aiCodeTracking.dailyStats.v1.5.%s", dateStr)

		// 使用 GlobalDBReader 读取数据
		value, err := s.globalDBReader.ReadValue(key)
		if err != nil {
			// 数据不存在时跳过，不返回错误
			current = current.AddDate(0, 0, 1)
			continue
		}

		// 解析数据
		stats, err := domainCursor.ParseAcceptanceStats(string(value), dateStr)
		if err != nil {
			// 解析失败时跳过
			current = current.AddDate(0, 0, 1)
			continue
		}

		results = append(results, stats)
		current = current.AddDate(0, 0, 1)
	}

	return results, nil
}

// GetConversationOverview 获取对话统计概览
// workspaceID: 工作区 ID
// 返回: ConversationOverview 和错误
func (s *StatsService) GetConversationOverview(workspaceID string) (*domainCursor.ConversationOverview, error) {
	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 获取工作区数据库路径
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace DB path: %w", err)
	}

	overview := &domainCursor.ConversationOverview{}

	// 读取 prompts
	promptsValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.prompts")
	if err == nil && len(promptsValue) > 0 {
		var prompts []map[string]interface{}
		if err := json.Unmarshal(promptsValue, &prompts); err == nil {
			overview.TotalChats = len(prompts)
		}
	}

	// 读取 generations
	generationsValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.generations")
	if err == nil && len(generationsValue) > 0 {
		generations, err := domainCursor.ParseGenerationsData(string(generationsValue))
		if err == nil {
			overview.TotalGenerations = len(generations)
			// 获取最近生成时间
			latestTime := domainCursor.GetLatestGenerationTime(generations)
			if !latestTime.IsZero() {
				overview.LatestChatTime = latestTime.Format(time.RFC3339)
				overview.TimeSinceLastGen = domainCursor.GetTimeSinceLastGeneration(generations)
			}
		}
	}

	// 读取 composer 数据统计活跃会话
	composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err == nil && len(composerDataValue) > 0 {
		composers, err := domainCursor.ParseComposerData(string(composerDataValue))
		if err == nil {
			// 统计未归档的会话
			for _, c := range composers {
				if !c.IsArchived {
					overview.ActiveSessions++
				}
			}
		}
	}

	return overview, nil
}

// GetFileReferences 获取文件引用分析（TopN）
// workspaceID: 工作区 ID
// topN: 返回前 N 个文件（默认 10，最大 50，最小 1）
// 返回: FileReference 切片和错误
func (s *StatsService) GetFileReferences(workspaceID string, topN int) ([]*domainCursor.FileReference, error) {
	// 验证 topN 参数
	if topN < 1 {
		topN = 10
	}
	if topN > 50 {
		topN = 50
	}

	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 获取工作区数据库路径
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace DB path: %w", err)
	}

	// 读取 composer 数据
	composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err != nil {
		return []*domainCursor.FileReference{}, nil // 返回空数组而非错误
	}

	composers, err := domainCursor.ParseComposerData(string(composerDataValue))
	if err != nil {
		return []*domainCursor.FileReference{}, nil
	}

	// 统计文件引用
	fileMap := make(map[string]int)
	for _, composer := range composers {
		if composer.Subtitle == "" {
			continue
		}
		// 解析 subtitle（逗号分隔的文件列表）
		files := strings.Split(composer.Subtitle, ",")
		for _, file := range files {
			file = strings.TrimSpace(file)
			if file != "" {
				fileMap[file]++
			}
		}
	}

	// 转换为 FileReference 切片
	var refs []*domainCursor.FileReference
	for fileName, count := range fileMap {
		ref := &domainCursor.FileReference{
			FileName:       fileName,
			ReferenceCount: count,
			FileType:       getFileType(fileName),
		}
		refs = append(refs, ref)
	}

	// 按引用次数排序
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ReferenceCount > refs[j].ReferenceCount
	})

	// 返回 TopN
	if len(refs) > topN {
		refs = refs[:topN]
	}

	return refs, nil
}

// getFileType 从文件名推断文件类型
func getFileType(fileName string) string {
	parts := strings.Split(fileName, ".")
	if len(parts) < 2 {
		return "unknown"
	}
	ext := strings.ToLower(parts[len(parts)-1])
	return ext
}

// GenerateDailyReport 生成日报（聚合多个数据源）
// workspaceID: 工作区 ID
// date: 日期 YYYY-MM-DD
// topNSessions: Top N 会话数（默认 5，最大 20）
// topNFiles: Top N 文件数（默认 10，最大 50）
// 返回: DailyReport 和错误
func (s *StatsService) GenerateDailyReport(workspaceID, date string, topNSessions, topNFiles int) (*domainCursor.DailyReport, error) {
	// 验证 topN 参数
	if topNSessions < 1 {
		topNSessions = 5
	}
	if topNSessions > 20 {
		topNSessions = 20
	}
	if topNFiles < 1 {
		topNFiles = 10
	}
	if topNFiles > 50 {
		topNFiles = 50
	}

	report := &domainCursor.DailyReport{
		Date:        date,
		WorkspaceID: workspaceID,
	}

	pathResolver := infraCursor.NewPathResolver()
	dbReader := infraCursor.NewDBReader()

	// 1. 获取接受率统计
	acceptanceStats, err := s.GetAcceptanceRateStats(date, date)
	if err == nil && len(acceptanceStats) > 0 {
		report.AcceptanceRate = acceptanceStats[0]
	}

	// 2. 获取对话统计概览
	overview, err := s.GetConversationOverview(workspaceID)
	if err == nil {
		report.AIUsage = &domainCursor.AIUsageSummary{
			TotalChats:       overview.TotalChats,
			TotalGenerations: overview.TotalGenerations,
			ActiveSessions:   overview.ActiveSessions,
		}
	}

	// 3. 获取工作区数据库路径并统计代码变更
	workspaceDBPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err == nil {
		s.populateComposerStats(report, dbReader, workspaceDBPath, date, topNSessions)

		// 4. 获取 Top 文件引用
		if topFiles, err := s.GetFileReferences(workspaceID, topNFiles); err == nil {
			report.TopFiles = topFiles
		}
	}

	return report, nil
}

// populateComposerStats 从 composer 数据填充报告的代码变更和 Top 会话
func (s *StatsService) populateComposerStats(
	report *domainCursor.DailyReport,
	dbReader *infraCursor.DBReader,
	workspaceDBPath, date string,
	topNSessions int,
) {
	composerDataValue, err := dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
	if err != nil || len(composerDataValue) == 0 {
		return
	}

	composers, err := domainCursor.ParseComposerData(string(composerDataValue))
	if err != nil {
		return
	}

	codeChanges, sessionSummaries := s.aggregateComposerData(composers, date)
	report.CodeChanges = codeChanges
	report.TopSessions = s.getTopSessions(sessionSummaries, topNSessions)
}

// aggregateComposerData 聚合 composer 数据，返回代码变更汇总和会话摘要列表
func (s *StatsService) aggregateComposerData(
	composers []domainCursor.ComposerData,
	date string,
) (*domainCursor.CodeChangeSummary, []*domainCursor.SessionSummary) {
	codeChanges := &domainCursor.CodeChangeSummary{}
	var sessionSummaries []*domainCursor.SessionSummary

	for _, composer := range composers {
		// 过滤指定日期的会话（根据创建时间或更新时间）
		createdDate := composer.GetCreatedAtTime().Format("2006-01-02")
		updatedDate := composer.GetLastUpdatedAtTime().Format("2006-01-02")
		if createdDate != date && updatedDate != date {
			continue
		}

		codeChanges.TotalLinesAdded += composer.TotalLinesAdded
		codeChanges.TotalLinesRemoved += composer.TotalLinesRemoved
		if composer.FilesChangedCount > 0 {
			codeChanges.FilesChanged++
		}

		// 构建会话摘要
		entropy := s.CalculateSessionEntropy(composer)
		sessionSummary := &domainCursor.SessionSummary{
			ComposerID:   composer.ComposerID,
			Name:         composer.Name,
			TotalLines:   composer.GetTotalLinesChanged(),
			FilesChanged: composer.FilesChangedCount,
			Entropy:      entropy,
			Duration:     composer.GetDurationMinutes(),
		}
		sessionSummaries = append(sessionSummaries, sessionSummary)
	}

	return codeChanges, sessionSummaries
}

// getTopSessions 按总变更行数排序并返回 Top N 会话
func (s *StatsService) getTopSessions(sessions []*domainCursor.SessionSummary, topN int) []*domainCursor.SessionSummary {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].TotalLines > sessions[j].TotalLines
	})
	if len(sessions) > topN {
		return sessions[:topN]
	}
	return sessions
}
