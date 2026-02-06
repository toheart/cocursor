import * as vscode from "vscode";
import axios from "axios";

// 会话数据接口（匹配后端 ComposerData 格式）
interface Session {
  composerId: string;
  name: string;
  type: string;
  createdAt: number;
  lastUpdatedAt: number;
  unifiedMode?: string;
  subtitle?: string;
  // workspaceId 不在 API 返回中，需要从上下文获取
}

// 项目数据接口
interface ProjectSessions {
  workspaceId: string;
  projectName: string;
  sessions: Session[];
  isCurrentProject: boolean;
}

/**
 * 会话列表 TreeDataProvider
 * 按项目组织会话，支持懒加载
 */
export class SessionProvider implements vscode.TreeDataProvider<SessionTreeItem> {
  private _onDidChangeTreeData: vscode.EventEmitter<SessionTreeItem | undefined | null | void> =
    new vscode.EventEmitter<SessionTreeItem | undefined | null | void>();
  readonly onDidChangeTreeData: vscode.Event<SessionTreeItem | undefined | null | void> =
    this._onDidChangeTreeData.event;

  // 项目缓存
  private projectsCache: Map<string, ProjectSessions> = new Map();
  // 当前工作区 ID
  private currentWorkspaceId: string | undefined;

  constructor(private context: vscode.ExtensionContext) {
    // 监听工作区变化
    vscode.window.onDidChangeActiveTextEditor(() => {
      this.detectCurrentWorkspace();
    });
    
    // 初始化检测当前工作区
    this.detectCurrentWorkspace();
  }

  /**
   * 检测当前工作区
   */
  private detectCurrentWorkspace(): void {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (workspaceFolders && workspaceFolders.length > 0) {
      const newWorkspaceId = workspaceFolders[0].uri.fsPath;
      if (newWorkspaceId !== this.currentWorkspaceId) {
        this.currentWorkspaceId = newWorkspaceId;
        this._onDidChangeTreeData.fire();
      }
    }
  }

  /**
   * 刷新列表
   */
  refresh(): void {
    this.projectsCache.clear();
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: SessionTreeItem): vscode.TreeItem {
    return element;
  }

  async getChildren(element?: SessionTreeItem): Promise<SessionTreeItem[]> {
    if (!element) {
      // 根节点 - 显示项目列表
      return this.getProjectItems();
    }

    if (element.type === "project") {
      // 项目节点 - 显示该项目的会话列表，使用项目名
      return this.getSessionItems(element.projectName || element.label.replace(/^★ /, ""));
    }

    return [];
  }

