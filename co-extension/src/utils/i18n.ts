import * as vscode from "vscode";
import i18next from "i18next";
import zhCN from "../webview/i18n/locales/zh-CN.json";
import en from "../webview/i18n/locales/en.json";

// 支持的语言类型
type SupportedLanguage = "zh-CN" | "en";

// 语言变化监听器类型
type LanguageChangeListener = (lang: SupportedLanguage) => void;

// 存储 Extension context 引用
let extensionContext: vscode.ExtensionContext | null = null;

// 语言变化事件监听器集合
const languageChangeListeners: Set<LanguageChangeListener> = new Set();

/**
 * 初始化 i18n（需要在扩展激活时调用）
 * @param context VSCode Extension Context
 */
export function initI18n(context: vscode.ExtensionContext): void {
  extensionContext = context;

  // 优先从 globalState 读取保存的语言设置
  const savedLanguage = context.globalState.get<string>("cocursor-language");
  let initialLang: SupportedLanguage;

  if (savedLanguage === "zh-CN" || savedLanguage === "en") {
    initialLang = savedLanguage;
  } else {
    // 如果没有保存的设置，使用 VSCode 语言设置
    const vscodeLanguage = vscode.env.language;
    initialLang = vscodeLanguage.toLowerCase().startsWith("zh")
      ? "zh-CN"
      : "en";
  }

  // 使用 i18next 初始化
  i18next.init({
    lng: initialLang,
    fallbackLng: "zh-CN",
    resources: {
      "zh-CN": { translation: zhCN },
      en: { translation: en },
    },
    interpolation: {
      escapeValue: false, // 不转义 HTML
    },
  });
}

/**
 * 获取翻译文本
 * @param key 翻译 key，支持嵌套 key（如 "panel.workAnalysis"）
 * @param params 插值参数
 * @returns 翻译后的文本，如果找不到返回 key 本身
 */
export function t(
  key: string,
  params?: Record<string, string | number>,
): string {
  return i18next.t(key, params);
}

/**
 * 获取当前语言
 * @returns 当前语言代码
 */
export function getCurrentLanguage(): SupportedLanguage {
  const lang = i18next.language;
  return lang === "zh-CN" || lang === "en" ? lang : "zh-CN";
}

/**
 * 切换语言
 * @param language 目标语言
 */
export async function changeLanguage(language: string): Promise<void> {
  if (language !== "zh-CN" && language !== "en") {
    console.warn(`Invalid language: ${language}`);
    return;
  }

  const typedLang = language as SupportedLanguage;

  // 更新 i18next 语言
  await i18next.changeLanguage(typedLang);

  // 持久化到 globalState
  if (extensionContext) {
    await extensionContext.globalState.update("cocursor-language", typedLang);
  }

  // 通知所有监听者
  languageChangeListeners.forEach((listener) => {
    try {
      listener(typedLang);
    } catch (error) {
      console.error("Error in language change listener:", error);
    }
  });
}

/**
 * 注册语言变化监听器
 * @param listener 监听器函数
 * @returns 取消监听的函数
 */
export function onLanguageChange(listener: LanguageChangeListener): () => void {
  languageChangeListeners.add(listener);
  return () => {
    languageChangeListeners.delete(listener);
  };
}

/**
 * 获取翻译资源（供 Webview 使用）
 * @returns 包含所有语言翻译资源的对象
 */
export function getTranslationResources(): Record<string, object> {
  return {
    "zh-CN": zhCN,
    en: en,
  };
}
