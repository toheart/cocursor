import React, { useState, useEffect } from "react";
import { apiService } from "../services/api";

interface Plugin {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  icon?: string;
  category: string;
  installed: boolean;
  installed_version?: string;
  skill: {
    skill_name: string;
  };
  mcp?: {
    server_name: string;
    transport: string;
    url: string;
  };
  command?: {
    commands: Array<{
      command_id: string;
    }>;
  };
}

interface Toast {
  id: string;
  message: string;
  type: "success" | "error";
}

export const Marketplace: React.FC = () => {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [installingPlugins, setInstallingPlugins] = useState<Set<string>>(new Set());
  const [expandedPlugins, setExpandedPlugins] = useState<Set<string>>(new Set());
  const [toasts, setToasts] = useState<Toast[]>([]);

  useEffect(() => {
    loadPlugins();
  }, [selectedCategory]);

  // 搜索防抖处理
  useEffect(() => {
    const timer = setTimeout(() => {
      loadPlugins();
    }, 300);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchQuery]);

  const getWorkspacePath = (): string => {
    // 从 window 对象获取工作区路径（由 webviewPanel 注入）
    const workspacePath = (window as any).__WORKSPACE_PATH__;
    if (!workspacePath) {
      console.warn("Workspace path not found, using current directory");
      return "";
    }
    return workspacePath;
  };

  const loadPlugins = async () => {
    setLoading(true);
    try {
      const response = await apiService.getPlugins(
        selectedCategory !== "all" ? selectedCategory : undefined,
        searchQuery || undefined,
        undefined
      ) as { plugins?: Plugin[]; total?: number };

      if (response.plugins) {
        setPlugins(response.plugins);
      } else {
        setPlugins([]);
      }
    } catch (error) {
      console.error("Failed to load plugins:", error);
      setPlugins([]);
    } finally {
      setLoading(false);
    }
  };

  const showToast = (message: string, type: "success" | "error") => {
    const id = Date.now().toString();
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  };

  const handleInstall = async (pluginId: string) => {
    setInstallingPlugins((prev) => new Set(prev).add(pluginId));
    try {
      const workspacePath = getWorkspacePath();
      const response = await apiService.installPlugin(pluginId, workspacePath) as {
        success?: boolean;
        message?: string;
        env_vars?: string[];
        error?: string;
      };

      if (response.error) {
        console.error("Failed to install plugin:", response.error);
        showToast(`安装失败: ${response.error}`, "error");
        return;
      }

      showToast("安装成功！", "success");
      // 刷新插件列表
      await loadPlugins();
    } catch (error) {
      console.error("Failed to install plugin:", error);
      showToast("安装失败，请稍后重试", "error");
    } finally {
      setInstallingPlugins((prev) => {
        const next = new Set(prev);
        next.delete(pluginId);
        return next;
      });
    }
  };

  const handleUninstall = async (pluginId: string) => {
    setInstallingPlugins((prev) => new Set(prev).add(pluginId));
    try {
      const workspacePath = getWorkspacePath();
      const response = await apiService.uninstallPlugin(pluginId, workspacePath) as {
        success?: boolean;
        message?: string;
        error?: string;
      };

      if (response.error) {
        console.error("Failed to uninstall plugin:", response.error);
        showToast(`卸载失败: ${response.error}`, "error");
        return;
      }

      showToast("卸载成功！", "success");
      // 刷新插件列表
      await loadPlugins();
    } catch (error) {
      console.error("Failed to uninstall plugin:", error);
      showToast("卸载失败，请稍后重试", "error");
    } finally {
      setInstallingPlugins((prev) => {
        const next = new Set(prev);
        next.delete(pluginId);
        return next;
      });
    }
  };

  const toggleExpand = (pluginId: string) => {
    setExpandedPlugins((prev) => {
      const next = new Set(prev);
      if (next.has(pluginId)) {
        next.delete(pluginId);
      } else {
        next.add(pluginId);
      }
      return next;
    });
  };

  const categories = ["all", "工具", "集成", "AI", "主题", "其他"];
  const filteredPlugins = plugins.filter((plugin) => {
    const matchesSearch =
      plugin.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      plugin.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory =
      selectedCategory === "all" || plugin.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  return (
    <div className="cocursor-marketplace">
      <div className="cocursor-marketplace-hero">
        <h1 className="cocursor-marketplace-title">插件市场</h1>
        <p className="cocursor-marketplace-subtitle">发现并安装强大的扩展插件</p>
      </div>
      <div className="cocursor-marketplace-header">
        <div className="cocursor-marketplace-search-wrapper">
          <div className="cocursor-marketplace-search-icon">🔍</div>
          <input
            type="text"
            placeholder="搜索插件..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="cocursor-marketplace-search-input"
          />
        </div>
        <div className="cocursor-marketplace-categories">
          {categories.map((category, index) => (
            <button
              key={category}
              className={`cocursor-marketplace-category ${
                selectedCategory === category ? "active" : ""
              }`}
              onClick={() => setSelectedCategory(category)}
              style={{ animationDelay: `${index * 50}ms` }}
            >
              {category === "all" ? "全部" : category}
            </button>
          ))}
        </div>
      </div>

      {/* Toast 通知 */}
      <div className="cocursor-marketplace-toasts">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`cocursor-marketplace-toast cocursor-marketplace-toast-${toast.type}`}
          >
            {toast.type === "success" ? "✓" : "✗"} {toast.message}
          </div>
        ))}
      </div>

      {loading ? (
        <div className="cocursor-marketplace-plugins">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="cocursor-marketplace-plugin-skeleton">
              <div className="cocursor-marketplace-plugin-skeleton-header">
                <div className="cocursor-marketplace-plugin-skeleton-icon"></div>
                <div className="cocursor-marketplace-plugin-skeleton-info">
                  <div className="cocursor-marketplace-plugin-skeleton-title"></div>
                  <div className="cocursor-marketplace-plugin-skeleton-meta"></div>
                </div>
                <div className="cocursor-marketplace-plugin-skeleton-button"></div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="cocursor-marketplace-plugins">
          {filteredPlugins.length === 0 ? (
            <div className="cocursor-marketplace-empty">
              <div className="cocursor-marketplace-empty-icon">📦</div>
              <p>暂无插件</p>
              <span>尝试调整搜索条件或筛选器</span>
            </div>
          ) : (
            filteredPlugins.map((plugin, index) => {
              // 生成使用说明
              const usageInstructions = [];
              if (plugin.skill) {
                usageInstructions.push({
                  type: "Skill",
                  title: "Skill 组件",
                  description: `此插件包含 Skill: ${plugin.skill.skill_name}。安装后，该 Skill 将自动添加到项目的 AGENTS.md 文件中，可在对话中使用。`
                });
              }
              if (plugin.mcp) {
                usageInstructions.push({
                  type: "MCP",
                  title: "MCP 组件",
                  description: `此插件包含 MCP 服务器: ${plugin.mcp.server_name}。安装后，MCP 配置将添加到 ~/.cursor/mcp.json 中，需要重启 Cursor 才能生效。`
                });
              }
              if (plugin.command && plugin.command.commands && plugin.command.commands.length > 0) {
                const commandNames = plugin.command.commands.map(cmd => `/${cmd.command_id}`).join("、");
                usageInstructions.push({
                  type: "Command",
                  title: "Command 组件",
                  description: `此插件包含命令: ${commandNames}。安装后，可在 Cursor 中使用此命令。`
                });
              }

              const isExpanded = expandedPlugins.has(plugin.id);
              const isInstalling = installingPlugins.has(plugin.id);

              return (
                <div 
                  key={plugin.id} 
                  className={`cocursor-marketplace-plugin ${plugin.installed ? "installed" : ""}`}
                  style={{ animationDelay: `${index * 80}ms` }}
                >
                  {/* 紧凑头部 - 图标、名称、组件标签、操作按钮一行 */}
                  <div className="cocursor-marketplace-plugin-header">
                    <div className="cocursor-marketplace-plugin-header-left">
                      <div className="cocursor-marketplace-plugin-icon">
                        {plugin.icon ? (
                          <img src={plugin.icon} alt={plugin.name} />
                        ) : (
                          <div className="cocursor-marketplace-plugin-icon-placeholder">
                            <span>{plugin.name.charAt(0)}</span>
                          </div>
                        )}
                      </div>
                      <div className="cocursor-marketplace-plugin-info">
                        <div className="cocursor-marketplace-plugin-title-row">
                          <h3 className="cocursor-marketplace-plugin-name">
                            {plugin.name}
                          </h3>
                          {plugin.installed && (
                            <span className="cocursor-marketplace-plugin-installed-badge">
                              ✓ 已安装
                            </span>
                          )}
                        </div>
                        <div className="cocursor-marketplace-plugin-meta">
                          <span className="cocursor-marketplace-plugin-author">
                            {plugin.author}
                          </span>
                          <span className="cocursor-marketplace-plugin-version">
                            v{plugin.version}
                          </span>
                          {plugin.installed && plugin.installed_version && (
                            <span className="cocursor-marketplace-plugin-installed-version">
                              (v{plugin.installed_version})
                            </span>
                          )}
                          <span className="cocursor-marketplace-plugin-category">
                            {plugin.category}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="cocursor-marketplace-plugin-header-right">
                      <div className="cocursor-marketplace-plugin-components">
                        <span className="cocursor-marketplace-plugin-component skill" title="Skill">
                          🎯
                        </span>
                        {plugin.mcp && (
                          <span className="cocursor-marketplace-plugin-component mcp" title="MCP">
                            🔌
                          </span>
                        )}
                        {plugin.command && (
                          <span className="cocursor-marketplace-plugin-component command" title="Command">
                            ⚡
                          </span>
                        )}
                      </div>
                      {plugin.installed ? (
                        <button
                          className="cocursor-marketplace-plugin-button uninstall"
                          onClick={() => handleUninstall(plugin.id)}
                          disabled={isInstalling}
                        >
                          {isInstalling ? (
                            <>
                              <span className="cocursor-marketplace-plugin-button-spinner"></span>
                              <span>卸载中...</span>
                            </>
                          ) : (
                            "卸载"
                          )}
                        </button>
                      ) : (
                        <button
                          className="cocursor-marketplace-plugin-button install"
                          onClick={() => handleInstall(plugin.id)}
                          disabled={isInstalling}
                        >
                          {isInstalling ? (
                            <>
                              <span className="cocursor-marketplace-plugin-button-spinner"></span>
                              <span>安装中...</span>
                            </>
                          ) : (
                            "安装"
                          )}
                        </button>
                      )}
                    </div>
                  </div>

                  {/* 可折叠内容区域 */}
                  <div className="cocursor-marketplace-plugin-content">
                    <div className={`cocursor-marketplace-plugin-description-preview ${isExpanded ? "expanded" : ""}`}>
                      <p>{plugin.description}</p>
                    </div>
                    
                    {usageInstructions.length > 0 && (
                      <>
                        <button
                          className="cocursor-marketplace-plugin-expand-button"
                          onClick={() => toggleExpand(plugin.id)}
                        >
                          {isExpanded ? (
                            <>
                              <span>收起详情</span>
                              <span className="cocursor-marketplace-plugin-expand-icon">▲</span>
                            </>
                          ) : (
                            <>
                              <span>查看详情</span>
                              <span className="cocursor-marketplace-plugin-expand-icon">▼</span>
                            </>
                          )}
                        </button>

                        {isExpanded && (
                          <div className="cocursor-marketplace-plugin-expanded-content">
                            <div className="cocursor-marketplace-plugin-usage-section">
                              <h4 className="cocursor-marketplace-plugin-section-title">使用说明</h4>
                              <div className="cocursor-marketplace-plugin-usage-list">
                                {usageInstructions.map((instruction, idx) => (
                                  <div key={idx} className="cocursor-marketplace-plugin-usage-item">
                                    <div className="cocursor-marketplace-plugin-usage-icon">
                                      {instruction.type === "Skill" && "🎯"}
                                      {instruction.type === "MCP" && "🔌"}
                                      {instruction.type === "Command" && "⚡"}
                                    </div>
                                    <div className="cocursor-marketplace-plugin-usage-content">
                                      <div className="cocursor-marketplace-plugin-usage-title">
                                        {instruction.title}
                                      </div>
                                      <div className="cocursor-marketplace-plugin-usage-description">
                                        {instruction.description}
                                      </div>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </div>
                          </div>
                        )}
                      </>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
};
