import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import zhCN from "./locales/zh-CN.json";
import en from "./locales/en.json";

/**
 * 获取初始语言
 * 优先使用 Extension 注入的语言，然后是浏览器语言
 */
const getLanguage = (): string => {
  // 1. 优先从 extension 传递的初始语言获取（通过 window.__INITIAL_LANGUAGE__）
  try {
    const initialLang = (window as any).__INITIAL_LANGUAGE__;
    if (initialLang === "zh-CN" || initialLang === "en") {
      return initialLang;
    }
  } catch (e) {
    // 忽略错误
  }

  // 2. 尝试从 VSCode API 获取语言设置
  try {
    const vscode = (window as any).vscode;
    if (vscode) {
      const vscodeLang =
        vscode.getState?.()?.language || (window as any).__VSCODE_LANGUAGE__;
      if (vscodeLang) {
        return vscodeLang.toLowerCase().startsWith("zh") ? "zh-CN" : "en";
      }
    }
  } catch (e) {
    // 忽略错误
  }

  // 3. 使用浏览器语言作为最后的降级方案
  const browserLang = navigator.language || (navigator as any).userLanguage;
  return browserLang.toLowerCase().startsWith("zh") ? "zh-CN" : "en";
};

/**
 * 获取翻译资源
 * 优先使用 Extension 注入的资源，否则使用本地导入的
 */
const getResources = () => {
  try {
    const injectedResources = (window as any).__I18N_RESOURCES__;
    if (
      injectedResources &&
      injectedResources["zh-CN"] &&
      injectedResources["en"]
    ) {
      return {
        "zh-CN": { translation: injectedResources["zh-CN"] },
        en: { translation: injectedResources["en"] },
      };
    }
  } catch (e) {
    // 忽略错误，使用本地资源
  }

  // 降级：使用本地导入的翻译文件
  return {
    "zh-CN": { translation: zhCN },
    en: { translation: en },
  };
};

// 初始化 i18next
i18n.use(initReactI18next).init({
  resources: getResources(),
  lng: getLanguage(),
  fallbackLng: "zh-CN",
  interpolation: {
    escapeValue: false, // React 已经转义了
  },
});

// 监听来自 extension 的语言变更消息
if (typeof window !== "undefined") {
  window.addEventListener("message", (event: MessageEvent) => {
    const data = event.data;
    if (
      data &&
      data.type === "languageChanged" &&
      data.data &&
      data.data.language
    ) {
      const newLang = data.data.language;
      if (newLang === "zh-CN" || newLang === "en") {
        i18n.changeLanguage(newLang);
        // 不再写入 localStorage，统一由 Extension 管理语言状态
      }
    }
  });
}

export default i18n;
