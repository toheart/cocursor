import React from "react";
import { useTranslation } from "react-i18next";
import { CodeChangeSummary } from "../../services/api";

interface CodeChangesStatsProps {
  codeChanges: CodeChangeSummary;
}

/**
 * 代码变更统计组件
 * 展示新增行、删除行、变更文件数
 */
export const CodeChangesStats: React.FC<CodeChangesStatsProps> = ({ codeChanges }) => {
  const { t } = useTranslation();

  // 检查是否有数据
  const { lines_added, lines_removed, files_changed } = codeChanges;
  if (lines_added === 0 && lines_removed === 0 && files_changed === 0) {
    return null;
  }

  return (
    <div className="cocursor-daily-report-section">
      <h4 className="cocursor-daily-report-section-title">
        <span className="section-icon">📈</span>
        {t("dailyReport.codeChanges")}
      </h4>
      <div className="cocursor-code-changes-stats">
        <div className="cocursor-code-change-item added">
          <span className="change-indicator">+</span>
          <span className="change-value">{lines_added}</span>
          <span className="change-label">{t("dailyReport.linesAdded")}</span>
        </div>
        <div className="cocursor-code-change-item removed">
          <span className="change-indicator">-</span>
          <span className="change-value">{lines_removed}</span>
          <span className="change-label">{t("dailyReport.linesRemoved")}</span>
        </div>
        <div className="cocursor-code-change-item files">
          <span className="change-indicator">📄</span>
          <span className="change-value">{files_changed}</span>
          <span className="change-label">{t("dailyReport.filesChanged")}</span>
        </div>
      </div>
    </div>
  );
};
