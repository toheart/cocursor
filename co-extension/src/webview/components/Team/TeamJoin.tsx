/**
 * 发现和加入团队组件
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { apiService } from "../../services/api";
import { DiscoveredTeam } from "../../types";
import { useApi } from "../../hooks";

interface TeamJoinProps {
  onClose: () => void;
  onSuccess: () => void;
}

export const TeamJoin: React.FC<TeamJoinProps> = ({ onClose, onSuccess }) => {
  const { t } = useTranslation();
  const [manualEndpoint, setManualEndpoint] = useState("");
  const [activeTab, setActiveTab] = useState<"discover" | "manual">("discover");
  const [loading, setLoading] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 发现团队
  const [discoveredTeams, setDiscoveredTeams] = useState<DiscoveredTeam[]>([]);

  const handleDiscover = useCallback(async () => {
    setDiscovering(true);
    setError(null);
    
    try {
      const resp = await apiService.discoverTeams(5) as { teams: DiscoveredTeam[] };
      setDiscoveredTeams(resp.teams || []);
      if (resp.teams?.length === 0) {
        setError(t("team.noTeamsFound"));
      }
    } catch (err: any) {
      setError(err.message || t("team.discoverFailed"));
    } finally {
      setDiscovering(false);
    }
  }, [t]);

  const handleJoinDiscovered = useCallback(async (endpoint: string) => {
    setLoading(true);
    setError(null);

    try {
      await apiService.joinTeam(endpoint);
      onSuccess();
    } catch (err: any) {
      setError(err.message || t("team.joinFailed"));
    } finally {
      setLoading(false);
    }
  }, [onSuccess, t]);

  const handleJoinManual = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();

    if (!manualEndpoint.trim()) {
      setError(t("team.endpointRequired"));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await apiService.joinTeam(manualEndpoint.trim());
      onSuccess();
    } catch (err: any) {
      setError(err.message || t("team.joinFailed"));
    } finally {
      setLoading(false);
    }
  }, [manualEndpoint, onSuccess, t]);

  return (
    <div className="cocursor-modal-overlay" onClick={onClose}>
      <div className="cocursor-modal wide" onClick={e => e.stopPropagation()}>
        <div className="cocursor-modal-header">
          <h2 className="cocursor-modal-title">{t("team.joinTeam")}</h2>
          <button className="cocursor-modal-close" onClick={onClose}>×</button>
        </div>

        <div className="cocursor-modal-body">
          {/* 选项卡 */}
          <div className="cocursor-team-join-tabs">
            <button
              className={`cocursor-team-join-tab ${activeTab === "discover" ? "active" : ""}`}
              onClick={() => setActiveTab("discover")}
            >
              <span className="cocursor-team-join-tab-icon">🔍</span>
              {t("team.discoverTeams")}
            </button>
            <button
              className={`cocursor-team-join-tab ${activeTab === "manual" ? "active" : ""}`}
              onClick={() => setActiveTab("manual")}
            >
              <span className="cocursor-team-join-tab-icon">✏️</span>
              {t("team.manualJoin")}
            </button>
          </div>

          {error && (
            <div className="cocursor-form-error">{error}</div>
          )}

          {/* 自动发现 */}
          {activeTab === "discover" && (
            <div className="cocursor-team-join-discover">
              <p className="cocursor-modal-desc">{t("team.discoverDesc")}</p>

              <button
                type="button"
                className="cocursor-btn primary full-width"
                onClick={handleDiscover}
                disabled={discovering}
              >
                {discovering ? (
                  <>
                    <span className="cocursor-btn-spinner"></span>
                    {t("team.discovering")}
                  </>
                ) : (
                  <>
                    <span className="cocursor-btn-icon">📡</span>
                    {t("team.startDiscover")}
                  </>
                )}
              </button>

              {discoveredTeams.length > 0 && (
                <div className="cocursor-team-join-list">
                  {discoveredTeams.map(team => (
                    <div key={team.team_id} className="cocursor-team-join-item">
                      <div className="cocursor-team-join-item-info">
                        <div className="cocursor-team-join-item-name">{team.name}</div>
                        <div className="cocursor-team-join-item-meta">
                          <span>{t("team.leaderLabel")}: {team.leader_name}</span>
                          <span>{t("team.members")}: {team.member_count}</span>
                        </div>
                      </div>
                      <button
                        className="cocursor-btn primary"
                        onClick={() => handleJoinDiscovered(team.endpoint)}
                        disabled={loading}
                      >
                        {loading ? t("common.loading") : t("team.join")}
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {discoveredTeams.length === 0 && !discovering && !error && (
                <div className="cocursor-team-join-empty">
                  <span className="cocursor-team-join-empty-icon">📡</span>
                  <p>{t("team.discoverHint")}</p>
                </div>
              )}
            </div>
          )}

          {/* 手动输入 */}
          {activeTab === "manual" && (
            <form className="cocursor-team-join-manual" onSubmit={handleJoinManual}>
              <p className="cocursor-modal-desc">{t("team.manualJoinDesc")}</p>

              <div className="cocursor-form-group">
                <label className="cocursor-form-label">{t("team.leaderEndpoint")}</label>
                <input
                  type="text"
                  className="cocursor-form-input"
                  value={manualEndpoint}
                  onChange={e => setManualEndpoint(e.target.value)}
                  placeholder={t("team.endpointPlaceholder")}
                />
                <p className="cocursor-form-help">{t("team.endpointHelp")}</p>
              </div>

              <div className="cocursor-modal-footer">
                <button
                  type="button"
                  className="cocursor-btn secondary"
                  onClick={onClose}
                  disabled={loading}
                >
                  {t("common.cancel")}
                </button>
                <button
                  type="submit"
                  className="cocursor-btn primary"
                  disabled={loading || !manualEndpoint.trim()}
                >
                  {loading ? t("common.loading") : t("team.join")}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
};
