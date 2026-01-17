import * as vscode from "vscode";

export class SidebarProvider implements vscode.TreeDataProvider<SidebarItem> {
  private _onDidChangeTreeData: vscode.EventEmitter<SidebarItem | undefined | null | void> =
    new vscode.EventEmitter<SidebarItem | undefined | null | void>();
  readonly onDidChangeTreeData: vscode.Event<SidebarItem | undefined | null | void> =
    this._onDidChangeTreeData.event;

  constructor(private context: vscode.ExtensionContext) {}

  refresh(): void {
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: SidebarItem): vscode.TreeItem {
    return element;
  }

  getChildren(element?: SidebarItem): Thenable<SidebarItem[]> {
    if (!element) {
      // 根节点 - 显示主要功能
      return Promise.resolve([
        new SidebarItem(
          "对话列表",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.openDashboard",
            title: "打开对话列表",
            arguments: []
          },
          "$(comment-discussion)"
        ),
        new SidebarItem(
          "团队管理",
          vscode.TreeItemCollapsibleState.Collapsed,
          undefined,
          "$(organization)"
        ),
        new SidebarItem(
          "节点列表",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.showPeers",
            title: "显示节点列表",
            arguments: []
          },
          "$(server-process)"
        ),
        new SidebarItem(
          "使用统计",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.showStats",
            title: "显示使用统计",
            arguments: []
          },
          "$(graph)"
        )
      ]);
    } else if (element.label === "团队管理") {
      // 团队管理子项
      return Promise.resolve([
        new SidebarItem(
          "当前团队",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.showCurrentTeam",
            title: "显示当前团队",
            arguments: []
          },
          "$(account)"
        ),
        new SidebarItem(
          "加入团队",
          vscode.TreeItemCollapsibleState.None,
          {
            command: "cocursor.joinTeam",
            title: "加入团队",
            arguments: []
          },
          "$(add)"
        )
      ]);
    }
    return Promise.resolve([]);
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

    if (icon) {
      this.iconPath = new vscode.ThemeIcon(icon);
    }

    if (command) {
      this.command = command;
    }
  }
}
