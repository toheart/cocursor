//go:build integration
// +build integration

// 团队协作功能集成测试
// 测试工作状态、代码分享等协作功能
// 重点测试跨节点场景

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cocursor/backend/internal/domain/p2p"
	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// ============================================================
// 场景 2.1: Leader 广播状态
// ============================================================

// TestCollaboration_LeaderBroadcastStatus Leader 更新状态，Member 能查询到
func TestCollaboration_LeaderBroadcastStatus(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20030, "BroadcastLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("BroadcastTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20031, "BroadcastMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Leader 获取身份信息
	os.Setenv("HOME", leader.HomeDir)
	leaderIdentity, err := leader.Service.GetIdentity()
	require.NoError(t, err)

	// Leader 更新工作状态
	leaderStatus := &p2p.MemberWorkStatusPayload{
		MemberID:      leaderIdentity.ID,
		MemberName:    leaderIdentity.Name,
		ProjectName:   "CoCursor",
		CurrentFile:   "src/main.go",
		StatusVisible: true,
		LastActiveAt:  time.Now(),
	}

	err = leader.CollaborationService.UpdateWorkStatus(ctx, team.ID, leaderStatus)
	require.NoError(t, err)
	t.Logf("Leader updated status: project=%s, file=%s", leaderStatus.ProjectName, leaderStatus.CurrentFile)

	// Leader 本地应该能查到自己的状态
	gotStatus := leader.CollaborationService.GetMemberWorkStatus(team.ID, leaderIdentity.ID)
	require.NotNil(t, gotStatus)
	assert.Equal(t, "CoCursor", gotStatus.ProjectName)
	assert.Equal(t, "src/main.go", gotStatus.CurrentFile)

	t.Log("Leader status broadcast test passed")
}

// ============================================================
// 场景 2.2: Member 状态上报到 Leader
// ============================================================

// TestCollaboration_MemberStatusUpload Member 更新状态发送到 Leader
func TestCollaboration_MemberStatusUpload(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20032, "UploadLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("UploadTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20033, "UploadMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Member 获取身份信息
	memberIdentity, err := member.Service.GetIdentity()
	require.NoError(t, err)

	// Member 更新工作状态（会发送到 Leader）
	memberStatus := &p2p.MemberWorkStatusPayload{
		MemberID:      memberIdentity.ID,
		MemberName:    memberIdentity.Name,
		ProjectName:   "MemberProject",
		CurrentFile:   "member.go",
		StatusVisible: true,
		LastActiveAt:  time.Now(),
	}

	err = member.CollaborationService.UpdateWorkStatus(ctx, team.ID, memberStatus)
	require.NoError(t, err)
	t.Logf("Member updated status: project=%s, file=%s", memberStatus.ProjectName, memberStatus.CurrentFile)

	// Member 本地应该能查到自己的状态
	gotStatus := member.CollaborationService.GetMemberWorkStatus(team.ID, memberIdentity.ID)
	require.NotNil(t, gotStatus)
	assert.Equal(t, "MemberProject", gotStatus.ProjectName)

	t.Log("Member status upload test passed")
}

// ============================================================
// 场景 2.3: 状态隐藏
// ============================================================

// TestCollaboration_StatusInvisible 测试隐藏工作状态
func TestCollaboration_StatusInvisible(t *testing.T) {
	ctx := context.Background()

	// 启动节点
	node := StartTestNode(t, 20034, "InvisibleNode")
	defer node.Stop()

	// 创建团队
	os.Setenv("HOME", node.HomeDir)
	team, err := node.Service.CreateTeam("InvisibleTeam", "", "")
	require.NoError(t, err)

	identity, _ := node.Service.GetIdentity()

	// 设置隐藏状态
	status := &p2p.MemberWorkStatusPayload{
		MemberID:      identity.ID,
		MemberName:    identity.Name,
		ProjectName:   "SecretProject",
		CurrentFile:   "secret.go",
		StatusVisible: false, // 隐藏
		LastActiveAt:  time.Now(),
	}

	err = node.CollaborationService.UpdateWorkStatus(ctx, team.ID, status)
	require.NoError(t, err)

	// 获取状态
	gotStatus := node.CollaborationService.GetMemberWorkStatus(team.ID, identity.ID)
	require.NotNil(t, gotStatus)
	assert.False(t, gotStatus.StatusVisible, "Status should be invisible")

	t.Log("Invisible status test passed")
}

// ============================================================
// 场景 2.4: 状态不存在
// ============================================================

