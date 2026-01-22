package cursor

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	"github.com/cocursor/backend/internal/infrastructure/log"
	"github.com/cocursor/backend/internal/infrastructure/storage"
)

// readDir 是 os.ReadDir 的封装
var readDir = os.ReadDir

// WorkAnalysisService 工作分析服务
type WorkAnalysisService struct {
	statsService          *StatsService
	projectManager        *ProjectManager
	sessionRepo           storage.WorkspaceSessionRepository
	summaryRepo           storage.DailySummaryRepository
	tokenService          *TokenService
	workspaceCacheService *WorkspaceCacheService // 用于获取活跃会话状态
	logger                *slog.Logger
}

// NewWorkAnalysisService 创建工作分析服务实例（接受 Repository 作为参数）
func NewWorkAnalysisService(
	statsService *StatsService,
	projectManager *ProjectManager,
	sessionRepo storage.WorkspaceSessionRepository,
	summaryRepo storage.DailySummaryRepository,
	tokenService *TokenService,
) *WorkAnalysisService {
	return &WorkAnalysisService{
		statsService:   statsService,
		projectManager: projectManager,
		sessionRepo:    sessionRepo,
		summaryRepo:    summaryRepo,
		tokenService:   tokenService,
		logger:         log.NewModuleLogger("cursor", "work_analysis"),
	}
}

// SetWorkspaceCacheService 设置工作区缓存服务（用于解决循环依赖）
func (s *WorkAnalysisService) SetWorkspaceCacheService(cacheService *WorkspaceCacheService) {
	s.workspaceCacheService = cacheService
}

// GetWorkAnalysis 获取工作分析数据（全局视图）
// startDate: 开始日期 YYYY-MM-DD
// endDate: 结束日期 YYYY-MM-DD
// 返回: WorkAnalysis 和错误
func (s *WorkAnalysisService) GetWorkAnalysis(startDate, endDate string) (*domainCursor.WorkAnalysis, error) {
	startTime := time.Now()
	defer func() {
		s.logger.Debug("GetWorkAnalysis completed", "duration", time.Since(startTime))
	}()

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

	// 构建概览
	if totalEntropyCount > 0 {
		analysis.Overview.TotalLinesAdded = s.sumDailyChanges(dailyChangesMap, "added")
		analysis.Overview.TotalLinesRemoved = s.sumDailyChanges(dailyChangesMap, "removed")
		analysis.Overview.FilesChanged = len(fileMap)
		analysis.Overview.ActiveSessions = totalEntropyCount
	}
	// prompts 和 generations 统计已删除（需求不大）
	analysis.Overview.TotalPrompts = 0
	analysis.Overview.TotalGenerations = 0

	// 从缓存表获取 Token 汇总（使用 SQL 聚合，性能更好）
	tokenStartTime := time.Now()
	dailyTokenMap := make(map[string]int)
	totalTokens := 0

	// 从 workspace_sessions 表聚合 Token
	dailyTokenUsage, err := s.sessionRepo.GetDailyTokenUsage(workspaceIDs, startDate, endDate)
	if err != nil {
		s.logger.Error("failed to get daily token usage", "error", err)
	} else {
		for _, usage := range dailyTokenUsage {
			dailyTokenMap[usage.Date] = usage.TokenCount
			totalTokens += usage.TokenCount
		}
	}
	analysis.Overview.TotalTokens = totalTokens

	// 计算趋势：与上一周期对比
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	days := int(end.Sub(start).Hours() / 24)
	prevEnd := start.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -days)
	prevStartStr := prevStart.Format("2006-01-02")
	prevEndStr := prevEnd.Format("2006-01-02")

	prevTotalTokens := 0
	prevDailyTokenUsage, err := s.sessionRepo.GetDailyTokenUsage(workspaceIDs, prevStartStr, prevEndStr)
	if err == nil {
		for _, usage := range prevDailyTokenUsage {
			prevTotalTokens += usage.TokenCount
		}
	}
	analysis.Overview.TokenTrend = s.calculateTrend(totalTokens, prevTotalTokens)
	s.logger.Debug("token calculation completed", "duration", time.Since(tokenStartTime), "source", "cache")

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

	// 查询日报状态
	dailyReportStatus := make(map[string]bool)
	if s.summaryRepo != nil {
		status, err := s.summaryRepo.FindDatesByRange(startDate, endDate)
		if err == nil {
			dailyReportStatus = status
		}
	}

	// 构建每日详情
	dailyDetailsMap := make(map[string]*domainCursor.DailyAnalysis)

	// 遍历日期范围，构建每日详情
	// 注意：start 和 end 变量在上面 Token 计算中已定义
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

		// 获取日报状态
		hasDailyReport := dailyReportStatus[dateStr]

		// 获取当日 Token 使用量
		tokenUsage := dailyTokenMap[dateStr]

		// 统计当日完成的 OpenSpec 变更数量
		completedChanges := s.countCompletedChangesOnDate(dateStr)

		dailyDetailsMap[dateStr] = &domainCursor.DailyAnalysis{
			Date:             dateStr,
			LinesAdded:       dailyChanges.LinesAdded,
			LinesRemoved:     dailyChanges.LinesRemoved,
			FilesChanged:     dailyChanges.FilesChanged,
			ActiveSessions:   dailySessionCount,
			TokenUsage:       tokenUsage,
			HasDailyReport:   hasDailyReport,
			CompletedChanges: completedChanges,
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

// calculateTrend 计算趋势（当前值与上一周期对比）
func (s *WorkAnalysisService) calculateTrend(current, previous int) string {
	if previous == 0 {
		if current > 0 {
			return "+100%"
		}
		return "0%"
	}

	change := float64(current-previous) / float64(previous) * 100
	if change > 0 {
		return fmt.Sprintf("+%.1f%%", change)
	} else if change < 0 {
		return fmt.Sprintf("%.1f%%", change)
	}
	return "0%"
}

// countCompletedChangesOnDate 统计指定日期完成的 OpenSpec 变更数量
// 通过扫描所有项目的 openspec/changes/archive/ 目录下文件的修改时间来判断
func (s *WorkAnalysisService) countCompletedChangesOnDate(date string) int {
	count := 0
	
	// 解析日期
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0
	}
	
	// 遍历所有已注册的项目
	projects := s.projectManager.ListAllProjects()
	for _, project := range projects {
		for _, ws := range project.Workspaces {
			projectPath := ws.Path
			
			// 检查 openspec/changes/archive 目录是否存在
			archiveDir := projectPath + "/openspec/changes/archive"
			entries, err := readDir(archiveDir)
			if err != nil {
				continue // 目录不存在或无法读取，跳过
			}
			
			// 遍历 archive 下的变更目录
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				
				// 获取目录的修改时间（代表归档时间）
				info, err := entry.Info()
				if err != nil {
					continue
				}
				
				modTime := info.ModTime()
				modDate := modTime.Format("2006-01-02")
				
				// 如果归档日期与目标日期匹配，计数 +1
				if modDate == targetDate.Format("2006-01-02") {
					count++
				}
			}
		}
	}
	
	return count
}

