package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"meridian-go-rewrite/internal/agentmeridian"
	"meridian-go-rewrite/internal/config"
	"meridian-go-rewrite/internal/logger"
)

func normalizeIntervals(intervals []string) []string {
	if len(intervals) == 0 {
		return []string{"5_MINUTE"}
	}
	var list []string
	for _, v := range intervals {
		up := strings.ToUpper(strings.TrimSpace(v))
		if up == "5_MINUTE" || up == "15_MINUTE" {
			list = append(list, up)
		}
	}
	if len(list) == 0 {
		return []string{"5_MINUTE"}
	}
	return list
}

func getFloatVal(m map[string]any, k string) *float64 {
	if v, ok := m[k].(float64); ok {
		return &v
	}
	return nil
}

func buildSignalSummary(payload map[string]any) map[string]any {
	latest, _ := payload["latest"].(map[string]any)
	if latest == nil {
		latest = map[string]any{}
	}
	candle, _ := latest["candle"].(map[string]any)
	if candle == nil {
		candle = map[string]any{}
	}
	prevCandle, _ := latest["previousCandle"].(map[string]any)
	if prevCandle == nil {
		prevCandle = map[string]any{}
	}
	rsiMap, _ := latest["rsi"].(map[string]any)
	if rsiMap == nil {
		rsiMap = map[string]any{}
	}
	bollinger, _ := latest["bollinger"].(map[string]any)
	if bollinger == nil {
		bollinger = map[string]any{}
	}
	supertrend, _ := latest["supertrend"].(map[string]any)
	if supertrend == nil {
		supertrend = map[string]any{}
	}
	fib, _ := latest["fibonacci"].(map[string]any)
	var fibLevels map[string]any
	if fib != nil {
		fibLevels, _ = fib["levels"].(map[string]any)
	}
	if fibLevels == nil {
		fibLevels = map[string]any{}
	}
	states, _ := latest["states"].(map[string]any)
	if states == nil {
		states = map[string]any{}
	}

	stDir, _ := supertrend["direction"].(string)
	if stDir == "" {
		stDir = "unknown"
	}

	breakUp, _ := states["supertrendBreakUp"].(bool)
	breakDown, _ := states["supertrendBreakDown"].(bool)

	return map[string]any{
		"close":               getFloatVal(candle, "close"),
		"previousClose":       getFloatVal(prevCandle, "close"),
		"rsi":                 getFloatVal(rsiMap, "value"),
		"lowerBand":           getFloatVal(bollinger, "lower"),
		"middleBand":          getFloatVal(bollinger, "middle"),
		"upperBand":           getFloatVal(bollinger, "upper"),
		"supertrendValue":     getFloatVal(supertrend, "value"),
		"supertrendDirection": stDir,
		"supertrendBreakUp":   breakUp,
		"supertrendBreakDown": breakDown,
		"fib50":               getFloatVal(fibLevels, "0.500"),
		"fib618":              getFloatVal(fibLevels, "0.618"),
		"fib786":              getFloatVal(fibLevels, "0.786"),
	}
}

type EvalResult struct {
	Confirmed bool
	Reason    string
	Signal    map[string]any
}

