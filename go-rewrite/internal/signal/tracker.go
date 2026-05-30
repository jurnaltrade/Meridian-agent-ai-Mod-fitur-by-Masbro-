package signal

import (
	"sync"
	"time"
)

type StagedSignal struct {
	Pool           string    `json:"pool"`
	PoolName       string    `json:"pool_name"`
	BaseMint       string    `json:"base_mint"`
	BinStep        int       `json:"bin_step"`
	FeePct         float64   `json:"base_fee_percentage"`
	Volatility     float64   `json:"volatility"`
	FeeTVLRatio    float64   `json:"fee_tvl_ratio"`
	OrganicScore   float64   `json:"organic_score"`
	TVL            float64   `json:"tvl"`
	Volume         float64   `json:"volume"`
	Holders        int       `json:"holders"`
	MCap           float64   `json:"mcap"`
	SmartWallets   int       `json:"smart_wallets_present"`
	NarrativeScore float64   `json:"narrative_score"`
	StudyWinRate   float64   `json:"study_win_rate"`
	HiveConsensus  float64   `json:"hive_consensus"`
	StagedAt       time.Time `json:"staged_at"`
}

type Tracker struct {
	mu    sync.Mutex
	pools map[string]StagedSignal
}

func NewTracker() *Tracker {
	return &Tracker{
		pools: make(map[string]StagedSignal),
	}
}

func (t *Tracker) Stage(s StagedSignal) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s.StagedAt = time.Now()
	t.pools[s.Pool] = s
}

func (t *Tracker) GetAndClear(pool string) *StagedSignal {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.pools[pool]
	if !ok {
		return nil
	}
	if time.Since(s.StagedAt) > 10*time.Minute {
		delete(t.pools, pool)
		return nil
	}
	delete(t.pools, pool)
	return &s
}

func (t *Tracker) PurgeExpired() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, s := range t.pools {
		if time.Since(s.StagedAt) > 10*time.Minute {
			delete(t.pools, k)
		}
	}
}
