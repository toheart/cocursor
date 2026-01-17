package main

import (
	"fmt"
	"os"

	"github.com/cocursor/backend/internal/infrastructure/cursor"
)

// 验收测试：输入项目路径 D:/code/cocursor，输出工作区 ID
func main() {
	resolver := cursor.NewPathResolver()

	// 测试路径
	testPath := "D:/code/cocursor"
	if len(os.Args) > 1 {
		testPath = os.Args[1]
	}

	fmt.Printf("正在查找项目路径: %s\n", testPath)

	workspaceID, err := resolver.GetWorkspaceIDByPath(testPath)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("工作区 ID: %s\n", workspaceID)
}
