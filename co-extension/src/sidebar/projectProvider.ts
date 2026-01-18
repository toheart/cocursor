import * as vscode from "vscode";
import axios from "axios";

// 项目信息接口
interface ProjectInfo {
  project_name: string;
  project_id: string;
  workspaces: WorkspaceInfo[];
  git_remote_url?: string;
  git_branch?: string;
  created_at?: string;
  last_updated_at?: string;
}

interface WorkspaceInfo {
  workspace_id: string;
  path: string;
  project_name: string;
  git_remote_url?: string;
  git_branch?: string;
  is_active?: boolean;
  is_primary?: boolean;
}

// 项目列表项
class ProjectItem extends vscode.TreeItem {
  constructor(
    public readonly project: ProjectInfo,
    public readonly collapsibleState: vscode.TreeItemCollapsibleState
  ) {
    super(project.project_name, collapsibleState);

    this.tooltip = `项目: ${project.project_name}\n工作区数: ${project.workspaces.length}`;
    this.description = `${project.workspaces.length} 个工作区`;

    // 检查是否有活跃工作区
    const hasActive = project.workspaces.some((ws) => ws.is_active);
    if (hasActive) {
      this.iconPath = new vscode.ThemeIcon("circle-filled", new vscode.ThemeColor("charts.green"));
    } else {
      this.iconPath = new vscode.ThemeIcon("circle-outline");
    }

    // 添加命令：点击项目时显示详情
    this.command = {
      command: "cocursor.showProjectDetail",
      title: "显示项目详情",
      arguments: [project.project_name]
    };
  }
}

// 工作区列表项
class WorkspaceItem extends vscode.TreeItem {
  constructor(public readonly workspace: WorkspaceInfo) {
    super(
      workspace.path.split(/[/\\]/).pop() || workspace.path,
      vscode.TreeItemCollapsibleState.None
    );

    this.tooltip = `路径: ${workspace.path}\n工作区 ID: ${workspace.workspace_id}`;
    this.description = workspace.path;

    // 标记活跃和主工作区
    if (workspace.is_active) {
      this.iconPath = new vscode.ThemeIcon("play-circle", new vscode.ThemeColor("charts.green"));
    } else if (workspace.is_primary) {
      this.iconPath = new vscode.ThemeIcon("star", new vscode.ThemeColor("charts.yellow"));
    } else {
      this.iconPath = new vscode.ThemeIcon("folder");
    }

    // 添加命令：点击工作区时显示统计信息
    this.command = {
      command: "cocursor.showWorkspaceStats",
      title: "显示工作区统计",
      arguments: [workspace.workspace_id]
    };
  }
}

export class ProjectProvider implements vscode.TreeDataProvider<ProjectItem | WorkspaceItem> {
  private _onDidChangeTreeData: vscode.EventEmitter<ProjectItem | WorkspaceItem | undefined | null | void> =
    new vscode.EventEmitter<ProjectItem | WorkspaceItem | undefined | null | void>();
  readonly onDidChangeTreeData: vscode.Event<ProjectItem | WorkspaceItem | undefined | null | void> =
    this._onDidChangeTreeData.event;

  private projects: ProjectInfo[] = [];
  private loading = false;

  constructor(private context: vscode.ExtensionContext) {}

  refresh(): void {
    this.loadProjects();
  }

  async loadProjects(): Promise<void> {
    if (this.loading) {
      return;
    }

    this.loading = true;
    try {
      const response = await axios.get("http://localhost:19960/api/v1/project/list", {
        timeout: 5000
      });

      if (response.data && response.data.data) {
        // 确保 data 是数组格式
        const data = response.data.data;
        if (data.projects && Array.isArray(data.projects)) {
          this.projects = data.projects as ProjectInfo[];
        } else if (Array.isArray(data)) {
          this.projects = data as ProjectInfo[];
        } else {
          console.warn("项目列表数据格式不正确:", data);
          this.projects = [];
        }
      } else {
        this.projects = [];
      }
    } catch (error) {
      console.error("加载项目列表失败:", error);
      this.projects = [];
      // 静默失败，不显示错误提示
    } finally {
      this.loading = false;
      this._onDidChangeTreeData.fire();
    }
  }

  getTreeItem(element: ProjectItem | WorkspaceItem): vscode.TreeItem {
    return element;
  }

  getChildren(element?: ProjectItem | WorkspaceItem): Thenable<(ProjectItem | WorkspaceItem)[]> {
    if (!element) {
      // 根节点 - 显示项目列表
      if (this.projects.length === 0 && !this.loading) {
        // 如果还没有加载过，尝试加载
        this.loadProjects();
        return Promise.resolve([
          new vscode.TreeItem(
            "加载中...",
            vscode.TreeItemCollapsibleState.None
          ) as ProjectItem
        ]);
      }

      // 确保 projects 是数组
      if (!Array.isArray(this.projects)) {
        console.warn("projects 不是数组:", this.projects);
        this.projects = [];
      }
      
      return Promise.resolve(
        this.projects.map((project) => new ProjectItem(project, vscode.TreeItemCollapsibleState.Collapsed))
      );
    } else if (element instanceof ProjectItem) {
      // 项目节点 - 显示工作区列表
      return Promise.resolve(
        element.project.workspaces.map((ws) => new WorkspaceItem(ws))
      );
    }

    return Promise.resolve([]);
  }
}
