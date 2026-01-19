/**
 * 向导进度条组件
 * 显示当前步骤和整体进度
 */

import React from "react";
import { useTranslation } from "react-i18next";
import { StepNumber } from "./types";

interface WizardProgressProps {
  currentStep: StepNumber;
  completedSteps: Set<StepNumber>;
}

export const WizardProgress: React.FC<WizardProgressProps> = ({
  currentStep,
  completedSteps,
}) => {
  const { t } = useTranslation();

  const steps: StepNumber[] = [1, 2, 3, 4, 5];

  const getStepStatus = (step: StepNumber) => {
    if (step === currentStep) {
      return 'current';
    } else if (completedSteps.has(step)) {
      return 'completed';
    } else if (step < currentStep) {
      return 'completed';
    } else {
      return 'pending';
    }
  };

  const getStepIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return '✓';
      case 'current':
        return '●';
      case 'pending':
        return '○';
      default:
        return '○';
    }
  };

  const getProgress = () => {
    if (currentStep === 5) {
      return 100;
    }
    return (currentStep / 5) * 100;
  };

  return (
    <div className="cocursor-rag-wizard-progress">
      {/* 步骤指示器 */}
      <div className="cocursor-rag-steps-indicator">
        {steps.map((step) => {
          const status = getStepStatus(step);
          return (
            <div
              key={step}
              className={`cocursor-rag-step-indicator ${status}`}
            >
              <span className="cocursor-rag-step-indicator-number">
                {getStepIcon(status)}
              </span>
              <span className="cocursor-rag-step-indicator-label">
                {t(`rag.config.wizard.step`)} {step}
              </span>
            </div>
          );
        })}
      </div>

      {/* 进度条 */}
      <div className="cocursor-rag-progress-bar">
        <div
          className="cocursor-rag-progress-fill"
          style={{ width: `${getProgress()}%` }}
        />
      </div>
    </div>
  );
};
