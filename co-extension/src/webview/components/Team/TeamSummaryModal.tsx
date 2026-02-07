/**
 * 团队周报素材汇总弹窗
 * 展示所有成员的周报 Markdown，支持复制到剪贴板
 */

import React, { useMemo, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { TeamMemberSummariesView } from "../../types";

interface TeamSummaryModalProps {
  data: TeamMemberSummariesView;
  onClose: () => void;
}

export const TeamSummaryModal: React.FC<TeamSummaryModalProps> = ({
  data,
  onClose,
}) => {
  const { t } = useTranslation();

  // 有周报的成员
  const membersWithSummary = useMemo(
    () => data.members.filter((m) => m.has_summary && m.summary),
    [data.members],
  );

  // 缺少周报的成员
  const missingMembers = useMemo(
    () => data.missing_members || [],
    [data.missing_members],
  );

  // 生成可复制的汇总文本
  const summaryText = useMemo(() => {
    const lines: string[] = [];
    lines.push(`# ${data.team_name} 周报素材`);
    lines.push(`> 周期: ${data.week_start} ~ ${data.week_end}`);
    lines.push("");

    if (missingMembers.length > 0) {
      lines.push(`> ⚠️ 以下成员未生成周报: ${missingMembers.join("、")}`);
      lines.push("");
    }

    for (const member of membersWithSummary) {
      lines.push(`## ${member.member_name}`);
      lines.push("");
      lines.push(member.summary);
      lines.push("");
      lines.push("---");
      lines.push("");
    }

    lines.push("");
    lines.push(
      "请基于以上成员周报，按项目分组汇总，生成团队周报。分析各项目本周进展、成员贡献和整体进度。",
    );

    return lines.join("\n");
  }, [data, membersWithSummary, missingMembers]);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(summaryText);
      // 使用简单的 alert 提示，因为 toast 在 modal 上层
      const btn = document.querySelector(
        ".ct-summary-copy-btn",
      ) as HTMLElement;
      if (btn) {
        const originalText = btn.textContent;
        btn.textContent = t("weeklyReport.copySuccess");
        btn.classList.add("copied");
        setTimeout(() => {
          btn.textContent = originalText;
          btn.classList.remove("copied");
        }, 2000);
      }
    } catch {
      // fallback: 使用 textarea 方式复制
      const textarea = document.createElement("textarea");
      textarea.value = summaryText;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
    }
  }, [summaryText, t]);

  // 所有成员都没有周报
  const noneReady = membersWithSummary.length === 0;

  return (
    <div className="ct-modal-overlay" onClick={onClose}>
      <div
        className="ct-modal ct-summary-modal"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 头部 */}
        <div className="ct-modal-header">
          <h3>{t("weeklyReport.summaryTitle")}</h3>
          <button className="ct-btn-icon" onClick={onClose}>
            <span className="codicon codicon-close" />
          </button>
        </div>

        {/* 说明 */}
        <div className="ct-summary-desc">
          <span className="codicon codicon-info" />
          <span>{t("weeklyReport.summaryDesc")}</span>
        </div>

        {/* 缺少周报警告 */}
        {missingMembers.length > 0 && (
          <div className="ct-summary-warning">
            <span className="codicon codicon-warning" />
            <div>
              <div>{t("weeklyReport.missingWarning")}</div>
              <div className="ct-summary-missing-list">
                {missingMembers.join("、")}
              </div>
            </div>
          </div>
        )}

        {/* 内容区 */}
        <div className="ct-summary-content">
          {noneReady ? (
            <div className="ct-summary-empty">
              <span className="codicon codicon-warning" />
              <h4>{t("weeklyReport.cannotGenerate")}</h4>
              <p>{t("weeklyReport.cannotGenerateDesc")}</p>
            </div>
          ) : (
            <div className="ct-summary-members">
              {data.members.map((member) => (
                <div key={member.member_id} className="ct-summary-member">
                  <div className="ct-summary-member-header">
                    <span className="ct-summary-member-name">
                      {member.member_name}
                    </span>
                    {member.has_summary ? (
                      <span className="ct-badge success small">
                        <span className="codicon codicon-check" />
                      </span>
                    ) : (
                      <span className="ct-badge warning small">
                        {!member.is_online
                          ? t("weeklyReport.offline")
                          : t("weeklyReport.noSummaryContent")}
                      </span>
                    )}
                  </div>
                  {member.has_summary && member.summary ? (
                    <div className="ct-summary-member-content">
                      <pre>{member.summary}</pre>
                    </div>
                  ) : (
                    <div className="ct-summary-member-empty">
                      {!member.is_online
                        ? t("weeklyReport.offlineMember")
                        : member.error
                          ? t("weeklyReport.fetchError")
                          : t("weeklyReport.noSummaryContent")}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* 底部操作 */}
        <div className="ct-modal-footer">
          <button className="ct-btn secondary" onClick={onClose}>
            {t("common.close")}
          </button>
          {!noneReady && (
            <button
              className="ct-btn primary ct-summary-copy-btn"
              onClick={handleCopy}
            >
              <span className="codicon codicon-copy" />
              {t("weeklyReport.copyToClipboard")}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};
