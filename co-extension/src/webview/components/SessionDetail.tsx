import React, { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import "highlight.js/styles/github-dark.css";
import { apiService } from "../services/api";

// 可折叠代码块组件
interface CollapsibleCodeBlockProps {
  language: string;
  code: string;
  defaultCollapsed?: boolean;
}

const CollapsibleCodeBlock: React.FC<CollapsibleCodeBlockProps> = ({ 
  language, 
  code, 
  defaultCollapsed = true 
}) => {
  const { t } = useTranslation();
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);
  const lineCount = code.split('\n').length;
  // 只有超过 5 行的代码块才默认折叠
  const shouldCollapse = lineCount > 5;
  const actualCollapsed = shouldCollapse ? isCollapsed : false;

  // 获取代码预览（前3行）
  const getPreview = () => {
    const lines = code.split('\n');
    return lines.slice(0, 3).join('\n') + (lines.length > 3 ? '\n...' : '');
  };

  return (
    <div className={`cocursor-collapsible-code ${actualCollapsed ? 'collapsed' : 'expanded'}`}>
      {shouldCollapse && (
        <div 
          className="cocursor-code-header"
          onClick={() => setIsCollapsed(!isCollapsed)}
        >
          <span className="cocursor-code-language">{language || 'code'}</span>
          <span className="cocursor-code-info">
            {lineCount} {t("sessions.lines")}
          </span>
          <span className="cocursor-code-toggle">
            {actualCollapsed ? '▶ ' + t("sessions.expandCode") : '▼ ' + t("sessions.collapseCode")}
          </span>
        </div>
      )}
      {actualCollapsed ? (
        <pre className="cocursor-markdown-code-block cocursor-code-preview" onClick={() => setIsCollapsed(false)}>
          <code className={`language-${language}`}>{getPreview()}</code>
        </pre>
      ) : (
        <pre className="cocursor-markdown-code-block">
          <code className={`language-${language}`}>{code}</code>
        </pre>
      )}
    </div>
  );
};

interface ToolCall {
  name: string;
  arguments: Record<string, string>;
}

interface Message {
  type: "user" | "ai";
  text: string;
  timestamp: number;
  code_blocks?: Array<{
    language: string;
    code: string;
  }>;
  files?: string[];
  tools?: ToolCall[];
}

interface SessionDetailData {
  session: {
    composerId: string;
    name: string;
    createdAt: number;
    lastUpdatedAt: number;
  };
  messages: Message[];
  total_messages: number;
  has_more: boolean;
}