func evaluatePreset(side, preset string, payload map[string]any) EvalResult {
	summary := buildSignalSummary(payload)
	cfg := config.Get()
	oversold := 30.0
	overbought := 80.0
	if cfg != nil && cfg.Indicators.RSIOversold != 0 {
		oversold = float64(cfg.Indicators.RSIOversold)
	}
	if cfg != nil && cfg.Indicators.RSIOverbought != 0 {
		overbought = float64(cfg.Indicators.RSIOverbought)
	}

	closePrice := summary["close"].(*float64)
	prevClose := summary["previousClose"].(*float64)
	lowerBand := summary["lowerBand"].(*float64)
	upperBand := summary["upperBand"].(*float64)
	rsi := summary["rsi"].(*float64)
	stDir := summary["supertrendDirection"].(string)
	stBreakUp := summary["supertrendBreakUp"].(bool)
	stBreakDown := summary["supertrendBreakDown"].(bool)
	stValue := summary["supertrendValue"].(*float64)

	fib50 := summary["fib50"].(*float64)
	fib618 := summary["fib618"].(*float64)
	fib786 := summary["fib786"].(*float64)

	isBullish := stDir == "bullish"
	isBearish := stDir == "bearish"

	crossedUp := func(level *float64) bool {
		return level != nil && closePrice != nil && prevClose != nil && *prevClose < *level && *closePrice >= *level
	}
	crossedDown := func(level *float64) bool {
		return level != nil && closePrice != nil && prevClose != nil && *prevClose > *level && *closePrice <= *level
	}

	formatReason := func(label string, val *float64, op string, target *float64) string {
		vStr := "n/a"
		if val != nil {
			vStr = fmt.Sprintf("%.2f", *val)
		}
		tStr := "n/a"
		if target != nil {
			tStr = fmt.Sprintf("%.2f", *target)
		}
		return fmt.Sprintf("%s %s %s target %s", label, vStr, op, tStr)
	}

	switch preset {
	case "supertrend_break":
		if side == "entry" {
			confirmed := stBreakUp || (isBullish && closePrice != nil && stValue != nil && *closePrice >= *stValue)
			reason := "Price is above bullish Supertrend"
			if stBreakUp {
				reason = "Supertrend flipped bullish"
			}
			return EvalResult{Confirmed: confirmed, Reason: reason, Signal: summary}
		} else {
			confirmed := stBreakDown || (isBearish && closePrice != nil && stValue != nil && *closePrice <= *stValue)
			reason := "Price is below bearish Supertrend"
			if stBreakDown {
				reason = "Supertrend flipped bearish"
			}
			return EvalResult{Confirmed: confirmed, Reason: reason, Signal: summary}
		}
	case "rsi_reversal":
		if side == "entry" {
			confirmed := rsi != nil && *rsi <= oversold
			reason := fmt.Sprintf("RSI <= oversold %.0f", oversold)
			if rsi != nil {
				reason = fmt.Sprintf("RSI %.2f <= oversold %.0f", *rsi, oversold)
			}
			return EvalResult{Confirmed: confirmed, Reason: reason, Signal: summary}
		} else {
			confirmed := rsi != nil && *rsi >= overbought
			reason := fmt.Sprintf("RSI >= overbought %.0f", overbought)
			if rsi != nil {
				reason = fmt.Sprintf("RSI %.2f >= overbought %.0f", *rsi, overbought)
			}
			return EvalResult{Confirmed: confirmed, Reason: reason, Signal: summary}
		}
	case "bollinger_reversion":
		if side == "entry" {
			confirmed := closePrice != nil && lowerBand != nil && *closePrice <= *lowerBand
			return EvalResult{Confirmed: confirmed, Reason: formatReason("Close", closePrice, "<=", lowerBand), Signal: summary}
		} else {
			confirmed := closePrice != nil && upperBand != nil && *closePrice >= *upperBand
			return EvalResult{Confirmed: confirmed, Reason: formatReason("Close", closePrice, ">=", upperBand), Signal: summary}
		}
	case "rsi_plus_supertrend":
		if side == "entry" {
			confirmed := (rsi != nil && *rsi <= oversold) && (stBreakUp || isBullish)
			return EvalResult{Confirmed: confirmed, Reason: "RSI oversold with bullish Supertrend context", Signal: summary}
		} else {
			confirmed := (rsi != nil && *rsi >= overbought) && (stBreakDown || isBearish)
			return EvalResult{Confirmed: confirmed, Reason: "RSI overbought with bearish Supertrend context", Signal: summary}
		}
	case "supertrend_or_rsi":
		if side == "entry" {
			confirmed := stBreakUp || (isBullish && closePrice != nil && stValue != nil && *closePrice >= *stValue) || (rsi != nil && *rsi <= oversold)
			return EvalResult{Confirmed: confirmed, Reason: "Supertrend bullish confirmation or RSI oversold", Signal: summary}
		} else {
			confirmed := stBreakDown || (isBearish && closePrice != nil && stValue != nil && *closePrice <= *stValue) || (rsi != nil && *rsi >= overbought)
			return EvalResult{Confirmed: confirmed, Reason: "Supertrend bearish confirmation or RSI overbought", Signal: summary}
		}
	case "bb_plus_rsi":
		if side == "entry" {
			confirmed := closePrice != nil && lowerBand != nil && *closePrice <= *lowerBand && rsi != nil && *rsi <= oversold
			return EvalResult{Confirmed: confirmed, Reason: "Close at/below lower band with RSI oversold", Signal: summary}
		} else {
			confirmed := closePrice != nil && upperBand != nil && *closePrice >= *upperBand && rsi != nil && *rsi >= overbought
			return EvalResult{Confirmed: confirmed, Reason: "Close at/above upper band with RSI overbought", Signal: summary}
		}
	case "fibo_reclaim":
		if side == "entry" {
			confirmed := crossedUp(fib618) || crossedUp(fib50) || crossedUp(fib786)
			return EvalResult{Confirmed: confirmed, Reason: "Price reclaimed a key Fibonacci level", Signal: summary}
		} else {
			confirmed := crossedUp(fib618) || crossedUp(fib50)
			return EvalResult{Confirmed: confirmed, Reason: "Price reclaimed a key Fibonacci level upward", Signal: summary}
		}
	case "fibo_reject":
		if side == "entry" {
			confirmed := crossedDown(fib618) || crossedDown(fib50)
			return EvalResult{Confirmed: confirmed, Reason: "Price rejected from a key Fibonacci level", Signal: summary}
		} else {
			confirmed := crossedDown(fib618) || crossedDown(fib50) || crossedDown(fib786)
			return EvalResult{Confirmed: confirmed, Reason: "Price rejected below a key Fibonacci level", Signal: summary}
		}
	default:
		return EvalResult{Confirmed: false, Reason: "Unknown preset " + preset, Signal: summary}
	}
}

