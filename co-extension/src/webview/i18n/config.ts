import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from './locales/zh-CN.json';
import en from './locales/en.json';

// 检测语言：优先使用 VSCode 的语言设置，否则使用浏览器语言
const getLanguage = (): string => {
  // 尝试从 VSCode API 获取语言设置
  try {
    const vscode = (window as any).vscode;
    if (vscode) {
      // VSCode 语言代码通常是 'zh-cn', 'en' 等
      const vscodeLang = vscode.getState?.()?.language || 
                        (window as any).__VSCODE_LANGUAGE__;
      if (vscodeLang) {
        return vscodeLang.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';
      }
    }
  } catch (e) {
    // 忽略错误
  }
  
  // 使用浏览器语言
  const browserLang = navigator.language || (navigator as any).userLanguage;
  return browserLang.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';
};

i18n
  .use(initReactI18next)
  .init({
    resources: {
      'zh-CN': {
        translation: zhCN,
      },
      en: {
        translation: en,
      },
    },
    lng: getLanguage(),
    fallbackLng: 'zh-CN',
    interpolation: {
      escapeValue: false, // React 已经转义了
    },
  });

export default i18n;
