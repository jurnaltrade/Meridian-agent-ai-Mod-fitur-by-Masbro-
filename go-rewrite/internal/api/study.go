package api

import (
	"encoding/json"
	"fmt"
	"math"

	"meridian-go-rewrite/internal/agentmeridian"
)

func StudyTopLpers(poolAddress string, limit int) (map[string]any, error) {
	if limit <= 0 {
		limit = 4
	}

	type Result struct {
		data map[string]any
		err  error
	}

	poolCh := make(chan Result)
	signalCh := make(chan Result)

	go func() {
		data, err := agentmeridian.AgentMeridianJSON("GET", fmt.Sprintf("/top-lp/%s", poolAddress), nil)
		if err != nil {
			poolCh <- Result{err: err}
			return
		}
		var d map[string]any
		json.Unmarshal(data, &d)
		poolCh <- Result{data: d}
	}()

	go func() {
		data, err := agentmeridian.AgentMeridianJSON("GET", fmt.Sprintf("/study-top-lp/%s", poolAddress), nil)
		if err != nil {
			signalCh <- Result{err: err}
			return
		}
		var d map[string]any
		json.Unmarshal(data, &d)
		signalCh <- Result{data: d}
	}()

	poolRes := <-poolCh
	signalRes := <-signalCh

	if poolRes.err != nil {
		return nil, poolRes.err
	}
	if signalRes.err != nil {
		return nil, signalRes.err
	}

	poolD := poolRes.data
	signalD := signalRes.data

	topLpers := make([]map[string]any, 0)
	if raw, ok := poolD["topLpers"].([]interface{}); ok {
		for _, v := range raw {
			if m, ok := v.(map[string]any); ok {
				topLpers = append(topLpers, m)
			}
		}
	}

	historicalOwners := make([]map[string]any, 0)
	if raw, ok := poolD["historicalOwners"].([]interface{}); ok {
		for _, v := range raw {
			if m, ok := v.(map[string]any); ok {
				historicalOwners = append(historicalOwners, m)
			}
		}
	}

	rankedLimit := limit
	if len(topLpers) < limit {
		rankedLimit = len(topLpers)
	}
	ranked := topLpers[:rankedLimit]

	if len(ranked) == 0 {
		return map[string]any{
			"pool":     poolAddress,
			"message":  "No LPAgent top LPer data found for this pool yet.",
			"patterns": map[string]any{},
			"lpers":    []any{},
		}, nil
	}

	historicalMap := make(map[string]map[string]any)
	for _, owner := range historicalOwners {
		if o, ok := owner["owner"].(string); ok {
			historicalMap[o] = owner
		}
	}

	lpers := make([]map[string]any, 0)
	for _, owner := range ranked {
		oStr, _ := owner["owner"].(string)
		history := historicalMap[oStr]

		ownerShort, _ := owner["ownerShort"].(string)
		if ownerShort == "" && len(oStr) > 8 {
			ownerShort = oStr[:8] + "..."
		}

		var tags []string
		if history != nil {
			if strat, ok := history["preferredStrategy"].(string); ok && strat != "" {
				tags = append(tags, "strategy:"+strat)
			}
			if rangeStyle, ok := history["preferredRangeStyle"].(string); ok && rangeStyle != "" {
				tags = append(tags, "range:"+rangeStyle)
			}
		}

		totalPositions := 0
		if tp, ok := owner["totalLp"].(float64); ok {
			totalPositions = int(tp)
		} else if history != nil {
			if topPos, ok := history["topPositions"].([]interface{}); ok {
				totalPositions = len(topPos)
			}
		}

		avgHoldHours := getFallbackFloat(owner, "avgAgeHours", history, "avgHoldHours")
		avgOpenPnlPct := getFallbackFloat(owner, "pnlPerInflowPct", history, "avgPnlPct")
		avgFeePerTvl24h := getFallbackFloat(owner, "feePercent", history, "avgFeePercent")

		totalPnlUsd := getFloat(owner, "totalPnlUsd")
		totalBalanceUsd := getFloat(owner, "totalInflowUsd")
		winRate := getFloat(owner, "winRatePct") / 100
		roi := getFloat(owner, "roiPct") / 100
		feePct := getFloat(owner, "feePercent")

		prefStrat := "unknown"
		prefRange := "unknown"
		if history != nil {
			if s, ok := history["preferredStrategy"].(string); ok {
				prefStrat = s
			}
			if r, ok := history["preferredRangeStyle"].(string); ok {
				prefRange = r
			}
		}

		positions := make([]map[string]any, 0)
		if history != nil {
			if topPos, ok := history["topPositions"].([]interface{}); ok {
				for _, posRaw := range topPos {
					if pos, ok := posRaw.(map[string]any); ok {
						pnlPctStr := fmt.Sprintf("%.2f%%", getFloat(pos, "pnlPct"))
						if getFloat(pos, "pnlPct") >= 0 {
							pnlPctStr = "+" + pnlPctStr
						}

						inRangePct := (*float64)(nil)
						if ir, ok := pos["inRange"].(bool); ok {
							val := 0.0
							if ir {
								val = 100.0
							}
							inRangePct = &val
						}

						overview, _ := poolD["overview"].(map[string]any)
						pair := "Unknown pool"
						if overview != nil {
							if n, ok := overview["name"].(string); ok {
								pair = n
							}
						}

						positions = append(positions, map[string]any{
							"pool":                   poolAddress,
							"pair":                   pair,
							"hold_hours":             round(getFloat(pos, "ageHours"), 2),
							"pnl_usd":                round(getFloat(pos, "pnlUsd"), 2),
							"pnl_pct":                pnlPctStr,
							"fee_usd":                round(getFloat(pos, "feeUsd"), 2),
							"in_range_pct":           inRangePct,
							"strategy":               getStringOrNil(pos, "strategy"),
							"closed_reason":          getStringOrNil(pos, "rangeStyle"),
							"balance_usd":            round(getFloat(pos, "inputValue"), 2),
							"fee_per_tvl_24h_pct":    round(getFloat(pos, "feePercent"), 2),
							"range_width_pct":        getFloatOrNil(pos, "widthBins"),
							"distance_to_active_pct": nil,
							"lower_bin_id":           getFloatOrNil(pos, "lowerBinId"),
							"upper_bin_id":           getFloatOrNil(pos, "upperBinId"),
						})
					}
				}
			}
		}

		lpers = append(lpers, map[string]any{
			"owner":       oStr,
			"owner_short": ownerShort,
			"signal_tags": tags,
			"summary": map[string]any{
				"total_positions":            totalPositions,
				"avg_hold_hours":             round(avgHoldHours, 2),
				"avg_open_pnl_pct":           round(avgOpenPnlPct, 2),
				"avg_fee_per_tvl_24h_pct":    round(avgFeePerTvl24h, 2),
				"total_pnl_usd":              round(totalPnlUsd, 2),
				"total_balance_usd":          round(totalBalanceUsd, 2),
				"avg_range_width_pct":        nil,
				"avg_distance_to_active_pct": nil,
				"win_rate":                   round(winRate, 2),
				"roi":                        round(roi, 4),
				"fee_pct_of_capital":         round(feePct, 2),
				"preferred_strategy":         prefStrat,
				"preferred_range_style":      prefRange,
			},
			"positions": positions,
		})
	}

	overview, _ := poolD["overview"].(map[string]any)
	if overview == nil {
		overview = map[string]any{}
	}

	patterns := buildPatterns(ranked, historicalOwners, signalD, overview)

	poolName := "TOKEN-SOL"
	if n, ok := overview["name"].(string); ok && n != "" {
		poolName = n
	} else {
		tx, _ := overview["tokenXSymbol"].(string)
		ty, _ := overview["tokenYSymbol"].(string)
		if tx == "" {
			tx = "TOKEN"
		}
		if ty == "" {
			ty = "SOL"
		}
		poolName = fmt.Sprintf("%s-%s", tx, ty)
	}

	return map[string]any{
		"pool":      poolAddress,
		"pool_name": poolName,
		"message":   "LPAgent-backed top LP study from Agent Meridian 30m cached owner aggregates plus owner historical positions.",
		"patterns":  patterns,
		"lpers":     lpers,
	}, nil
}

