import { useState, useCallback, RefObject } from "react";
import html2canvas from "html2canvas";
import { getVscodeApi } from "../../services/api";

interface UseScreenshotOptions {
  filename?: string;
  watermark?: string;
}

interface UseScreenshotReturn {
  takeScreenshot: () => Promise<void>;
  copyToClipboard: () => Promise<void>;
  isCapturing: boolean;
  error: string | null;
}

/**
 * 解析 CSS 变量为实际颜色值
 * VS Code Webview 中 html2canvas 可能无法正确解析 CSS 变量
 */
function resolveCSSVariables(element: HTMLElement): void {
  const computedStyle = getComputedStyle(element);
  const allElements = element.querySelectorAll("*");
  
  // 处理容器元素
  const bgColor = computedStyle.getPropertyValue("background-color");
  if (bgColor && bgColor !== "rgba(0, 0, 0, 0)") {
    element.style.backgroundColor = bgColor;
  }
  element.style.color = computedStyle.getPropertyValue("color");
  
  // 处理所有子元素
  allElements.forEach((el) => {
    if (el instanceof HTMLElement) {
      const style = getComputedStyle(el);
      el.style.backgroundColor = style.getPropertyValue("background-color");
      el.style.color = style.getPropertyValue("color");
      el.style.borderColor = style.getPropertyValue("border-color");
    }
  });
}

/**
 * 截图功能 Hook
 * 支持保存到本地和复制到剪贴板
 */
export function useScreenshot(
  containerRef: RefObject<HTMLElement>,
  options: UseScreenshotOptions = {}
): UseScreenshotReturn {
  const [isCapturing, setIsCapturing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 执行截图
  const captureElement = useCallback(async (): Promise<string | null> => {
    if (!containerRef.current) {
      setError("Container element not found");
      return null;
    }

    try {
      setError(null);
      
      // 使用 html2canvas 渲染
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const canvas = await html2canvas(containerRef.current, {
        backgroundColor: "#1e1e1e", // VS Code 深色背景
        scale: 2, // 2x 分辨率
        useCORS: true,
        logging: true, // 开启日志便于调试
        allowTaint: true, // 允许污染画布
        // 在克隆 DOM 时解析 CSS 变量
        onclone: (clonedDoc: Document) => {
          const clonedElement = clonedDoc.body.querySelector("[data-screenshot-target]") as HTMLElement;
          if (clonedElement) {
            resolveCSSVariables(clonedElement);
          }
        },
        // 忽略某些元素
        ignoreElements: (element: Element) => {
          return element.classList.contains("screenshot-ignore");
        },
      } as any);

      // 添加水印
      if (options.watermark) {
        const ctx = canvas.getContext("2d");
        if (ctx) {
          ctx.font = "24px -apple-system, BlinkMacSystemFont, sans-serif";
          ctx.fillStyle = "rgba(255, 255, 255, 0.5)";
          ctx.textAlign = "center";
          ctx.fillText(options.watermark, canvas.width / 2, canvas.height - 20);
        }
      }

      // 直接使用 toDataURL 获取 base64，避免 blob 问题
      const dataUrl = canvas.toDataURL("image/png");
      return dataUrl;
    } catch (err) {
      console.error("Screenshot failed:", err);
      setError(err instanceof Error ? err.message : "Screenshot failed");
      
      // 显示错误提示
      const vscode = getVscodeApi();
      vscode.postMessage({
        command: "showMessage",
        payload: {
          type: "error",
          message: `截图失败: ${err instanceof Error ? err.message : "未知错误"}`,
        },
      });
      
      return null;
    }
  }, [containerRef, options.watermark]);

  // 保存到本地
  const takeScreenshot = useCallback(async () => {
    setIsCapturing(true);
    try {
      const dataUrl = await captureElement();
      if (!dataUrl) return;

      // 提取 base64 数据（去掉 data:image/png;base64, 前缀）
      const base64Data = dataUrl.split(",")[1];

      // 发送到 extension 保存文件
      const vscode = getVscodeApi();
      vscode.postMessage({
        command: "saveDailyReportScreenshot",
        payload: {
          filename: options.filename || `daily-report-${new Date().toISOString().split("T")[0]}.png`,
          data: base64Data,
        },
      });
    } finally {
      setIsCapturing(false);
    }
  }, [captureElement, options.filename]);

  // 复制到剪贴板 - 在 VS Code Webview 中可能受限，降级为保存
  const copyToClipboard = useCallback(async () => {
    setIsCapturing(true);
    try {
      const dataUrl = await captureElement();
      if (!dataUrl) return;

      // 尝试使用 Clipboard API
      try {
        // 将 base64 转换为 blob
        const response = await fetch(dataUrl);
        const blob = await response.blob();
        
        await navigator.clipboard.write([
          new ClipboardItem({
            "image/png": blob,
          }),
        ]);
        
        // 通知成功
        const vscode = getVscodeApi();
        vscode.postMessage({
          command: "showMessage",
          payload: {
            type: "info",
            message: "截图已复制到剪贴板",
          },
        });
      } catch (clipboardErr) {
        // 剪贴板 API 在 VS Code Webview 中可能不可用，降级为保存文件
        console.warn("Clipboard API failed, falling back to download:", clipboardErr);
        
        const base64Data = dataUrl.split(",")[1];
        const vscode = getVscodeApi();
        vscode.postMessage({
          command: "saveDailyReportScreenshot",
          payload: {
            filename: options.filename || `daily-report-${new Date().toISOString().split("T")[0]}.png`,
            data: base64Data,
          },
        });
        
        vscode.postMessage({
          command: "showMessage",
          payload: {
            type: "info",
            message: "剪贴板不可用，已改为保存文件",
          },
        });
      }
    } finally {
      setIsCapturing(false);
    }
  }, [captureElement, options.filename]);

  return {
    takeScreenshot,
    copyToClipboard,
    isCapturing,
    error,
  };
}