export const SessionDetail: React.FC = () => {
  const { t } = useTranslation();
  const { sessionId } = useParams<{ sessionId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<SessionDetailData | null>(null);
  const [highlightedIndex, setHighlightedIndex] = useState<number | null>(null);
  const isMountedRef = React.useRef(true);
  const highlightTimeoutRef = React.useRef<NodeJS.Timeout | null>(null);
  const messageRefs = React.useRef<Map<number, HTMLDivElement>>(new Map());

  // 解析 URL 参数
  const targetTimestamp = searchParams.get('ts') ? parseInt(searchParams.get('ts')!) : null;
  const targetTurnIndex = searchParams.get('turn') ? parseInt(searchParams.get('turn')!) : null;
  const shouldHighlight = searchParams.get('highlight') === 'true';

  // 查找最匹配的消息索引
  const findTargetMessageIndex = useCallback((messages: Message[]): number | null => {
    if (!messages || messages.length === 0) return null;
    
    // 优先使用 turn index
    if (targetTurnIndex !== null && targetTurnIndex >= 0 && targetTurnIndex < messages.length) {
      return targetTurnIndex;
    }
    
    // 使用时间戳匹配
    if (targetTimestamp) {
      // 查找时间戳最接近的消息
      let closestIndex = 0;
      let closestDiff = Math.abs(messages[0].timestamp - targetTimestamp);
      
      for (let i = 1; i < messages.length; i++) {
        const diff = Math.abs(messages[i].timestamp - targetTimestamp);
        if (diff < closestDiff) {
          closestDiff = diff;
          closestIndex = i;
        }
      }
      
      // 时间差在 5 分钟内才认为是匹配的
      if (closestDiff < 5 * 60 * 1000) {
        return closestIndex;
      }
    }
    
    return null;
  }, [targetTimestamp, targetTurnIndex]);

  // 滚动到目标消息并高亮
  const scrollToAndHighlight = useCallback((index: number) => {
    const element = messageRefs.current.get(index);
    if (element) {
      // 滚动到元素
      element.scrollIntoView({ behavior: 'smooth', block: 'center' });
      
      // 设置高亮
      setHighlightedIndex(index);
      
      // 3 秒后移除高亮
      if (highlightTimeoutRef.current) {
        clearTimeout(highlightTimeoutRef.current);
      }
      highlightTimeoutRef.current = setTimeout(() => {
        setHighlightedIndex(null);
      }, 3000);
    }
  }, []);

  useEffect(() => {
    isMountedRef.current = true;
    if (sessionId) {
      loadSessionDetail();
    }
    
    return () => {
      isMountedRef.current = false;
      if (highlightTimeoutRef.current) {
        clearTimeout(highlightTimeoutRef.current);
        highlightTimeoutRef.current = null;
      }
    };
  }, [sessionId]);

  // 数据加载完成后定位到目标消息
  useEffect(() => {
    if (data && shouldHighlight && data.messages) {
      const targetIndex = findTargetMessageIndex(data.messages);
      if (targetIndex !== null) {
        // 延迟执行，确保 DOM 已渲染
        setTimeout(() => {
          scrollToAndHighlight(targetIndex);
        }, 100);
      }
    }
  }, [data, shouldHighlight, findTargetMessageIndex, scrollToAndHighlight]);

  const loadSessionDetail = async (): Promise<void> => {
    if (!sessionId || !isMountedRef.current) return;

    try {
      setLoading(true);
      setError(null);
      const result = await apiService.getSessionDetail(sessionId);
      
      // 检查组件是否已卸载
      if (!isMountedRef.current) return;
      
      const sessionData = result as SessionDetailData;
      // 合并连续的AI消息
      if (sessionData.messages) {
        sessionData.messages = mergeAIMessages(sessionData.messages);
      }
      setData(sessionData);
      
      // rehype-highlight 会自动处理代码高亮，无需手动调用 hljs
    } catch (err) {
      // 组件已卸载，不更新状态
      if (!isMountedRef.current) return;
      setError(err instanceof Error ? err.message : t("common.error"));
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
      }
    }
  };

  const formatTimestamp = (timestamp: number): string => {
    const date = new Date(timestamp);
    return date.toLocaleString();
  };

  // 合并连续的AI消息
  const mergeAIMessages = (messages: Message[]): Message[] => {
    if (!messages || messages.length === 0) return messages;

    const merged: Message[] = [];
    let currentAIGroup: Message[] = [];
    const AI_MERGE_TIME_THRESHOLD = 30000; // 30秒内的AI消息合并

    for (let i = 0; i < messages.length; i++) {
      const msg = messages[i];

      if (msg.type === "ai") {
        // 检查是否应该与上一个AI消息合并
        if (currentAIGroup.length > 0) {
          const lastMsg = currentAIGroup[currentAIGroup.length - 1];
          const timeDiff = msg.timestamp - lastMsg.timestamp;

          if (timeDiff <= AI_MERGE_TIME_THRESHOLD) {
            // 合并到当前组
            currentAIGroup.push(msg);
            continue;
          } else {
            // 时间间隔太长，先保存当前组
            merged.push(mergeAIGroup(currentAIGroup));
            currentAIGroup = [msg];
          }
        } else {
          // 开始新的AI消息组
          currentAIGroup = [msg];
        }
      } else {
        // 用户消息，先保存当前的AI组
        if (currentAIGroup.length > 0) {
          merged.push(mergeAIGroup(currentAIGroup));
          currentAIGroup = [];
        }
        merged.push(msg);
      }
    }

    // 保存最后的AI组
    if (currentAIGroup.length > 0) {
      merged.push(mergeAIGroup(currentAIGroup));
    }

    return merged;
  };

  // 合并AI消息组
  const mergeAIGroup = (group: Message[]): Message => {
    if (group.length === 1) return group[0];

    // 合并文本
    const texts = group.map(m => m.text).filter(t => t.trim());
    const mergedText = texts.join("\n\n");

    // 合并代码块
    const allCodeBlocks: Array<{ language: string; code: string }> = [];
    group.forEach(m => {
      if (m.code_blocks) {
        allCodeBlocks.push(...m.code_blocks);
      }
    });

    // 合并工具调用
    const allTools: ToolCall[] = [];
    group.forEach(m => {
      if (m.tools) {
        allTools.push(...m.tools);
      }
    });

    // 合并文件引用
    const allFiles: string[] = [];
    group.forEach(m => {
      if (m.files) {
        allFiles.push(...m.files);
      }
    });

    // 使用第一条消息的时间戳
    return {
      type: "ai",
      text: mergedText,
      timestamp: group[0].timestamp,
      code_blocks: allCodeBlocks.length > 0 ? allCodeBlocks : undefined,
      tools: allTools.length > 0 ? allTools : undefined,
      files: allFiles.length > 0 ? Array.from(new Set(allFiles)) : undefined, // 去重
    };
  };

  // Markdown 组件配置
  const markdownComponents = {
    // 代码块
    code({ node, inline, className, children, ...props }: any) {
      const match = /language-(\w+)/.exec(className || "");
      return !inline && match ? (
        <pre className="cocursor-markdown-code-block">
          <code className={className} {...props}>
            {children}
          </code>
        </pre>
      ) : (
        <code className="cocursor-markdown-inline-code" {...props}>
          {children}
        </code>
      );
    },
    // 段落
    p: ({ children }: any) => <p className="cocursor-markdown-paragraph">{children}</p>,
    // 标题
    h1: ({ children }: any) => <h1 className="cocursor-markdown-h1">{children}</h1>,
    h2: ({ children }: any) => <h2 className="cocursor-markdown-h2">{children}</h2>,
    h3: ({ children }: any) => <h3 className="cocursor-markdown-h3">{children}</h3>,
    // 列表
    ul: ({ children }: any) => <ul className="cocursor-markdown-ul">{children}</ul>,
    ol: ({ children }: any) => <ol className="cocursor-markdown-ol">{children}</ol>,
    li: ({ children }: any) => <li className="cocursor-markdown-li">{children}</li>,
    // 链接
    a: ({ href, children }: any) => (
      <a href={href} className="cocursor-markdown-link" target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    ),
    // 引用
    blockquote: ({ children }: any) => (
      <blockquote className="cocursor-markdown-blockquote">{children}</blockquote>
    ),
    // 表格
    table: ({ children }: any) => (
      <div className="cocursor-markdown-table-wrapper">
        <table className="cocursor-markdown-table">{children}</table>
      </div>
    ),
    // 分隔线
    hr: () => <hr className="cocursor-markdown-hr" />,
    // 强调
    strong: ({ children }: any) => <strong className="cocursor-markdown-strong">{children}</strong>,
    em: ({ children }: any) => <em className="cocursor-markdown-em">{children}</em>,
  };

  return (
    <div className="cocursor-session-detail">
      {data && (
        <div style={{ 
          padding: "16px 24px", 
          borderBottom: "1px solid var(--vscode-panel-border)",
          background: "var(--vscode-sideBar-background)"
        }}>
          <h2 style={{ 
            margin: 0, 
            fontSize: "16px", 
            fontWeight: 600,
            color: "var(--vscode-foreground)",
            letterSpacing: "-0.3px"
          }}>
            {data.session.name || t("sessions.detail")}
          </h2>
        </div>
      )}

      {loading && <div className="cocursor-loading">{t("sessions.loading")}</div>}
      {error && <div className="cocursor-error">{t("sessions.error")}: {error}</div>}
      {data && (
        <>
          <div className="cocursor-session-info">
            <div className="cocursor-info-item">
              <span className="cocursor-info-label">{t("sessions.createdAt")}</span>
              <span className="cocursor-info-value">{formatTimestamp(data.session.createdAt)}</span>
            </div>
            <div className="cocursor-info-item">
              <span className="cocursor-info-label">{t("sessions.lastUpdated")}</span>
              <span className="cocursor-info-value">{formatTimestamp(data.session.lastUpdatedAt)}</span>
            </div>
            <div className="cocursor-info-item">
              <span className="cocursor-info-label">{t("sessions.totalMessages")}</span>
              <span className="cocursor-info-value">{data.total_messages}</span>
            </div>
          </div>

          <div className="cocursor-messages">
            {data.messages && Array.isArray(data.messages) && data.messages.length > 0 ? (
              <>
                {data.messages.map((message, index) => (
                  <div
                    key={index}
                    ref={(el) => {
                      if (el) messageRefs.current.set(index, el);
                    }}
                    className={`cocursor-message cocursor-message-${message.type}${highlightedIndex === index ? ' cocursor-message-highlighted' : ''}`}
                  >
                    <div className="cocursor-message-header">
                      <span className="cocursor-message-type">
                        {message.type === "user" ? t("sessions.user") : t("sessions.ai")}
                      </span>
                      <span className="cocursor-message-time">
                        {formatTimestamp(message.timestamp)}
                      </span>
                    </div>
                    <div className="cocursor-message-content">
                      {message.text && (
                        <ReactMarkdown
                          remarkPlugins={[remarkGfm]}
                          rehypePlugins={[rehypeHighlight]}
                          components={markdownComponents}
                        >
                          {message.text}
                        </ReactMarkdown>
                      )}
                      {message.code_blocks && Array.isArray(message.code_blocks) && message.code_blocks.map((block, i) => (
                        <CollapsibleCodeBlock
                          key={i}
                          language={block.language}
                          code={block.code}
                          defaultCollapsed={true}
                        />
                      ))}
                      {message.tools && Array.isArray(message.tools) && message.tools.length > 0 && (
                        <div className="cocursor-message-tools">
                          <div className="cocursor-tools-compact">
                            <span className="cocursor-tools-label">{t("sessions.tools")}:</span>
                            {message.tools.map((tool, i) => (
                              <span key={i} className="cocursor-tool-badge" title={Object.keys(tool.arguments).length > 0 ? `${tool.name}(${Object.keys(tool.arguments).join(', ')})` : tool.name}>
                                {tool.name}
                                {Object.keys(tool.arguments).length > 0 && (
                                  <span className="cocursor-tool-arg-count">({Object.keys(tool.arguments).length})</span>
                                )}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}
                      {message.files && Array.isArray(message.files) && message.files.length > 0 && (
                        <div className="cocursor-message-files">
                          <strong>{t("sessions.referencedFiles")}</strong>
                          <ul>
                            {message.files.map((file, i) => (
                              <li key={i}>{file}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>
                  </div>
                ))}
                {data.has_more && (
                  <div className="cocursor-load-more-messages">
                    <button onClick={() => {/* TODO: 加载更多消息 */}}>
                      {t("sessions.loadMoreMessages")}
                    </button>
                  </div>
                )}
              </>
            ) : (
              <div className="cocursor-empty" style={{ 
                padding: "60px 20px", 
                textAlign: "center", 
                color: "var(--vscode-descriptionForeground)",
                fontSize: "14px"
              }}>
                {t("sessions.noMessages")}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
};
