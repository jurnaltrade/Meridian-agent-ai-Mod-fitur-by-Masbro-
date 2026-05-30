package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	okxBase        = "https://www.okx.com"
	okxPublicAgent = "agent-cli"
)

type OKXRiskFlags struct {
	IsRugpull bool `json:"is_rugpull"`
	IsWash    bool `json:"is_wash"`
	RiskLevel int  `json:"risk_level"`
}

type OKXAdvancedInfo struct {
	RiskLevel       int      `json:"risk_level"`
	BundlePct       float64  `json:"bundle_pct"`
	SniperPct       float64  `json:"sniper_pct"`
	SuspiciousPct   float64  `json:"suspicious_pct"`
	NewWalletPct    float64  `json:"new_wallet_pct"`
	DevHoldingPct   float64  `json:"dev_holding_pct"`
	Top10Pct        float64  `json:"top10_pct"`
	LPBurnedPct     float64  `json:"lp_burned_pct"`
	TotalFeeSol     float64  `json:"total_fee_sol"`
	DevRugCount     int      `json:"dev_rug_count"`
	DevTokenCount   int      `json:"dev_token_count"`
	SmartMoneyBuy   bool     `json:"smart_money_buy"`
	DevSoldAll      bool     `json:"dev_sold_all"`
	DexBoost        bool     `json:"dex_boost"`
	DexScreenerPaid bool     `json:"dex_screener_paid"`
	Creator         string   `json:"creator"`
	Tags            []string `json:"tags"`
}

type OKXClusterInfo struct {
	HoldingPct   float64 `json:"holding_pct"`
	Trend        string  `json:"trend"`
	AvgHoldDays  float64 `json:"avg_hold_days"`
	PnLPct       float64 `json:"pnl_pct"`
	HasKOL       bool    `json:"has_kol"`
	AddressCount int     `json:"address_count"`
}

type OKXPriceInfo struct {
	Price         float64 `json:"price"`
	ATH           float64 `json:"ath"`
	ATL           float64 `json:"atl"`
	PriceVsAThPct float64 `json:"price_vs_ath_pct"`
	PriceChange5m float64 `json:"price_change_5m"`
	PriceChange1h float64 `json:"price_change_1h"`
	Volume5m      float64 `json:"volume_5m"`
	Volume1h      float64 `json:"volume_1h"`
	Holders       int     `json:"holders"`
	MarketCap     float64 `json:"market_cap"`
	Liquidity     float64 `json:"liquidity"`
}

type OKXFullAnalysis struct {
	Advanced *OKXAdvancedInfo `json:"advanced"`
	Clusters []OKXClusterInfo `json:"clusters"`
	Price    *OKXPriceInfo    `json:"price"`
	Risk     *OKXRiskFlags    `json:"risk"`
}

func hasOKXAuth() bool {
	return os.Getenv("OKX_API_KEY") != "" &&
		os.Getenv("OKX_SECRET_KEY") != "" &&
		os.Getenv("OKX_PASSPHRASE") != ""
}

func buildOKXHeaders(method, path string, body []byte) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")

	if hasOKXAuth() {
		timestamp := time.Now().UTC().Format(time.RFC3339)
		signStr := timestamp + method + path + string(body)
		mac := hmac.New(sha256.New, []byte(os.Getenv("OKX_SECRET_KEY")))
		mac.Write([]byte(signStr))
		sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		h.Set("OK-ACCESS-KEY", os.Getenv("OKX_API_KEY"))
		h.Set("OK-ACCESS-SIGN", sign)
		h.Set("OK-ACCESS-TIMESTAMP", timestamp)
		h.Set("OK-ACCESS-PASSPHRASE", os.Getenv("OKX_PASSPHRASE"))
	} else {
		h.Set("Ok-Access-Client-type", okxPublicAgent)
	}
	return h
}

