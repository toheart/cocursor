/**
 * 共享会话详情组件
 * 显示会话内容和评论区
 * - 左右布局：User 靠左，Assistant 靠右
 * - 连续同角色消息自动合并为一个气泡
 * - AI 回复过长时默认截断，可展开
 */

import React, { useState, useCallback, useRef, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";
import { apiService } from "../../services/api";
import { useApi, useToast } from "../../hooks";
import { ToastContainer } from "../shared/ToastContainer";

// 会话消息
interface SessionMessage {
  role: string;
  content: string;
}

// 合并后的消息组（连续同角色消息合并为一组）
interface MergedMessage {
  role: string;
  contents: string[];
}

// 共享会话详情
interface SharedSession {
  id: string;
  sharer_id: string;
  sharer_name: string;
  title: string;
  messages: SessionMessage[];
  message_count: number;
  description?: string;
  shared_at: string;
  comment_count: number;
}

// 评论
interface SessionComment {
  id: string;
  author_id: string;
  author_name: string;
  content: string;
  mentions?: string[];
  created_at: string;
}

interface SharedSessionDetailProps {
  teamId: string;
  sessionId: string;
  onBack: () => void;
}

/** 将连续同角色消息合并为一组 */
function mergeMessages(messages: SessionMessage[]): MergedMessage[] {
  if (!messages || messages.length === 0) return [];

  const merged: MergedMessage[] = [];
  let current: MergedMessage | null = null;

  for (const msg of messages) {
    if (current && current.role === msg.role) {
      // 同角色，追加到当前组
      current.contents.push(msg.content);
    } else {
      // 新角色，开始新组
      if (current) merged.push(current);
      current = { role: msg.role, contents: [msg.content] };
    }
  }
  if (current) merged.push(current);

  return merged;
}

/** 可折叠的消息内容组件 */
const CollapsibleContent: React.FC<{
  contents: string[];
  role: string;
}> = ({ contents, role }) => {
  const [expanded, setExpanded] = useState(false);
  const contentRef = useRef<HTMLDivElement>(null);
  const [needsCollapse, setNeedsCollapse] = useState(false);

  // 内容渲染后检测是否需要折叠（高度超过 150px）
  useEffect(() => {
    if (contentRef.current) {
      setNeedsCollapse(contentRef.current.scrollHeight > 150);
    }
  }, [contents]);

  // Assistant 消息才做折叠，User 消息通常很短
  const shouldCollapse = role === "assistant" && needsCollapse && !expanded;

  return (
    <>
      <div
        ref={contentRef}
        className={`cocursor-shared-session-message-content${shouldCollapse ? " collapsed" : ""}`}
      >
        {contents.map((content, i) => (
          <React.Fragment key={i}>
            {i > 0 && <hr className="cocursor-msg-merged-separator" />}
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeHighlight]}
            >
              {content}
            </ReactMarkdown>
          </React.Fragment>
        ))}
      </div>
      {role === "assistant" && needsCollapse && (
        <button
          className={`cocursor-msg-expand-btn${expanded ? " expanded" : ""}`}
          onClick={() => setExpanded(!expanded)}
        >
          <span className="cocursor-msg-expand-arrow">▼</span>
          {expanded ? "收起" : "展开全文"}
        </button>
      )}
    </>
  );
};