func FetchChartIndicatorsForMint(mint, interval string, candles, rsiLength int, refresh bool) (map[string]any, error) {
	norm := strings.ToUpper(strings.TrimSpace(interval))
	if norm == "" {
		norm = "15_MINUTE"
	}

	u := url.Values{}
	u.Set("interval", norm)
	u.Set("candles", strconv.Itoa(candles))
	u.Set("rsiLength", strconv.Itoa(rsiLength))
	if refresh {
		u.Set("refresh", "1")
	}

	path := fmt.Sprintf("/chart-indicators/%s?%s", mint, u.Encode())
	data, err := agentmeridian.AgentMeridianJSON("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	json.Unmarshal(data, &payload)
	return payload, nil
}

type ConfirmIndicatorArgs struct {
	Mint      string
	Side      string
	Preset    string
	Intervals []string
	Refresh   bool
}

func ConfirmIndicatorPreset(args ConfirmIndicatorArgs) map[string]any {
	cfg := config.Get()
	if cfg == nil || !cfg.Indicators.Enabled || args.Mint == "" {
		return map[string]any{
			"enabled":   false,
			"confirmed": true,
			"reason":    "Indicators disabled or not configured",
			"intervals": []any{},
		}
	}

	preset := args.Preset
	if preset == "" {
		if args.Side == "entry" {
			preset = cfg.Indicators.EntryPreset
		} else {
			preset = cfg.Indicators.ExitPreset
		}
	}

	if preset == "" {
		return map[string]any{
			"enabled":   false,
			"confirmed": true,
			"reason":    "Indicators disabled or not configured",
			"intervals": []any{},
		}
	}

	intervals := args.Intervals
	if len(intervals) == 0 {
		intervals = cfg.Indicators.Intervals
	}

	targets := normalizeIntervals(intervals)
	if len(targets) == 0 {
		return map[string]any{
			"enabled":   false,
			"confirmed": true,
			"reason":    "No indicator intervals configured",
			"intervals": []any{},
		}
	}

	candles := 298
	if cfg.Indicators.Candles != 0 {
		candles = cfg.Indicators.Candles
	}
	rsiLength := 2
	if cfg.Indicators.RSILength != 0 {
		rsiLength = cfg.Indicators.RSILength
	}

	var results []map[string]any
	var successful []map[string]any

	for _, interval := range targets {
		payload, err := FetchChartIndicatorsForMint(args.Mint, interval, candles, rsiLength, args.Refresh)
		if err != nil {
			logger.Log("indicators_warn", fmt.Sprintf("Indicator fetch failed for %s %s: %v", args.Mint[:8], interval, err))
			results = append(results, map[string]any{
				"interval":  interval,
				"ok":        false,
				"confirmed": nil,
				"reason":    err.Error(),
				"signal":    nil,
				"latest":    nil,
			})
			continue
		}

		eval := evaluatePreset(args.Side, preset, payload)
		res := map[string]any{
			"interval":  interval,
			"ok":        true,
			"confirmed": eval.Confirmed,
			"reason":    eval.Reason,
			"signal":    eval.Signal,
			"latest":    payload["latest"],
		}
		results = append(results, res)
		successful = append(successful, res)
	}

	if len(successful) == 0 {
		return map[string]any{
			"enabled":   true,
			"confirmed": true,
			"skipped":   true,
			"preset":    preset,
			"side":      args.Side,
			"reason":    "Indicator API unavailable; falling back to existing logic",
			"intervals": results,
		}
	}

	requireAll := cfg.Indicators.RequireAllIntervals
	confirmed := false

	if requireAll {
		confirmed = true
		for _, s := range successful {
			if !s["confirmed"].(bool) {
				confirmed = false
				break
			}
		}
	} else {
		for _, s := range successful {
			if s["confirmed"].(bool) {
				confirmed = true
				break
			}
		}
	}

	var successIntervals []string
	var allIntervals []string
	for _, s := range successful {
		interval := s["interval"].(string)
		allIntervals = append(allIntervals, interval)
		if s["confirmed"].(bool) {
			successIntervals = append(successIntervals, interval)
		}
	}

	reason := fmt.Sprintf("%s not confirmed on %s", preset, strings.Join(allIntervals, ", "))
	if confirmed {
		reason = fmt.Sprintf("%s confirmed on %s", preset, strings.Join(successIntervals, ", "))
	}

	return map[string]any{
		"enabled":             true,
		"confirmed":           confirmed,
		"skipped":             false,
		"preset":              preset,
		"side":                args.Side,
		"requireAllIntervals": requireAll,
		"reason":              reason,
		"intervals":           results,
	}
}
