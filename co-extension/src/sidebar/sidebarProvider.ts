import * as vscode from "vscode";
import axios from "axios";

// Token 使用统计接口
interface TokenUsage {
  date: string;
  total_tokens: number;
  by_type: {
    tab: number;
    composer: number;
    chat: number;
  };
  trend: string;
}

export class SidebarProvider implements vscode.TreeDataProvider<SidebarItem> {
  private _onDidChangeTreeData: vscode.EventEmitter<SidebarItem | undefined | null | void> =
    new vscode.EventEmitter<SidebarItem | undefined | null | void>();
  readonly onDidChangeTreeData: vscode.Event<SidebarItem | undefined | null | void> =
    this._onDidChangeTreeData.event;

  private tokenUsage: TokenUsage | null = null;
  private refreshInterval: NodeJS.Timeout | null = null;

  constructor(private context: vscode.ExtensionContext) {
    // 加载 Token 数据
    this.loadTokenUsage();
    
    // 定时刷新（每 5 分钟）
    this.refreshInterval = setInterval(() => {
      this.loadTokenUsage();
    }, 5 * 60 * 1000);

    // 注册清理
    context.subscriptions.push({
      dispose: () => {
        if (this.refreshInterval) {
          clearInterval(this.refreshInterval);
        }
      }
    });
  }

  async loadTokenUsage(): Promise<void> {
    try {
      const response = await axios.get("http://localhost:19960/api/v1/stats/token-usage", {
        timeout: 5000
      });

      if (response.data && response.data.code === 0 && response.data.data) {
        this.tokenUsage = response.data.data as TokenUsage;
        this._onDidChangeTreeData.fire(); // 触发 UI 更新
      }
    } catch (error) {
      // 静默失败，不显示错误
      console.log("加载 Token 使用统计失败:", error);
    }
  }

  refresh(): void {
    this.loadTokenUsage();
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: SidebarItem): vscode.TreeItem {
    return element;
  }

  getChildren(element?: SidebarItem): Thenable<SidebarItem[]> {
    if (!element) {
      // 根节点 - 显示主要功能
      const items: SidebarItem[] = [];

      // Token 消耗（可展开）
      if (this.tokenUsage) {
        const tokenText = this.formatTokenCount(this.tokenUsage.total_tokens);
        const trendIcon = this.tokenUsage.trend.startsWith("+") ? "↑" : this.tokenUsage.trend.startsWith("-") ? "↓" : "";
        items.push(
          new SidebarItem(
            `今日 Token: ${tokenText} ${trendIcon} ${this.tokenUsage.trend}`,
            vscode.TreeItemCollapsibleState.Collapsed,
            undefined,
            "pulse"
          )
        );
      } else {
        items.push(
          new SidebarItem(
            "今日 Token: 加载中...",
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "sync~spin"
          )
        );
      }

      items.push(
        new SidebarItem(
          "工作分析",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openWorkAnalysis",
            title: "打开工作分析",
            arguments: []
          },
          "graph"
        ),
        new SidebarItem(
          "最近对话",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openSessions",
            title: "打开最近对话",
            arguments: []
          },
          "comment-discussion"
        ),
        new SidebarItem(
          "插件市场",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openMarketplace",
            title: "打开插件市场",
            arguments: []
          },
          "extensions"
        )
      );

      return Promise.resolve(items);
    } else if (element.label && element.label.startsWith("今日 Token:")) {
      // Token 消耗详情
      if (this.tokenUsage) {
        return Promise.resolve([
          new SidebarItem(
            `Tab: ${this.formatTokenCount(this.tokenUsage.by_type.tab)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "symbol-keyword"
          ),
          new SidebarItem(
            `Composer: ${this.formatTokenCount(this.tokenUsage.by_type.composer)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "code"
          ),
          new SidebarItem(
            `Chat: ${this.formatTokenCount(this.tokenUsage.by_type.chat)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "comment"
          )
        ]);
      }
      return Promise.resolve([]);
    }
    return Promise.resolve([]);
  }

  formatTokenCount(count: number): string {
    if (count >= 1000000) {
      return `${(count / 1000000).toFixed(1)}M`;
    } else if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}K`;
    }
    return count.toString();
  }
}

class SidebarItem extends vscode.TreeItem {
  constructor(
    public readonly label: string,
    public readonly collapsibleState: vscode.TreeItemCollapsibleState,
    public readonly command?: vscode.Command,
    public readonly icon?: string
  ) {
    super(label, collapsibleState);

    this.tooltip = label;
    this.description = "";

    // 确保图标正确设置
    if (icon) {
      try {
        // 移除可能的 $() 包装，ThemeIcon 构造函数不需要
        const iconName = icon.replace(/^\$\((.+)\)$/, "$1");
        this.iconPath = new vscode.ThemeIcon(iconName);
      } catch (error) {
        // 如果图标无效，不设置图标（避免显示错误图标）
        console.warn(`Invalid icon: ${icon}`, error);
        this.iconPath = undefined;
      }
    }
    // 如果没有提供图标，不设置 iconPath（让 VS Code 使用默认行为）

    if (command) {
      this.command = command;
    }
  }
}
