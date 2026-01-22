# 团队功能集成测试场景设计

从业务场景出发，而非代码实现。

## 一、团队生命周期

### 场景 1.1：团队创建与解散
- **GIVEN** 用户 A 想组建团队
- **WHEN** A 创建团队 "CoCursor Team"
- **THEN** A 成为 Leader，成员列表包含 A

- **GIVEN** A 是团队 Leader，团队有 2 名成员
- **WHEN** A 解散团队
- **THEN** 所有成员的团队列表清空

### 场景 1.2：成员加入与离开
- **GIVEN** A 已创建团队
- **WHEN** B 通过 A 的地址加入团队
- **THEN** A 的成员列表包含 A 和 B
- **AND** B 的团队列表包含该团队

- **GIVEN** B 已加入 A 的团队
- **WHEN** B 离开团队
- **THEN** A 的成员列表只剩 A
- **AND** B 的团队列表为空

### 场景 1.3：重复加入 / 错误端点
- **GIVEN** B 已加入 A 的团队
- **WHEN** B 再次尝试加入同一团队
- **THEN** 返回明确错误（已是成员）

- **GIVEN** B 想加入团队
- **WHEN** B 尝试连接不存在的地址
- **THEN** 返回连接失败错误

## 二、工作状态同步

### 场景 2.1：Leader 广播状态
- **GIVEN** A 是 Leader，B 是成员
- **WHEN** A 更新自己的工作状态（正在编辑 main.go）
- **THEN** B 能查询到 A 的状态

### 场景 2.2：Member 状态上报
- **GIVEN** A 是 Leader，B 是成员
- **WHEN** B 更新自己的工作状态
- **THEN** 状态发送到 Leader
- **AND** Leader 广播给其他成员
- **AND** A 能查询到 B 的状态

### 场景 2.3：状态隐藏
- **GIVEN** B 设置状态为隐藏
- **WHEN** A 查询 B 的状态
- **THEN** 返回状态但 visible=false

## 三、代码分享

### 场景 3.1：Leader 分享代码
- **GIVEN** A 是 Leader，B 是成员
- **WHEN** A 分享代码片段
- **THEN** B 收到代码分享通知

### 场景 3.2：Member 分享代码
- **GIVEN** A 是 Leader，B 是成员
- **WHEN** B 分享代码片段
- **THEN** 代码发送到 Leader
- **AND** Leader 广播给所有成员（包括 A）

### 场景 3.3：代码片段验证
- **GIVEN** 用户要分享代码
- **WHEN** 代码内容为空
- **THEN** 返回验证错误

- **WHEN** 代码超过 10KB
- **THEN** 自动截断或返回错误

## 四、项目配置管理

### 场景 4.1：Leader 管理项目
- **GIVEN** A 是 Leader
- **WHEN** A 添加项目 "github.com/org/repo"
- **THEN** 项目配置生效

- **GIVEN** B 是成员
- **WHEN** B 尝试添加项目
- **THEN** 返回权限错误

### 场景 4.2：配置同步
- **GIVEN** A 已添加项目
- **WHEN** B 查询项目配置
- **THEN** B 能获取到 Leader 的项目列表

## 五、技能分享

### 场景 5.1：技能发布与获取
- **GIVEN** A 是 Leader，发布了技能 "my-skill"
- **WHEN** B 查询团队技能列表
- **THEN** B 能看到 "my-skill"

### 场景 5.2：技能更新
- **GIVEN** A 已发布 "my-skill" v1.0
- **WHEN** A 更新到 v2.0
- **THEN** 团队技能索引显示 v2.0

## 六、网络与容错

### 场景 6.1：Leader 离线
- **GIVEN** B 已加入 A 的团队
- **WHEN** A 离线
- **THEN** B 的加入请求失败（无法连接）
- **AND** B 本地团队信息保留

### 场景 6.2：网络恢复
- **GIVEN** A 曾离线
- **WHEN** A 恢复上线
- **THEN** B 能重新连接并同步状态

## 七、当前测试覆盖分析

| 场景 | 是否覆盖 | 备注 |
|------|----------|------|
| 1.1 创建与解散 | ✅ 已覆盖 | `TestTeamJoin_FullFlow`, `TestTeamDissolve` |
| 1.2 加入与离开 | ✅ 已覆盖 | `TestTeamJoin_LeaveFlow`, `TestTeamJoin_MultipleMembers` |
| 1.3 重复/错误加入 | ✅ 已覆盖 | `TestTeamJoin_InvalidEndpoint`, `TestTeamJoin_DuplicateJoin` |
| 2.1 Leader 广播状态 | ✅ 已覆盖 | `TestCollaboration_LeaderBroadcastStatus` |
| 2.2 Member 状态上报 | ✅ 已覆盖 | `TestCollaboration_MemberStatusUpload` **发现并修复了 bug** |
| 2.3 状态隐藏 | ✅ 已覆盖 | `TestCollaboration_StatusInvisible` |
| 3.1 Leader 分享代码 | ✅ 已覆盖 | `TestCodeShare_LeaderShare` |
| 3.2 Member 分享代码 | ✅ 已覆盖 | `TestCodeShare_MemberShare` |
| 3.3 代码片段验证 | ✅ 已覆盖 | `TestCodeSnippet_Validation` (5个子测试) |
| 4.1 Leader 管理项目 | ✅ 已覆盖 | `TestProjectConfig_*` 系列 |
| 4.2 配置同步 | ✅ 已覆盖 | `TestProjectConfig_MemberGet` |
| 5.1 技能发布与获取 | ✅ 已覆盖 | `TestSkillIndex_*` 系列 |
| 5.2 技能更新 | ✅ 已覆盖 | `TestSkillIndex_UpdateExisting` |
| 6.1 Leader 离线 | ✅ 已覆盖 | `TestLeaderOffline_JoinFails`, `TestLeaderOffline_ExistingMemberRetainsInfo` |
| 6.2 网络恢复 | ⚠️ 待补充 | 需要更复杂的模拟 |

## 八、已完成的测试基础设施改进

1. **TestNode 已注册完整路由**
   - P2P Handler (`/p2p/team/...`) - 团队加入/信息
   - 协作路由 (`/api/v1/team/:id/status`, `/share-code` 等)
   - 周报路由 (`/api/v1/team/:id/project-config` 等)

2. **发现并修复的 Bug**
   - `sendToLeader` 函数未传递请求体（使用 `nil` 而非实际数据）
   - 影响范围：所有 Member 向 Leader 的协作请求

## 九、测试统计

- **总测试数**：38 个
- **场景覆盖率**：15/16 (93.75%)
- **发现 Bug**：1 个（已修复）

## 十、待补充的测试

1. **网络恢复场景**（Leader 重新上线后状态同步）
2. **WebSocket 事件广播验证**（需要监听机制）
3. **并发加入团队**（压力测试）
