package screening

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"meridian-go-rewrite/internal/api"
	"meridian-go-rewrite/internal/config"
)

type Candidate struct {
	PoolAddress       string               `json:"pool_address"`
	Name              string               `json:"name"`
	TokenX            string               `json:"token_x"`
	TokenXMint        string               `json:"token_x_mint"`
	TokenY            string               `json:"token_y"`
	BinStep           int                  `json:"bin_step"`
	FeePct            float64              `json:"base_fee_percentage"`
	TVL               float64              `json:"tvl"`
	ActiveTVL         float64              `json:"active_tvl"`
	Volume            float64              `json:"volume"`
	FeeTVLRatio       float64              `json:"fee_tvl_ratio"`
	Volatility        float64              `json:"volatility"`
	Holders           int                  `json:"holders"`
	MCap              float64              `json:"mcap"`
	OrganicScore      float64              `json:"organic_score"`
	ActivePositions   int                  `json:"active_positions"`
	Price             float64              `json:"pool_price"`
	PriceChangePct    float64              `json:"pool_price_change_pct"`
	SwapCount         int                  `json:"swap_count"`
	UniqueTraders     int                  `json:"unique_traders"`
	HasSafetyWarnings bool                 `json:"has_safety_warnings"`
	HasSupplyRisk     bool                 `json:"has_supply_risk"`
	Dev               string               `json:"dev"`
	AgeDays           float64              `json:"age_days"`
	CreatedAt         float64              `json:"created_at"`
	VolumeChangePct   float64              `json:"volume_change_pct"`
	FeeChangePct      float64              `json:"fee_change_pct"`
	OKXAdvanced       *api.OKXAdvancedInfo `json:"okx_advanced,omitempty"`
	OKXPrice          *api.OKXPriceInfo    `json:"okx_price,omitempty"`
	OKXClusters       []api.OKXClusterInfo `json:"okx_clusters,omitempty"`
	Score             float64              `json:"score"`
	WeightedScore     float64              `json:"weighted_score"`
	RejectionReason   string               `json:"rejection_reason,omitempty"`
	Passed            bool                 `json:"passed"`
}

type ScreeningResult struct {
	Candidates    []Candidate `json:"candidates"`
	TotalFetched  int         `json:"total_fetched"`
	TotalScreened int         `json:"total_screened"`
	TotalPassed   int         `json:"total_passed"`
}

func DiscoverAndScore(cfg *config.Config) (*ScreeningResult, error) {
	discovery, err := api.FetchDiscoveryPools(
		cfg.Screening.PageSize,
		cfg.Screening.FilterBy,
		cfg.Screening.Timeframe,
		cfg.Screening.Category,
	)
	if err != nil {
		return nil, fmt.Errorf("discovery API failed: %w", err)
	}

	candidates := make([]Candidate, 0, len(discovery.Data))
	for _, pool := range discovery.Data {
		c := Candidate{
			PoolAddress:       pool.PoolAddress,
			Name:              pool.Name,
			TokenX:            pool.TokenX.Symbol,
			TokenXMint:        pool.TokenX.Address,
			TokenY:            pool.TokenY.Symbol,
			BinStep:           pool.DLMMParams.BinStep,
			FeePct:            pool.FeePct,
			TVL:               pool.Tvl,
			ActiveTVL:         pool.ActiveTvl,
			Volume:            pool.Volume,
			FeeTVLRatio:       pool.FeeActiveTvlRatio,
			Volatility:        pool.Volatility,
			Holders:           int(pool.BaseTokenHolders),
			MCap:              pool.TokenX.MarketCap,
			OrganicScore:      pool.TokenX.OrganicScore,
			ActivePositions:   int(pool.ActivePositions),
			Price:             pool.PoolPrice,
			PriceChangePct:    pool.PoolPriceChangePct,
			SwapCount:         int(pool.SwapCount),
			UniqueTraders:     int(pool.UniqueTraders),
			HasSafetyWarnings: pool.BaseTokenHasCriticalWarnings || pool.QuoteTokenHasCriticalWarnings,
			HasSupplyRisk:     pool.BaseTokenHasHighSupplyConcentration || pool.BaseTokenHasHighSingleOwnership,
			Dev:               pool.TokenX.Dev,
			AgeDays:           pool.TokenX.CreatedAt,
			VolumeChangePct:   pool.VolumeChangePct,
			FeeChangePct:      pool.FeeChangePct,
		}
		candidates = append(candidates, c)
	}

	passed := filterAndScore(candidates, cfg)

	sort.Slice(passed, func(i, j int) bool {
		return passed[i].WeightedScore > passed[j].WeightedScore
	})

	return &ScreeningResult{
		Candidates:    passed,
		TotalFetched:  len(discovery.Data),
		TotalScreened: len(candidates),
		TotalPassed:   len(passed),
	}, nil
}

