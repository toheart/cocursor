package lifecycle

import (
	"testing"
	"time"

	"github.com/cocursor/backend/internal/domain/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowManager_Heartbeat(t *testing.T) {
	t.Run("注册新窗口", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		resp := wm.Heartbeat("window-1", "/path/to/project")

		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, 1, resp.ActiveWindows)

		windows := wm.GetActiveWindows()
		require.Len(t, windows, 1)
		assert.Equal(t, "window-1", windows[0].WindowID)
		assert.Equal(t, "/path/to/project", windows[0].ProjectPath)
	})

	t.Run("更新已有窗口心跳", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		// 第一次心跳
		wm.Heartbeat("window-1", "/path/to/project1")
		firstWindow := wm.GetActiveWindows()[0]
		firstSeen := firstWindow.LastSeen

		// 等待一小段时间
		time.Sleep(10 * time.Millisecond)

		// 第二次心跳（更新项目路径）
		resp := wm.Heartbeat("window-1", "/path/to/project2")

		assert.Equal(t, 1, resp.ActiveWindows)

		windows := wm.GetActiveWindows()
		require.Len(t, windows, 1)
		assert.Equal(t, "/path/to/project2", windows[0].ProjectPath)
		assert.True(t, windows[0].LastSeen.After(firstSeen))
	})

	t.Run("注册多个窗口", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		wm.Heartbeat("window-1", "/project1")
		wm.Heartbeat("window-2", "/project2")
		resp := wm.Heartbeat("window-3", "/project3")

		assert.Equal(t, 3, resp.ActiveWindows)
		assert.Equal(t, 3, wm.ActiveWindowCount())
	})
}

func TestWindowManager_GetActiveWindows(t *testing.T) {
	t.Run("返回所有活跃窗口", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		wm.Heartbeat("window-a", "/projectA")
		wm.Heartbeat("window-b", "/projectB")

		windows := wm.GetActiveWindows()
		assert.Len(t, windows, 2)

		// 验证返回的是副本
		windowIDs := make(map[string]bool)
		for _, w := range windows {
			windowIDs[w.WindowID] = true
		}
		assert.True(t, windowIDs["window-a"])
		assert.True(t, windowIDs["window-b"])
	})

	t.Run("空列表", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		windows := wm.GetActiveWindows()
		assert.Len(t, windows, 0)
	})
}

func TestWindowManager_GetStatus(t *testing.T) {
	t.Run("有活跃窗口时无空闲时间", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		wm.Heartbeat("window-1", "/project")

		status := wm.GetStatus()
		assert.Len(t, status.ActiveWindows, 1)
		assert.Nil(t, status.IdleSince)
		assert.Nil(t, status.WillShutdownAt)
	})
}

func TestWindowManager_Cleanup(t *testing.T) {
	t.Run("清理超时窗口", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		// 注册窗口
		wm.Heartbeat("window-1", "/project")

		// 手动修改最后心跳时间为超时时间之前
		wm.mu.Lock()
		wm.windows["window-1"].LastSeen = time.Now().Add(-lifecycle.HeartbeatTimeout - time.Second)
		wm.mu.Unlock()

		// 执行清理
		wm.cleanup()

		// 验证窗口已被清理
		assert.Equal(t, 0, wm.ActiveWindowCount())
	})

	t.Run("保留未超时窗口", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		// 注册两个窗口
		wm.Heartbeat("window-1", "/project1")
		wm.Heartbeat("window-2", "/project2")

		// 只让 window-1 超时
		wm.mu.Lock()
		wm.windows["window-1"].LastSeen = time.Now().Add(-lifecycle.HeartbeatTimeout - time.Second)
		wm.mu.Unlock()

		// 执行清理
		wm.cleanup()

		// 验证只有 window-2 保留
		assert.Equal(t, 1, wm.ActiveWindowCount())
		windows := wm.GetActiveWindows()
		assert.Equal(t, "window-2", windows[0].WindowID)
	})

	t.Run("清理后开始空闲计时", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		// 注册并超时
		wm.Heartbeat("window-1", "/project")
		wm.mu.Lock()
		wm.windows["window-1"].LastSeen = time.Now().Add(-lifecycle.HeartbeatTimeout - time.Second)
		wm.mu.Unlock()

		// 执行清理
		wm.cleanup()

		// 验证开始空闲计时
		wm.mu.RLock()
		idleSince := wm.idleSince
		wm.mu.RUnlock()

		assert.NotNil(t, idleSince)
	})

	t.Run("新心跳重置空闲状态", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		// 设置空闲状态
		now := time.Now()
		wm.mu.Lock()
		wm.idleSince = &now
		wm.mu.Unlock()

		// 发送心跳
		wm.Heartbeat("window-1", "/project")

		// 验证空闲状态被重置
		wm.mu.RLock()
		idleSince := wm.idleSince
		wm.mu.RUnlock()

		assert.Nil(t, idleSince)
	})
}

func TestWindowManager_IdleShutdown(t *testing.T) {
	t.Run("有活跃窗口时不触发关闭检查", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		wm.Heartbeat("window-1", "/project")

		// 这不应该触发任何操作
		wm.checkIdleShutdown()

		// 窗口仍在
		assert.Equal(t, 1, wm.ActiveWindowCount())
	})

	t.Run("空闲时间未到不触发关闭", func(t *testing.T) {
		wm := NewWindowManager()
		defer wm.Stop()

		// 设置刚开始空闲
		now := time.Now()
		wm.mu.Lock()
		wm.idleSince = &now
		wm.mu.Unlock()

		// 检查不应该触发关闭（因为空闲时间不足）
		wm.checkIdleShutdown()

		// 验证没有触发关闭（通过检查 idleSince 仍存在）
		wm.mu.RLock()
		idleSince := wm.idleSince
		wm.mu.RUnlock()

		assert.NotNil(t, idleSince)
	})
}

func TestWindowManager_ActiveWindowCount(t *testing.T) {
	wm := NewWindowManager()
	defer wm.Stop()

	assert.Equal(t, 0, wm.ActiveWindowCount())

	wm.Heartbeat("w1", "")
	assert.Equal(t, 1, wm.ActiveWindowCount())

	wm.Heartbeat("w2", "")
	assert.Equal(t, 2, wm.ActiveWindowCount())

	wm.Heartbeat("w1", "") // 重复心跳不增加计数
	assert.Equal(t, 2, wm.ActiveWindowCount())
}

func TestWindowManager_StartStop(t *testing.T) {
	t.Run("启动和停止", func(t *testing.T) {
		wm := NewWindowManager()

		// 启动
		wm.Start()

		// 发送心跳验证正常工作
		resp := wm.Heartbeat("window-1", "/project")
		assert.Equal(t, "ok", resp.Status)

		// 停止
		wm.Stop()

		// 停止后仍可以查询（但不会有后台清理）
		assert.Equal(t, 1, wm.ActiveWindowCount())
	})
}