func buildPatterns(ranked, historicalOwners []map[string]any, signalData, overview map[string]any) map[string]any {
	var holdSum, openPnlSum, feeSum, roiSum float64
	var holdC, openPnlC, feeC, roiC int

	for _, o := range ranked {
		if v, ok := o["avgAgeHours"].(float64); ok {
			holdSum += v
			holdC++
		}
		if v, ok := o["pnlPerInflowPct"].(float64); ok {
			openPnlSum += v
			openPnlC++
		}
		if v, ok := o["feePercent"].(float64); ok {
			feeSum += v
			feeC++
		}
		if v, ok := o["roiPct"].(float64); ok {
			roiSum += v
			roiC++
		}
	}

	avgHold := 0.0
	if holdC > 0 {
		avgHold = holdSum / float64(holdC)
	}
	avgOpenPnl := 0.0
	if openPnlC > 0 {
		avgOpenPnl = openPnlSum / float64(openPnlC)
	}
	avgFee := 0.0
	if feeC > 0 {
		avgFee = feeSum / float64(feeC)
	}
	avgRoi := 0.0
	if roiC > 0 {
		avgRoi = roiSum / float64(roiC)
	}

	prefStrats := make(map[string]int)
	prefRanges := make(map[string]int)
	for _, o := range historicalOwners {
		if s, ok := o["preferredStrategy"].(string); ok && s != "" {
			prefStrats[s]++
		}
		if r, ok := o["preferredRangeStyle"].(string); ok && r != "" {
			prefRanges[r]++
		}
	}

	poolName := "TOKEN-SOL"
	if n, ok := overview["name"].(string); ok && n != "" {
		poolName = n
	} else {
		tx, _ := overview["tokenXSymbol"].(string)
		ty, _ := overview["tokenYSymbol"].(string)
		if tx == "" {
			tx = "TOKEN"
		}
		if ty == "" {
			ty = "SOL"
		}
		poolName = fmt.Sprintf("%s-%s", tx, ty)
	}

	actPosCount := float64(len(ranked))
	if apc, ok := signalData["activePositionCount"].(float64); ok {
		actPosCount = apc
	}
	ownerCount := float64(len(ranked))
	if oc, ok := signalData["ownerCount"].(float64); ok {
		ownerCount = oc
	}

	bestOpenPnl := (*string)(nil)
	if len(ranked) > 0 {
		if v, ok := ranked[0]["pnlPerInflowPct"].(float64); ok {
			s := fmt.Sprintf("%.2f%%", v)
			bestOpenPnl = &s
		}
	}

	scalperC := 0
	holderC := 0
	for _, o := range ranked {
		age := getFloat(o, "avgAgeHours")
		if age < 1 {
			scalperC++
		} else if age >= 4 {
			holderC++
		}
	}

	topHist := []any{}
	if th, ok := signalData["topHistoricalOwners"].([]interface{}); ok {
		if len(th) > 3 {
			topHist = th[:3]
		} else {
			topHist = th
		}
	}

	return map[string]any{
		"top_lper_count":         len(ranked),
		"study_mode":             "lpagent_top_lpers",
		"pool_name":              poolName,
		"active_position_count":  actPosCount,
		"owner_count":            ownerCount,
		"avg_hold_hours":         round(avgHold, 2),
		"avg_open_pnl_pct":       round(avgOpenPnl, 2),
		"avg_fee_percent":        round(avgFee, 2),
		"avg_roi_pct":            round(avgRoi, 2),
		"best_open_pnl_pct":      bestOpenPnl,
		"scalper_count":          scalperC,
		"holder_count":           holderC,
		"preferred_strategies":   prefStrats,
		"preferred_range_styles": prefRanges,
		"top_historical_owners":  topHist,
		"suggested_style":        getStringOrNil(signalData, "suggestedStyle"),
	}
}

func getFloat(m map[string]any, k string) float64 {
	if v, ok := m[k].(float64); ok {
		return v
	}
	return 0
}

func getFallbackFloat(m1 map[string]any, k1 string, m2 map[string]any, k2 string) float64 {
	if v, ok := m1[k1].(float64); ok {
		return v
	}
	if m2 != nil {
		if v, ok := m2[k2].(float64); ok {
			return v
		}
	}
	return 0
}

func getStringOrNil(m map[string]any, k string) *string {
	if v, ok := m[k].(string); ok && v != "" {
		return &v
	}
	return nil
}

func getFloatOrNil(m map[string]any, k string) *float64 {
	if v, ok := m[k].(float64); ok {
		return &v
	}
	return nil
}

func round(val float64, precision int) float64 {
	shift := math.Pow(10, float64(precision))
	return math.Round(val*shift) / shift
}