func filterAndScore(candidates []Candidate, cfg *config.Config) []Candidate {
	passed := make([]Candidate, 0)
	maxVol := 999.0
	if cfg.Screening.MaxVolatility != nil {
		maxVol = *cfg.Screening.MaxVolatility
	}

	for _, c := range candidates {
		rejectReason := ""

		if c.HasSafetyWarnings {
			rejectReason = "safety warnings"
		} else if c.HasSupplyRisk {
			rejectReason = "supply concentration risk"
		} else if IsTokenBlacklisted(c.TokenXMint) {
			rejectReason = "token blacklisted"
		} else if IsDevBlocked(c.Dev) {
			rejectReason = "dev blocked"
		} else if float64(c.BinStep) > cfg.Screening.MaxBinStep {
			rejectReason = fmt.Sprintf("bin_step %d > max %.0f", c.BinStep, cfg.Screening.MaxBinStep)
		} else if c.TVL < cfg.Screening.MinTvl {
			rejectReason = fmt.Sprintf("TVL %.0f < min %.0f", c.TVL, cfg.Screening.MinTvl)
		} else if c.OrganicScore < cfg.Screening.MinOrganic {
			rejectReason = fmt.Sprintf("organic %d < min %d", int(c.OrganicScore), int(cfg.Screening.MinOrganic))
		} else if c.ActivePositions < cfg.Screening.MinActivePositions {
			rejectReason = fmt.Sprintf("active positions %d < min %d", c.ActivePositions, cfg.Screening.MinActivePositions)
		} else if c.FeeTVLRatio < cfg.Screening.MinFeeActiveTvlRatio {
			rejectReason = fmt.Sprintf("fee/TVL %.4f < min %.4f", c.FeeTVLRatio, cfg.Screening.MinFeeActiveTvlRatio)
		} else if c.Volatility > maxVol {
			rejectReason = fmt.Sprintf("volatility %.1f > max %.1f", c.Volatility, maxVol)
		} else if float64(c.Holders) < cfg.Screening.MinHolders {
			rejectReason = fmt.Sprintf("holders %d < min %.0f", c.Holders, cfg.Screening.MinHolders)
		}

		if rejectReason != "" {
			c.RejectionReason = rejectReason
			continue
		}

		c.Score = scoreCandidate(c)
		c.Passed = true
		passed = append(passed, c)
	}

	return passed
}

func scoreCandidate(c Candidate) float64 {
	return (c.FeeTVLRatio * 1000) + (c.OrganicScore * 10) + (math.Log10(c.Volume+1) * 10) + (float64(c.Holders) / 100) + (math.Log10(c.TVL) * 5) + (float64(c.UniqueTraders) / 50) - (c.Volatility / 10) + (float64(c.SwapCount) / 100)
}

func (c *Candidate) ApplyWeights(weights map[string]float64) {
	if c.Score == 0 {
		c.Score = scoreCandidate(*c)
	}
	c.WeightedScore = c.Score
	if w, ok := weights["organic_score"]; ok {
		c.WeightedScore += c.OrganicScore * 10 * (w - 1)
	}
	if w, ok := weights["fee_tvl_ratio"]; ok {
		c.WeightedScore += c.FeeTVLRatio * 1000 * (w - 1)
	}
	if w, ok := weights["volume"]; ok {
		c.WeightedScore += math.Log10(c.Volume+1) * 10 * (w - 1)
	}
	if w, ok := weights["holder_count"]; ok {
		c.WeightedScore += float64(c.Holders) / 100 * (w - 1)
	}
	if w, ok := weights["volatility"]; ok {
		c.WeightedScore -= (c.Volatility / 10) * (w - 1)
	}
}

func (c *Candidate) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s [%s] %s/%s\n", c.Name, c.PoolAddress[:8], c.TokenX, c.TokenY))
	sb.WriteString(fmt.Sprintf("  tvl=%.0f vol=%.0f fee_tvl=%.4f organic=%d holders=%d vola=%.0f active=%d\n",
		c.TVL, c.Volume, c.FeeTVLRatio, int(c.OrganicScore), c.Holders, c.Volatility, c.ActivePositions))
	sb.WriteString(fmt.Sprintf("  bin=%d fee=%.2f%% price=%.6f mcap=%.0f swaps=%d\n",
		c.BinStep, c.FeePct, c.Price, c.MCap, c.SwapCount))
	sb.WriteString(fmt.Sprintf("  score=%.1f weighted=%.1f", c.Score, c.WeightedScore))
	if c.AgeDays > 0 {
		sb.WriteString(fmt.Sprintf(" age=%.0fd", c.AgeDays))
	}
	if c.OKXPrice != nil {
		sb.WriteString(fmt.Sprintf(" | ATHdist=%.0f%% holders=%d", c.OKXPrice.PriceVsAThPct, c.OKXPrice.Holders))
	}
	if c.OKXAdvanced != nil {
		sb.WriteString(fmt.Sprintf(" | risk=%d bundle=%.0f%% snip=%.0f%%", c.OKXAdvanced.RiskLevel, c.OKXAdvanced.BundlePct, c.OKXAdvanced.SniperPct))
		if c.OKXAdvanced.SmartMoneyBuy {
			sb.WriteString(" smart_money")
		}
	}
	return sb.String()
}
