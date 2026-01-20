import * as vscode from "vscode";
import axios from "axios";
import { t, changeLanguage } from "../utils/i18n";

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
  method: string; // "tiktoken" 或 "estimate"
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

  // 刷新语言并更新侧边栏
  refreshLanguage(): void {
    const savedLanguage = this.context.globalState.get<string>('cocursor-language');
    if (savedLanguage === 'zh-CN' || savedLanguage === 'en') {
      changeLanguage(savedLanguage);
      this._onDidChangeTreeData.fire(); // 刷新侧边栏
    }
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
      console.log("Failed to load token usage statistics:", error);
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

      // 今日效率（可展开）
      if (this.tokenUsage) {
        const tokenText = this.formatTokenCount(this.tokenUsage.total_tokens);
        const trendIcon = this.tokenUsage.trend.startsWith("+") ? "↑" : this.tokenUsage.trend.startsWith("-") ? "↓" : "";
        items.push(
          new SidebarItem(
            `${t("sidebar.todayEfficiency")}: ${tokenText} ${trendIcon} ${this.tokenUsage.trend}`,
            vscode.TreeItemCollapsibleState.Collapsed,
            undefined,
            "dashboard"
          )
        );
      } else {
        items.push(
          new SidebarItem(
            `${t("sidebar.todayEfficiency")}: ${t("sidebar.tokenLoading")}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "sync~spin"
          )
        );
      }

      items.push(
        new SidebarItem(
          t("sidebar.workAnalysis"),
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openWorkAnalysis",
            title: t("sidebar.openWorkAnalysis"),
            arguments: []
          },
          "graph"
        ),
        new SidebarItem(
          t("sidebar.workflow"),
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openWorkflows",
            title: t("sidebar.openWorkflow"),
            arguments: []
          },
          "git-branch"
        ),
        // 隐藏最近对话功能
        // new SidebarItem(
        //   t("sidebar.recentSessions"),
        //   vscode.TreeItemCollapsibleState.None,
        //   {
        //     command: "cocursor.openSessions",
        //     title: t("sidebar.openSessions"),
        //     arguments: []
        //   },
        //   "comment-discussion"
        // ),
        // RAG 搜索功能（Beta）
        new SidebarItem(
          `${t("sidebar.ragSearch")} Beta`,
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openRAGSearch",
            title: t("sidebar.openRAGSearch"),
            arguments: []
          },
          "search"
        ),
        new SidebarItem(
          t("sidebar.marketplace"),
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openMarketplace",
            title: t("sidebar.openMarketplace"),
            arguments: []
          },
          "extensions"
        ),
        // 团队功能
        new SidebarItem(
          t("sidebar.team"),
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openTeam",
            title: t("sidebar.openTeam"),
            arguments: []
          },
          "organization"
        )
      );

      return Promise.resolve(items);
    } else if (element.label && element.label.includes(t("sidebar.todayEfficiency"))) {
      // 今日效率详情
      if (this.tokenUsage) {
        const items: SidebarItem[] = [];
        
        // Token 统计标题
        items.push(
          new SidebarItem(
            `Token: ${this.formatTokenCount(this.tokenUsage.total_tokens)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "pulse"
          )
        );
        
        // Token 分类详情
        items.push(
          new SidebarItem(
            `  ${t("sidebar.tokenTypes.tab")}: ${this.formatTokenCount(this.tokenUsage.by_type.tab)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "symbol-keyword"
          )
        );
        items.push(
          new SidebarItem(
            `  ${t("sidebar.tokenTypes.composer")}: ${this.formatTokenCount(this.tokenUsage.by_type.composer)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "code"
          )
        );
        items.push(
          new SidebarItem(
            `  ${t("sidebar.tokenTypes.chat")}: ${this.formatTokenCount(this.tokenUsage.by_type.chat)}`,
            vscode.TreeItemCollapsibleState.None,
            undefined,
            "comment"
          )
        );
        
        return Promise.resolve(items);
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
