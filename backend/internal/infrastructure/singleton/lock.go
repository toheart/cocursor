package singleton

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

const (
	// DefaultPort 默认监听端口
	DefaultPort = ":19960"
	// HealthCheckTimeout 健康检查超时时间
	HealthCheckTimeout = 2 * time.Second
)

// CheckAndLock 检查端口是否被占用，如果被占用则检查是否有实例在运行
// 返回 listener 和 error
// 如果已有实例运行，返回 nil listener 和 nil error（调用者应退出）
// 如果端口被占用但实例不健康，返回错误
func CheckAndLock(port string) (net.Listener, error) {
	// 尝试监听端口
	listener, err := net.Listen("tcp", port)
	if err == nil {
		// 端口可用，返回 listener
		return listener, nil
	}

	// 端口被占用，检查是否是地址已在使用错误
	if isAddrInUse(err) {
		// 检查是否有实例在运行
		if isInstanceRunning(port) {
			// 已有实例运行，返回 nil 表示应该退出
			return nil, nil
		}
		// 端口被占用但实例不健康，返回错误
		return nil, fmt.Errorf("端口 %s 被占用，但健康检查失败，可能存在死锁", port)
	}

	// 其他错误直接返回
	return nil, fmt.Errorf("监听端口失败: %w", err)
}

// isAddrInUse 检查错误是否是地址已在使用
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}

	// 首先检查错误字符串（最通用的方法）
	errStr := err.Error()
	if errStr == "bind: address already in use" ||
		errStr == "bind: Only one usage of each socket address (protocol/network address/port) is normally permitted" {
		return true
	}

	// 尝试类型断言检查
	opErr, ok := err.(*net.OpError)
	if !ok {
		return false
	}

	sysErr, ok := opErr.Err.(*os.SyscallError)
	if !ok {
		return false
	}

	// 检查错误码
	errno, ok := sysErr.Err.(syscall.Errno)
	if ok {
		// Windows: WSAEADDRINUSE (10048)
		// Linux/Unix: EADDRINUSE (98)
		return errno == 10048 || errno == syscall.EADDRINUSE
	}

	// 最后检查错误字符串
	errStr = sysErr.Err.Error()
	return errStr == "address already in use" ||
		errStr == "Only one usage of each socket address (protocol/network address/port) is normally permitted"
}

// isInstanceRunning 检查是否有实例在运行
func isInstanceRunning(port string) bool {
	client := &http.Client{
		Timeout: HealthCheckTimeout,
	}

	url := fmt.Sprintf("http://localhost%s/health", port)
	resp, err := client.Get(url)
	if err != nil {
		// 请求失败，说明实例不在运行或不可访问
		return false
	}
	defer resp.Body.Close()

	// 检查状态码
	return resp.StatusCode == http.StatusOK
}
