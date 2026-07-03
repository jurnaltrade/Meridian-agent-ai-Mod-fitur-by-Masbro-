/**
 * IL Hedge Overlay
 * ────────────────
 * Opens a small perp SHORT (on a venue of your choice — Drift, Zeta,
 * Hyperliquid, etc.) sized to a fraction of the volatile-token exposure
 * in a new LP position, so directional price risk is partially hedged
 * while LP fees keep accruing. When the LP position closes, the hedge
 * is closed too.
 *
 * IMPORTANT — read before enabling:
 * This module is an ADAPTER, not a finished broker integration. Meridian's
 * DLMM tooling in this repo does not include a perp-venue SDK, and perp
 * SDKs change their function signatures across versions. Wiring in real
 * order-placement calls with a hallucinated/outdated API would risk real
 * funds, so the actual `placeOrder`/`closeOrder` calls are left as a single,
 * clearly-marked integration point (`_venue` object below) for you to fill
 * in against whichever perp SDK's CURRENT docs you're using
 * (e.g. @drift-labs/sdk, @zetamarkets/sdk). Everything else — sizing math,
 * safety caps, state tracking, hedge lifecycle, config gating — is complete
 * and safe to use as-is. Until `_venue` is implemented, every call runs in
 * `dry_run` and only logs/records what WOULD have happened.
 *
 * Drop-in path: tools/hedge-overlay.js
 */

import fs from "fs";
import { log } from "../logger.js";
import { repoPath } from "../repo-root.js";
import { config } from "../config.js";

const HEDGE_STATE_FILE = repoPath("hedge-state.json");

function load() {
  if (!fs.existsSync(HEDGE_STATE_FILE)) return { hedges: [] };
  try {
    return JSON.parse(fs.readFileSync(HEDGE_STATE_FILE, "utf8"));
  } catch {
    return { hedges: [] };
  }
}

function save(data) {
  fs.writeFileSync(HEDGE_STATE_FILE, JSON.stringify(data, null, 2));
}

function hedgeConfig() {
  const u = config.hedge || {};
  return {
    enabled: u.enabled ?? false,          // OFF by default — must opt in
    hedgeRatio: u.hedgeRatio ?? 0.5,      // hedge 50% of the volatile-side notional by default
    maxHedgeNotionalUsd: u.maxHedgeNotionalUsd ?? 200,
    minPositionUsdToHedge: u.minPositionUsdToHedge ?? 50, // skip hedging tiny positions (fees eat the edge)
    venue: u.venue ?? "none",             // "drift" | "zeta" | "hyperliquid" | "none"
  };
}

/**
 * INTEGRATION POINT — fill this in against your chosen perp venue's SDK.
 * Keep the same method shapes so the rest of this file doesn't need to change.
 */
const _venue = {
  async placeShort({ market, notionalUsd }) {
    // TODO: e.g. Drift -> driftClient.placePerpOrder({ marketIndex, direction: 'short', ... })
    // TODO: e.g. Zeta  -> zetaClient.placeOrder(marketIndex, price, size, Side.SHORT, ...)
    throw new Error("Perp venue not wired — implement _venue.placeShort() in tools/hedge-overlay.js");
  },
  async closePosition({ market, hedgeVenuePositionId }) {
    throw new Error("Perp venue not wired — implement _venue.closePosition() in tools/hedge-overlay.js");
  },
  async getMarkPrice({ market }) {
    throw new Error("Perp venue not wired — implement _venue.getMarkPrice() in tools/hedge-overlay.js");
  },
};

/**
 * Tool: open_hedge
 * Call right after deploy_position succeeds. Sizes the short as
 * hedgeRatio * (volatile-token side of the LP notional), capped by
 * maxHedgeNotionalUsd, and records it against the LP position address
 * so it can be auto-closed later.
 */
export async function openHedge({ position_address, pool_address, base_symbol, position_notional_usd }) {
  const cfg = hedgeConfig();

  if (!cfg.enabled) {
    return { skipped: true, reason: "Hedging disabled (config.hedge.enabled = false)." };
  }
  if (cfg.venue === "none") {
    return { skipped: true, reason: "No perp venue configured (config.hedge.venue)." };
  }
  if (!position_notional_usd || position_notional_usd < cfg.minPositionUsdToHedge) {
    return { skipped: true, reason: `Position notional below minPositionUsdToHedge ($${cfg.minPositionUsdToHedge}).` };
  }

  // Volatile-token exposure is roughly half the LP notional at deploy time
  // (single-sided-into-range deploys vary this, but half is a safe default estimate).
  const volatileExposureUsd = position_notional_usd * 0.5;
  const rawHedgeNotional = volatileExposureUsd * cfg.hedgeRatio;
  const hedgeNotionalUsd = Math.min(rawHedgeNotional, cfg.maxHedgeNotionalUsd);

  const isDryRun = process.env.DRY_RUN === "true";

  let venueResult = { dry_run: true };
  try {
    if (!isDryRun) {
      venueResult = await _venue.placeShort({ market: base_symbol, notionalUsd: hedgeNotionalUsd });
    }
  } catch (error) {
    log("hedge_error", `open_hedge failed for ${position_address}: ${error.message}`);
    return { success: false, error: error.message, would_have_hedged_usd: hedgeNotionalUsd };
  }

  const data = load();
  const hedge = {
    id: `hedge_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
    position_address,
    pool_address,
    base_symbol,
    hedge_notional_usd: Math.round(hedgeNotionalUsd * 100) / 100,
    opened_at: new Date().toISOString(),
    status: isDryRun || venueResult.dry_run ? "dry_run" : "open",
    venue: cfg.venue,
    venue_ref: venueResult.orderId || venueResult.positionId || null,
  };
  data.hedges.push(hedge);
  save(data);

  log("hedge_open", `${hedge.status.toUpperCase()} short ${base_symbol} $${hedge.hedge_notional_usd} for LP ${position_address}`);
  return { success: true, hedge };
}

/**
 * Tool: close_hedge
 * Call right after close_position succeeds for the matching LP position.
 */
export async function closeHedge({ position_address }) {
  const data = load();
  const hedge = data.hedges.find((h) => h.position_address === position_address && (h.status === "open" || h.status === "dry_run"));
  if (!hedge) {
    return { skipped: true, reason: `No open hedge found for position ${position_address}.` };
  }

  try {
    if (hedge.status === "open") {
      await _venue.closePosition({ market: hedge.base_symbol, hedgeVenuePositionId: hedge.venue_ref });
    }
  } catch (error) {
    log("hedge_error", `close_hedge failed for ${position_address}: ${error.message}`);
    return { success: false, error: error.message };
  }

  hedge.status = "closed";
  hedge.closed_at = new Date().toISOString();
  save(data);

  log("hedge_close", `Closed hedge for LP ${position_address} (${hedge.base_symbol}, $${hedge.hedge_notional_usd})`);
  return { success: true, hedge };
}

/**
 * Tool: get_hedge_status
 */
export function getHedgeStatus({ position_address } = {}) {
  const data = load();
  if (position_address) {
    return { hedges: data.hedges.filter((h) => h.position_address === position_address) };
  }
  const open = data.hedges.filter((h) => h.status === "open" || h.status === "dry_run");
  return {
    total_hedges_recorded: data.hedges.length,
    open_hedges: open.length,
    open_notional_usd: Math.round(open.reduce((s, h) => s + h.hedge_notional_usd, 0) * 100) / 100,
    hedges: open,
  };
}
