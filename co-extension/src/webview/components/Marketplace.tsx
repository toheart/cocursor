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
    command_id: string;
    scope: string;
  };
}

export const Marketplace: React.FC = () => {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");

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

  const handleInstall = async (pluginId: string) => {
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
        return;
      }

      // 刷新插件列表
      await loadPlugins();
    } catch (error) {
      console.error("Failed to install plugin:", error);
    }
  };

  const handleUninstall = async (pluginId: string) => {
    try {
      const workspacePath = getWorkspacePath();
      const response = await apiService.uninstallPlugin(pluginId, workspacePath) as {
        success?: boolean;
        message?: string;
        error?: string;
      };

      if (response.error) {
        console.error("Failed to uninstall plugin:", response.error);
        return;
      }

      // 刷新插件列表
      await loadPlugins();
    } catch (error) {
      console.error("Failed to uninstall plugin:", error);
    }
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

      {loading ? (
        <div className="cocursor-marketplace-loading">
          <div className="cocursor-marketplace-loading-spinner"></div>
          <p>加载中...</p>
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
              if (plugin.command) {
                usageInstructions.push({
                  type: "Command",
                  title: "Command 组件",
                  description: `此插件包含命令: /${plugin.command.command_id}。安装后，可在 Cursor 中使用此命令。`
                });
              }

              return (
                <div 
                  key={plugin.id} 
                  className="cocursor-marketplace-plugin"
                  style={{ animationDelay: `${index * 80}ms` }}
                >
                  {/* Banner 区域 - 包含图标、信息、组件标签和操作按钮 */}
                  <div className="cocursor-marketplace-plugin-banner">
                    <div className="cocursor-marketplace-plugin-banner-left">
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
                          <div className="cocursor-marketplace-plugin-components">
                            <span className="cocursor-marketplace-plugin-component skill">
                              Skill
                            </span>
                            {plugin.mcp && (
                              <span className="cocursor-marketplace-plugin-component mcp">
                                MCP
                              </span>
                            )}
                            {plugin.command && (
                              <span className="cocursor-marketplace-plugin-component command">
                                Command
                              </span>
                            )}
                          </div>
                        </div>
                        <div className="cocursor-marketplace-plugin-meta">
                          <span className="cocursor-marketplace-plugin-author">
                            {plugin.author}
                          </span>
                          <span className="cocursor-marketplace-plugin-version">
                            v{plugin.version}
                          </span>
                          {plugin.installed && plugin.installed_version && (
                            <span className="cocursor-marketplace-plugin-installed">
                              ✓ 已安装 v{plugin.installed_version}
                            </span>
                          )}
                          <span className="cocursor-marketplace-plugin-category">
                            {plugin.category}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="cocursor-marketplace-plugin-banner-right">
                      {plugin.installed ? (
                        <button
                          className="cocursor-marketplace-plugin-button uninstall"
                          onClick={() => handleUninstall(plugin.id)}
                        >
                          卸载
                        </button>
                      ) : (
                        <button
                          className="cocursor-marketplace-plugin-button install"
                          onClick={() => handleInstall(plugin.id)}
                        >
                          安装
                        </button>
                      )}
                    </div>
                  </div>

                  {/* 内容区域 - 描述和使用说明 */}
                  <div className="cocursor-marketplace-plugin-content">
                    <div className="cocursor-marketplace-plugin-description-section">
                      <h4 className="cocursor-marketplace-plugin-section-title">插件说明</h4>
                      <p className="cocursor-marketplace-plugin-description">
                        {plugin.description}
                      </p>
                    </div>

                    {usageInstructions.length > 0 && (
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