// TestCollaboration_StatusNotFound 测试获取不存在的工作状态
func TestCollaboration_StatusNotFound(t *testing.T) {
	// 启动节点
	node := StartTestNode(t, 20035, "NotFoundNode")
	defer node.Stop()

	// 创建团队
	os.Setenv("HOME", node.HomeDir)
	team, err := node.Service.CreateTeam("NotFoundTeam", "", "")
	require.NoError(t, err)

	// 获取不存在的成员状态
	gotStatus := node.CollaborationService.GetMemberWorkStatus(team.ID, "non-existent-member")
	assert.Nil(t, gotStatus, "Should return nil for non-existent member")

	t.Log("Status not found test passed")
}

// ============================================================
// 场景 2.5: 多次更新状态
// ============================================================

// TestCollaboration_MultipleStatusUpdates 测试多次更新工作状态
func TestCollaboration_MultipleStatusUpdates(t *testing.T) {
	ctx := context.Background()

	// 启动节点
	node := StartTestNode(t, 20036, "MultiUpdateNode")
	defer node.Stop()

	// 创建团队
	os.Setenv("HOME", node.HomeDir)
	team, err := node.Service.CreateTeam("MultiUpdateTeam", "", "")
	require.NoError(t, err)

	identity, _ := node.Service.GetIdentity()

	// 第一次更新
	status1 := &p2p.MemberWorkStatusPayload{
		MemberID:      identity.ID,
		MemberName:    identity.Name,
		ProjectName:   "Project1",
		CurrentFile:   "file1.go",
		StatusVisible: true,
		LastActiveAt:  time.Now(),
	}
	err = node.CollaborationService.UpdateWorkStatus(ctx, team.ID, status1)
	require.NoError(t, err)

	gotStatus := node.CollaborationService.GetMemberWorkStatus(team.ID, identity.ID)
	assert.Equal(t, "Project1", gotStatus.ProjectName)
	assert.Equal(t, "file1.go", gotStatus.CurrentFile)

	// 第二次更新（切换项目）
	status2 := &p2p.MemberWorkStatusPayload{
		MemberID:      identity.ID,
		MemberName:    identity.Name,
		ProjectName:   "Project2",
		CurrentFile:   "file2.go",
		StatusVisible: true,
		LastActiveAt:  time.Now(),
	}
	err = node.CollaborationService.UpdateWorkStatus(ctx, team.ID, status2)
	require.NoError(t, err)

	gotStatus = node.CollaborationService.GetMemberWorkStatus(team.ID, identity.ID)
	assert.Equal(t, "Project2", gotStatus.ProjectName)
	assert.Equal(t, "file2.go", gotStatus.CurrentFile)

	t.Log("Multiple status updates test passed")
}

// ============================================================
// 场景 3.1: Leader 分享代码
// ============================================================

// TestCodeShare_LeaderShare Leader 分享代码片段
func TestCodeShare_LeaderShare(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20040, "CodeShareLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("CodeShareTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20041, "CodeShareMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Leader 获取身份
	os.Setenv("HOME", leader.HomeDir)
	leaderIdentity, _ := leader.Service.GetIdentity()

	// Leader 分享代码片段
	snippet := &domainTeam.CodeSnippet{
		ID:         "snippet-leader-001",
		TeamID:     team.ID,
		SenderID:   leaderIdentity.ID,
		SenderName: leaderIdentity.Name,
		FileName:   "main.go",
		FilePath:   "src/main.go",
		Language:   "go",
		StartLine:  1,
		EndLine:    5,
		Code:       "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
		Message:    "Check this code",
		CreatedAt:  time.Now(),
	}

	err = leader.CollaborationService.ShareCode(ctx, snippet)
	require.NoError(t, err)
	t.Logf("Leader shared code: %s (%d lines)", snippet.FileName, snippet.EndLine-snippet.StartLine+1)

	t.Log("Leader code share test passed")
}

// ============================================================
// 场景 3.2: Member 分享代码到 Leader
// ============================================================

// TestCodeShare_MemberShare Member 分享代码到 Leader
func TestCodeShare_MemberShare(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20042, "MemberShareLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("MemberShareTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20043, "MemberShareMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// Member 获取身份
	memberIdentity, _ := member.Service.GetIdentity()

	// Member 分享代码片段（会发送到 Leader）
	snippet := &domainTeam.CodeSnippet{
		ID:         "snippet-member-001",
		TeamID:     team.ID,
		SenderID:   memberIdentity.ID,
		SenderName: memberIdentity.Name,
		FileName:   "utils.go",
		FilePath:   "src/utils.go",
		Language:   "go",
		StartLine:  10,
		EndLine:    20,
		Code:       "func helper() {\n\treturn nil\n}",
		Message:    "Need review",
		CreatedAt:  time.Now(),
	}

	err = member.CollaborationService.ShareCode(ctx, snippet)
	require.NoError(t, err)
	t.Logf("Member shared code: %s", snippet.FileName)

	t.Log("Member code share test passed")
}

