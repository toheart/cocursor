package cursor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	domainCursor "github.com/cocursor/backend/internal/domain/cursor"
	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
)

// WorkAnalysisService 工作分析服务
type WorkAnalysisService struct {
	statsService   *StatsService
	projectManager *ProjectManager
	pathResolver   *infraCursor.PathResolver
	dbReader       *infraCursor.DBReader
}

// NewWorkAnalysisService 创建工作分析服务实例
func NewWorkAnalysisService(statsService *StatsService, projectManager *ProjectManager) *WorkAnalysisService {
	return &WorkAnalysisService{
		statsService:   statsService,
		projectManager: projectManager,
		pathResolver:   infraCursor.NewPathResolver(),
		dbReader:       infraCursor.NewDBReader(),
	}
}

// GetWorkAnalysis 获取工作分析数据
// startDate: 开始日期 YYYY-MM-DD
// endDate: 结束日期 YYYY-MM-DD
// projectName: 项目名称（可选），如果不提供则跨项目聚合
// 返回: WorkAnalysis 和错误
func (s *WorkAnalysisService) GetWorkAnalysis(startDate, endDate, projectName string) (*domainCursor.WorkAnalysis, error) {
	// 验证日期格式
	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date: %w", err)
	}
	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date: %w", err)
	}

	// 获取工作区列表
	var workspaceIDs []string
	if projectName != "" {
		// 指定项目
		project := s.projectManager.GetProject(projectName)
		if project == nil {
			return nil, fmt.Errorf("项目不存在: %s", projectName)
		}
		for _, ws := range project.Workspaces {
			workspaceIDs = append(workspaceIDs, ws.WorkspaceID)
		}
	} else {
		// 跨项目聚合：获取所有工作区
		projects := s.projectManager.ListAllProjects()
		for _, project := range projects {
			for _, ws := range project.Workspaces {
				workspaceIDs = append(workspaceIDs, ws.WorkspaceID)
			}
		}
	}

	// 聚合数据
	analysis := &domainCursor.WorkAnalysis{
		Overview:         &domainCursor.WorkAnalysisOverview{},
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
	var totalPrompts int
	var totalGenerations int

	// 遍历所有工作区
	for _, workspaceID := range workspaceIDs {
		workspaceDBPath, err := s.pathResolver.GetWorkspaceDBPath(workspaceID)
		if err != nil {
			continue
		}

		// 读取 composer 数据
		composerDataValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "composer.composerData")
		if err != nil {
			continue
		}

		composers, err := domainCursor.ParseComposerData(string(composerDataValue))
		if err != nil {
			continue
		}

		// 处理每个会话
		for _, composer := range composers {
			// 检查是否在日期范围内
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

		// 统计该工作区的 prompts 和 generations（用于概览）
		prompts, generations := s.countWorkspacePromptsAndGenerations(workspaceDBPath, startDate, endDate)
		totalPrompts += prompts
		totalGenerations += generations
	}

	// 构建概览
	if totalEntropyCount > 0 {
		analysis.Overview.TotalLinesAdded = s.sumDailyChanges(dailyChangesMap, "added")
		analysis.Overview.TotalLinesRemoved = s.sumDailyChanges(dailyChangesMap, "removed")
		analysis.Overview.FilesChanged = len(fileMap)
		analysis.Overview.AcceptanceRate = 0 // TODO: 从接受率统计中获取
		analysis.Overview.ActiveSessions = totalEntropyCount
	}
	// 填充 prompts 和 generations 统计
	analysis.Overview.TotalPrompts = totalPrompts
	analysis.Overview.TotalGenerations = totalGenerations

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

// countWorkspacePromptsAndGenerations 统计工作区的 prompts 和 generations 总数
// 用于概览统计
func (s *WorkAnalysisService) countWorkspacePromptsAndGenerations(
	workspaceDBPath string,
	startDate, endDate string,
) (int, int) {
	// 读取 prompts 和 generations
	promptsValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.prompts")
	if err != nil {
		return 0, 0
	}

	generationsValue, err := s.dbReader.ReadValueFromWorkspaceDB(workspaceDBPath, "aiService.generations")
	if err != nil {
		return 0, 0
	}

	// 解析数据
	var prompts []map[string]interface{}
	if err := json.Unmarshal(promptsValue, &prompts); err != nil {
		return 0, 0
	}

	generations, err := domainCursor.ParseGenerationsData(string(generationsValue))
	if err != nil {
		return 0, 0
	}

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, 0
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, 0
	}
	startMs := start.Unix() * 1000
	endMs := end.AddDate(0, 0, 1).Unix()*1000 - 1

	// 统计 Composer 模式的 prompts（commandType 4）
	promptCount := 0
	for _, prompt := range prompts {
		text, ok := prompt["text"].(string)
		if !ok || text == "" {
			continue
		}
		commandType, ok := prompt["commandType"].(float64)
		if !ok || int(commandType) != 4 {
			continue
		}
		// 由于 prompts 没有时间戳，我们统计所有 Composer 模式的 prompts
		// 这是一个近似值
		promptCount++
	}

	// 统计 Composer 模式的 generations（type "composer"）且在日期范围内
	generationCount := 0
	for _, gen := range generations {
		if gen.Type != "composer" {
			continue
		}
		if gen.UnixMs >= startMs && gen.UnixMs <= endMs {
			generationCount++
		}
	}

	return promptCount, generationCount
}
