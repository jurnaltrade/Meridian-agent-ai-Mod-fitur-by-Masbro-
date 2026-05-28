package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DiscoveryBase = "https://pool-discovery-api.datapi.meteora.ag"
	DLMMBase      = "https://dlmm.datapi.meteora.ag"
)

type DiscoveryResponse struct {
	Total int             `json:"total"`
	Data  []DiscoveryPool `json:"data"`
}

type DiscoveryPool struct {
	PoolAddress                         string    `json:"pool_address"`
	Name                                string    `json:"name"`
	TokenX                              TokenSide `json:"token_x"`
	TokenY                              TokenSide `json:"token_y"`
	PoolType                            string    `json:"pool_type"`
	FeePct                              float64   `json:"fee_pct"`
	Tvl                                 float64   `json:"tvl"`
	ActiveTvl                           float64   `json:"active_tvl"`
	Fee                                 float64   `json:"fee"`
	Volume                              float64   `json:"volume"`
	FeeActiveTvlRatio                   float64   `json:"fee_active_tvl_ratio"`
	Volatility                          float64   `json:"volatility"`
	BaseTokenHolders                    int       `json:"base_token_holders"`
	ActivePositions                     int       `json:"active_positions"`
	ActivePositionsPct                  float64   `json:"active_positions_pct"`
	OpenPositions                       int       `json:"open_positions"`
	PoolPrice                           float64   `json:"pool_price"`
	PoolPriceChangePct                  float64   `json:"pool_price_change_pct"`
	PriceTrend                          string    `json:"price_trend"`
	MinPrice                            float64   `json:"min_price"`
	MaxPrice                            float64   `json:"max_price"`
	VolumeChangePct                     float64   `json:"volume_change_pct"`
	FeeChangePct                        float64   `json:"fee_change_pct"`
	SwapCount                           int       `json:"swap_count"`
	UniqueTraders                       int       `json:"unique_traders"`
	BaseTokenHasCriticalWarnings        bool      `json:"base_token_has_critical_warnings"`
	QuoteTokenHasCriticalWarnings       bool      `json:"quote_token_has_critical_warnings"`
	BaseTokenHasHighSupplyConcentration bool      `json:"base_token_has_high_supply_concentration"`
	BaseTokenHasHighSingleOwnership     bool      `json:"base_token_has_high_single_ownership"`
	DLMMParams                          struct {
		BinStep int `json:"bin_step"`
	} `json:"dlmm_params"`
}

type TokenSide struct {
	Symbol       string  `json:"symbol"`
	Address      string  `json:"address"`
	OrganicScore float64 `json:"organic_score"`
	MarketCap    float64 `json:"market_cap"`
	Warnings     int     `json:"warnings"`
	Dev          string  `json:"dev"`
	CreatedAt    float64 `json:"created_at"`
}

type PoolMetadata struct {
	Address      string `json:"address"`
	Name         string `json:"name"`
	TokenXSymbol string `json:"token_x_symbol"`
	TokenYSymbol string `json:"token_y_symbol"`
}

func FetchDiscoveryPools(pageSize int, filters, timeframe, category string) (*DiscoveryResponse, error) {
	url := fmt.Sprintf("%s/pools?page_size=%d&filter_by=%s&timeframe=%s&category=%s",
		DiscoveryBase, pageSize, filters, timeframe, category)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery API error: %d %s", resp.StatusCode, string(body))
	}

	var data DiscoveryResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func FetchPoolDetail(poolAddress, timeframe string) (*DiscoveryPool, error) {
	url := fmt.Sprintf("%s/pools?page_size=1&filter_by=pool_address=%s&timeframe=%s",
		DiscoveryBase, poolAddress, timeframe)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data DiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if len(data.Data) == 0 {
		return nil, fmt.Errorf("pool %s not found", poolAddress)
	}
	return &data.Data[0], nil
}

type PortfolioPool struct {
	PoolAddress         string   `json:"poolAddress"`
	TokenX              string   `json:"tokenX"`
	TokenY              string   `json:"tokenY"`
	TokenXMint          string   `json:"tokenXMint"`
	OutOfRange          bool     `json:"outOfRange"`
	ListPositions       []string `json:"listPositions"`
	PositionsOutOfRange []string `json:"positionsOutOfRange,omitempty"`
	TotalPositions      int      `json:"totalPositions"`
}

type PortfolioResponse struct {
	Pools []PortfolioPool `json:"pools"`
}