export const SharedSessionDetail: React.FC<SharedSessionDetailProps> = ({
  teamId,
  sessionId,
  onBack,
}) => {
  const { t } = useTranslation();
  const { showToast, toasts } = useToast();
  const [newComment, setNewComment] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const commentsEndRef = useRef<HTMLDivElement>(null);

  // 获取会话详情
  const fetchDetail = useCallback(async () => {
    const resp = await apiService.getSharedSessionDetail(teamId, sessionId) as {
      session: SharedSession;
      comments: SessionComment[];
    };
    return resp;
  }, [teamId, sessionId]);

  const { data, loading, refetch } = useApi<{
    session: SharedSession;
    comments: SessionComment[];
  }>(fetchDetail);

  const session = data?.session;
  const comments = data?.comments || [];

  // 合并连续同角色消息
  const mergedMessages = useMemo(
    () => mergeMessages(session?.messages || []),
    [session?.messages]
  );

  // 滚动到评论底部
  useEffect(() => {
    if (comments.length > 0) {
      commentsEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [comments.length]);

  // 格式化时间
  const formatTime = (dateStr: string): string => {
    const date = new Date(dateStr);
    return date.toLocaleString();
  };

  // 提交评论
  const handleSubmitComment = useCallback(async () => {
    if (!newComment.trim()) return;

    setSubmitting(true);
    try {
      // 解析 @提及
      const mentionRegex = /@(\w+)/g;
      const mentions: string[] = [];
      let match;
      while ((match = mentionRegex.exec(newComment)) !== null) {
        mentions.push(match[1]);
      }

      await apiService.addSessionComment(teamId, sessionId, {
        content: newComment,
        mentions: mentions.length > 0 ? mentions : undefined,
      });

      setNewComment("");
      refetch();
      showToast(t("session.commentSuccess"), "success");
    } catch (error: any) {
      showToast(error.message || t("session.commentFailed"), "error");
    } finally {
      setSubmitting(false);
    }
  }, [teamId, sessionId, newComment, refetch, showToast, t]);

  // 处理按键事件（Ctrl/Cmd + Enter 提交）
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        handleSubmitComment();
      }
    },
    [handleSubmitComment]
  );

  if (loading) {
    return (
      <div className="cocursor-team-loading">
        <div className="cocursor-team-loading-spinner"></div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="cocursor-shared-session-error">
        <span>{t("session.notFound")}</span>
        <button className="cocursor-btn secondary" onClick={onBack}>
          {t("common.back")}
        </button>
      </div>
    );
  }

  return (
    <div className="cocursor-shared-session-detail">
      <ToastContainer toasts={toasts} />

      {/* 头部 */}
      <div className="cocursor-shared-session-detail-header">
        <button className="cocursor-team-detail-back" onClick={onBack}>
          ← {t("common.back")}
        </button>
        <div className="cocursor-shared-session-detail-info">
          <h2>{session.title}</h2>
          <div className="cocursor-shared-session-detail-meta">
            <span className="cocursor-shared-session-author">
              {t("session.sharedBy")} {session.sharer_name}
            </span>
            <span className="cocursor-shared-session-time">
              {formatTime(session.shared_at)}
            </span>
          </div>
          {session.description && (
            <p className="cocursor-shared-session-description">
              {session.description}
            </p>
          )}
        </div>
      </div>

      {/* 会话内容 */}
      <div className="cocursor-shared-session-messages">
        <h3>{t("session.conversation")}</h3>

        {mergedMessages.length === 0 ? (
          <div className="cocursor-shared-session-empty">
            <span className="cocursor-shared-session-empty-icon">💬</span>
            <span className="cocursor-shared-session-empty-title">
              {t("session.noMessages", "暂无对话内容")}
            </span>
            <span className="cocursor-shared-session-empty-desc">
              {t("session.noMessagesDesc", "该会话未包含可显示的对话记录")}
            </span>
          </div>
        ) : (
          <div className="cocursor-shared-session-message-list">
            {mergedMessages.map((msg, index) => (
              <div
                key={index}
                className={`cocursor-shared-session-message ${msg.role}`}
              >
                <div className="cocursor-shared-session-message-role">
                  <span className="cocursor-msg-role-icon">
                    {msg.role === "user" ? "U" : "A"}
                  </span>
                  {msg.role === "user" ? "User" : "Assistant"}
                  {msg.contents.length > 1 && (
                    <span className="cocursor-msg-merged-badge">
                      {msg.contents.length} 条合并
                    </span>
                  )}
                </div>
                <CollapsibleContent contents={msg.contents} role={msg.role} />
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 评论区 */}
      <div className="cocursor-shared-session-comments">
        <h3>
          {t("session.comments")} ({comments.length})
        </h3>

        {comments.length === 0 ? (
          <div className="cocursor-shared-session-no-comments">
            <span>{t("session.noComments")}</span>
          </div>
        ) : (
          <div className="cocursor-shared-session-comment-list">
            {comments.map((comment) => (
              <div key={comment.id} className="cocursor-shared-session-comment">
                <div className="cocursor-shared-session-comment-header">
                  <div className="cocursor-shared-session-comment-avatar">
                    {comment.author_name.charAt(0).toUpperCase()}
                  </div>
                  <div className="cocursor-shared-session-comment-meta">
                    <span className="cocursor-shared-session-comment-author">
                      {comment.author_name}
                    </span>
                    <span className="cocursor-shared-session-comment-time">
                      {formatTime(comment.created_at)}
                    </span>
                  </div>
                </div>
                <div className="cocursor-shared-session-comment-content">
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={[rehypeHighlight]}
                  >
                    {comment.content}
                  </ReactMarkdown>
                </div>
              </div>
            ))}
            <div ref={commentsEndRef} />
          </div>
        )}

        {/* 评论输入 */}
        <div className="cocursor-shared-session-comment-input">
          <textarea
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t("session.commentPlaceholder")}
            rows={3}
            disabled={submitting}
          />
          <div className="cocursor-shared-session-comment-actions">
            <span className="cocursor-shared-session-comment-hint">
              {t("session.commentHint")}
            </span>
            <button
              className="cocursor-btn primary"
              onClick={handleSubmitComment}
              disabled={submitting || !newComment.trim()}
            >
              {submitting ? (
                <>
                  <span className="cocursor-btn-spinner"></span>
                  {t("session.submitting")}
                </>
              ) : (
                t("session.submitComment")
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
