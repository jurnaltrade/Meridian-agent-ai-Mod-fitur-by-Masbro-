# Meridian — Saran Perbaikan

## Portfolio SL Protection (belum diimplementasi)

**Status:** Menunggu fork GitHub repo  
**Repo:** https://github.com/yunus-0x/meridian  
**Priority:** Medium

### Problem
Saat ini Meridian hanya punya **per-position SL** (default -50%, user set -20%).  
Tidak ada **portfolio-level SL** yang memantau total wallet + semua posisi terbuka.

### User Need
- Per-position SL: -20% (sudah di-set di `user-config.json`)
- Portfolio SL: -10% dari total portfolio value (wallet + open positions)

### Implementation Notes
1. **Track initial portfolio value** saat pertama kali start / saat deploy pertama
2. **Di setiap management cycle** (tiap 10 menit), hitung:
   - `currentValue = walletBalance + Σ(position.total_value_usd)`
   - `portfolioPnl = (currentValue - initialValue) / initialValue * 100`
3. **Kalau portfolioPnl ≤ -10%** → EMERGENCY CLOSE ALL positions
4. Simpan `initialValue` di state file (persist across restarts)

### Edge Cases
- Posisi baru dibuka → initialValue harus update
- Fee sudah di-claim → wallet balance naik, jangan false trigger
- Posisi partial close → handle partial value
- DRY_RUN mode → jangan close beneran

### Relevant Code
- `index.js:933` — per-position SL check
- `index.js:281` — `totalValue` calculation (reporting only)
- `tools/executor.js:378` — `stopLossPct` config mapping

---

## Dry Run Bug Fix (sudah di-fix 2026-06-02)

**Bug:** `DRY_RUN` precedence salah, bot deploy 0.21 SOL beneran saat dry run  
**Root cause:** 
- `.env` → `DRY_RUN=false` (loaded duluan via dotenv)
- `ecosystem.config.cjs` → `DRY_RUN: "false"` (string = truthy)
- `config.js` → `||=` gak bisa override

**Fix:** Ubah di 3 tempat: `.env`, `ecosystem.config.cjs`, `user-config.json`