// GetActiveSessionsOverview 获取当前工作区的活跃会话概览
// 如果 workspaceID 为空，则聚合所有工作区的活跃会话
func (s *WorkAnalysisService) GetActiveSessionsOverview(workspaceID string) (*domainCursor.ActiveSessionsOverview, error) {
	if s.workspaceCacheService == nil {
		return nil, fmt.Errorf("workspace cache service not initialized")
	}

	// 如果指定了工作区，直接返回该工作区的概览
	if workspaceID != "" {
		return s.workspaceCacheService.GetActiveSessionsOverview(workspaceID)
	}

	// 否则，聚合所有工作区的活跃会话
	projects := s.projectManager.ListAllProjects()

	aggregatedOverview := &domainCursor.ActiveSessionsOverview{
		OpenSessions:  make([]*domainCursor.ActiveSession, 0),
		ClosedCount:   0,
		ArchivedCount: 0,
	}

	for _, project := range projects {
		for _, ws := range project.Workspaces {
			overview, err := s.workspaceCacheService.GetActiveSessionsOverview(ws.WorkspaceID)
			if err != nil {
				s.logger.Error("failed to get active sessions for workspace", "workspace_id", ws.WorkspaceID, "error", err)
				continue
			}

			// 聚合统计
			aggregatedOverview.ClosedCount += overview.ClosedCount
			aggregatedOverview.ArchivedCount += overview.ArchivedCount

			// 聚焦会话：只保留一个（通常只有一个工作区是活跃的）
			if overview.Focused != nil {
				// 如果已有聚焦会话，比较更新时间，保留最新的
				if aggregatedOverview.Focused == nil || overview.Focused.LastUpdatedAt > aggregatedOverview.Focused.LastUpdatedAt {
					aggregatedOverview.Focused = overview.Focused
				}
			}

			// 合并打开会话列表
			aggregatedOverview.OpenSessions = append(aggregatedOverview.OpenSessions, overview.OpenSessions...)
		}
	}

	// 按熵值降序排序打开会话
	sort.Slice(aggregatedOverview.OpenSessions, func(i, j int) bool {
		return aggregatedOverview.OpenSessions[i].Entropy > aggregatedOverview.OpenSessions[j].Entropy
	})

	return aggregatedOverview, nil
}
