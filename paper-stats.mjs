#!/usr/bin/env node
/**
 * paper-stats.mjs — Charonica-style validation gate for the paper simulator.
 * Reads paper-positions.json and prints win-rate, profit factor, avg W/L,
 * exit breakdown, and a gate verdict. Run anytime: `node paper-stats.mjs`
 */
import { readFileSync, existsSync } from "fs";
import { fileURLToPath } from "url";
import path from "path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const FILE = path.join(__dirname, "paper-positions.json");

// Gates (tweak to taste)
const GATE_SAMPLE = 30;
const GATE_PF     = 1.2;
const GATE_WR     = 40; // %

if (!existsSync(FILE)) {
  console.log("No paper-positions.json yet — run the agent (npm start) to accumulate paper trades first.");
  process.exit(0);
}

const state = JSON.parse(readFileSync(FILE, "utf8"));
const all = Object.values(state.positions || {});
const open = all.filter((p) => p.status === "open");
const closed = all.filter((p) => p.status === "closed");

const pct = (p) => (p.deposit_amount > 0 ? (p.net_pnl / p.deposit_amount) * 100 : 0);
const wins = closed.filter((p) => p.net_pnl > 0);
const losses = closed.filter((p) => p.net_pnl <= 0);
const sum = (arr, f) => arr.reduce((a, x) => a + f(x), 0);

const grossWin = sum(wins, (p) => p.net_pnl);
const grossLoss = Math.abs(sum(losses, (p) => p.net_pnl));
const pf = grossLoss > 0 ? grossWin / grossLoss : (grossWin > 0 ? Infinity : 0);
const wr = closed.length ? (wins.length / closed.length) * 100 : 0;
const avgW = wins.length ? sum(wins, pct) / wins.length : 0;
const avgL = losses.length ? sum(losses, pct) / losses.length : 0;
const netUsd = sum(closed, (p) => p.net_pnl);
const netFees = sum(closed, (p) => p.fees_earned || 0);
const best = closed.length ? Math.max(...closed.map(pct)) : 0;
const worst = closed.length ? Math.min(...closed.map(pct)) : 0;

// Exit breakdown by close_reason category
const cat = (r) => {
  const s = String(r || "").toLowerCase();
  if (s.includes("stop loss")) return "SL";
  if (s.includes("take profit")) return "TP";
  if (s.includes("trailing")) return "TRAILING";
  if (s.includes("stale")) return "STALE";
  if (s.includes("range")) return "OOR";
  return "OTHER";
};
const exits = {};
for (const p of closed) {
  const c = cat(p.close_reason);
  exits[c] = exits[c] || { n: 0, sumPct: 0 };
  exits[c].n++;
  exits[c].sumPct += pct(p);
}

const pf_str = pf === Infinity ? "∞" : pf.toFixed(3);
const f = (v) => (v >= 0 ? "+" : "") + v.toFixed(2);

// ── Open positions (live condition right now) ──
if (open.length) {
  console.log(`\n🟢 OPEN POSITIONS (${open.length})\n${"─".repeat(40)}`);
  for (const p of open) {
    const pnlPct = p.deposit_amount > 0 ? (p.net_pnl / p.deposit_amount) * 100 : 0;
    const inRange = p.candles_total > 0 ? (p.candles_in_range / p.candles_total) * 100 : 0;
    const ageMin = Math.floor((Date.now() / 1000 - (p.entry_timestamp || 0)) / 60);
    const live = p.last_price >= p.lower_price && p.last_price <= p.upper_price ? "IN RANGE" : "OUT OF RANGE";
    console.log(`  ${p.pair}  $${p.deposit_amount}  ${p.strategy_type}`);
    console.log(`    PnL: $${f(p.net_pnl)} (${f(pnlPct)}%) | fees $${(p.fees_earned||0).toFixed(4)} | IL $${(p.il_usd||0).toFixed(4)}`);
    console.log(`    peak: ${f(p.peak_pnl_pct||0)}% | in-range: ${inRange.toFixed(0)}% | now: ${live} | age: ${ageMin}m`);
  }
}

console.log(`\n📊 PAPER VALIDATION GATE\n${"─".repeat(40)}`);
console.log(`Sample: ${closed.length} closed / ${open.length} open`);
console.log("");
console.log(`${closed.length >= GATE_SAMPLE ? "✅" : "❌"} Sample ${closed.length}/${GATE_SAMPLE}`);
console.log(`${pf >= GATE_PF ? "✅" : "❌"} PF ${pf_str} (gate ≥${GATE_PF})`);
console.log(`${wr >= GATE_WR ? "✅" : "❌"} WR ${wr.toFixed(1)}% (gate ≥${GATE_WR}%)`);
console.log(`\nMetrics:`);
console.log(`  Win/Loss: ${wins.length} / ${losses.length}`);
console.log(`  Avg W: ${f(avgW)}% · Avg L: ${f(avgL)}%`);
console.log(`  Net PnL: $${f(netUsd)} · Fees: $${netFees.toFixed(2)}`);
console.log(`  Best: ${f(best)}% · Worst: ${f(worst)}%`);
console.log(`\nExit breakdown:`);
for (const [k, v] of Object.entries(exits)) {
  console.log(`  ${k}: ${v.n} · avg ${f(v.sumPct / v.n)}%`);
}

const verdict = closed.length >= GATE_SAMPLE && pf >= GATE_PF && wr >= GATE_WR;
console.log(`\nVerdict: ${closed.length < GATE_SAMPLE ? "🟡 PENDING — need " + (GATE_SAMPLE - closed.length) + " more trades" : verdict ? "🟢 PASS — edge confirmed" : "🔴 FAIL — strategy not profitable enough"}`);
console.log("");