func FetchPortfolio(walletAddr string) (*PortfolioResponse, error) {
	url := fmt.Sprintf("%s/portfolio/open?user=%s", DLMMBase, walletAddr)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data PortfolioResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

type PnLPosition struct {
	PositionAddress    string        `json:"positionAddress"`
	LowerBinID         int           `json:"lowerBinId"`
	UpperBinID         int           `json:"upperBinId"`
	PoolActiveBinID    int           `json:"poolActiveBinId"`
	IsOutOfRange       bool          `json:"isOutOfRange"`
	FeePerTvl24h       string        `json:"feePerTvl24h"`
	PnlUsd             float64       `json:"pnlUsd"`
	PnlSol             float64       `json:"pnlSol"`
	PnlPctChange       float64       `json:"pnlPctChange"`
	PnlSolPctChange    float64       `json:"pnlSolPctChange"`
	CreatedAt          int64         `json:"createdAt"`
	UnrealizedPnl      PnLUnrealized `json:"unrealizedPnl"`
	AllTimeFees        PnLTotal      `json:"allTimeFees"`
	AllTimeDeposits    PnLTotal      `json:"allTimeDeposits"`
	AllTimeWithdrawals PnLTotal      `json:"allTimeWithdrawals"`
}

type PnLUnrealized struct {
	Balances           float64      `json:"balances"`
	BalancesSol        float64      `json:"balancesSol"`
	UnclaimedFeeTokenX PnLFeeAmount `json:"unclaimedFeeTokenX"`
	UnclaimedFeeTokenY PnLFeeAmount `json:"unclaimedFeeTokenY"`
}

type PnLFeeAmount struct {
	Usd       float64 `json:"usd"`
	AmountSol float64 `json:"amountSol"`
}

type PnLTotal struct {
	Total PnLValue `json:"total"`
}

type PnLValue struct {
	Usd float64 `json:"usd"`
	Sol float64 `json:"sol"`
}

type PnLResponse struct {
	Positions []PnLPosition `json:"positions"`
	Data      []PnLPosition `json:"data"`
}

func FetchDlmmPnL(poolAddr, walletAddr string) (*PnLResponse, error) {
	url := fmt.Sprintf("%s/positions/%s/pnl?user=%s&status=open&pageSize=100&page=1",
		DLMMBase, poolAddr, walletAddr)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data PnLResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func FetchClosedPnL(poolAddr, walletAddr string) (*PnLResponse, error) {
	url := fmt.Sprintf("%s/positions/%s/pnl?user=%s&status=closed&pageSize=50&page=1",
		DLMMBase, poolAddr, walletAddr)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data PnLResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func FetchPoolMetadata(poolAddr string) (*PoolMetadata, error) {
	url := fmt.Sprintf("%s/pools/%s", DLMMBase, poolAddr)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("pool metadata API: %w", err)
	}
	defer resp.Body.Close()

	var data PoolMetadata
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return &PoolMetadata{Address: poolAddr}, nil
	}
	return &data, nil
}

var poolMetadataCache = make(map[string]*PoolMetadata)

func GetPoolMetadata(poolAddr string) (*PoolMetadata, error) {
	if meta, ok := poolMetadataCache[poolAddr]; ok {
		return meta, nil
	}
	meta, err := FetchPoolMetadata(poolAddr)
	if err != nil || meta == nil {
		meta = &PoolMetadata{Address: poolAddr}
	}
	poolMetadataCache[poolAddr] = meta
	return meta, nil
}

// Jupiter token info API
const JupiterAPI = "https://datapi.jup.ag/v1"

type JupiterTokenInfo struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Symbol        string        `json:"symbol"`
	MarketCap     float64       `json:"mcap"`
	Price         float64       `json:"usdPrice"`
	Liquidity     float64       `json:"liquidity"`
	HolderCount   int           `json:"holderCount"`
	OrganicScore  float64       `json:"organicScore"`
	Launchpad     string        `json:"launchpad"`
	GraduatedPool bool          `json:"graduatedPool"`
	Fees          float64       `json:"fees"`
	TotalSupply   float64       `json:"totalSupply"`
	CircSupply    float64       `json:"circSupply"`
	Dev           string        `json:"dev"`
	CreatedAt     string        `json:"createdAt"`
	Audit         *JupiterAudit `json:"audit"`
	Stats1H       *JupiterStats `json:"stats1h"`
	Stats24H      *JupiterStats `json:"stats24h"`
}

type JupiterAudit struct {
	MintAuthorityDisabled   bool    `json:"mintAuthorityDisabled"`
	FreezeAuthorityDisabled bool    `json:"freezeAuthorityDisabled"`
	TopHoldersPercentage    float64 `json:"topHoldersPercentage"`
	BotHoldersPercentage    float64 `json:"botHoldersPercentage"`
	DevMigrations           bool    `json:"devMigrations"`
}

type JupiterStats struct {
	PriceChange      float64 `json:"priceChange"`
	BuyVolume        float64 `json:"buyVolume"`
	SellVolume       float64 `json:"sellVolume"`
	NumOrganicBuyers int     `json:"numOrganicBuyers"`
	NumNetBuyers     int     `json:"numNetBuyers"`
}

func FetchTokenInfo(query string) ([]JupiterTokenInfo, error) {
	url := fmt.Sprintf("%s/assets/search?query=%s", JupiterAPI, query)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data []JupiterTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
