import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from './locales/zh-CN.json';
import en from './locales/en.json';

// 检测语言：优先使用 extension 传递的语言，然后是 localStorage，最后使用浏览器语言
const getLanguage = (): string => {
  // 1. 优先从 extension 传递的初始语言获取（通过 window.__INITIAL_LANGUAGE__）
  try {
    const initialLang = (window as any).__INITIAL_LANGUAGE__;
    if (initialLang === 'zh-CN' || initialLang === 'en') {
      return initialLang;
    }
  } catch (e) {
    // 忽略错误
  }

  // 2. 从 localStorage 读取用户保存的语言偏好（向后兼容）
  try {
    const savedLang = localStorage.getItem('cocursor-language');
    if (savedLang === 'zh-CN' || savedLang === 'en') {
      return savedLang;
    }
  } catch (e) {
    // localStorage 可能不可用，忽略错误
  }

  // 3. 尝试从 VSCode API 获取语言设置
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
  
  // 4. 使用浏览器语言
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

// 监听来自 extension 的语言变更消息
if (typeof window !== 'undefined') {
  window.addEventListener('message', (event: MessageEvent) => {
    const data = event.data;
    if (data && data.type === 'languageChanged' && data.data && data.data.language) {
      const newLang = data.data.language;
      if (newLang === 'zh-CN' || newLang === 'en') {
        i18n.changeLanguage(newLang);
        // 同步更新 localStorage（向后兼容）
        try {
          localStorage.setItem('cocursor-language', newLang);
        } catch (e) {
          // localStorage 可能不可用，忽略错误
        }
      }
    }
  });
}

export default i18n;
