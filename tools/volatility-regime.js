/**
 * Volatility-Regime Adaptive Ranging
 * ───────────────────────────────────
 * Instead of deploying every position with the same static bin width
 * (config.strategy.defaultBinsBelow / fixed "bid_ask"), this module
 * classifies the CURRENT volatility regime of a pool and recommends a
 * bin width + LP strategy suited to that regime:
 *
 *   LOW     (sideways / low realized vol) -> narrow bins, "spot"    -> maximize fee capture
 *   MEDIUM  (normal trending)             -> default width, "bid_ask"
 *   HIGH    (choppy / fast moves)         -> wide bins, "bid_ask"   -> reduce OOR + IL
 *   EXTREME (parabolic / crashing)        -> widest bins or SKIP    -> capital preservation
 *
 * It reuses pool metrics Meridian already fetches (getPoolDetail) — no
 * new external API needed — and keeps a rolling history per pool so the
 * classification is based on more than one snapshot.
 *
 * Drop-in path: tools/volatility-regime.js
 */

import fs from "fs";
import { log } from "../logger.js";
import { repoPath } from "../repo-root.js";
import { config } from "../config.js";
import { getPoolDetail } from "./screening.js";

const REGIME_HISTORY_FILE = repoPath("volatility-regime-history.json");
const MAX_HISTORY_PER_POOL = 20;

function load() {
  if (!fs.existsSync(REGIME_HISTORY_FILE)) return {};
  try {
    return JSON.parse(fs.readFileSync(REGIME_HISTORY_FILE, "utf8"));
  } catch {
    return {};
  }
}

function save(data) {
  fs.writeFileSync(REGIME_HISTORY_FILE, JSON.stringify(data, null, 2));
}

// ─── Regime thresholds (tunable via user-config.json -> volatilityRegime.*) ───
function thresholds() {
  const u = config.volatilityRegime || {};
  return {
    lowMax: u.lowMaxPct ?? 1.5,       // volatility% below this => LOW
    mediumMax: u.mediumMaxPct ?? 4,   // below this => MEDIUM
    highMax: u.highMaxPct ?? 9,       // below this => HIGH, above => EXTREME
    extremePriceSwingPct: u.extremePriceSwingPct ?? 25, // |price_change_pct| beyond this forces EXTREME
  };
}

function classify(volatilityPct, priceChangePct) {
  const t = thresholds();
  const absSwing = Math.abs(Number(priceChangePct) || 0);
  const vol = Number(volatilityPct) || 0;

  if (absSwing >= t.extremePriceSwingPct || vol > t.highMax) return "EXTREME";
  if (vol > t.mediumMax) return "HIGH";
  if (vol > t.lowMax) return "MEDIUM";
  return "LOW";
}

// ─── Regime -> bin width / strategy recommendation ────────────────────
function recommendationFor(regime) {
  const s = config.strategy; // { minBinsBelow, maxBinsBelow, defaultBinsBelow }
  const range = Math.max(1, s.maxBinsBelow - s.minBinsBelow);

  switch (regime) {
    case "LOW":
      return {
        strategy: "spot",
        bins_below: s.minBinsBelow,
        bins_above: 0,
        position_size_multiplier: 1.15, // sideways = safer to size up slightly
        rationale: "Low realized volatility — narrow spot range maximizes fee/TVL capture with low OOR risk.",
      };
    case "MEDIUM":
      return {
        strategy: "bid_ask",
        bins_below: s.defaultBinsBelow,
        bins_above: 0,
        position_size_multiplier: 1.0,
        rationale: "Normal trending volatility — default bid_ask width balances fee capture and range coverage.",
      };
    case "HIGH":
      return {
        strategy: "bid_ask",
        bins_below: Math.min(s.maxBinsBelow, s.defaultBinsBelow + Math.round(range * 0.4)),
        bins_above: Math.round(range * 0.15),
        position_size_multiplier: 0.7, // reduce size, wider range costs more in idle liquidity
        rationale: "High volatility — widen range and add upside coverage to reduce OOR churn; smaller size to limit IL exposure.",
      };
    case "EXTREME":
      return {
        strategy: "bid_ask",
        bins_below: s.maxBinsBelow,
        bins_above: Math.round(range * 0.3),
        position_size_multiplier: 0.4,
        rationale: "Extreme volatility (parabolic or crashing) — widest allowed range, heavily reduced size. Consider skipping entirely.",
        skip_recommended: true,
      };
    default:
      return {
        strategy: s.strategy || "bid_ask",
        bins_below: s.defaultBinsBelow,
        bins_above: 0,
        position_size_multiplier: 1.0,
        rationale: "Unclassified — using configured defaults.",
      };
  }
}

/**
 * Tool: get_volatility_regime
 * Fetches live pool metrics, classifies the regime, records it to history,
 * and returns a concrete bin/strategy/size recommendation the screener
 * agent can pass straight into deploy_position.
 */
export async function getVolatilityRegime({ pool_address, timeframe }) {
  if (!pool_address) return { error: "pool_address is required" };

  const pool = await getPoolDetail({
    pool_address,
    timeframe: timeframe || config.screening.timeframe,
  });

  const volatilityPct = (pool.volatility ?? 0) * 100; // volatility is stored as a fraction upstream
  const priceChangePct = pool.price_change_pct ?? 0;
  const regime = classify(volatilityPct, priceChangePct);
  const rec = recommendationFor(regime);

  // ─── persist rolling history for this pool ───
  const data = load();
  const entry = {
    ts: new Date().toISOString(),
    volatility_pct: Math.round(volatilityPct * 100) / 100,
    price_change_pct: priceChangePct,
    regime,
  };
  data[pool_address] = data[pool_address] || [];
  data[pool_address].push(entry);
  data[pool_address] = data[pool_address].slice(-MAX_HISTORY_PER_POOL);
  save(data);

  const history = data[pool_address];
  const regimeFlips = history.slice(1).filter((h, i) => h.regime !== history[i].regime).length;

  log("volatility_regime", `${pool_address.slice(0, 8)} -> ${regime} (vol=${entry.volatility_pct}%, price_chg=${priceChangePct}%)`);

  return {
    pool: pool_address,
    regime,
    volatility_pct: entry.volatility_pct,
    price_change_pct: priceChangePct,
    recommendation: rec,
    stability: {
      samples: history.length,
      regime_flips_in_window: regimeFlips,
      note: regimeFlips >= 3
        ? "Regime is flip-flopping — treat this pool as unstable, prefer HIGH-regime sizing regardless of latest reading."
        : "Regime reading looks stable across recent samples.",
    },
  };
}

/**
 * Convenience helper for other modules (e.g. bandit-allocator.js) that
 * just need a quick regime label without the full tool-call shape.
 */
export async function quickRegimeLabel(pool_address, timeframe) {
  const result = await getVolatilityRegime({ pool_address, timeframe });
  return result.regime || "MEDIUM";
}
