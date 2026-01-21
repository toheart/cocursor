import React from "react";
import { useTranslation } from "react-i18next";
import { WorkCategories } from "../../services/api";

interface WorkCategoriesChartProps {
  categories: WorkCategories;
}

// 工作分类配置：图标、颜色、翻译key
const CATEGORY_CONFIG: Record<keyof WorkCategories, { icon: string; color: string; key: string }> = {
  coding: { icon: "💻", color: "#4CAF50", key: "coding" },
  problem_solving: { icon: "🔍", color: "#FF9800", key: "problemSolving" },
  refactoring: { icon: "♻️", color: "#2196F3", key: "refactoring" },
  code_review: { icon: "👀", color: "#9C27B0", key: "codeReview" },
  documentation: { icon: "📝", color: "#00BCD4", key: "documentation" },
  testing: { icon: "🧪", color: "#607D8B", key: "testing" },
  requirements_discussion: { icon: "💬", color: "#E91E63", key: "requirementsDiscussion" },
  other: { icon: "📌", color: "#9E9E9E", key: "other" },
};

/**
 * 工作分类图表组件
 * 展示各类工作的占比和次数
 */
export const WorkCategoriesChart: React.FC<WorkCategoriesChartProps> = ({ categories }) => {
  const { t } = useTranslation();

  // 计算总数和排序
  const total = Object.values(categories).reduce((sum, val) => sum + val, 0);
  if (total === 0) return null;

  // 按次数从高到低排序，过滤掉 0 值
  const sortedCategories = (Object.entries(categories) as [keyof WorkCategories, number][])
    .filter(([, count]) => count > 0)
    .sort((a, b) => b[1] - a[1]);

  return (
    <div className="cocursor-daily-report-section">
      <h4 className="cocursor-daily-report-section-title">
        <span className="section-icon">📊</span>
        {t("dailyReport.workCategories")}
      </h4>
      <div className="cocursor-work-categories-chart">
        {sortedCategories.map(([category, count]) => {
          const config = CATEGORY_CONFIG[category];
          const percentage = Math.round((count / total) * 100);
          return (
            <div key={category} className="cocursor-work-category-row">
              <div className="cocursor-work-category-label">
                <span className="category-icon">{config.icon}</span>
                <span className="category-name">{t(`dailyReport.category.${config.key}`)}</span>
              </div>
              <div className="cocursor-work-category-bar-container">
                <div
                  className="cocursor-work-category-bar"
                  style={{ width: `${percentage}%`, backgroundColor: config.color }}
                />
              </div>
              <div className="cocursor-work-category-stats">
                <span className="category-percentage">{percentage}%</span>
                <span className="category-count">({count}{t("dailyReport.times")})</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
