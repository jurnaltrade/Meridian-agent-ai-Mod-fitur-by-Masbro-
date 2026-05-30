package signal

import (
	"fmt"
	"strings"
	"time"

	"meridian-go-rewrite/internal/persistence"
	"meridian-go-rewrite/internal/solana/types"
)

type Weights struct {
	store *persistence.Store[types.SignalWeightsData]
}

func NewWeights(path string) (*Weights, error) {
	initial := types.SignalWeightsData{
		Weights: defaultWeights(),
	}
	s, err := persistence.NewStore(path, initial)
	if err != nil {
		return nil, err
	}
	return &Weights{store: s}, nil
}

func defaultWeights() map[string]float64 {
	return map[string]float64{
		"organic_score":         1.0,
		"fee_tvl_ratio":         1.0,
		"volume":                1.0,
		"mcap":                  1.0,
		"holder_count":          1.0,
		"smart_wallets_present": 1.0,
		"narrative_quality":     1.0,
		"study_win_rate":        1.0,
		"hive_consensus":        1.0,
		"volatility":            1.0,
	}
}

func (w *Weights) GetWeights() map[string]float64 {
	return w.store.Read().Weights
}

func (w *Weights) Recalculate(performances []types.Performance) {
	if len(performances) < 5 {
		return
	}

	wins := make([]types.Performance, 0)
	losses := make([]types.Performance, 0)
	for _, p := range performances {
		if p.PnLPct >= 0 {
			wins = append(wins, p)
		} else {
			losses = append(losses, p)
		}
	}
	if len(wins) < 2 || len(losses) < 2 {
		return
	}

	changes := make([]types.SignalWeightChange, 0)
	weights := w.store.Read().Weights

	signals := []struct {
		key    string
		getter func(types.Performance) float64
	}{
		{"organic_score", func(p types.Performance) float64 { return p.OrganicScore }},
		{"fee_tvl_ratio", func(p types.Performance) float64 { return p.FeeTVLRatio }},
		{"volume", func(p types.Performance) float64 { return p.SignalSnapshot["volume"].(float64) }},
		{"mcap", func(p types.Performance) float64 { return p.SignalSnapshot["mcap"].(float64) }},
		{"holder_count", func(p types.Performance) float64 { return p.SignalSnapshot["holder_count"].(float64) }},
		{"volatility", func(p types.Performance) float64 { return p.Volatility }},
	}

	for _, sig := range signals {
		winAvg, lossAvg := avgSignal(sig.getter, wins, losses)
		if winAvg == 0 && lossAvg == 0 {
			continue
		}
		oldWeight := weights[sig.key]
		if oldWeight == 0 {
			oldWeight = 1.0
		}

		lift := 0.0
		if (winAvg + lossAvg) > 0 {
			lift = (winAvg - lossAvg) / (winAvg + lossAvg)
		}

		newWeight := oldWeight
		if lift > 0.1 {
			newWeight = oldWeight * 1.05
		} else if lift < -0.05 {
			newWeight = oldWeight * 0.95
		}

		if newWeight < 0.3 {
			newWeight = 0.3
		}
		if newWeight > 2.5 {
			newWeight = 2.5
		}

		if newWeight != oldWeight {
			changes = append(changes, types.SignalWeightChange{
				Signal: sig.key, From: oldWeight, To: newWeight,
				Lift: lift, Action: actionForLift(lift),
			})
			weights[sig.key] = newWeight
		}
	}

	boolSignals := []struct {
		key    string
		getter func(types.Performance) float64
	}{
		{"smart_wallets_present", func(p types.Performance) float64 {
			if v, ok := p.SignalSnapshot["smart_wallets_present"]; ok {
				if b, ok := v.(bool); ok && b {
					return 1
				}
			}
			return 0
		}},
		{"narrative_quality", func(p types.Performance) float64 {
			if v, ok := p.SignalSnapshot["narrative_quality"]; ok {
				if f, ok := v.(float64); ok {
					return f
				}
			}
			return 0
		}},
		{"study_win_rate", func(p types.Performance) float64 {
			if v, ok := p.SignalSnapshot["study_win_rate"]; ok {
				if f, ok := v.(float64); ok {
					return f
				}
			}
			return 0
		}},
		{"hive_consensus", func(p types.Performance) float64 {
			if v, ok := p.SignalSnapshot["hive_consensus"]; ok {
				if f, ok := v.(float64); ok {
					return f
				}
			}
			return 0
		}},
	}

	for _, sig := range boolSignals {
		winAvg, lossAvg := avgSignal(sig.getter, wins, losses)
		oldWeight := weights[sig.key]
		if oldWeight == 0 {
			oldWeight = 1.0
		}

		var lift float64
		if lossAvg > 0 {
			lift = (winAvg/lossAvg - 1)
		} else if winAvg > 0 {
			lift = 1.0
		}

		newWeight := oldWeight
		if lift > 0.1 {
			newWeight = oldWeight * 1.05
		} else if lift < -0.05 {
			newWeight = oldWeight * 0.95
		}

		if newWeight < 0.3 {
			newWeight = 0.3
		}
		if newWeight > 2.5 {
			newWeight = 2.5
		}

		if newWeight != oldWeight {
			changes = append(changes, types.SignalWeightChange{
				Signal: sig.key, From: oldWeight, To: newWeight,
				Lift: lift, Action: actionForLift(lift),
			})
			weights[sig.key] = newWeight
		}
	}

	if len(changes) > 0 {
		w.store.Update(func(data *types.SignalWeightsData) error {
			data.Weights = weights
			data.LastRecalc = time.Now().Format(time.RFC3339)
			data.RecalcCount++
			data.History = append(data.History, types.SignalWeightHistory{
				Timestamp:  time.Now().Format(time.RFC3339),
				Changes:    changes,
				WindowSize: len(performances),
				WinCount:   len(wins),
				LossCount:  len(losses),
			})
			if len(data.History) > 50 {
				data.History = data.History[len(data.History)-50:]
			}
			return nil
		})
	}
}

func avgSignal(getter func(types.Performance) float64, wins, losses []types.Performance) (winAvg, lossAvg float64) {
	for _, p := range wins {
		winAvg += getter(p)
	}
	for _, p := range losses {
		lossAvg += getter(p)
	}
	if len(wins) > 0 {
		winAvg /= float64(len(wins))
	}
	if len(losses) > 0 {
		lossAvg /= float64(len(losses))
	}
	return winAvg, lossAvg
}

func actionForLift(lift float64) string {
	if lift > 0.2 {
		return "boosted"
	}
	if lift < -0.1 {
		return "decayed"
	}
	return "neutral"
}

func (w *Weights) GetSummary() string {
	weights := w.GetWeights()
	var lines []string
	for name, val := range weights {
		bar := ""
		bars := int(val * 5)
		if bars > 20 {
			bars = 20
		}
		for i := 0; i < bars; i++ {
			bar += "█"
		}
		label := strings.ReplaceAll(name, "_", " ")
		lines = append(lines, fmt.Sprintf("  %-22s %4.2f %s", label, val, bar))
	}
	return strings.Join(lines, "\n")
}