  /**
   * 获取项目列表
   */
  private async getProjectItems(): Promise<SessionTreeItem[]> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/project/list", {
        timeout: 5000
      });

      if (response.data && response.data.code === 0 && response.data.data) {
        // API 返回 ProjectInfo 格式
        const projects = response.data.data.projects as Array<{
          project_name: string;
          project_id: string;
          workspaces: Array<{
            workspace_id: string;
            path: string;
            is_active: boolean;
          }>;
          last_updated_at: string;
        }>;

        // 按最后更新时间排序，当前项目置顶
        const sortedProjects = projects.sort((a, b) => {
          // 检查是否为当前工作区的项目
          const aIsCurrent = a.workspaces.some(w => w.path === this.currentWorkspaceId);
          const bIsCurrent = b.workspaces.some(w => w.path === this.currentWorkspaceId);
          
          if (aIsCurrent && !bIsCurrent) return -1;
          if (!aIsCurrent && bIsCurrent) return 1;
          
          // 按最后更新时间倒序
          const aTime = new Date(a.last_updated_at).getTime();
          const bTime = new Date(b.last_updated_at).getTime();
          return bTime - aTime;
        });

        return sortedProjects.map(project => {
          const isCurrent = project.workspaces.some(w => w.path === this.currentWorkspaceId);
          const label = isCurrent ? `★ ${project.project_name}` : project.project_name;
          // 使用第一个工作区的 ID 作为项目标识（用于后续获取会话）
          const primaryWorkspace = project.workspaces.find(w => w.is_active) || project.workspaces[0];
          
          return new SessionTreeItem(
            label,
            vscode.TreeItemCollapsibleState.Collapsed,
            "project",
            primaryWorkspace?.workspace_id || project.project_id,
            undefined,
            `${project.workspaces.length} 个工作区`,
            "folder",
            undefined,
            project.project_name // 保存项目名用于后续查询
          );
        });
      }

      return [];
    } catch (error) {
      console.error("Failed to load projects:", error);
      return [
        new SessionTreeItem(
          "加载失败，点击重试",
          vscode.TreeItemCollapsibleState.None,
          "error",
          undefined,
          {
            command: "cocursor.refreshSessions",
            title: "刷新",
            arguments: []
          },
          undefined,
          "warning"
        )
      ];
    }
  }

  /**
   * 获取指定项目的会话列表
   */
  private async getSessionItems(projectName: string): Promise<SessionTreeItem[]> {
    try {
      // 检查缓存
      const cached = this.projectsCache.get(projectName);
      if (cached) {
        return this.sessionsToItems(cached.sessions, projectName);
      }

      // 从 API 加载，使用 project_name 参数
      const response = await axios.get("http://localhost:19960/api/v1/sessions/list", {
        params: {
          project_name: projectName,
          limit: 50
        },
        timeout: 10000
      });

      if (response.data && response.data.code === 0 && response.data.data) {
        // API 返回的是数组（分页格式下 data 直接是数组）
        const rawData = response.data.data;
        const sessions = (Array.isArray(rawData) ? rawData : rawData.list || []) as Session[];
        
        // 缓存结果
        this.projectsCache.set(projectName, {
          workspaceId: projectName,
          projectName,
          sessions,
          isCurrentProject: false
        });

        return this.sessionsToItems(sessions, projectName);
      }

      return [];
    } catch (error) {
      console.error("Failed to load sessions:", error);
      return [
        new SessionTreeItem(
          "加载失败",
          vscode.TreeItemCollapsibleState.None,
          "error",
          undefined,
          undefined,
          undefined,
          "warning"
        )
      ];
    }
  }

  /**
   * 将会话转换为树项
   */
  private sessionsToItems(sessions: Session[], projectName: string): SessionTreeItem[] {
    // 过滤掉无效会话，按最后更新时间倒序排序（使用 createdAt 作为 fallback）
    const validSessions = sessions.filter(s => s.composerId);
    const sortedSessions = validSessions.sort((a, b) => {
      const aTime = a.lastUpdatedAt || a.createdAt || 0;
      const bTime = b.lastUpdatedAt || b.createdAt || 0;
      return bTime - aTime;
    });

    return sortedSessions.map(session => {
      // 优先使用 name，然后 subtitle 前50字符，最后使用默认名
      let title = session.name;
      if (!title && session.subtitle) {
        title = session.subtitle.length > 50 
          ? session.subtitle.substring(0, 50) + "..." 
          : session.subtitle;
      }
      if (!title) {
        title = "未命名会话";
      }

      const timestamp = session.lastUpdatedAt || session.createdAt || 0;
      const date = new Date(timestamp);
      const timeStr = timestamp > 0 ? this.formatTime(date) : "";

      return new SessionTreeItem(
        title,
        vscode.TreeItemCollapsibleState.None,
        "session",
        projectName, // 使用项目名作为 workspaceId（用于后续查询）
        undefined,
        timeStr,
        "comment-discussion",
        session.composerId,
        projectName
      );
    });
  }

  /**
   * 格式化时间
   */
  private formatTime(date: Date): string {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return "刚刚";
    if (minutes < 60) return `${minutes} 分钟前`;
    if (hours < 24) return `${hours} 小时前`;
    if (days < 7) return `${days} 天前`;
    
    return date.toLocaleDateString();
  }

  /**
   * 获取会话详情（用于分享）
   */
  async getSessionForShare(composerId: string, workspaceId: string): Promise<Session | undefined> {
    const cached = this.projectsCache.get(workspaceId);
    if (cached) {
      return cached.sessions.find(s => s.composerId === composerId);
    }
    
    // 如果没有缓存，从 API 获取
    try {
      const response = await axios.get(`http://localhost:19960/api/v1/sessions/${composerId}/detail`, {
        params: { workspace_id: workspaceId },
        timeout: 10000
      });
      
      if (response.data && response.data.code === 0 && response.data.data) {
        return response.data.data.session as Session;
      }
    } catch (error) {
      console.error("Failed to get session for share:", error);
    }
    
    return undefined;
  }
}

/**
 * 会话树项
 */
export class SessionTreeItem extends vscode.TreeItem {
  constructor(
    public readonly label: string,
    public readonly collapsibleState: vscode.TreeItemCollapsibleState,
    public readonly type: "project" | "session" | "error",
    public readonly workspaceId?: string,
    public readonly command?: vscode.Command,
    public readonly description?: string,
    public readonly icon?: string,
    public readonly composerId?: string,
    public readonly projectName?: string
  ) {
    super(label, collapsibleState);

    this.tooltip = label;
    
    if (description) {
      this.description = description;
    }

    if (icon) {
      try {
        const iconName = icon.replace(/^\$\((.+)\)$/, "$1");
        this.iconPath = new vscode.ThemeIcon(iconName);
      } catch (error) {
        console.warn(`Invalid icon: ${icon}`, error);
        this.iconPath = undefined;
      }
    }

    if (command) {
      this.command = command;
    }

    // 设置上下文值，用于菜单条件
    if (type === "session") {
      this.contextValue = "session";
    } else if (type === "project") {
      this.contextValue = "project";
    }
  }
}
