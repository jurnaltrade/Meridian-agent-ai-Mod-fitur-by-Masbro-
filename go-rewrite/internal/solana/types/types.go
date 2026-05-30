package types

type Performance struct {
	Position        string         `json:"position"`
	Pool            string         `json:"pool"`
	PoolName        string         `json:"pool_name"`
	BaseMint        string         `json:"base_mint"`
	Strategy        string         `json:"strategy"`
	BinRange        BinRange       `json:"bin_range"`
	BinStep         int            `json:"bin_step"`
	Volatility      float64        `json:"volatility"`
	FeeTVLRatio     float64        `json:"fee_tvl_ratio"`
	OrganicScore    float64        `json:"organic_score"`
	AmountSOL       float64        `json:"amount_sol"`
	FeesEarnedUSD   float64        `json:"fees_earned_usd"`
	FinalValueUSD   float64        `json:"final_value_usd"`
	InitialValueUSD float64        `json:"initial_value_usd"`
	MinutesInRange  float64        `json:"minutes_in_range"`
	MinutesHeld     float64        `json:"minutes_held"`
	CloseReason     string         `json:"close_reason"`
	SignalSnapshot  map[string]any `json:"signal_snapshot"`
	PnLUSD          float64        `json:"pnl_usd"`
	PnLPct          float64        `json:"pnl_pct"`
	RangeEfficiency float64        `json:"range_efficiency"`
	DeployedAt      string         `json:"deployed_at"`
	RecordedAt      string         `json:"recorded_at"`
}

type BinRange struct {
	Min       int `json:"min"`
	Max       int `json:"max"`
	BinsBelow int `json:"bins_below"`
	BinsAbove int `json:"bins_above"`
	Active    int `json:"active,omitempty"`
}

type PositionState struct {
	Position          string         `json:"position"`
	Pool              string         `json:"pool"`
	PoolName          string         `json:"pool_name"`
	Strategy          string         `json:"strategy"`
	BinRange          BinRange       `json:"bin_range"`
	AmountSOL         float64        `json:"amount_sol"`
	AmountX           float64        `json:"amount_x"`
	ActiveBinAtDeploy int            `json:"active_bin_at_deploy"`
	BinStep           int            `json:"bin_step"`
	Volatility        float64        `json:"volatility"`
	FeeTVLRatio       float64        `json:"fee_tvl_ratio"`
	InitialFeeTVL24h  float64        `json:"initial_fee_tvl_24h"`
	OrganicScore      float64        `json:"organic_score"`
	InitialValueUSD   float64        `json:"initial_value_usd"`
	DeployedAt        string         `json:"deployed_at"`
	OORSince          string         `json:"out_of_range_since"`
	LastClaimAt       string         `json:"last_claim_at"`
	TotalFeesClaimed  float64        `json:"total_fees_claimed_usd"`
	RebalanceCount    int            `json:"rebalance_count"`
	Closed            bool           `json:"closed"`
	ClosedAt          string         `json:"closed_at"`
	Notes             []string       `json:"notes"`
	PeakPnLPct        float64        `json:"peak_pnl_pct"`
	PendingPeakPct    float64        `json:"pending_peak_pnl_pct"`
	PendingPeakAt     string         `json:"pending_peak_started_at"`
	TrailingActive    bool           `json:"trailing_active"`
	Instruction       string         `json:"instruction"`
	BaseMint          string         `json:"base_mint"`
	SignalSnapshot    map[string]any `json:"signal_snapshot"`
}

type Lesson struct {
	ID              int      `json:"id"`
	Rule            string   `json:"rule"`
	Tags            []string `json:"tags"`
	Outcome         string   `json:"outcome"`
	SourceType      string   `json:"sourceType"`
	Pinned          bool     `json:"pinned"`
	Role            string   `json:"role"`
	Confidence      float64  `json:"confidence,omitempty"`
	CreatedAt       string   `json:"created_at"`
	Pool            string   `json:"pool,omitempty"`
	PnLPct          float64  `json:"pnl_pct,omitempty"`
	FeesUSD         float64  `json:"fees_earned_usd,omitempty"`
	InitialValue    float64  `json:"initial_value_usd,omitempty"`
	RangeEfficiency float64  `json:"range_efficiency,omitempty"`
	CloseReason     string   `json:"close_reason,omitempty"`
}

type LessonsData struct {
	Lessons     []Lesson      `json:"lessons"`
	Performance []Performance `json:"performance"`
}

type StateData struct {
	Positions        map[string]PositionState `json:"positions"`
	RecentEvents     []StateEvent             `json:"recentEvents"`
	LastUpdated      string                   `json:"lastUpdated"`
	LastBriefingDate string                   `json:"_lastBriefingDate"`
}

type StateEvent struct {
	TS       string `json:"ts"`
	Action   string `json:"action"`
	Position string `json:"position"`
	PoolName string `json:"pool_name"`
	Reason   string `json:"reason,omitempty"`
}

type PoolMemoryData map[string]PoolMemoryEntry