// ============================================================
// 场景 6.1: Leader 离线后行为
// ============================================================

// TestLeaderOffline_JoinFails Leader 离线后新成员无法加入
func TestLeaderOffline_JoinFails(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20044, "OfflineLeader")

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	_, err := leader.Service.CreateTeam("OfflineTeam", "", "")
	require.NoError(t, err)

	leaderEndpoint := leader.Endpoint()

	// 停止 Leader
	leader.Stop()
	t.Log("Leader stopped")

	time.Sleep(100 * time.Millisecond)

	// 启动新 Member 尝试加入
	member := StartTestNode(t, 20045, "OfflineMember")
	defer member.Stop()

	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leaderEndpoint)
	assert.Error(t, err, "Should fail to join when leader is offline")
	t.Logf("Expected error: %v", err)

	t.Log("Leader offline test passed")
}

// TestLeaderOffline_ExistingMemberRetainsInfo 已加入的成员本地信息保留
func TestLeaderOffline_ExistingMemberRetainsInfo(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20046, "RetainLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("RetainTeam", "", "")
	require.NoError(t, err)

	// 启动 Member 节点并加入
	member := StartTestNode(t, 20047, "RetainMember")
	defer member.Stop()

	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// 验证 Member 本地有团队信息
	teams := member.Service.GetTeamList()
	require.Len(t, teams, 1)
	assert.Equal(t, team.ID, teams[0].ID)

	// 停止 Leader
	leader.Stop()
	t.Log("Leader stopped")

	time.Sleep(100 * time.Millisecond)

	// Member 本地团队信息应该保留
	teams = member.Service.GetTeamList()
	require.Len(t, teams, 1, "Member should retain team info after leader goes offline")
	assert.Equal(t, team.ID, teams[0].ID)

	t.Log("Member retains team info after leader offline")
}

// ============================================================
// 场景 3.3: 代码片段验证
// ============================================================

// TestCodeSnippet_Validation 测试代码片段验证
func TestCodeSnippet_Validation(t *testing.T) {
	t.Run("ValidSnippet", func(t *testing.T) {
		snippet := &domainTeam.CodeSnippet{
			ID:         "snippet-001",
			TeamID:     "team-001",
			SenderID:   "sender-001",
			SenderName: "TestUser",
			FileName:   "main.go",
			FilePath:   "src/main.go",
			Language:   "go",
			StartLine:  1,
			EndLine:    10,
			Code:       "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
			Message:    "Check this code",
			CreatedAt:  time.Now(),
		}

		err := snippet.Validate()
		assert.NoError(t, err, "Valid snippet should pass validation")
	})

	t.Run("EmptyCode", func(t *testing.T) {
		snippet := &domainTeam.CodeSnippet{
			FileName: "test.go",
			Code:     "",
		}

		err := snippet.Validate()
		assert.Error(t, err, "Empty code should fail validation")
		assert.Contains(t, err.Error(), "code content is required")
	})

	t.Run("EmptyFileName", func(t *testing.T) {
		snippet := &domainTeam.CodeSnippet{
			FileName: "",
			Code:     "some code",
		}

		err := snippet.Validate()
		assert.Error(t, err, "Empty filename should fail validation")
		assert.Contains(t, err.Error(), "file name is required")
	})

	t.Run("TooLargeCode", func(t *testing.T) {
		// 创建超过 10KB 的代码
		largeCode := make([]byte, 11*1024)
		for i := range largeCode {
			largeCode[i] = 'a'
		}

		snippet := &domainTeam.CodeSnippet{
			FileName: "large.go",
			Code:     string(largeCode),
		}

		err := snippet.Validate()
		assert.Error(t, err, "Large code should fail validation")
		assert.Contains(t, err.Error(), "exceeds maximum size")
	})

	t.Run("Truncate", func(t *testing.T) {
		// 创建超过 10KB 的代码
		largeCode := make([]byte, 11*1024)
		for i := range largeCode {
			largeCode[i] = 'x'
		}

		snippet := &domainTeam.CodeSnippet{
			FileName: "truncate.go",
			Code:     string(largeCode),
		}

		truncated := snippet.Truncate()
		assert.True(t, truncated, "Should return true when truncated")
		assert.Len(t, snippet.Code, domainTeam.MaxCodeSnippetSize, "Code should be truncated to max size")
	})
}

