package cursor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// WorkAnalysisService 工作分析服务
type WorkAnalysisService struct {
	statsService   *StatsService
	projectManager *ProjectManager
	sessionRepo    storage.WorkspaceSessionRepository
	dataMerger     *DataMerger
}

// NewWorkAnalysisService 创建工作分析服务实例（接受 Repository 作为参数）
func NewWorkAnalysisService(
	statsService *StatsService,
	projectManager *ProjectManager,
	sessionRepo storage.WorkspaceSessionRepository,
	dataMerger *DataMerger,
) *WorkAnalysisService {
	return &WorkAnalysisService{
		statsService:   statsService,
		projectManager: projectManager,
		sessionRepo:    sessionRepo,
		dataMerger:     dataMerger,
	}
}

// GetWorkAnalysis 获取工作分析数据（全局视图）
// startDate: 开始日期 YYYY-MM-DD
// endDate: 结束日期 YYYY-MM-DD
// 返回: WorkAnalysis 和错误
func (s *WorkAnalysisService) GetWorkAnalysis(startDate, endDate string) (*domainCursor.WorkAnalysis, error) {
	// 验证日期格式
	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}

	// 获取所有工作区（全局聚合）
	var workspaceIDs []string
	projects := s.projectManager.ListAllProjects()
	for _, project := range projects {
		for _, ws := range project.Workspaces {
			workspaceIDs = append(workspaceIDs, ws.WorkspaceID)
		}
	}

	// 聚合数据
	analysis := &domainCursor.WorkAnalysis{
		Overview:         &domainCursor.WorkAnalysisOverview{},
		DailyDetails:     []*domainCursor.DailyAnalysis{},
		CodeChangesTrend: []*domainCursor.DailyCodeChanges{},
		TopFiles:         []*domainCursor.FileReference{},
		TimeDistribution: []*domainCursor.TimeDistributionItem{},
		EfficiencyMetrics: &domainCursor.EfficiencyMetrics{
			EntropyTrend: []*domainCursor.EntropyTrendItem{},
		},
	}

	// 文件引用统计（跨工作区聚合）
	fileMap := make(map[string]int)

	// 按日期统计代码变更
	dailyChangesMap := make(map[string]*domainCursor.DailyCodeChanges)

	// 时间分布统计（小时 × 星期）
	timeDistMap := make(map[int]map[int]int) // hour -> day -> count

	// 熵值趋势统计
	entropyTrendMap := make(map[string][]float64) // date -> []entropy

	var totalEntropySum float64
	var totalEntropyCount int
	var totalContextUsageSum float64
	var totalContextUsageCount int

	// 从缓存表查询会话数据
	sessions, err := s.sessionRepo.FindByWorkspacesAndDateRange(workspaceIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions from cache: %w", err)
	}

	// 处理每个会话
	for _, session := range sessions {
		// 转换为 ComposerData
		composer := s.sessionToComposerData(session)

		// 检查是否在日期范围内（已经在查询中过滤，但为了安全再次检查）
		createdDate := composer.GetCreatedAtTime().Format("2006-01-02")
		updatedDate := composer.GetLastUpdatedAtTime().Format("2006-01-02")

		// 如果创建或更新日期在范围内，则包含
		createdInRange := s.isDateInRange(createdDate, startDate, endDate)
		updatedInRange := s.isDateInRange(updatedDate, startDate, endDate)
		if !createdInRange && !updatedInRange {
			continue
		}

		// 统计文件引用
		if composer.Subtitle != "" {
			files := s.parseFileList(composer.Subtitle)
			for _, file := range files {
				fileMap[file]++
			}
		}

		// 统计代码变更（按日期）
		date := createdDate
		if updatedInRange {
			date = updatedDate
		}
		if dailyChangesMap[date] == nil {
			dailyChangesMap[date] = &domainCursor.DailyCodeChanges{
				Date:         date,
				LinesAdded:   0,
				LinesRemoved: 0,
				FilesChanged: 0,
			}
		}
		dailyChangesMap[date].LinesAdded += composer.TotalLinesAdded
		dailyChangesMap[date].LinesRemoved += composer.TotalLinesRemoved
		if composer.FilesChangedCount > 0 {
			dailyChangesMap[date].FilesChanged++
		}

		// 统计时间分布（活跃时段）
		composerTime := composer.GetCreatedAtTime()
		hour := composerTime.Hour()
		day := int(composerTime.Weekday())
		if timeDistMap[hour] == nil {
			timeDistMap[hour] = make(map[int]int)
		}
		timeDistMap[hour][day]++

		// 统计熵值和上下文使用率
		entropy := s.statsService.CalculateSessionEntropy(composer)
		totalEntropySum += entropy
		totalEntropyCount++
		totalContextUsageSum += composer.ContextUsagePercent
		totalContextUsageCount++

		// 熵值趋势（按日期）
		if entropyTrendMap[date] == nil {
			entropyTrendMap[date] = []float64{}
		}
		entropyTrendMap[date] = append(entropyTrendMap[date], entropy)
	}

	// 获取接受率统计数据（整周汇总）
	mergedAcceptanceStats, _, err := s.dataMerger.MergeAcceptanceStats(startDate, endDate)
	if err != nil {
		// 如果获取失败，设置为 0，不返回错误
		mergedAcceptanceStats = &domainCursor.DailyAcceptanceStats{}
	}

	// 计算整体接受率
	// 注意：如果 ComposerSuggestedLines 为 0 但 ComposerAcceptedLines > 0，说明数据异常
	// 这种情况下，应该只使用有有效建议行数的类型来计算接受率
	var overallAcceptanceRate float64
	totalSuggested := mergedAcceptanceStats.TabSuggestedLines + mergedAcceptanceStats.ComposerSuggestedLines
	totalAccepted := mergedAcceptanceStats.TabAcceptedLines + mergedAcceptanceStats.ComposerAcceptedLines

	// 如果总建议行数 > 0，且接受行数 <= 建议行数，正常计算
	if totalSuggested > 0 && totalAccepted <= totalSuggested {
		overallAcceptanceRate = float64(totalAccepted) / float64(totalSuggested) * 100
	} else if totalSuggested > 0 {
		// 如果接受行数 > 建议行数，说明数据异常，使用加权平均
		// 只使用有有效建议行数的类型
		if mergedAcceptanceStats.TabSuggestedLines > 0 && mergedAcceptanceStats.ComposerSuggestedLines > 0 {
			// 两个类型都有数据，使用加权平均
			tabWeight := float64(mergedAcceptanceStats.TabSuggestedLines) / float64(totalSuggested)
			composerWeight := float64(mergedAcceptanceStats.ComposerSuggestedLines) / float64(totalSuggested)
			overallAcceptanceRate = mergedAcceptanceStats.TabAcceptanceRate*tabWeight + mergedAcceptanceStats.ComposerAcceptanceRate*composerWeight
		} else if mergedAcceptanceStats.TabSuggestedLines > 0 {
			// 只有 Tab 有数据
			overallAcceptanceRate = mergedAcceptanceStats.TabAcceptanceRate
		} else if mergedAcceptanceStats.ComposerSuggestedLines > 0 {
			// 只有 Composer 有数据
			overallAcceptanceRate = mergedAcceptanceStats.ComposerAcceptanceRate
		} else {
			// 都没有有效数据，设为 0
			overallAcceptanceRate = 0
		}
	}

	// 构建概览
	if totalEntropyCount > 0 {
		analysis.Overview.TotalLinesAdded = s.sumDailyChanges(dailyChangesMap, "added")
		analysis.Overview.TotalLinesRemoved = s.sumDailyChanges(dailyChangesMap, "removed")
		analysis.Overview.FilesChanged = len(fileMap)
		analysis.Overview.AcceptanceRate = overallAcceptanceRate
		analysis.Overview.TabAcceptanceRate = mergedAcceptanceStats.TabAcceptanceRate
		analysis.Overview.ComposerAcceptanceRate = mergedAcceptanceStats.ComposerAcceptanceRate
		analysis.Overview.ActiveSessions = totalEntropyCount
	}
	// prompts 和 generations 统计已删除（需求不大）
	analysis.Overview.TotalPrompts = 0
	analysis.Overview.TotalGenerations = 0

	// 统计每日活跃会话数（避免重复计算）
	dailySessionCountMap := make(map[string]map[string]bool) // date -> composerID -> true
	for _, session := range sessions {
		composer := s.sessionToComposerData(session)
		createdDate := composer.GetCreatedAtTime().Format("2006-01-02")
		updatedDate := composer.GetLastUpdatedAtTime().Format("2006-01-02")

		// 如果创建日期在范围内，计入当日
		if s.isDateInRange(createdDate, startDate, endDate) {
			if dailySessionCountMap[createdDate] == nil {
				dailySessionCountMap[createdDate] = make(map[string]bool)
			}
			dailySessionCountMap[createdDate][composer.ComposerID] = true
		}

		// 如果更新日期在范围内且与创建日期不同，计入当日
		if s.isDateInRange(updatedDate, startDate, endDate) && updatedDate != createdDate {
			if dailySessionCountMap[updatedDate] == nil {
				dailySessionCountMap[updatedDate] = make(map[string]bool)
			}
			dailySessionCountMap[updatedDate][composer.ComposerID] = true
		}
	}

	// 构建每日详情
	dailyDetailsMap := make(map[string]*domainCursor.DailyAnalysis)

	// 遍历日期范围，构建每日详情
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	current := start
	for !current.After(end) {
		dateStr := current.Format("2006-01-02")

		// 获取当日代码变更数据
		dailyChanges := dailyChangesMap[dateStr]
		if dailyChanges == nil {
			dailyChanges = &domainCursor.DailyCodeChanges{
				Date:         dateStr,
				LinesAdded:   0,
				LinesRemoved: 0,
				FilesChanged: 0,
			}
		}

		// 获取当日活跃会话数
		dailySessionCount := 0
		if dailySessionCountMap[dateStr] != nil {
			dailySessionCount = len(dailySessionCountMap[dateStr])
		}

		dailyDetailsMap[dateStr] = &domainCursor.DailyAnalysis{
			Date:           dateStr,
			LinesAdded:     dailyChanges.LinesAdded,
			LinesRemoved:   dailyChanges.LinesRemoved,
			FilesChanged:   dailyChanges.FilesChanged,
			ActiveSessions: dailySessionCount,
		}

		current = current.AddDate(0, 0, 1)
	}

	// 将每日详情转换为切片并排序
	for _, detail := range dailyDetailsMap {
		analysis.DailyDetails = append(analysis.DailyDetails, detail)
	}
	sort.Slice(analysis.DailyDetails, func(i, j int) bool {
		return analysis.DailyDetails[i].Date < analysis.DailyDetails[j].Date
	})

	// 构建代码变更趋势（按日期排序）
	for date, changes := range dailyChangesMap {
		analysis.CodeChangesTrend = append(analysis.CodeChangesTrend, &domainCursor.DailyCodeChanges{
			Date:         date,
			LinesAdded:   changes.LinesAdded,
			LinesRemoved: changes.LinesRemoved,
			FilesChanged: changes.FilesChanged,
		})
	}
	sort.Slice(analysis.CodeChangesTrend, func(i, j int) bool {
		return analysis.CodeChangesTrend[i].Date < analysis.CodeChangesTrend[j].Date
	})

	// 构建 Top 文件（取前 10）
	fileRefs := s.buildFileReferences(fileMap, 10)
	analysis.TopFiles = fileRefs

	// 构建时间分布（用于热力图）
	for hour := 0; hour < 24; hour++ {
		for day := 0; day < 7; day++ {
			count := 0
			if timeDistMap[hour] != nil {
				count = timeDistMap[hour][day]
			}
			analysis.TimeDistribution = append(analysis.TimeDistribution, &domainCursor.TimeDistributionItem{
				Hour:  hour,
				Day:   day,
				Count: count,
			})
		}
	}

	// 构建效率指标
	if totalEntropyCount > 0 {
		analysis.EfficiencyMetrics.AvgSessionEntropy = totalEntropySum / float64(totalEntropyCount)
		analysis.EfficiencyMetrics.AvgContextUsage = totalContextUsageSum / float64(totalContextUsageCount)
	}

	// 构建熵值趋势（按日期平均）
	for date, entropies := range entropyTrendMap {
		sum := 0.0
		for _, e := range entropies {
			sum += e
		}
		avg := sum / float64(len(entropies))
		analysis.EfficiencyMetrics.EntropyTrend = append(analysis.EfficiencyMetrics.EntropyTrend, &domainCursor.EntropyTrendItem{
			Date:  date,
			Value: avg,
		})
	}
	sort.Slice(analysis.EfficiencyMetrics.EntropyTrend, func(i, j int) bool {
		return analysis.EfficiencyMetrics.EntropyTrend[i].Date < analysis.EfficiencyMetrics.EntropyTrend[j].Date
	})

	return analysis, nil
}

