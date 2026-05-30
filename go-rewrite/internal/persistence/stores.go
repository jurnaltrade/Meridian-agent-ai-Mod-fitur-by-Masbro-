package persistence

import (
	"fmt"
	"strings"
	"time"

	"meridian-go-rewrite/internal/solana/types"
)

type LessonsStore struct {
	store *Store[types.LessonsData]
}

func NewLessonsStore(path string) (*LessonsStore, error) {
	initial := types.LessonsData{
		Lessons:     make([]types.Lesson, 0),
		Performance: make([]types.Performance, 0),
	}
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &LessonsStore{store: s}, nil
}

func (ls *LessonsStore) AddLesson(lesson types.Lesson) {
	ls.store.Update(func(data *types.LessonsData) error {
		data.Lessons = append(data.Lessons, lesson)
		if len(data.Lessons) > 500 {
			data.Lessons = data.Lessons[len(data.Lessons)-500:]
		}
		return nil
	})
}

func (ls *LessonsStore) AddPerformance(perf types.Performance) {
	ls.store.Update(func(data *types.LessonsData) error {
		data.Performance = append(data.Performance, perf)
		if len(data.Performance) > 1000 {
			data.Performance = data.Performance[len(data.Performance)-1000:]
		}
		return nil
	})
}

func (ls *LessonsStore) GetPerformanceSummary() *map[string]any {
	data := ls.store.Read()
	if len(data.Performance) == 0 {
		return nil
	}
	var totalPnL, totalPnlPct, totalRangeEff float64
	wins := 0
	for _, p := range data.Performance {
		totalPnL += p.PnLUSD
		totalPnlPct += p.PnLPct
		totalRangeEff += p.RangeEfficiency
		if p.PnLUSD > 0 {
			wins++
		}
	}
	n := float64(len(data.Performance))
	summary := map[string]any{
		"total_positions_closed":   len(data.Performance),
		"total_pnl_usd":            totalPnL,
		"avg_pnl_pct":              roundFloat(totalPnlPct/n, 2),
		"avg_range_efficiency_pct": roundFloat(totalRangeEff/n, 1),
		"win_rate_pct":             roundFloat(float64(wins)/n*100, 0),
		"total_lessons":            len(data.Lessons),
	}
	return &summary
}

func (ls *LessonsStore) GetLessonsForPrompt(agentType string, maxLessons int) string {
	data := ls.store.Read()
	if len(data.Lessons) == 0 {
		return ""
	}
	var lines []string
	added := 0
	for i := len(data.Lessons) - 1; i >= 0 && added < maxLessons; i-- {
		l := data.Lessons[i]
		if l.Role != "" && l.Role != agentType && agentType != "GENERAL" {
			continue
		}
		date := l.CreatedAt
		if len(date) > 16 {
			date = date[:16]
		}
		pin := ""
		if l.Pinned {
			pin = "📌 "
		}
		lines = append(lines, pin+"["+strings.ToUpper(l.Outcome)+"] ["+date+"] "+l.Rule)
		added++
	}
	return strings.Join(lines, "\n")
}

func (ls *LessonsStore) GetRecentPerformance(limit int) []types.Performance {
	data := ls.store.Read()
	p := data.Performance
	if len(p) > limit {
		return p[len(p)-limit:]
	}
	return p
}

type DecisionLogStore struct {
	store *Store[types.DecisionLogData]
}

func NewDecisionLogStore(path string) (*DecisionLogStore, error) {
	initial := types.DecisionLogData{Decisions: make([]types.Decision, 0)}
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &DecisionLogStore{store: s}, nil
}

func (dl *DecisionLogStore) Append(decision types.Decision) {
	dl.store.Update(func(data *types.DecisionLogData) error {
		data.Decisions = append([]types.Decision{decision}, data.Decisions...)
		if len(data.Decisions) > 100 {
			data.Decisions = data.Decisions[:100]
		}
		return nil
	})
}

func (dl *DecisionLogStore) GetRecent(limit int) []types.Decision {
	data := dl.store.Read()
	if len(data.Decisions) > limit {
		return data.Decisions[:limit]
	}
	return data.Decisions
}