type PoolMemoryEntry struct {
	Name                   string             `json:"name"`
	BaseMint               string             `json:"base_mint"`
	Deploys                []DeployRecord     `json:"deploys"`
	TotalDeploys           int                `json:"total_deploys"`
	AvgPnLPct              float64            `json:"avg_pnl_pct"`
	WinRate                float64            `json:"win_rate"`
	AdjustedWinRate        float64            `json:"adjusted_win_rate"`
	AdjustedSampleCount    int                `json:"adjusted_win_rate_sample_count"`
	LastDeployedAt         string             `json:"last_deployed_at"`
	LastOutcome            string             `json:"last_outcome"`
	Notes                  []PoolNote         `json:"notes"`
	Snapshots              []PositionSnapshot `json:"snapshots"`
	CooldownUntil          string             `json:"cooldown_until"`
	CooldownReason         string             `json:"cooldown_reason"`
	BaseMintCooldownUntil  string             `json:"base_mint_cooldown_until"`
	BaseMintCooldownReason string             `json:"base_mint_cooldown_reason"`
}

type DeployRecord struct {
	DeployedAt         string  `json:"deployed_at"`
	ClosedAt           string  `json:"closed_at"`
	PnLPct             float64 `json:"pnl_pct"`
	PnLUSD             float64 `json:"pnl_usd"`
	FeesEarnedUSD      float64 `json:"fees_earned_usd"`
	FeesEarnedSOL      float64 `json:"fees_earned_sol"`
	FeeEarnedPct       float64 `json:"fee_earned_pct"`
	RangeEfficiency    float64 `json:"range_efficiency"`
	MinutesHeld        float64 `json:"minutes_held"`
	CloseReason        string  `json:"close_reason"`
	Strategy           string  `json:"strategy"`
	VolatilityAtDeploy float64 `json:"volatility_at_deploy"`
}

type PoolNote struct {
	Note    string `json:"note"`
	AddedAt string `json:"added_at"`
}

type PositionSnapshot struct {
	TS            string  `json:"ts"`
	Position      string  `json:"position"`
	PnLPct        float64 `json:"pnl_pct"`
	PnLUSD        float64 `json:"pnl_usd"`
	InRange       bool    `json:"in_range"`
	UnclaimedFees float64 `json:"unclaimed_fees_usd"`
	OORMinutes    int     `json:"minutes_out_of_range"`
	AgeMinutes    int     `json:"age_minutes"`
}

type StrategyData struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Author        string         `json:"author"`
	LPStrategy    string         `json:"lp_strategy"`
	TokenCriteria map[string]any `json:"token_criteria"`
	Entry         map[string]any `json:"entry"`
	Range         map[string]any `json:"range"`
	Exit          map[string]any `json:"exit"`
	BestFor       string         `json:"best_for"`
	Raw           string         `json:"raw"`
	AddedAt       string         `json:"added_at"`
	UpdatedAt     string         `json:"updated_at"`
}

type StrategyLibraryData struct {
	Active     string                  `json:"active"`
	Strategies map[string]StrategyData `json:"strategies"`
}

type Decision struct {
	ID       string         `json:"id"`
	TS       string         `json:"ts"`
	Type     string         `json:"type"`
	Actor    string         `json:"actor"`
	Pool     string         `json:"pool"`
	PoolName string         `json:"pool_name"`
	Position string         `json:"position"`
	Summary  string         `json:"summary"`
	Reason   string         `json:"reason"`
	Risks    []string       `json:"risks"`
	Metrics  map[string]any `json:"metrics"`
	Rejected []string       `json:"rejected"`
}

type DecisionLogData struct {
	Decisions []Decision `json:"decisions"`
}

type SmartWallet struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Category string `json:"category"`
	Type     string `json:"type"`
	AddedAt  string `json:"addedAt"`
}

type SmartWalletData struct {
	Wallets []SmartWallet `json:"wallets"`
}

type TokenBlacklistData struct {
	Blacklist []BlacklistEntry `json:"blacklist"`
}

type BlacklistEntry struct {
	Mint    string `json:"mint"`
	Symbol  string `json:"symbol"`
	Reason  string `json:"reason"`
	AddedAt string `json:"added_at"`
}

type DeployerBlacklistData struct {
	Blocked map[string]DeployerBlockEntry `json:"blocked"`
}

type DeployerBlockEntry struct {
	Wallet  string `json:"wallet"`
	Label   string `json:"label"`
	Reason  string `json:"reason"`
	AddedAt string `json:"added_at"`
}

type SignalWeightsData struct {
	Weights     map[string]float64    `json:"weights"`
	LastRecalc  string                `json:"last_recalc"`
	RecalcCount int                   `json:"recalc_count"`
	History     []SignalWeightHistory `json:"history"`
}

type SignalWeightHistory struct {
	Timestamp  string               `json:"timestamp"`
	Changes    []SignalWeightChange `json:"changes"`
	WindowSize int                  `json:"window_size"`
	WinCount   int                  `json:"win_count"`
	LossCount  int                  `json:"loss_count"`
}

type SignalWeightChange struct {
	Signal string  `json:"signal"`
	From   float64 `json:"from"`
	To     float64 `json:"to"`
	Lift   float64 `json:"lift"`
	Action string  `json:"action"`
}
