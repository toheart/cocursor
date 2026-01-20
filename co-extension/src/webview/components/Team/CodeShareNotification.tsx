/**
 * 代码分享通知组件
 */

import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { CodeSnippet } from "../../types";

interface CodeShareNotificationProps {
  snippet: CodeSnippet;
  onDismiss: () => void;
}

export const CodeShareNotification: React.FC<CodeShareNotificationProps> = ({
  snippet,
  onDismiss,
}) => {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  const getLanguageLabel = (lang: string) => {
    const langMap: Record<string, string> = {
      typescript: "TypeScript",
      javascript: "JavaScript",
      python: "Python",
      go: "Go",
      rust: "Rust",
      java: "Java",
      cpp: "C++",
      c: "C",
      css: "CSS",
      html: "HTML",
      json: "JSON",
      yaml: "YAML",
      markdown: "Markdown",
    };
    return langMap[lang.toLowerCase()] || lang;
  };

  return (
    <div className="cocursor-code-share-notification">
      <div className="cocursor-code-share-header" onClick={() => setExpanded(!expanded)}>
        <div className="cocursor-code-share-info">
          <span className="cocursor-code-share-sender">{snippet.sender_name}</span>
          <span className="cocursor-code-share-action">{t("team.sharedCode")}</span>
        </div>
        <div className="cocursor-code-share-meta">
          <span className="cocursor-code-share-file">
            {snippet.file_name}
            {snippet.start_line > 0 && `:${snippet.start_line}-${snippet.end_line}`}
          </span>
          <span className="cocursor-code-share-time">{formatTime(snippet.created_at)}</span>
        </div>
        <button className="cocursor-code-share-dismiss" onClick={(e) => { e.stopPropagation(); onDismiss(); }}>
          ×
        </button>
      </div>

      {snippet.message && (
        <div className="cocursor-code-share-message">{snippet.message}</div>
      )}

      {expanded && (
        <div className="cocursor-code-share-content">
          <div className="cocursor-code-share-lang-badge">
            {getLanguageLabel(snippet.language)}
          </div>
          <pre className="cocursor-code-share-code">
            <code>{snippet.code}</code>
          </pre>
        </div>
      )}

      <div className="cocursor-code-share-expand" onClick={() => setExpanded(!expanded)}>
        {expanded ? t("common.collapse") : t("common.expand")}
      </div>
    </div>
  );
};

// 代码分享列表组件
interface CodeShareListProps {
  snippets: CodeSnippet[];
  onDismiss: (id: string) => void;
  onClear: () => void;
}

export const CodeShareList: React.FC<CodeShareListProps> = ({
  snippets,
  onDismiss,
  onClear,
}) => {
  const { t } = useTranslation();

  if (snippets.length === 0) {
    return null;
  }

  return (
    <div className="cocursor-code-share-list">
      <div className="cocursor-code-share-list-header">
        <span className="cocursor-code-share-list-title">
          {t("team.sharedCodeSnippets")} ({snippets.length})
        </span>
        {snippets.length > 0 && (
          <button className="cocursor-code-share-clear" onClick={onClear}>
            {t("common.clearAll")}
          </button>
        )}
      </div>
      <div className="cocursor-code-share-list-content">
        {snippets.map((snippet) => (
          <CodeShareNotification
            key={snippet.id}
            snippet={snippet}
            onDismiss={() => onDismiss(snippet.id)}
          />
        ))}
      </div>
    </div>
  );
};
