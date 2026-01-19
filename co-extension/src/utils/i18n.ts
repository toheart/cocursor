import * as vscode from "vscode";
import * as fs from "fs";
import * as path from "path";

interface Translations {
  [key: string]: string | Translations;
}

let translations: Translations = {};
let currentLanguage = "zh-CN";
let extensionPath: string | undefined;

// 初始化 i18n（需要在扩展激活时调用，传入 extensionPath）
export function initI18n(context: vscode.ExtensionContext): void {
  extensionPath = context.extensionPath;
  
  // 优先从 globalState 读取保存的语言设置
  const savedLanguage = context.globalState.get<string>('cocursor-language');
  if (savedLanguage === 'zh-CN' || savedLanguage === 'en') {
    currentLanguage = savedLanguage;
  } else {
    // 如果没有保存的设置，获取 VSCode 语言设置
    const vscodeLanguage = vscode.env.language;
    currentLanguage = vscodeLanguage.toLowerCase().startsWith("zh") ? "zh-CN" : "en";
  }
  
  translations = loadTranslations(currentLanguage);
}

// 加载翻译文件
function loadTranslations(language: string): Translations {
  if (!extensionPath) {
    return {};
  }

  // 尝试多个可能的路径（开发环境和生产环境）
  const possiblePaths = [
    path.join(extensionPath, "src", "webview", "i18n", "locales", `${language}.json`), // 开发环境
    path.join(extensionPath, "dist", "webview", "i18n", "locales", `${language}.json`), // 生产环境（如果文件被复制）
    path.join(extensionPath, "webview", "i18n", "locales", `${language}.json`), // 备用路径
  ];
  
  for (const localePath of possiblePaths) {
    try {
      if (fs.existsSync(localePath)) {
        const content = fs.readFileSync(localePath, "utf-8");
        return JSON.parse(content);
      }
    } catch (error) {
      // 继续尝试下一个路径
      continue;
    }
  }
  
  // 如果加载失败，尝试加载英文作为后备
  if (language !== "en") {
    return loadTranslations("en");
  }
  return {};
}

// 获取翻译
export function t(key: string, params?: Record<string, string | number>): string {
  const keys = key.split(".");
  let value: any = translations;
  
  for (const k of keys) {
    if (value && typeof value === "object" && k in value) {
      value = value[k];
    } else {
      // 如果找不到翻译，返回 key
      return key;
    }
  }
  
  if (typeof value !== "string") {
    return key;
  }
  
  // 替换参数
  if (params) {
    return value.replace(/\{\{(\w+)\}\}/g, (match: string, paramKey: string) => {
      return params[paramKey]?.toString() || match;
    });
  }
  
  return value;
}

// 获取当前语言
export function getCurrentLanguage(): string {
  return currentLanguage;
}

// 切换语言（用于响应语言变更）
export function changeLanguage(language: string): void {
  if (language !== "zh-CN" && language !== "en") {
    console.warn(`Invalid language: ${language}`);
    return;
  }
  
  currentLanguage = language;
  translations = loadTranslations(language);
}
