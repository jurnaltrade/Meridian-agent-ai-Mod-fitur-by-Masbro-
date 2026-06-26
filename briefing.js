import fs from "fs";
import { log } from "./logger.js";
import { getPerformanceSummary } from "./lessons.js";
import { repoPath } from "./repo-root.js";

const STATE_FILE = repoPath("state.json");
const LESSONS_FILE = repoPath("lessons.json");

export async function generateBriefing() {
  const state = loadJson(STATE_FILE) || { positions: {}, recentEvents: [] };
  const lessonsData = loadJson(LESSONS_FILE) || { lessons: [], performance: [] };

  const now = new Date();
  const last24h = new Date(now.getTime() - 24 * 60 * 60 * 1000);

  // 1. Positions Activity
  const allPositions = Object.values(state.positions || {});
  const openedLast24h = allPositions.filter(p => new Date(p.deployed_at) > last24h);
  const closedLast24h = allPositions.filter(p => p.closed && new Date(p.closed_at) > last24h);

  // 2. Performance Activity (from performance log)
  const perfLast24h = (lessonsData.performance || []).filter(p => new Date(p.recorded_at) > last24h);
  const totalPnLUsd = perfLast24h.reduce((sum, p) => sum + (p.pnl_usd || 0), 0);
  const totalFeesUsd = perfLast24h.reduce((sum, p) => sum + (p.fees_earned_usd || 0), 0);

  // 3. Lessons Learned
  const lessonsLast24h = (lessonsData.lessons || []).filter(l => new Date(l.created_at) > last24h);

  // 4. Current State
  const openPositions = allPositions.filter(p => !p.closed);
  const perfSummary = getPerformanceSummary();

  // 5. Format Message
  const winRate = perfLast24h.length > 0
    ? `${Math.round((perfLast24h.filter(p => p.pnl_usd > 0).length / perfLast24h.length) * 100)}%`
    : "n/a";
  const sign = totalPnLUsd >= 0 ? "+" : "";
  const lines = [
    `☀️ <b>Morning Briefing</b> — last 24h`,
    `Opened ${openedLast24h.length} | Closed ${closedLast24h.length}`,
    `PnL ${sign}$${totalPnLUsd.toFixed(2)} | Fees $${totalFeesUsd.toFixed(2)} | WR ${winRate}`,
    "",
    lessonsLast24h.length > 0
      ? `📚 <b>Lessons</b>\n${lessonsLast24h.slice(0, 5).map(l => `• ${l.rule}`).join("\n")}`
      : "",
    "",
    `📊 <b>Portfolio</b> · ${openPositions.length} open${perfSummary ? ` | All-time PnL $${perfSummary.total_pnl_usd.toFixed(2)} (${perfSummary.win_rate_pct}%)` : ""}`,
  ];

  return lines.filter(Boolean).join("\n");
}

function loadJson(file) {
  if (!fs.existsSync(file)) return null;
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (err) {
    log("briefing_error", `Failed to read ${file}: ${err.message}`);
    return null;
  }
}