func okxRequest(method, path string, body []byte) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}
	req, err := http.NewRequest(method, okxBase+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header = buildOKXHeaders(method, path, body)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OKX request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("OKX HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func GetOKXAdvancedInfo(tokenAddress string) (*OKXAdvancedInfo, error) {
	path := fmt.Sprintf("/api/v6/dex/market/token/advanced-info?chainIndex=501&tokenAddress=%s", tokenAddress)
	data, err := okxRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code string            `json:"code"`
		Data []OKXAdvancedInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "0" {
		return nil, fmt.Errorf("OKX advanced info error: code=%s", resp.Code)
	}
	if len(resp.Data) > 0 {
		return &resp.Data[0], nil
	}
	return nil, fmt.Errorf("no data in OKX advanced info response")
}

func GetOKXClusterList(tokenAddress string) ([]OKXClusterInfo, error) {
	path := fmt.Sprintf("/api/v6/dex/market/token/cluster/list?chainIndex=501&tokenAddress=%s&limit=5", tokenAddress)
	data, err := okxRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code string           `json:"code"`
		Data []OKXClusterInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "0" {
		return nil, fmt.Errorf("OKX cluster list error: code=%s", resp.Code)
	}
	return resp.Data, nil
}

func GetOKXPriceInfo(tokenAddress string) (*OKXPriceInfo, error) {
	path := "/api/v6/dex/market/price-info"
	body, _ := json.Marshal(map[string]any{
		"chainIndex":   "501",
		"tokenAddress": tokenAddress,
	})
	data, err := okxRequest("POST", path, body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code string         `json:"code"`
		Data []OKXPriceInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "0" {
		return nil, fmt.Errorf("OKX price info error: code=%s", resp.Code)
	}
	if len(resp.Data) > 0 {
		return &resp.Data[0], nil
	}
	return nil, fmt.Errorf("no data in OKX price info response")
}

func GetOKXRiskFlags(tokenAddress string) (*OKXRiskFlags, error) {
	path := fmt.Sprintf("/priapi/v1/dx/market/v2/risk/new/check?chainId=501&tokenAddress=%s", tokenAddress)
	data, err := okxRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Code string `json:"code"`
		Data struct {
			AllAnalysis      []map[string]any `json:"allAnalysis"`
			SwapAnalysis     []map[string]any `json:"swapAnalysis"`
			ContractAnalysis []map[string]any `json:"contractAnalysis"`
			ExtraAnalysis    []map[string]any `json:"extraAnalysis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil
	}

	flags := &OKXRiskFlags{}

	collectFrom := func(entries []map[string]any) {
		for _, e := range entries {
			name, _ := e["riskName"].(string)
			lvl, _ := e["riskLevel"].(float64)
			if lvl > float64(flags.RiskLevel) {
				flags.RiskLevel = int(lvl)
			}
			switch name {
			case "is_rugpull":
				flags.IsRugpull = true
			case "is_wash":
				flags.IsWash = true
			}
		}
	}
	collectFrom(resp.Data.AllAnalysis)
	collectFrom(resp.Data.SwapAnalysis)
	collectFrom(resp.Data.ContractAnalysis)
	collectFrom(resp.Data.ExtraAnalysis)

	return flags, nil
}

func GetOKXFullAnalysis(tokenAddress string) *OKXFullAnalysis {
	result := &OKXFullAnalysis{}

	ch := make(chan func(), 3)
	calls := 0

	go func() {
		adv, err := GetOKXAdvancedInfo(tokenAddress)
		ch <- func() {
			if err == nil {
				result.Advanced = adv
			}
		}
	}()
	calls++
	go func() {
		clusters, err := GetOKXClusterList(tokenAddress)
		ch <- func() {
			if err == nil {
				result.Clusters = clusters
			}
		}
	}()
	calls++
	go func() {
		price, err := GetOKXPriceInfo(tokenAddress)
		ch <- func() {
			if err == nil {
				result.Price = price
			}
		}
	}()
	calls++

	for i := 0; i < calls; i++ {
		(<-ch)()
	}

	return result
}
