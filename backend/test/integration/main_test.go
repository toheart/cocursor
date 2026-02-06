//go:build integration
// +build integration

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/cocursor/backend/test/integration/framework"
)

func TestMain(m *testing.M) {
	// 编译 daemon 二进制
	fmt.Println("=== Building cocursor-daemon binary ===")
	if err := framework.BuildDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build daemon: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("=== Binary built at: %s ===\n", framework.BinaryPath)

	// 运行测试
	code := m.Run()

	// 清理
	framework.Cleanup()

	os.Exit(code)
}
