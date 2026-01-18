import React, { useState, useEffect } from "react";

interface Plugin {
  id: string;
  name: string;
  description: string;
  author: string;
  version: string;
  icon?: string;
  category: string;
  installed?: boolean;
  rating?: number;
  downloads?: number;
}

export const Marketplace: React.FC = () => {
  const [plugins, setPlugins] = useState<Plugin[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");

  useEffect(() => {
    // 模拟加载插件列表
    loadPlugins();
  }, []);

  const loadPlugins = async () => {
    setLoading(true);
    try {
      // TODO: 从后端API加载插件列表
      // 目前使用模拟数据
      await new Promise((resolve) => setTimeout(resolve, 500));
      const mockPlugins: Plugin[] = [
        {
          id: "1",
          name: "代码格式化工具",
          description: "自动格式化代码，支持多种编程语言",
          author: "CoCursor Team",
          version: "1.0.0",
          category: "工具",
          rating: 4.5,
          downloads: 1234,
          installed: false
        },
        {
          id: "2",
          name: "Git 集成",
          description: "增强的 Git 功能，支持可视化提交历史",
          author: "CoCursor Team",
          version: "2.1.0",
          category: "集成",
          rating: 4.8,
          downloads: 5678,
          installed: true
        },
        {
          id: "3",
          name: "代码审查助手",
          description: "AI 驱动的代码审查建议",
          author: "CoCursor Team",
          version: "1.5.0",
          category: "AI",
          rating: 4.7,
          downloads: 3456,
          installed: false
        }
      ];
      setPlugins(mockPlugins);
    } catch (error) {
      console.error("加载插件列表失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleInstall = async (pluginId: string) => {
    try {
      // TODO: 调用后端API安装插件
      setPlugins((prev) =>
        prev.map((plugin) =>
          plugin.id === pluginId ? { ...plugin, installed: true } : plugin
        )
      );
    } catch (error) {
      console.error("安装插件失败:", error);
    }
  };

  const handleUninstall = async (pluginId: string) => {
    try {
      // TODO: 调用后端API卸载插件
      setPlugins((prev) =>
        prev.map((plugin) =>
          plugin.id === pluginId ? { ...plugin, installed: false } : plugin
        )
      );
    } catch (error) {
      console.error("卸载插件失败:", error);
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
      <div className="cocursor-marketplace-header">
        <div className="cocursor-marketplace-search">
          <input
            type="text"
            placeholder="搜索插件..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="cocursor-marketplace-search-input"
          />
        </div>
        <div className="cocursor-marketplace-categories">
          {categories.map((category) => (
            <button
              key={category}
              className={`cocursor-marketplace-category ${
                selectedCategory === category ? "active" : ""
              }`}
              onClick={() => setSelectedCategory(category)}
            >
              {category === "all" ? "全部" : category}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="cocursor-loading">加载中...</div>
      ) : (
        <div className="cocursor-marketplace-plugins">
          {filteredPlugins.length === 0 ? (
            <div className="cocursor-empty">暂无插件</div>
          ) : (
            filteredPlugins.map((plugin) => (
              <div key={plugin.id} className="cocursor-marketplace-plugin">
                <div className="cocursor-marketplace-plugin-header">
                  <div className="cocursor-marketplace-plugin-icon">
                    {plugin.icon ? (
                      <img src={plugin.icon} alt={plugin.name} />
                    ) : (
                      <div className="cocursor-marketplace-plugin-icon-placeholder">
                        {plugin.name.charAt(0)}
                      </div>
                    )}
                  </div>
                  <div className="cocursor-marketplace-plugin-info">
                    <h3 className="cocursor-marketplace-plugin-name">
                      {plugin.name}
                    </h3>
                    <div className="cocursor-marketplace-plugin-meta">
                      <span className="cocursor-marketplace-plugin-author">
                        {plugin.author}
                      </span>
                      <span className="cocursor-marketplace-plugin-version">
                        v{plugin.version}
                      </span>
                      {plugin.rating && (
                        <span className="cocursor-marketplace-plugin-rating">
                          ⭐ {plugin.rating}
                        </span>
                      )}
                      {plugin.downloads && (
                        <span className="cocursor-marketplace-plugin-downloads">
                          📥 {plugin.downloads}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <p className="cocursor-marketplace-plugin-description">
                  {plugin.description}
                </p>
                <div className="cocursor-marketplace-plugin-footer">
                  <span className="cocursor-marketplace-plugin-category">
                    {plugin.category}
                  </span>
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
            ))
          )}
        </div>
      )}
    </div>
  );
};
