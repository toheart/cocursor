/**
 * 发现和加入团队页面（路由页面，替代弹窗）
 */

import React, { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { apiService } from "../../../services/api";
import { DiscoveredTeam } from "../../../types";
import { useToast } from "../../../hooks";
import { useTeamStore } from "../stores";
import { PageHeader, EmptyState, LoadingState } from "../shared";
import { ToastContainer } from "../../shared/ToastContainer";

export const TeamJoinPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showToast, toasts } = useToast();
  const { fetchTeams } = useTeamStore();

  const [manualEndpoint, setManualEndpoint] = useState("");
  const [activeTab, setActiveTab] = useState<"discover" | "manual">(
    "discover",
  );
  const [loading, setLoading] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [discoveredTeams, setDiscoveredTeams] = useState<DiscoveredTeam[]>([]);

  const handleDiscover = useCallback(async () => {
    setDiscovering(true);
    setError(null);

    try {
      const resp = (await apiService.discoverTeams(5)) as {
        teams: DiscoveredTeam[];
      };
      setDiscoveredTeams(resp.teams || []);
      if (resp.teams?.length === 0) {
        setError(t("team.noTeamsFound"));
      }
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : t("team.discoverFailed");
      setError(message);
    } finally {
      setDiscovering(false);
    }
  }, [t]);

  const handleJoin = useCallback(
    async (endpoint: string) => {
      setLoading(true);
      setError(null);

      try {
        await apiService.joinTeam(endpoint);
        await fetchTeams();
        showToast(t("team.joinSuccess"), "success");
        navigate("/");
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t("team.joinFailed");
        setError(message);
      } finally {
        setLoading(false);
      }
    },
    [fetchTeams, showToast, navigate, t],
  );

  const handleJoinManual = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!manualEndpoint.trim()) {
        setError(t("team.endpointRequired"));
        return;
      }
      await handleJoin(manualEndpoint.trim());
    },
    [manualEndpoint, handleJoin, t],
  );

  return (
    <div className="ct-page">
      <ToastContainer toasts={toasts} />
      <PageHeader title={t("team.joinTeam")} backTo="/" />

      <div className="ct-page-body">
        {/* Tab 切换 */}
        <div className="ct-toggle-tabs">
          <button
            className={`ct-toggle-tab ${activeTab === "discover" ? "active" : ""}`}
            onClick={() => setActiveTab("discover")}
          >
            <span className="codicon codicon-search" />
            {t("team.discoverTeams")}
          </button>
          <button
            className={`ct-toggle-tab ${activeTab === "manual" ? "active" : ""}`}
            onClick={() => setActiveTab("manual")}
          >
            <span className="codicon codicon-edit" />
            {t("team.manualJoin")}
          </button>
        </div>

        {error && <div className="ct-form-error">{error}</div>}

        {/* 自动发现 */}
        {activeTab === "discover" && (
          <div className="ct-form-section">
            <p className="ct-form-desc">{t("team.discoverDesc")}</p>

            <button
              type="button"
              className="ct-btn primary full-width"
              onClick={handleDiscover}
              disabled={discovering}
            >
              {discovering ? (
                <>
                  <span className="ct-btn-spinner" />
                  {t("team.discovering")}
                </>
              ) : (
                <>
                  <span className="codicon codicon-broadcast" />
                  {t("team.startDiscover")}
                </>
              )}
            </button>

            {discovering && <LoadingState compact />}

            {discoveredTeams.length > 0 && (
              <div className="ct-discover-list">
                {discoveredTeams.map((team) => (
                  <div key={team.team_id} className="ct-discover-item">
                    <div className="ct-discover-item-info">
                      <div className="ct-discover-item-name">{team.name}</div>
                      <div className="ct-discover-item-meta">
                        <span>
                          {t("team.leaderLabel")}: {team.leader_name}
                        </span>
                        <span>
                          {t("team.members")}: {team.member_count}
                        </span>
                      </div>
                    </div>
                    <button
                      className="ct-btn primary small"
                      onClick={() => handleJoin(team.endpoint)}
                      disabled={loading}
                    >
                      {loading ? (
                        <span className="ct-btn-spinner" />
                      ) : (
                        t("team.join")
                      )}
                    </button>
                  </div>
                ))}
              </div>
            )}

            {discoveredTeams.length === 0 && !discovering && !error && (
              <EmptyState
                icon="broadcast"
                title={t("team.discoverHint")}
              />
            )}
          </div>
        )}

        {/* 手动输入 */}
        {activeTab === "manual" && (
          <form className="ct-form-section" onSubmit={handleJoinManual}>
            <p className="ct-form-desc">{t("team.manualJoinDesc")}</p>

            <div className="ct-form-group">
              <label className="ct-form-label">
                {t("team.leaderEndpoint")}
              </label>
              <input
                type="text"
                className="ct-form-input"
                value={manualEndpoint}
                onChange={(e) => setManualEndpoint(e.target.value)}
                placeholder={t("team.endpointPlaceholder")}
              />
              <p className="ct-form-help">{t("team.endpointHelp")}</p>
            </div>

            <div className="ct-page-actions">
              <button
                type="button"
                className="ct-btn secondary"
                onClick={() => navigate("/")}
                disabled={loading}
              >
                {t("common.cancel")}
              </button>
              <button
                type="submit"
                className="ct-btn primary"
                disabled={loading || !manualEndpoint.trim()}
              >
                {loading && <span className="ct-btn-spinner" />}
                {loading ? t("common.loading") : t("team.join")}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
};