// isDateInRange 检查日期是否在范围内
func (s *WorkAnalysisService) isDateInRange(dateStr, startStr, endStr string) bool {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return false
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return false
	}
	return (date.Equal(start) || date.After(start)) && (date.Equal(end) || date.Before(end))
}

// parseFileList 解析文件列表（逗号分隔）
func (s *WorkAnalysisService) parseFileList(subtitle string) []string {
	var files []string
	parts := strings.Split(subtitle, ",")
	for _, part := range parts {
		file := strings.TrimSpace(part)
		if file != "" {
			files = append(files, file)
		}
	}
	return files
}

// sumDailyChanges 汇总每日变更
func (s *WorkAnalysisService) sumDailyChanges(dailyMap map[string]*domainCursor.DailyCodeChanges, field string) int {
	sum := 0
	for _, changes := range dailyMap {
		switch field {
		case "added":
			sum += changes.LinesAdded
		case "removed":
			sum += changes.LinesRemoved
		case "files":
			sum += changes.FilesChanged
		}
	}
	return sum
}

// buildFileReferences 构建文件引用列表
func (s *WorkAnalysisService) buildFileReferences(fileMap map[string]int, topN int) []*domainCursor.FileReference {
	var refs []*domainCursor.FileReference
	for fileName, count := range fileMap {
		refs = append(refs, &domainCursor.FileReference{
			FileName:       fileName,
			ReferenceCount: count,
			FileType:       s.getFileType(fileName),
		})
	}

	// 按引用次数排序
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].ReferenceCount > refs[j].ReferenceCount
	})

	// 返回 TopN
	if len(refs) > topN {
		refs = refs[:topN]
	}

	return refs
}

// getFileType 从文件名推断文件类型
func (s *WorkAnalysisService) getFileType(fileName string) string {
	parts := strings.Split(fileName, ".")
	if len(parts) < 2 {
		return "unknown"
	}
	ext := strings.ToLower(parts[len(parts)-1])
	return ext
}

// sessionToComposerData 将 WorkspaceSession 转换为 ComposerData
func (s *WorkAnalysisService) sessionToComposerData(session *storage.WorkspaceSession) domainCursor.ComposerData {
	return domainCursor.ComposerData{
		Type:                session.Type,
		ComposerID:          session.ComposerID,
		Name:                session.Name,
		CreatedAt:           session.CreatedAt,
		LastUpdatedAt:       session.LastUpdatedAt,
		UnifiedMode:         session.UnifiedMode,
		ContextUsagePercent: session.ContextUsagePercent,
		TotalLinesAdded:     session.TotalLinesAdded,
		TotalLinesRemoved:   session.TotalLinesRemoved,
		FilesChangedCount:   session.FilesChangedCount,
		Subtitle:            session.Subtitle,
		IsArchived:          session.IsArchived,
		CreatedOnBranch:     session.CreatedOnBranch,
	}
}
