package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	infraCursor "github.com/cocursor/backend/internal/infrastructure/cursor"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法:")
		fmt.Println("  diagnose_workspace <路径>              - 诊断工作区路径")
		fmt.Println("  diagnose_workspace --db <数据库路径>   - 诊断 SQLite 数据库")
		fmt.Println("  diagnose_workspace --workspace-id <ID> - 通过工作区 ID 诊断数据库")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  diagnose_workspace \"c:\\Users\\TANG\\Videos\\goanalysis\"")
		fmt.Println("  diagnose_workspace --db \"C:\\Users\\TANG\\AppData\\Roaming\\Cursor\\User\\workspaceStorage\\861ca156f6e5c2aad73afd2854c92261\\state.vscdb\"")
		fmt.Println("  diagnose_workspace --workspace-id 861ca156f6e5c2aad73afd2854c92261")
		os.Exit(1)
	}

	// 检查是否是数据库诊断模式
	if os.Args[1] == "--db" {
		if len(os.Args) < 3 {
			fmt.Println("错误: 请提供数据库路径")
			os.Exit(1)
		}
		dbPath := os.Args[2]
		diagnoseDatabase(dbPath)
		return
	}

	// 检查是否是通过工作区 ID 诊断
	if os.Args[1] == "--workspace-id" {
		if len(os.Args) < 3 {
			fmt.Println("错误: 请提供工作区 ID")
			os.Exit(1)
		}
		workspaceID := os.Args[2]
		diagnoseWorkspaceByID(workspaceID)
		return
	}

	// 默认模式：诊断工作区路径
	targetPath := os.Args[1]
	fmt.Printf("诊断路径: %s\n", targetPath)
	fmt.Println(strings.Repeat("=", 80))

	pathResolver := infraCursor.NewPathResolver()

	// 1. 规范化目标路径
	normalizedTarget, err := pathResolver.GetWorkspaceIDByPath(targetPath)
	if err != nil {
		fmt.Printf("❌ 无法找到工作区 ID: %v\n\n", err)
	} else {
		fmt.Printf("✅ 找到工作区 ID: %s\n\n", normalizedTarget)
	}

	// 2. 获取工作区存储目录
	workspaceDir, err := pathResolver.GetWorkspaceStorageDir()
	if err != nil {
		log.Fatalf("无法获取工作区存储目录: %v", err)
	}
	fmt.Printf("工作区存储目录: %s\n\n", workspaceDir)

	// 3. 列出所有工作区
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		log.Fatalf("无法读取工作区存储目录: %v", err)
	}

	fmt.Printf("发现 %d 个工作区目录:\n\n", len(entries))

	// 规范化目标路径用于比较
	normalizedTargetPath, _ := normalizePathForCompare(targetPath)

	// 4. 检查每个工作区
	for i, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		workspaceID := entry.Name()
		workspaceJSONPath := filepath.Join(workspaceDir, workspaceID, "workspace.json")

		fmt.Printf("[%d] 工作区 ID: %s\n", i+1, workspaceID)

		// 读取 workspace.json
		data, err := os.ReadFile(workspaceJSONPath)
		if err != nil {
			fmt.Printf("  ⚠️  workspace.json 不存在或无法读取: %v\n", err)
			fmt.Println()
			continue
		}

		// 解析 JSON
		var workspace struct {
			Folder string `json:"folder"`
		}
		if err := json.Unmarshal(data, &workspace); err != nil {
			fmt.Printf("  ⚠️  无法解析 workspace.json: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Printf("  Folder URI: %s\n", workspace.Folder)

		// 解析 folder URI
		folderPath, err := parseFolderURI(workspace.Folder)
		if err != nil {
			fmt.Printf("  ⚠️  无法解析 Folder URI: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Printf("  Folder 路径: %s\n", folderPath)

		// 规范化并比较（使用 PathResolver 的 normalizePath 逻辑）
		normalizedFolderPath, err := normalizePathForCompare(folderPath)
		if err != nil {
			fmt.Printf("  ⚠️  无法规范化路径: %v\n", err)
			fmt.Println()
			continue
		}
		fmt.Printf("  规范化路径: %s\n", normalizedFolderPath)
		fmt.Printf("  目标路径:   %s\n", normalizedTargetPath)

		// 比较路径
		if strings.EqualFold(normalizedTargetPath, normalizedFolderPath) {
			fmt.Printf("  ✅ 路径匹配！\n")
		} else {
			// 计算相似度
			similarity := calculatePathSimilarity(normalizedTargetPath, normalizedFolderPath)
			fmt.Printf("  ❌ 路径不匹配 (相似度: %.2f%%)\n", similarity*100)
		}

		// 检查数据库文件
		dbPath := filepath.Join(workspaceDir, workspaceID, "state.vscdb")
		if _, err := os.Stat(dbPath); err == nil {
			fmt.Printf("  ✅ 数据库文件存在\n")
		} else {
			fmt.Printf("  ⚠️  数据库文件不存在\n")
		}

		fmt.Println()
	}

	// 5. 提供建议
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("建议:")
	fmt.Println("1. 确保路径已经在 Cursor 中打开过")
	fmt.Println("2. 检查路径格式是否正确（大小写、斜杠方向）")
	fmt.Println("3. 尝试在 Cursor 中重新打开该文件夹")
}

// normalizePathForCompare 规范化路径用于比较（与 PathResolver.normalizePath 逻辑相同）
func normalizePathForCompare(path string) (string, error) {
	// 移除开头的反斜杠（Windows 路径问题）
	path = strings.TrimPrefix(path, "\\")

	// 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	// 统一使用正斜杠
	normalized := filepath.ToSlash(absPath)

	// 移除末尾的斜杠
	normalized = strings.TrimSuffix(normalized, "/")

	// 转换为小写（Windows 路径不区分大小写）
	normalized = strings.ToLower(normalized)

	return normalized, nil
}

// parseFolderURI 解析 folder URI（与 PathResolver 逻辑相同）
func parseFolderURI(uri string) (string, error) {
	// 使用 url.Parse 解析 URI
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse URI: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme != "file" {
		return "", fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	// 获取路径部分
	path := parsedURL.Path

	// 手动处理 URL 编码
	decodedPath, err := url.PathUnescape(path)
	if err != nil {
		decodedPath = path
	}

	// 区分 Windows 和 Unix 路径
	if len(decodedPath) > 2 && decodedPath[1] == ':' {
		// Windows 路径: 移除开头的斜杠
		if len(decodedPath) > 0 && decodedPath[0] == '/' {
			decodedPath = decodedPath[1:]
		}
	}

	// 转换为系统路径格式
	systemPath := filepath.FromSlash(decodedPath)

	return systemPath, nil
}

// calculatePathSimilarity 计算路径相似度
func calculatePathSimilarity(path1, path2 string) float64 {
	parts1 := strings.Split(path1, "/")
	parts2 := strings.Split(path2, "/")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	if maxLen == 0 {
		return 0.0
	}

	commonParts := 0
	minLen := len(parts1)
	if len(parts2) < minLen {
		minLen = len(parts2)
	}

	for i := 0; i < minLen; i++ {
		if parts1[i] == parts2[i] {
			commonParts++
		} else {
			break
		}
	}

	return float64(commonParts) / float64(maxLen)
}

// diagnoseDatabase 诊断 SQLite 数据库
func diagnoseDatabase(dbPath string) {
	fmt.Printf("诊断 SQLite 数据库: %s\n", dbPath)
	fmt.Println(strings.Repeat("=", 80))

	// 检查文件是否存在
	if _, err := os.Stat(dbPath); err != nil {
		fmt.Printf("❌ 数据库文件不存在: %v\n", err)
		os.Exit(1)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		fmt.Printf("❌ 无法获取文件信息: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("文件大小: %d 字节 (%.2f KB)\n", fileInfo.Size(), float64(fileInfo.Size())/1024)
	fmt.Println()

	// 尝试打开数据库
	fmt.Println("1. 检查数据库连接...")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("❌ 无法打开数据库: %v\n", err)
		fmt.Println("\n建议:")
		fmt.Println("- 检查文件权限")
		fmt.Println("- 确保文件没有被其他程序锁定")
		fmt.Println("- 尝试关闭 Cursor 后重试")
		os.Exit(1)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		fmt.Println("\n建议:")
		fmt.Println("- 数据库可能已损坏")
		fmt.Println("- 尝试使用 --repair 选项修复（需要实现）")
		os.Exit(1)
	}
	fmt.Println("✅ 数据库连接成功")
	fmt.Println()

	// 检查完整性
	fmt.Println("2. 检查数据库完整性...")
	var integrityResult string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&integrityResult)
	if err != nil {
		fmt.Printf("❌ 完整性检查失败: %v\n", err)
	} else {
		if integrityResult == "ok" {
			fmt.Println("✅ 数据库完整性检查通过")
		} else {
			fmt.Printf("❌ 数据库已损坏: %s\n", integrityResult)
			fmt.Println("\n建议:")
			fmt.Println("- 数据库文件可能已损坏")
			fmt.Println("- 尝试从备份恢复")
			fmt.Println("- 如果数据不重要，可以删除该工作区的数据库文件，Cursor 会重新创建")
		}
	}
	fmt.Println()

	// 检查快速完整性
	fmt.Println("3. 快速完整性检查...")
	var quickCheck string
	err = db.QueryRow("PRAGMA quick_check").Scan(&quickCheck)
	if err != nil {
		fmt.Printf("⚠️  快速检查失败: %v\n", err)
	} else {
		if quickCheck == "ok" {
			fmt.Println("✅ 快速完整性检查通过")
		} else {
			fmt.Printf("❌ 快速检查发现问题: %s\n", quickCheck)
		}
	}
	fmt.Println()

	// 获取数据库信息
	fmt.Println("4. 数据库信息...")
	var pageCount, pageSize, freelistCount int64
	db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	db.QueryRow("PRAGMA freelist_count").Scan(&freelistCount)
	fmt.Printf("  页数: %d\n", pageCount)
	fmt.Printf("  页大小: %d 字节\n", pageSize)
	fmt.Printf("  数据库大小: %d 字节 (%.2f KB)\n", pageCount*pageSize, float64(pageCount*pageSize)/1024)
	fmt.Printf("  空闲页数: %d\n", freelistCount)
	fmt.Println()

	// 检查表结构
	fmt.Println("5. 检查表结构...")
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		fmt.Printf("⚠️  无法查询表: %v\n", err)
	} else {
		defer rows.Close()
		tableCount := 0
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				continue
			}
			tableCount++
			fmt.Printf("  - %s\n", tableName)

			// 获取表的记录数
			var count int
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
			if err := db.QueryRow(countQuery).Scan(&count); err == nil {
				fmt.Printf("    记录数: %d\n", count)
			}
		}
		if tableCount == 0 {
			fmt.Println("  ⚠️  没有找到表")
		}
	}
	fmt.Println()

	// 尝试读取一些数据
	fmt.Println("6. 测试数据读取...")
	var keyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ItemTable").Scan(&keyCount)
	if err != nil {
		fmt.Printf("⚠️  无法读取 ItemTable: %v\n", err)
		fmt.Println("   这可能是正常的，如果数据库中没有这个表")
	} else {
		fmt.Printf("✅ ItemTable 中有 %d 条记录\n", keyCount)

		// 尝试读取一些键
		rows, err := db.Query("SELECT key FROM ItemTable LIMIT 10")
		if err == nil {
			defer rows.Close()
			fmt.Println("   示例键:")
			for rows.Next() {
				var key string
				if err := rows.Scan(&key); err == nil {
					fmt.Printf("     - %s\n", key)
				}
			}
		}
	}
	fmt.Println()

	// 测试使用 DBReader 读取（模拟实际使用场景）
	fmt.Println("7. 测试使用 DBReader 读取（模拟实际场景）...")
	dbReader := infraCursor.NewDBReader()

	// 测试读取 composer.composerData
	testKey := "composer.composerData"
	fmt.Printf("   尝试读取键: %s\n", testKey)
	value, err := dbReader.ReadValueFromWorkspaceDB(dbPath, testKey)
	if err != nil {
		fmt.Printf("   ❌ 读取失败: %v\n", err)
		fmt.Println("\n   这可能是导致错误的根本原因！")
		fmt.Println("   可能的原因:")
		fmt.Println("   - 文件复制过程中出现问题")
		fmt.Println("   - 数据库文件在复制时被修改")
		fmt.Println("   - 临时文件系统问题")
	} else {
		fmt.Printf("   ✅ 读取成功，数据大小: %d 字节\n", len(value))
	}
	fmt.Println()

	// 总结
	fmt.Println(strings.Repeat("=", 80))
	if integrityResult == "ok" && quickCheck == "ok" {
		fmt.Println("✅ 数据库状态良好")
	} else {
		fmt.Println("❌ 数据库可能存在问题")
		fmt.Println("\n修复建议:")
		fmt.Println("1. 关闭 Cursor 编辑器")
		fmt.Println("2. 备份当前数据库文件")
		fmt.Println("3. 删除损坏的数据库文件（Cursor 会在下次打开时重新创建）")
		fmt.Println("4. 或者尝试使用 SQLite 工具修复:")
		fmt.Println("   sqlite3 database.db \".dump\" | sqlite3 database_new.db")
	}
}

// diagnoseWorkspaceByID 通过工作区 ID 诊断数据库
func diagnoseWorkspaceByID(workspaceID string) {
	fmt.Printf("诊断工作区 ID: %s\n", workspaceID)
	fmt.Println(strings.Repeat("=", 80))

	pathResolver := infraCursor.NewPathResolver()

	// 获取工作区数据库路径
	dbPath, err := pathResolver.GetWorkspaceDBPath(workspaceID)
	if err != nil {
		fmt.Printf("❌ 无法获取工作区数据库路径: %v\n", err)
		fmt.Println("\n建议:")
		fmt.Println("- 检查工作区 ID 是否正确")
		fmt.Println("- 确保该工作区在 Cursor 中打开过")
		os.Exit(1)
	}

	fmt.Printf("数据库路径: %s\n\n", dbPath)

	// 调用数据库诊断
	diagnoseDatabase(dbPath)
}