func (dl *DecisionLogStore) GetDecisionSummary(limit int) string {
	decisions := dl.GetRecent(limit)
	if len(decisions) == 0 {
		return "No recent structured decisions yet."
	}
	var lines []string
	for i, d := range decisions {
		line := fmt.Sprintf("%d. [%s] %s %s", i+1, d.Actor, strings.ToUpper(d.Type), d.PoolName)
		if d.Summary != "" {
			line += fmt.Sprintf(" | summary: %s", d.Summary)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

type StrategyLibraryStore struct {
	store *Store[types.StrategyLibraryData]
}

func NewStrategyLibraryStore(path string) (*StrategyLibraryStore, error) {
	initial := types.StrategyLibraryData{
		Strategies: defaultStrategies(),
		Active:     "custom_ratio_spot",
	}
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &StrategyLibraryStore{store: s}, nil
}

func defaultStrategies() map[string]types.StrategyData {
	return map[string]types.StrategyData{
		"custom_ratio_spot":   {ID: "custom_ratio_spot", Name: "Custom Ratio Spot", Author: "meridian", LPStrategy: "spot", BestFor: "Expressing directional bias while earning fees"},
		"single_sided_reseed": {ID: "single_sided_reseed", Name: "Single-Sided Bid-Ask", Author: "meridian", LPStrategy: "bid_ask", BestFor: "Riding volatile tokens down"},
		"fee_compounding":     {ID: "fee_compounding", Name: "Fee Compounding", Author: "meridian", LPStrategy: "any", BestFor: "Maximizing yield on stable pools"},
		"multi_layer":         {ID: "multi_layer", Name: "Multi-Layer", Author: "meridian", LPStrategy: "mixed", BestFor: "Custom liquidity distributions"},
		"partial_harvest":     {ID: "partial_harvest", Name: "Partial Harvest", Author: "meridian", LPStrategy: "any", BestFor: "Progressive profit-taking"},
	}
}

func (sl *StrategyLibraryStore) GetActiveStrategy() *types.StrategyData {
	data := sl.store.Read()
	if s, ok := data.Strategies[data.Active]; ok {
		return &s
	}
	return nil
}

func (sl *StrategyLibraryStore) ListStrategies() []types.StrategyData {
	data := sl.store.Read()
	result := make([]types.StrategyData, 0, len(data.Strategies))
	for _, s := range data.Strategies {
		result = append(result, s)
	}
	return result
}

func (sl *StrategyLibraryStore) SetActive(id string) error {
	return sl.store.Update(func(data *types.StrategyLibraryData) error {
		if _, ok := data.Strategies[id]; !ok {
			return fmt.Errorf("strategy %s not found", id)
		}
		data.Active = id
		return nil
	})
}

func (sl *StrategyLibraryStore) AddStrategy(s types.StrategyData) error {
	return sl.store.Update(func(data *types.StrategyLibraryData) error {
		data.Strategies[s.ID] = s
		return nil
	})
}

type TokenBlacklistStore struct {
	store *Store[types.TokenBlacklistData]
}

func NewTokenBlacklistStore(path string) (*TokenBlacklistStore, error) {
	initial := types.TokenBlacklistData{Blacklist: make([]types.BlacklistEntry, 0)}
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &TokenBlacklistStore{store: s}, nil
}

func (tb *TokenBlacklistStore) IsBlacklisted(mint string) bool {
	data := tb.store.Read()
	for _, e := range data.Blacklist {
		if e.Mint == mint {
			return true
		}
	}
	return false
}

func (tb *TokenBlacklistStore) Add(mint, symbol, reason string) {
	tb.store.Update(func(data *types.TokenBlacklistData) error {
		data.Blacklist = append(data.Blacklist, types.BlacklistEntry{
			Mint: mint, Symbol: symbol, Reason: reason, AddedAt: time.Now().Format(time.RFC3339),
		})
		return nil
	})
}

type SmartWalletStore struct {
	store *Store[types.SmartWalletData]
}

func NewSmartWalletStore(path string) (*SmartWalletStore, error) {
	initial := types.SmartWalletData{Wallets: make([]types.SmartWallet, 0)}
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &SmartWalletStore{store: s}, nil
}

func (sw *SmartWalletStore) List() []types.SmartWallet {
	return sw.store.Read().Wallets
}

func (sw *SmartWalletStore) Add(wallet types.SmartWallet) error {
	return sw.store.Update(func(data *types.SmartWalletData) error {
		data.Wallets = append(data.Wallets, wallet)
		return nil
	})
}

func (sw *SmartWalletStore) Remove(address string) error {
	return sw.store.Update(func(data *types.SmartWalletData) error {
		filtered := make([]types.SmartWallet, 0)
		for _, w := range data.Wallets {
			if w.Address != address {
				filtered = append(filtered, w)
			}
		}
		data.Wallets = filtered
		return nil
	})
}
