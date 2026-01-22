//go:build integration
// +build integration

// 团队技能管理集成测试
// 测试技能索引的添加、获取、移除功能

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainTeam "github.com/cocursor/backend/internal/domain/team"
)

// TestSkillIndex_Empty 测试新团队的技能索引为空
func TestSkillIndex_Empty(t *testing.T) {
	// 启动 Leader 节点
	leader := StartTestNode(t, 20001, "SkillLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("SkillTeam", "", "")
	require.NoError(t, err)

	// 获取技能索引
	index, err := leader.Service.GetSkillIndex(team.ID)
	require.NoError(t, err)
	assert.NotNil(t, index)
	assert.Empty(t, index.Skills, "New team should have empty skill index")

	t.Logf("Team %s has %d skills (expected 0)", team.Name, len(index.Skills))
}

// TestSkillIndex_AddAndGet 测试 Leader 添加技能到索引
func TestSkillIndex_AddAndGet(t *testing.T) {
	// 启动 Leader 节点
	leader := StartTestNode(t, 20002, "AddSkillLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("AddSkillTeam", "", "")
	require.NoError(t, err)

	// 添加技能到索引
	skillEntry := &domainTeam.TeamSkillEntry{
		PluginID:       "test-skill-001",
		Name:           "Test Skill",
		Description:    "A test skill for integration testing",
		Version:        "1.0.0",
		AuthorID:       "author-001",
		AuthorName:     "Test Author",
		AuthorEndpoint: leader.Endpoint(),
		FileCount:      3,
		TotalSize:      1024,
		Checksum:       "abc123def456",
		PublishedAt:    time.Now(),
	}

	err = leader.Service.AddSkillToIndex(team.ID, skillEntry)
	require.NoError(t, err)
	t.Logf("Added skill: %s v%s", skillEntry.Name, skillEntry.Version)

	// 获取技能索引
	index, err := leader.Service.GetSkillIndex(team.ID)
	require.NoError(t, err)
	assert.Len(t, index.Skills, 1, "Should have 1 skill")
	assert.Equal(t, "test-skill-001", index.Skills[0].PluginID)
	assert.Equal(t, "Test Skill", index.Skills[0].Name)
	assert.Equal(t, "1.0.0", index.Skills[0].Version)

	// 添加第二个技能
	skillEntry2 := &domainTeam.TeamSkillEntry{
		PluginID:       "test-skill-002",
		Name:           "Another Skill",
		Description:    "Another test skill",
		Version:        "2.0.0",
		AuthorID:       "author-001",
		AuthorName:     "Test Author",
		AuthorEndpoint: leader.Endpoint(),
		FileCount:      5,
		TotalSize:      2048,
		Checksum:       "xyz789",
		PublishedAt:    time.Now(),
	}

	err = leader.Service.AddSkillToIndex(team.ID, skillEntry2)
	require.NoError(t, err)

	// 验证有 2 个技能
	index, err = leader.Service.GetSkillIndex(team.ID)
	require.NoError(t, err)
	assert.Len(t, index.Skills, 2, "Should have 2 skills")

	t.Logf("Team has %d skills", len(index.Skills))
	for _, s := range index.Skills {
		t.Logf("  - %s v%s by %s", s.Name, s.Version, s.AuthorName)
	}
}

// TestSkillIndex_MemberGet 测试 Member 从 Leader 获取技能索引
func TestSkillIndex_MemberGet(t *testing.T) {
	ctx := context.Background()

	// 启动 Leader 节点
	leader := StartTestNode(t, 20003, "MemberGetLeader")
	defer leader.Stop()

	// Leader 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("MemberGetTeam", "", "")
	require.NoError(t, err)

	// Leader 添加技能
	skillEntry := &domainTeam.TeamSkillEntry{
		PluginID:       "shared-skill",
		Name:           "Shared Skill",
		Description:    "A skill shared with team members",
		Version:        "1.0.0",
		AuthorID:       "leader-id",
		AuthorName:     "MemberGetLeader",
		AuthorEndpoint: leader.Endpoint(),
		FileCount:      2,
		TotalSize:      512,
		Checksum:       "shared123",
		PublishedAt:    time.Now(),
	}
	err = leader.Service.AddSkillToIndex(team.ID, skillEntry)
	require.NoError(t, err)

	// 启动 Member 节点
	member := StartTestNode(t, 20004, "MemberGetMember")
	defer member.Stop()

	// Member 加入团队
	os.Setenv("HOME", member.HomeDir)
	_, err = member.Service.JoinTeam(ctx, leader.Endpoint())
	require.NoError(t, err)

	// 等待同步
	time.Sleep(200 * time.Millisecond)

	// Member 获取技能索引（通过 Leader 的 P2P 接口）
	index, err := member.Service.GetSkillIndex(team.ID)
	require.NoError(t, err)
	assert.Len(t, index.Skills, 1, "Member should see 1 skill from leader")
	assert.Equal(t, "shared-skill", index.Skills[0].PluginID)
	assert.Equal(t, "Shared Skill", index.Skills[0].Name)

	t.Logf("Member retrieved %d skills from leader", len(index.Skills))
}

// TestSkillIndex_Remove 测试从索引移除技能
func TestSkillIndex_Remove(t *testing.T) {
	// 启动 Leader 节点
	leader := StartTestNode(t, 20005, "RemoveSkillLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("RemoveSkillTeam", "", "")
	require.NoError(t, err)

	// 添加两个技能
	skill1 := &domainTeam.TeamSkillEntry{
		PluginID:    "skill-to-keep",
		Name:        "Skill To Keep",
		Version:     "1.0.0",
		PublishedAt: time.Now(),
	}
	skill2 := &domainTeam.TeamSkillEntry{
		PluginID:    "skill-to-remove",
		Name:        "Skill To Remove",
		Version:     "1.0.0",
		PublishedAt: time.Now(),
	}

	err = leader.Service.AddSkillToIndex(team.ID, skill1)
	require.NoError(t, err)
	err = leader.Service.AddSkillToIndex(team.ID, skill2)
	require.NoError(t, err)

	// 验证有 2 个技能
	index, _ := leader.Service.GetSkillIndex(team.ID)
	assert.Len(t, index.Skills, 2)

	// 移除一个技能
	err = leader.Service.RemoveSkillFromIndex(team.ID, "skill-to-remove")
	require.NoError(t, err)
	t.Log("Removed skill: skill-to-remove")

	// 验证只剩 1 个技能
	index, err = leader.Service.GetSkillIndex(team.ID)
	require.NoError(t, err)
	assert.Len(t, index.Skills, 1, "Should have 1 skill after removal")
	assert.Equal(t, "skill-to-keep", index.Skills[0].PluginID)

	t.Logf("Remaining skill: %s", index.Skills[0].Name)
}

// TestSkillIndex_UpdateExisting 测试更新已存在的技能
func TestSkillIndex_UpdateExisting(t *testing.T) {
	// 启动 Leader 节点
	leader := StartTestNode(t, 20006, "UpdateSkillLeader")
	defer leader.Stop()

	// 创建团队
	os.Setenv("HOME", leader.HomeDir)
	team, err := leader.Service.CreateTeam("UpdateSkillTeam", "", "")
	require.NoError(t, err)

	// 添加技能 v1.0.0
	skillEntry := &domainTeam.TeamSkillEntry{
		PluginID:    "updatable-skill",
		Name:        "Updatable Skill",
		Description: "Original description",
		Version:     "1.0.0",
		PublishedAt: time.Now(),
	}
	err = leader.Service.AddSkillToIndex(team.ID, skillEntry)
	require.NoError(t, err)

	// 更新技能到 v2.0.0
	skillEntry.Version = "2.0.0"
	skillEntry.Description = "Updated description"
	skillEntry.PublishedAt = time.Now()
	err = leader.Service.AddSkillToIndex(team.ID, skillEntry)
	require.NoError(t, err)

	// 验证技能已更新（应该还是 1 个，但版本变了）
	index, err := leader.Service.GetSkillIndex(team.ID)
	require.NoError(t, err)
	assert.Len(t, index.Skills, 1, "Should still have 1 skill (updated)")
	assert.Equal(t, "2.0.0", index.Skills[0].Version)
	assert.Equal(t, "Updated description", index.Skills[0].Description)

	t.Logf("Skill updated to v%s", index.Skills[0].Version)
}
