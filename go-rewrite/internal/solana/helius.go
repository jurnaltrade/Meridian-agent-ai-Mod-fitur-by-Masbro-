package solana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type HeliusBalances struct {
	Wallet   string         `json:"wallet"`
	SOL      float64        `json:"sol"`
	SOLPrice float64        `json:"sol_price"`
	SOLUSD   float64        `json:"sol_usd"`
	USDC     float64        `json:"usdc"`
	Tokens   []TokenBalance `json:"tokens"`
	TotalUSD float64        `json:"total_usd"`
	Error    string         `json:"error,omitempty"`
}

type TokenBalance struct {
	Mint    string  `json:"mint"`
	Symbol  string  `json:"symbol"`
	Balance float64 `json:"balance"`
	USD     float64 `json:"usd"`
}

type HeliusAsset struct {
	Mint          string  `json:"mint"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	ImageURI      string  `json:"image_uri"`
	Supply        float64 `json:"supply"`
	PricePerToken float64 `json:"price_per_token,omitempty"`
}

type HeliusTokenHolder struct {
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
	Pct     float64 `json:"pct"`
	IsPool  bool    `json:"is_pool,omitempty"`
}

type heliusBalancesResponse struct {
	Balances []struct {
		Mint          string  `json:"mint"`
		Symbol        string  `json:"symbol"`
		Balance       float64 `json:"balance"`
		PricePerToken float64 `json:"pricePerToken"`
		USDValue      float64 `json:"usdValue"`
	} `json:"balances"`
	TotalUSDValue float64 `json:"totalUsdValue"`
}

type heliusHoldersResponse struct {
	Holders []struct {
		Address             string  `json:"ownerAddress"`
		Amount              float64 `json:"amount"`
		DelegatedAmount     float64 `json:"delegatedAmount"`
		Pct                 float64 `json:"percentage"`
		Classification      string  `json:"classification"`
		TokenAccountAddress string  `json:"tokenAccountAddress"`
	} `json:"holders"`
	TotalHolders int `json:"totalHolders"`
}

func GetHeliusBalances(walletAddr, solMint, usdcMint string) (*HeliusBalances, error) {
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("HELIUS_API_KEY not set")
	}
	url := fmt.Sprintf("https://api.helius.xyz/v1/wallet/%s/balances?api-key=%s", walletAddr, apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
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
		return nil, fmt.Errorf("Helius error %d: %s", resp.StatusCode, string(body))
	}
	var data heliusBalancesResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	result := &HeliusBalances{Wallet: walletAddr, TotalUSD: data.TotalUSDValue}
	for _, b := range data.Balances {
		if b.Mint == solMint || b.Symbol == "SOL" {
			result.SOL = b.Balance
			result.SOLPrice = b.PricePerToken
			result.SOLUSD = b.USDValue
		}
		if b.Mint == usdcMint || b.Symbol == "USDC" {
			result.USDC = b.Balance
		}
		result.Tokens = append(result.Tokens, TokenBalance{
			Mint: b.Mint, Symbol: b.Symbol, Balance: b.Balance, USD: b.USDValue,
		})
	}
	return result, nil
}

func GetHeliusSolPrice() (float64, error) {
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("HELIUS_API_KEY not set")
	}
	url := fmt.Sprintf("https://api.helius.xyz/v1/token/%s/asset?api-key=%s", WSOL(), apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Helius asset error %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var asset HeliusAsset
	if err := json.Unmarshal(body, &asset); err != nil {
		return 0, err
	}
	if asset.PricePerToken > 0 {
		return asset.PricePerToken, nil
	}
	return 0, fmt.Errorf("SOL price not available")
}

func GetHeliusTokenHolders(mint string, limit int) ([]HeliusTokenHolder, int, error) {
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		return nil, 0, fmt.Errorf("HELIUS_API_KEY not set")
	}
	if limit <= 0 {
		limit = 100
	}
	url := fmt.Sprintf("https://api.helius.xyz/v1/token/%s/holders?api-key=%s&limit=%d", mint, apiKey, limit)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("Helius holders error %d: %s", resp.StatusCode, string(body))
	}
	var data heliusHoldersResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, 0, err
	}
	holders := make([]HeliusTokenHolder, 0, len(data.Holders))
	for _, h := range data.Holders {
		isPool := h.Classification == "pool" || h.Classification == "lp-pool"
		holders = append(holders, HeliusTokenHolder{
			Address: h.Address,
			Amount:  h.Amount,
			Pct:     h.Pct,
			IsPool:  isPool,
		})
	}
	return holders, data.TotalHolders, nil
}
