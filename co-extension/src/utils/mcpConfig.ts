import * as fs from "fs";
import * as path from "path";
import * as os from "os";

/**
 * 获取 Cursor 配置文件路径（根据操作系统）
 */
export function getCursorSettingsPath(): string {
  const platform = process.platform;
  const homeDir = os.homedir();

  switch (platform) {
    case "win32":
      // Windows: %APPDATA%\Cursor\User\settings.json
      const appData = process.env.APPDATA || path.join(homeDir, "AppData", "Roaming");
      return path.join(appData, "Cursor", "User", "settings.json");
    
    case "darwin":
      // macOS: ~/Library/Application Support/Cursor/User/settings.json
      return path.join(
        homeDir,
        "Library",
        "Application Support",
        "Cursor",
        "User",
        "settings.json"
      );
    
    case "linux":
      // Linux: ~/.config/Cursor/User/settings.json
      return path.join(homeDir, ".config", "Cursor", "User", "settings.json");
    
    default:
      throw new Error(`不支持的操作系统: ${platform}`);
  }
}

/**
 * 读取 Cursor 设置文件
 */
export function readCursorSettings(): any {
  const settingsPath = getCursorSettingsPath();
  
  // 如果文件不存在，返回空对象
  if (!fs.existsSync(settingsPath)) {
    console.log(`Cursor 设置文件不存在: ${settingsPath}`);
    return {};
  }

  try {
    const content = fs.readFileSync(settingsPath, "utf-8");
    return JSON.parse(content);
  } catch (error) {
    console.error(`读取 Cursor 设置文件失败: ${error}`);
    throw error;
  }
}

/**
 * 写入 Cursor 设置文件
 */
export function writeCursorSettings(settings: any): void {
  const settingsPath = getCursorSettingsPath();
  const dir = path.dirname(settingsPath);

  // 确保目录存在
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }

  try {
    // 格式化 JSON（2 空格缩进）
    const content = JSON.stringify(settings, null, 2);
    fs.writeFileSync(settingsPath, content, "utf-8");
    console.log(`Cursor 设置文件已更新: ${settingsPath}`);
  } catch (error) {
    console.error(`写入 Cursor 设置文件失败: ${error}`);
    throw error;
  }
}

/**
 * 配置 MCP 服务器
 * @param mcpUrl MCP 服务器 URL（默认: http://localhost:19960/mcp/sse）
 * @param serverName MCP 服务器名称（默认: cocursor）
 */
export function configureMCPServer(
  mcpUrl: string = "http://localhost:19960/mcp/sse",
  serverName: string = "cocursor"
): boolean {
  try {
    const settings = readCursorSettings();

    // 初始化 mcp.servers 对象（如果不存在）
    if (!settings["mcp.servers"]) {
      settings["mcp.servers"] = {};
    }

    // 检查是否已配置
    const servers = settings["mcp.servers"] as Record<string, any>;
    if (servers[serverName] && servers[serverName].url === mcpUrl) {
      console.log(`MCP 服务器 ${serverName} 已配置，跳过`);
      return false; // 已存在，无需更新
    }

    // 添加或更新 MCP 服务器配置
    servers[serverName] = {
      url: mcpUrl,
      transport: "sse"
    };

    // 写入设置文件
    writeCursorSettings(settings);
    console.log(`MCP 服务器 ${serverName} 配置成功: ${mcpUrl}`);
    return true; // 已更新
  } catch (error) {
    console.error(`配置 MCP 服务器失败: ${error}`);
    return false;
  }
}

/**
 * 移除 MCP 服务器配置
 */
export function removeMCPServer(serverName: string = "cocursor"): boolean {
  try {
    const settings = readCursorSettings();

    if (!settings["mcp.servers"]) {
      return false; // 不存在，无需移除
    }

    const servers = settings["mcp.servers"] as Record<string, any>;
    if (!servers[serverName]) {
      return false; // 不存在，无需移除
    }

    delete servers[serverName];

    // 如果 servers 对象为空，删除整个 mcp.servers
    if (Object.keys(servers).length === 0) {
      delete settings["mcp.servers"];
    }

    writeCursorSettings(settings);
    console.log(`MCP 服务器 ${serverName} 已移除`);
    return true;
  } catch (error) {
    console.error(`移除 MCP 服务器失败: ${error}`);
    return false;
  }
}
