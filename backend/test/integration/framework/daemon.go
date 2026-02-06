//go:build integration
// +build integration

// TestDaemon 管理独立 cocursor-daemon 进程的启动与关闭
package framework

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// TestDaemon 测试守护进程
type TestDaemon struct {
	Name     string // 角色名称（如 "leader", "member"）
	HTTPPort int    // HTTP 端口
	MCPPort  int    // MCP 端口
	DataDir  string // 数据目录（隔离）

	cmd     *exec.Cmd
	baseURL string
}

// DaemonOption 守护进程配置选项
type DaemonOption func(*TestDaemon)

// NewTestDaemon 创建测试守护进程
func NewTestDaemon(binaryPath, name string, opts ...DaemonOption) (*TestDaemon, error) {
	// 分配空闲端口
	httpPort, err := getFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate HTTP port: %w", err)
	}
	mcpPort, err := getFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate MCP port: %w", err)
	}

	// 创建隔离的数据目录
	dataDir, err := os.MkdirTemp("", fmt.Sprintf("cocursor-test-%s-", name))
	if err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	d := &TestDaemon{
		Name:     name,
		HTTPPort: httpPort,
		MCPPort:  mcpPort,
		DataDir:  dataDir,
		baseURL:  fmt.Sprintf("http://localhost:%d", httpPort),
	}

	for _, opt := range opts {
		opt(d)
	}

	// 确保子目录存在
	for _, sub := range []string{"team"} {
		if err := os.MkdirAll(filepath.Join(dataDir, sub), 0755); err != nil {
			return nil, fmt.Errorf("failed to create subdirectory %s: %w", sub, err)
		}
	}

	// 构建进程命令
	d.cmd = exec.Command(binaryPath)
	d.cmd.Env = append(os.Environ(),
		fmt.Sprintf("COCURSOR_DATA_DIR=%s", dataDir),
		fmt.Sprintf("COCURSOR_HTTP_PORT=:%d", httpPort),
		fmt.Sprintf("COCURSOR_MCP_PORT=:%d", mcpPort),
		"GIN_MODE=test",
	)
	d.cmd.Stdout = os.Stdout
	d.cmd.Stderr = os.Stderr

	return d, nil
}

// Start 启动守护进程并等待就绪
func (d *TestDaemon) Start() error {
	if err := d.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon %s: %w", d.Name, err)
	}

	// 等待 health 端点就绪
	return d.waitForReady(30 * time.Second)
}

// Stop 停止守护进程并清理数据目录
func (d *TestDaemon) Stop() error {
	return d.StopWithCleanup(true)
}

// StopWithCleanup 停止守护进程，可选择是否清理数据目录
func (d *TestDaemon) StopWithCleanup(cleanup bool) error {
	if d.cmd.Process != nil {
		// 发送关闭信号
		_ = d.cmd.Process.Signal(os.Interrupt)

		// 等待进程退出（最多 5 秒）
		done := make(chan error, 1)
		go func() {
			done <- d.cmd.Wait()
		}()

		select {
		case <-done:
			// 正常退出
		case <-time.After(5 * time.Second):
			// 强制杀进程
			_ = d.cmd.Process.Kill()
			<-done
		}
	}

	// 可选清理数据目录
	if cleanup {
		return os.RemoveAll(d.DataDir)
	}
	return nil
}

// BaseURL 返回 HTTP 基础 URL
func (d *TestDaemon) BaseURL() string {
	return d.baseURL
}

// NewTestDaemonWithConfig 使用指定配置创建守护进程（用于重启场景）
func NewTestDaemonWithConfig(binaryPath, name, dataDir string, httpPort, mcpPort int) (*TestDaemon, error) {
	d := &TestDaemon{
		Name:     name,
		HTTPPort: httpPort,
		MCPPort:  mcpPort,
		DataDir:  dataDir,
		baseURL:  fmt.Sprintf("http://localhost:%d", httpPort),
	}

	d.cmd = exec.Command(binaryPath)
	d.cmd.Env = append(os.Environ(),
		fmt.Sprintf("COCURSOR_DATA_DIR=%s", dataDir),
		fmt.Sprintf("COCURSOR_HTTP_PORT=:%d", httpPort),
		fmt.Sprintf("COCURSOR_MCP_PORT=:%d", mcpPort),
		"GIN_MODE=test",
	)
	d.cmd.Stdout = os.Stdout
	d.cmd.Stderr = os.Stderr

	return d, nil
}

// waitForReady 等待守护进程 health 端点就绪
func (d *TestDaemon) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(d.baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("daemon %s failed to become ready within %v", d.Name, timeout)
}

// getFreePort 获取一个空闲的 TCP 端口
func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
