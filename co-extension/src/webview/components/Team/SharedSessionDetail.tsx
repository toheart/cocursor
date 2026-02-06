/**
 * 共享会话详情组件
 * 显示会话内容和评论区
 */

import React, { useState, useCallback, useRef, useEffect } from "react";
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
        <div className="cocursor-shared-session-message-list">
          {session.messages.map((msg, index) => (
            <div
              key={index}
              className={`cocursor-shared-session-message ${msg.role}`}
            >
              <div className="cocursor-shared-session-message-role">
                {msg.role === "user" ? "👤 User" : "🤖 Assistant"}
              </div>
              <div className="cocursor-shared-session-message-content">
                <ReactMarkdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeHighlight]}
                >
                  {msg.content}
                </ReactMarkdown>
              </div>
            </div>
          ))}
        </div>
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
