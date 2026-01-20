/**
 * 技能市场组件（重构版）
 */

import React, { useState, useCallback, useMemo, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../services/api";
import { 
  Plugin, 
  PageResponse, 
  UsageInstruction 
} from "../types";
import { useApi, useDebounce, useToast } from "../hooks";
import { 
  generateUsageInstructions, 
  getComponentIcon 
} from "../utils/pluginUtils";

// ========== 常量定义 ==========

const CATEGORIES = ["all", "productivity", "creative", "design", "tools", "other"] as const;
const SOURCES = ["all", "builtin", "team_global"] as const;
const DEBOUNCE_DELAY = 300;
const SKELETON_COUNT = 6;

interface GetPluginsParams {
  category?: string;
  search?: string;
  installed?: boolean;
  source?: string;
  team_id?: string;
}

interface PluginInstallResponse {
  success?: boolean;
  message?: string;
  env_vars?: string[];
  error?: string;
  // 冲突相关
  conflict?: boolean;
  conflict_type?: "other_plugin" | "manual_install" | "unknown";
  skill_name?: string;
}

// ========== 主组件 ==========

export const Marketplace: React.FC = () => {
  const { t, i18n } = useTranslation();
  // ========== 状态 ==========
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("all");
  const [selectedSource, setSelectedSource] = useState<string>("all");

  // ========== 自定义 Hooks ==========
  const { showToast, toasts } = useToast();
  const debouncedSearchQuery = useDebounce(searchQuery, DEBOUNCE_DELAY);

  // ========== API 请求 ==========
  const fetchPlugins = useCallback(async () => {
    try {
      const lang = i18n.language; // 获取当前语言
      const params: GetPluginsParams = {
        category: selectedCategory !== "all" ? selectedCategory : undefined,
        search: debouncedSearchQuery || undefined,
        source: selectedSource !== "all" ? selectedSource : undefined,
        lang: lang || "zh-CN", // 传递语言参数，默认中文
      };
      console.log("Marketplace: 获取技能列表", params);
      
      // 后端返回格式: { plugins: Plugin[], total: number }
      const response = await apiService.getPlugins(
        params.category,
        params.search,
        undefined,
        params.lang,
        params.source,
        params.team_id
      ) as { plugins?: Plugin[]; total?: number };
      
      console.log("Marketplace: API 响应", response);
      
      // 处理响应格式
      if (!response) {
        console.warn("Marketplace: API 返回空响应");
        return { plugins: [], total: 0 };
      }
      
      // 如果响应有 plugins 字段
      if (response.plugins && Array.isArray(response.plugins)) {
        return { plugins: response.plugins, total: response.total || response.plugins.length };
      }
      
      // 如果响应直接是数组（向后兼容）
      if (Array.isArray(response)) {
        console.warn("Marketplace: API 返回数组，转换为对象格式");
        return { plugins: response, total: response.length };
      }
      
      console.warn("Marketplace: 无法解析技能数据", response);
      return { plugins: [], total: 0 };
    } catch (error) {
      console.error("Marketplace: 获取技能列表失败", error);
      throw error;
    }
  }, [selectedCategory, debouncedSearchQuery, selectedSource, i18n.language]);

  const {
    data: pluginsResponse,
    loading,
    error,
    refetch: loadPlugins,
  } = useApi<{ plugins: Plugin[]; total: number }>(fetchPlugins);

  const plugins = useMemo(() => {
    if (!pluginsResponse) {
      return [];
    }
    
    // 后端返回格式: { plugins: Plugin[], total: number }
    if (pluginsResponse.plugins && Array.isArray(pluginsResponse.plugins)) {
      return pluginsResponse.plugins;
    }
    
    // 向后兼容：如果直接是数组
    if (Array.isArray(pluginsResponse)) {
      return pluginsResponse;
    }
    
    console.warn("Marketplace: 无法解析技能数据", pluginsResponse);
    return [];
  }, [pluginsResponse]);

  // ========== 事件处理 ==========
  
  // 执行安装（支持强制覆盖）
  const doInstall = useCallback(async (pluginId: string, force: boolean = false) => {
    const workspacePath = (window as any).__WORKSPACE_PATH__ || "";
    
    const response = await apiService.installPlugin(
      pluginId, 
      workspacePath,
      force
    ) as PluginInstallResponse;

    return response;
  }, []);

  const handleInstall = useCallback(async (pluginId: string) => {
    try {
      const response = await doInstall(pluginId, false);

      // 处理冲突情况
      if (response.conflict) {
        if (response.conflict_type === "manual_install") {
          // 手动安装的冲突，通过 VSCode API 询问用户是否覆盖
          const confirmOverwrite = await apiService.showConfirmDialog(
            t("marketplace.conflictManualInstall", { skillName: response.skill_name }),
            t("common.confirm"),
            t("common.cancel")
          );
          
          if (confirmOverwrite) {
            // 用户确认覆盖，强制安装
            const forceResponse = await doInstall(pluginId, true);
            if (forceResponse.error) {
              console.error("Failed to install plugin:", forceResponse.error);
              showToast(`${t("marketplace.installFailed")}: ${forceResponse.error}`, "error");
              return;
            }
            showToast(t("marketplace.installSuccess"), "success");
            await loadPlugins();
          }
          // 用户取消，不做任何操作
          return;
        } else {
          // 其他类型的冲突（被其他插件占用等），显示错误
          showToast(`${t("marketplace.installFailed")}: ${response.message || t("marketplace.conflictOtherPlugin")}`, "error");
          return;
        }
      }

      if (response.error) {
        console.error("Failed to install plugin:", response.error);
        showToast(`${t("marketplace.installFailed")}: ${response.error}`, "error");
        return;
      }

      showToast(t("marketplace.installSuccess"), "success");
      await loadPlugins();
    } catch (error) {
      console.error("Failed to install plugin:", error);
      showToast(t("marketplace.installFailedRetry"), "error");
    }
  }, [showToast, loadPlugins, doInstall, t]);

  const handleUninstall = useCallback(async (pluginId: string) => {
    const workspacePath = (window as any).__WORKSPACE_PATH__ || "";
    
    try {
      const response = await apiService.uninstallPlugin(
        pluginId, 
        workspacePath
      ) as PluginInstallResponse;

      if (response.error) {
        console.error("Failed to uninstall plugin:", response.error);
        showToast(`${t("marketplace.uninstallFailed")}: ${response.error}`, "error");
        return;
      }

      showToast(t("marketplace.uninstallSuccess"), "success");
      await loadPlugins();
    } catch (error) {
      console.error("Failed to uninstall plugin:", error);
      showToast(t("marketplace.uninstallFailedRetry"), "error");
    }
  }, [showToast, loadPlugins]);

  // ========== 计算 ==========
  const filteredPlugins = useMemo(() => {
    return plugins.filter((plugin) => {
      const matchesSearch =
        plugin.name.toLowerCase().includes(debouncedSearchQuery.toLowerCase()) ||
        plugin.description.toLowerCase().includes(debouncedSearchQuery.toLowerCase());
      const matchesCategory =
        selectedCategory === "all" || plugin.category === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  }, [plugins, debouncedSearchQuery, selectedCategory]);

  // ========== 渲染 ==========
  return (
    <div className="cocursor-marketplace">
      <MarketplaceHeader
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        selectedCategory={selectedCategory}
        onCategoryChange={setSelectedCategory}
        selectedSource={selectedSource}
        onSourceChange={setSelectedSource}
      />

      <ToastContainer toasts={toasts} />

      {error && (
        <div className="cocursor-marketplace-error-alert">
          <strong>{t("marketplace.loadFailed")}</strong>
          {typeof error === 'string' ? error : String(error)}
          <button
            onClick={() => loadPlugins()}
            className="cocursor-marketplace-retry-button"
          >
            {t("common.retry")}
          </button>
        </div>
      )}

      {loading ? (
        <SkeletonContainer count={SKELETON_COUNT} />
      ) : (
        <PluginList
          plugins={filteredPlugins}
          onInstall={handleInstall}
          onUninstall={handleUninstall}
        />
      )}
    </div>
  );
};

// ========== 子组件 ==========

interface MarketplaceHeaderProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  selectedCategory: string;
  onCategoryChange: (category: string) => void;
  selectedSource: string;
  onSourceChange: (source: string) => void;
}

const MarketplaceHeader: React.FC<MarketplaceHeaderProps> = ({
  searchQuery,
  onSearchChange,
  selectedCategory,
  onCategoryChange,
  selectedSource,
  onSourceChange,
}) => {
  const { t } = useTranslation();
  return (
    <>
      <div className="cocursor-marketplace-hero">
        <h1 className="cocursor-marketplace-title">{t("marketplace.title")}</h1>
        <p className="cocursor-marketplace-subtitle">{t("marketplace.subtitle")}</p>
      </div>
      
      <div className="cocursor-marketplace-header">
        <div className="cocursor-marketplace-search-wrapper">
          <div className="cocursor-marketplace-search-icon">🔍</div>
          <input
            type="text"
            placeholder={t("marketplace.searchPlaceholder")}
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="cocursor-marketplace-search-input"
          />
        </div>
        
      <div className="cocursor-marketplace-filters">
          <div className="cocursor-marketplace-categories">
            {CATEGORIES.map((category) => (
              <button
                key={category}
                className={`cocursor-marketplace-category ${
                  selectedCategory === category ? "active" : ""
                }`}
                onClick={() => onCategoryChange(category)}
              >
                {category === "all" ? t("marketplace.categories.all") : t(`marketplace.categories.${category}`)}
              </button>
            ))}
          </div>
          
          <div className="cocursor-marketplace-sources">
            {SOURCES.map((source) => (
              <button
                key={source}
                className={`cocursor-marketplace-source ${
                  selectedSource === source ? "active" : ""
                }`}
                onClick={() => onSourceChange(source)}
              >
                {t(`marketplace.sources.${source}`)}
              </button>
            ))}
          </div>
        </div>
      </div>
    </>
  );
};

interface ToastContainerProps {
  toasts: Array<{
    id: string;
    message: string;
    type: "success" | "error";
  }>;
}

const ToastContainer: React.FC<ToastContainerProps> = ({ toasts }) => {
  return (
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
  );
};

interface SkeletonContainerProps {
  count: number;
}

const SkeletonContainer: React.FC<SkeletonContainerProps> = ({ count }) => {
  return (
    <div className="cocursor-marketplace-plugins">
      {Array.from({ length: count }).map((_, i) => (
        <div 
          key={i} 
          className="cocursor-marketplace-plugin-skeleton"
          style={{ animationDelay: `${i * 100}ms` }}
        >
          <div className="cocursor-marketplace-plugin-skeleton-header">
            <div className="cocursor-marketplace-plugin-skeleton-icon"></div>
            <div className="cocursor-marketplace-plugin-skeleton-info">
              <div className="cocursor-marketplace-plugin-skeleton-title"></div>
              <div className="cocursor-marketplace-plugin-skeleton-meta"></div>
            </div>
            <div className="cocursor-marketplace-plugin-skeleton-button"></div>
          </div>
          <div className="cocursor-marketplace-plugin-skeleton-content">
            <div className="cocursor-marketplace-plugin-skeleton-desc"></div>
            <div className="cocursor-marketplace-plugin-skeleton-desc"></div>
            <div className="cocursor-marketplace-plugin-skeleton-desc"></div>
          </div>
        </div>
      ))}
    </div>
  );
};

interface PluginListProps {
  plugins: Plugin[];
  onInstall: (pluginId: string) => Promise<void>;
  onUninstall: (pluginId: string) => Promise<void>;
}

const PluginList: React.FC<PluginListProps> = ({ 
  plugins, 
  onInstall, 
  onUninstall 
}) => {
  const [installingPlugins, setInstallingPlugins] = useState<Set<string>>(new Set());
  const [expandedPlugins, setExpandedPlugins] = useState<Set<string>>(new Set());

  const handleInstallClick = useCallback(async (pluginId: string) => {
    if (installingPlugins.has(pluginId)) return;
    
    setInstallingPlugins(prev => new Set(prev).add(pluginId));
    await onInstall(pluginId);
    setInstallingPlugins(prev => {
      const next = new Set(prev);
      next.delete(pluginId);
      return next;
    });
  }, [installingPlugins, onInstall]);

  const handleUninstallClick = useCallback(async (pluginId: string) => {
    if (installingPlugins.has(pluginId)) return;
    
    setInstallingPlugins(prev => new Set(prev).add(pluginId));
    await onUninstall(pluginId);
    setInstallingPlugins(prev => {
      const next = new Set(prev);
      next.delete(pluginId);
      return next;
    });
  }, [installingPlugins, onUninstall]);

  const handleToggleExpand = useCallback((pluginId: string) => {
    setExpandedPlugins(prev => {
      const next = new Set(prev);
      if (next.has(pluginId)) {
        next.delete(pluginId);
      } else {
        next.add(pluginId);
      }
      return next;
    });
  }, []);

  const { t } = useTranslation();
  return (
    <div className="cocursor-marketplace-plugins">
      {plugins.length === 0 ? (
        <div className="cocursor-marketplace-empty">
          <div className="cocursor-marketplace-empty-icon">📦</div>
          <p>{t("marketplace.noPlugins")}</p>
          <span>{t("marketplace.noPluginsDesc")}</span>
        </div>
      ) : (
        plugins.map((plugin, index) => (
          <PluginCard
            key={plugin.id}
            plugin={plugin}
            index={index}
            isExpanded={expandedPlugins.has(plugin.id)}
            isInstalling={installingPlugins.has(plugin.id)}
            onInstall={handleInstallClick}
            onUninstall={handleUninstallClick}
            onToggleExpand={handleToggleExpand}
          />
        ))
      )}
    </div>
  );
};

interface PluginCardProps {
  plugin: Plugin;
  index: number;
  isExpanded: boolean;
  isInstalling: boolean;
  onInstall: (pluginId: string) => void;
  onUninstall: (pluginId: string) => void;
  onToggleExpand: (pluginId: string) => void;
}

const PluginCard: React.FC<PluginCardProps> = ({
  plugin,
  index,
  isExpanded,
  isInstalling,
  onInstall,
  onUninstall,
  onToggleExpand,
}) => {
  const { t } = useTranslation();
  const usageInstructions: UsageInstruction[] = useMemo(
    () => generateUsageInstructions(plugin, t),
    [plugin, t]
  );

  return (
    <div 
      className={`cocursor-marketplace-plugin ${plugin.installed ? "installed" : ""}`}
      style={{ animationDelay: `${index * 60}ms` }}
      data-category={plugin.category}
    >
      <PluginCardHeader
        plugin={plugin}
        isInstalling={isInstalling}
        onInstall={onInstall}
        onUninstall={onUninstall}
      />

      <PluginCardContent
        plugin={plugin}
        isExpanded={isExpanded}
        usageInstructions={usageInstructions}
        onToggleExpand={onToggleExpand}
      />
    </div>
  );
};

interface PluginCardHeaderProps {
  plugin: Plugin;
  isInstalling: boolean;
  onInstall: (pluginId: string) => void;
  onUninstall: (pluginId: string) => void;
}

const PluginCardHeader: React.FC<PluginCardHeaderProps> = ({
  plugin,
  isInstalling,
  onInstall,
  onUninstall,
}) => {
  const { t } = useTranslation();
  return (
    <div className="cocursor-marketplace-plugin-header">
      <div className="cocursor-marketplace-plugin-header-left">
        <PluginIcon plugin={plugin} />
        
        <div className="cocursor-marketplace-plugin-info">
          <div className="cocursor-marketplace-plugin-title-row">
            <h3 className="cocursor-marketplace-plugin-name">
              {plugin.name}
            </h3>
            {plugin.installed && (
              <span className="cocursor-marketplace-plugin-installed-badge">
                ✓ {t("marketplace.installed")}
              </span>
            )}
          </div>
          
          <PluginMeta plugin={plugin} />
        </div>
      </div>

      <PluginActionSection
        plugin={plugin}
        isInstalling={isInstalling}
        onInstall={onInstall}
        onUninstall={onUninstall}
      />
    </div>
  );
};

interface PluginIconProps {
  plugin: Plugin;
}

const PluginIcon: React.FC<PluginIconProps> = ({ plugin }) => {
  return (
    <div className="cocursor-marketplace-plugin-icon">
      {plugin.icon ? (
        <img src={plugin.icon} alt={plugin.name} />
      ) : (
        <div className="cocursor-marketplace-plugin-icon-placeholder">
          <span>{plugin.name.charAt(0)}</span>
        </div>
      )}
    </div>
  );
};

interface PluginMetaProps {
  plugin: Plugin;
}

const PluginMeta: React.FC<PluginMetaProps> = ({ plugin }) => {
  const { t } = useTranslation();
  
  // 获取来源标签
  const getSourceLabel = () => {
    if (!plugin.source || plugin.source === "builtin") {
      return null;
    }
    if (plugin.source === "team_global" || plugin.source === "team_project") {
      return (
        <span className={`cocursor-marketplace-plugin-source source-team ${plugin.author_online ? "online" : "offline"}`}>
          {plugin.team_name || t("marketplace.sources.team_global")}
          {plugin.author_name && ` · ${plugin.author_name}`}
        </span>
      );
    }
    return (
      <span className={`cocursor-marketplace-plugin-source source-${plugin.source}`}>
        {t(`marketplace.sources.${plugin.source}`)}
      </span>
    );
  };
  
  return (
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
        {plugin.category_display || plugin.category}
      </span>
      {getSourceLabel()}
    </div>
  );
};

interface PluginActionSectionProps {
  plugin: Plugin;
  isInstalling: boolean;
  onInstall: (pluginId: string) => void;
  onUninstall: (pluginId: string) => void;
}

const PluginActionSection: React.FC<PluginActionSectionProps> = ({
  plugin,
  isInstalling,
  onInstall,
  onUninstall,
}) => {
  const { t } = useTranslation();
  return (
    <div className="cocursor-marketplace-plugin-header-right">
      <PluginComponents plugin={plugin} />
      
      {plugin.installed ? (
        <button
          className="cocursor-marketplace-plugin-button uninstall"
          onClick={() => onUninstall(plugin.id)}
          disabled={isInstalling}
        >
          {isInstalling ? (
            <>
              <span className="cocursor-marketplace-plugin-button-spinner"></span>
              <span>{t("marketplace.uninstalling")}</span>
            </>
          ) : (
            t("marketplace.uninstall")
          )}
        </button>
      ) : (
        <button
          className="cocursor-marketplace-plugin-button install"
          onClick={() => onInstall(plugin.id)}
          disabled={isInstalling}
        >
          {isInstalling ? (
            <>
              <span className="cocursor-marketplace-plugin-button-spinner"></span>
              <span>{t("marketplace.installing")}</span>
            </>
          ) : (
            t("marketplace.install")
          )}
        </button>
      )}
    </div>
  );
};

interface PluginComponentsProps {
  plugin: Plugin;
}

const PluginComponents: React.FC<PluginComponentsProps> = ({ plugin }) => {
  const { t } = useTranslation();
  return (
    <div className="cocursor-marketplace-plugin-components">
      <span className="cocursor-marketplace-plugin-component skill" title={t("marketplace.componentTitles.skill")}>
        {getComponentIcon("Skill")}
      </span>
      {plugin.mcp && (
        <span className="cocursor-marketplace-plugin-component mcp" title={t("marketplace.componentTitles.mcp")}>
          {getComponentIcon("MCP")}
        </span>
      )}
      {plugin.command && (
        <span className="cocursor-marketplace-plugin-component command" title={t("marketplace.componentTitles.command")}>
          {getComponentIcon("Command")}
        </span>
      )}
    </div>
  );
};

interface PluginCardContentProps {
  plugin: Plugin;
  isExpanded: boolean;
  usageInstructions: UsageInstruction[];
  onToggleExpand: (pluginId: string) => void;
}

const PluginCardContent: React.FC<PluginCardContentProps> = ({
  plugin,
  isExpanded,
  usageInstructions,
  onToggleExpand,
}) => {
  const { t } = useTranslation();
  return (
    <div className="cocursor-marketplace-plugin-content">
      <div className={`cocursor-marketplace-plugin-description-preview ${isExpanded ? "expanded" : ""}`}>
        <p>{plugin.description}</p>
      </div>
      
      {usageInstructions.length > 0 && (
        <>
          <button
            className="cocursor-marketplace-plugin-expand-button"
            onClick={() => onToggleExpand(plugin.id)}
          >
            {isExpanded ? (
              <>
                <span>{t("marketplace.collapseDetails")}</span>
                <span className="cocursor-marketplace-plugin-expand-icon">▲</span>
              </>
            ) : (
              <>
                <span>{t("marketplace.viewDetails")}</span>
                <span className="cocursor-marketplace-plugin-expand-icon">▼</span>
              </>
            )}
          </button>

          {isExpanded && (
            <div className="cocursor-marketplace-plugin-expanded-content">
              <div className="cocursor-marketplace-plugin-usage-section">
                <h4 className="cocursor-marketplace-plugin-section-title">{t("marketplace.usageInstructions")}</h4>
                <div className="cocursor-marketplace-plugin-usage-list">
                  {usageInstructions.map((instruction, idx) => (
                    <UsageItem key={idx} instruction={instruction} />
                  ))}
                </div>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};

interface UsageItemProps {
  instruction: UsageInstruction;
}

const UsageItem: React.FC<UsageItemProps> = ({ instruction }) => {
  return (
    <div className="cocursor-marketplace-plugin-usage-item">
      <div className="cocursor-marketplace-plugin-usage-icon">
        {getComponentIcon(instruction.type)}
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
  );
};
