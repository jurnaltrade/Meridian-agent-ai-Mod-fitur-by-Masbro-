package persistence

import (
	"fmt"
	"math"
	"time"

	"meridian-go-rewrite/internal/solana/types"
)

func roundFloat(val float64, decimals int) float64 {
	pow := math.Pow10(decimals)
	return math.Round(val*pow) / pow
}

type PoolMemoryStore struct {
	store *Store[types.PoolMemoryData]
}

func NewPoolMemoryStore(path string) (*PoolMemoryStore, error) {
	initial := make(types.PoolMemoryData)
	s, err := NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &PoolMemoryStore{store: s}, nil
}

func (pm *PoolMemoryStore) RecallForPool(poolAddress string) string {
	data := pm.store.Read()
	entry, ok := data[poolAddress]
	if !ok {
		return ""
	}
	var lines []string
	if entry.TotalDeploys > 0 {
		lines = append(lines, fmt.Sprintf("POOL MEMORY [%s]: %d past deploy(s), avg PnL %.2f%%, win rate %.2f%%",
			entry.Name, entry.TotalDeploys, entry.AvgPnLPct, entry.WinRate))
	}
	if entry.CooldownUntil != "" {
		t, err := time.Parse(time.RFC3339, entry.CooldownUntil)
		if err == nil && time.Now().Before(t) {
			lines = append(lines, "POOL COOLDOWN: active until "+entry.CooldownUntil)
		}
	}
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}

func (pm *PoolMemoryStore) RecordDeploy(poolAddr string, record types.DeployRecord) {
	pm.store.Update(func(data *types.PoolMemoryData) error {
		entry := (*data)[poolAddr]
		entry.Deploys = append(entry.Deploys, record)
		entry.TotalDeploys = len(entry.Deploys)
		entry.LastDeployedAt = record.ClosedAt
		if record.PnLPct >= 0 {
			entry.LastOutcome = "profit"
		} else {
			entry.LastOutcome = "loss"
		}
		var sum float64
		wins := 0
		for _, d := range entry.Deploys {
			sum += d.PnLPct
			if d.PnLPct >= 0 {
				wins++
			}
		}
		withPnl := len(entry.Deploys)
		if withPnl > 0 {
			entry.AvgPnLPct = roundFloat(sum/float64(withPnl), 2)
			entry.WinRate = roundFloat(float64(wins)/float64(withPnl)*100, 2)
		}
		(*data)[poolAddr] = entry
		return nil
	})
}

func (pm *PoolMemoryStore) RecordSnapshot(poolAddr string, snap types.PositionSnapshot) {
	pm.store.Update(func(data *types.PoolMemoryData) error {
		entry := (*data)[poolAddr]
		entry.Snapshots = append(entry.Snapshots, snap)
		if len(entry.Snapshots) > 48 {
			entry.Snapshots = entry.Snapshots[len(entry.Snapshots)-48:]
		}
		(*data)[poolAddr] = entry
		return nil
	})
}
