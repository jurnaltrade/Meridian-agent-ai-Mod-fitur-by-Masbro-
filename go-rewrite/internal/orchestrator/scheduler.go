package orchestrator

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"meridian-go-rewrite/internal/agent"
	"meridian-go-rewrite/internal/config"
	"meridian-go-rewrite/internal/logger"
)

var (
	managementBusy atomic.Bool
	screeningBusy  atomic.Bool
	cronTasks      []*cron.Cron
	pnlPollStop    chan struct{}
)

var (
	ManagementLastRun time.Time
	ScreeningLastRun  time.Time
)

func StartCronJobs(cfg *config.Config) {
	StopCronJobs()

	tasks := cron.New(cron.WithSeconds())

	mgmtInterval := cfg.Schedule.ManagementIntervalMin
	tasks.AddFunc(fmt.Sprintf("@every %dm", mgmtInterval), func() {
		if managementBusy.Load() {
			return
		}
		managementBusy.Store(true)
		ManagementLastRun = time.Now()
		defer managementBusy.Store(false)
		logger.Log("cron", "Starting management cycle")
		runManagementCycle(cfg, false)
	})

	screenInterval := cfg.Schedule.ScreeningIntervalMin
	tasks.AddFunc(fmt.Sprintf("@every %dm", screenInterval), func() {
		if screeningBusy.Load() {
			return
		}
		screeningBusy.Store(true)
		ScreeningLastRun = time.Now()
		defer screeningBusy.Store(false)
		logger.Log("cron", "Starting screening cycle")
		runScreeningCycle(cfg, false)
	})

	healthInterval := cfg.Schedule.HealthCheckIntervalMin
	tasks.AddFunc(fmt.Sprintf("@every %dm", healthInterval), func() {
		if managementBusy.Load() {
			return
		}
		managementBusy.Store(true)
		defer managementBusy.Store(false)
		logger.Log("cron", "Starting health check")
		agent.AgentLoop("\nHEALTH CHECK\nSummarize the current portfolio health and total fees earned.\n", cfg.LLM.MaxSteps, nil, "MANAGER", "", 0, false, nil)
	})

	tasks.AddFunc("0 1 * * *", func() {
		logger.Log("cron", "Starting morning briefing")
		runBriefing(cfg)
	})

	tasks.Start()
	cronTasks = append(cronTasks, tasks)

	pnlPollInterval := time.Duration(cfg.Schedule.PnlPollIntervalSec) * time.Second
	pnlPollStop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(pnlPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if managementBusy.Load() || screeningBusy.Load() {
					continue
				}
				pollPnL(cfg)
			case <-pnlPollStop:
				return
			}
		}
	}()

	logger.Log("cron", fmt.Sprintf("Cycles started — management %dm, screening %dm, health %dm, PnL poll %ds",
		mgmtInterval, screenInterval, healthInterval, cfg.Schedule.PnlPollIntervalSec))
}

func StopCronJobs() {
	for _, c := range cronTasks {
		c.Stop()
	}
	cronTasks = nil
	if pnlPollStop != nil {
		close(pnlPollStop)
		pnlPollStop = nil
	}
}

func runManagementCycle(cfg *config.Config, silent bool) {
	logger.Log("cron", "Management cycle — position evaluation")
	result, err := agent.AgentLoop(
		`MANAGEMENT CYCLE
You have open positions to evaluate. For each position:
- If there's a CLOSE action: call close_position with the rule name as the reason.
- If there's a CLAIM action: call claim_fees.
- If STAY: no action needed — report that all positions are healthy.
First call get_my_positions to check current state, then act accordingly.
Be concise — one line per action.
`,
		cfg.LLM.MaxSteps, nil, "MANAGER", "", 0, false, nil)
	if err != nil {
		logger.Error("cron", err)
		return
	}
	logger.Log("cron", "Management cycle complete")
	_ = result
}

func runScreeningCycle(cfg *config.Config, silent bool) {
	logger.Log("cron", "Screening cycle — pool discovery")

	goal := fmt.Sprintf(`SCREENING CYCLE
Find and deploy into the best candidate.
Call get_top_candidates(limit=10) first, then evaluate candidates.
If one is clearly worth it, call deploy_position with:
- amount_y = compute_deploy_amount from wallet
- strategy = the active strategy or "bid_ask"
- bins_below based on volatility (round(%d + (volatility/5) × %d))
- bins_above = 0
- pool_name, base_mint, bin_step, volatility, fee_tvl_ratio, organic_score from the candidate
If no pool qualifies, report ⛔ NO DEPLOY with clear reasoning.
If deploy succeeds, report 🚀 DEPLOYED with range and pool metrics.
`,
		cfg.Strategy.MinBinsBelow, cfg.Strategy.MaxBinsBelow-cfg.Strategy.MinBinsBelow)

	result, err := agent.AgentLoop(goal, cfg.LLM.MaxSteps, nil, "SCREENER", "", 0, false, nil)
	if err != nil {
		logger.Error("cron", err)
		return
	}
	logger.Log("cron", "Screening cycle complete")
	_ = result
}

func runBriefing(cfg *config.Config) {
	goal := `MORNING BRIEFING
Generate a morning briefing for the last 24h. Include:
- Positions opened and closed
- Net PnL and fees earned
- Win rate
- Current portfolio status (open positions count)
Format as clean HTML with <b> for section headers. Keep it under 4000 chars for Telegram.
`
	result, err := agent.AgentLoop(goal, cfg.LLM.MaxSteps, nil, "GENERAL", "", 2048, false, nil)
	if err != nil {
		logger.Error("briefing", err)
		return
	}
	logger.Log("briefing", "Briefing generated")
	_ = result
}

func pollPnL(cfg *config.Config) {
	logger.Log("pnl", "PnL poll tick")
}

func NextRunIn(lastRun time.Time, intervalMin int) string {
	if lastRun.IsZero() {
		return "now"
	}
	elapsed := time.Since(lastRun)
	remaining := time.Duration(intervalMin)*time.Minute - elapsed
	if remaining <= 0 {
		return "now"
	}
	m := int(remaining.Minutes())
	s := int(remaining.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func ManagementBusy() bool {
	return managementBusy.Load()
}

func ScreeningBusy() bool {
	return screeningBusy.Load()
}
