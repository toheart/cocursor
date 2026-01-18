import * as fs from "fs";
import * as path from "path";
import * as os from "os";

/**
 * 获取 Cursor MCP 配置文件路径（根据操作系统）
 * MCP 配置存储在 ~/.cursor/mcp.json（全局配置）
 */
export function getCursorMCPConfigPath(): string {
  const platform = process.platform;
  const homeDir = os.homedir();

  switch (platform) {
    case "win32":
      // Windows: %USERPROFILE%\.cursor\mcp.json
      return path.join(homeDir, ".cursor", "mcp.json");
    
    case "darwin":
      // macOS: ~/.cursor/mcp.json
      return path.join(homeDir, ".cursor", "mcp.json");
    
    case "linux":
      // Linux: ~/.cursor/mcp.json
      return path.join(homeDir, ".cursor", "mcp.json");
    
    default:
      throw new Error(`不支持的操作系统: ${platform}`);
  }
}

/**
 * 移除 JSON 中的注释（支持 JSONC 格式）
 * 移除单行注释 (//) 和块注释
 */
function removeJsonComments(jsonString: string): string {
  let result = "";
  let inString = false;
  let escapeNext = false;
  let inSingleLineComment = false;
  let inMultiLineComment = false;
  
  for (let i = 0; i < jsonString.length; i++) {
    const char = jsonString[i];
    const nextChar = i + 1 < jsonString.length ? jsonString[i + 1] : "";
    
    if (escapeNext) {
      result += char;
      escapeNext = false;
      continue;
    }
    
    if (char === "\\" && inString) {
      result += char;
      escapeNext = true;
      continue;
    }
    
    if (char === '"' && !inSingleLineComment && !inMultiLineComment) {
      inString = !inString;
      result += char;
      continue;
    }
    
    if (inString) {
      result += char;
      continue;
    }
    
    // 检查单行注释
    if (char === "/" && nextChar === "/" && !inMultiLineComment) {
      inSingleLineComment = true;
      i++; // 跳过下一个字符
      continue;
    }
    
    if (inSingleLineComment && char === "\n") {
      inSingleLineComment = false;
      result += char;
      continue;
    }
    
    if (inSingleLineComment) {
      continue;
    }
    
    // 检查多行注释
    if (char === "/" && nextChar === "*" && !inMultiLineComment) {
      inMultiLineComment = true;
      i++; // 跳过下一个字符
      continue;
    }
    
    if (inMultiLineComment && char === "*" && nextChar === "/") {
      inMultiLineComment = false;
      i++; // 跳过下一个字符
      continue;
    }
    
    if (inMultiLineComment) {
      continue;
    }
    
    result += char;
  }
  
  return result;
}

/**
 * 读取 Cursor MCP 配置文件
 */
export function readCursorMCPConfig(): any {
  const configPath = getCursorMCPConfigPath();
  
  // 如果文件不存在，返回空对象
  if (!fs.existsSync(configPath)) {
    console.log(`Cursor MCP 配置文件不存在: ${configPath}`);
    return {};
  }

  try {
    const content = fs.readFileSync(configPath, "utf-8");
    // 移除注释（支持 JSONC 格式）
    const cleanedContent = removeJsonComments(content);
    return JSON.parse(cleanedContent);
  } catch (error) {
    console.error(`读取 Cursor MCP 配置文件失败: ${error}`);
    throw error;
  }
}

/**
 * 写入 Cursor MCP 配置文件
 */
export function writeCursorMCPConfig(config: any): void {
  const configPath = getCursorMCPConfigPath();
  const dir = path.dirname(configPath);

  // 确保目录存在
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }

  try {
    // 格式化 JSON（2 空格缩进）
    const content = JSON.stringify(config, null, 2);
    fs.writeFileSync(configPath, content, "utf-8");
    console.log(`Cursor MCP 配置文件已更新: ${configPath}`);
  } catch (error) {
    console.error(`写入 Cursor MCP 配置文件失败: ${error}`);
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
    const config = readCursorMCPConfig();

    // 初始化 mcpServers 对象（如果不存在）
    if (!config.mcpServers) {
      config.mcpServers = {};
    }

    // 检查是否已配置
    const servers = config.mcpServers as Record<string, any>;
    if (servers[serverName] && servers[serverName].url === mcpUrl) {
      console.log(`MCP 服务器 ${serverName} 已配置，跳过`);
      return false; // 已存在，无需更新
    }

    // 添加或更新 MCP 服务器配置
    servers[serverName] = {
      url: mcpUrl,
      transport: "sse"
    };

    // 写入配置文件
    writeCursorMCPConfig(config);
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
    const config = readCursorMCPConfig();

    if (!config.mcpServers) {
      return false; // 不存在，无需移除
    }

    const servers = config.mcpServers as Record<string, any>;
    if (!servers[serverName]) {
      return false; // 不存在，无需移除
    }

    delete servers[serverName];

    // 如果 servers 对象为空，删除整个 mcpServers
    if (Object.keys(servers).length === 0) {
      delete config.mcpServers;
    }

    writeCursorMCPConfig(config);
    console.log(`MCP 服务器 ${serverName} 已移除`);
    return true;
  } catch (error) {
    console.error(`移除 MCP 服务器失败: ${error}`);
    return false;
  }
}
