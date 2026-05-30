package jupiter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"meridian-go-rewrite/internal/config"
)

const (
	SwapV2API     = "https://api.jup.ag/swap/v2"
	PriceAPI      = "https://api.jup.ag/price/v3"
	DefaultAPIKey = "b15d42e9-e0e4-4f90-a424-ae41ceeaa382"
	WSOL          = "So11111111111111111111111111111111111111112"
)

type SwapOrder struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"requestId"`
	FeeBps       int    `json:"feeBps"`
	FeeMint      string `json:"feeMint"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type SwapOutput struct {
	Success       bool   `json:"success"`
	Tx            string `json:"tx"`
	InputMint     string `json:"input_mint"`
	OutputMint    string `json:"output_mint"`
	AmountIn      string `json:"amount_in"`
	AmountOut     string `json:"amount_out"`
	Error         string `json:"error,omitempty"`
	FeeBpsApplied int    `json:"fee_bps_applied"`
}

func NormalizeMint(mint string) string {
	if mint == "SOL" || mint == "native" {
		return WSOL
	}
	return mint
}

func CreateSwapOrder(inputMint, outputMint, walletPubKey string, amount float64, cfg *config.Config) (*SwapOrder, error) {
	inputMint = NormalizeMint(inputMint)
	outputMint = NormalizeMint(outputMint)

	apiKey := cfg.Jupiter.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("JUPITER_API_KEY")
	}
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}

	amountLamports := fmt.Sprintf("%.0f", amount*1e9)
	params := url.Values{
		"inputMint":  {inputMint},
		"outputMint": {outputMint},
		"amount":     {amountLamports},
		"taker":      {walletPubKey},
	}

	if cfg.Jupiter.ReferralAccount != "" && cfg.Jupiter.ReferralFeeBps >= 50 && cfg.Jupiter.ReferralFeeBps <= 255 {
		params.Set("referralAccount", cfg.Jupiter.ReferralAccount)
		params.Set("referralFee", fmt.Sprintf("%d", cfg.Jupiter.ReferralFeeBps))
	}

	orderURL := fmt.Sprintf("%s/order?%s", SwapV2API, params.Encode())
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", orderURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("swap order failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("swap order HTTP %d: %s", resp.StatusCode, string(body))
	}

	var order SwapOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("parse order: %w", err)
	}
	if order.ErrorCode != "" || order.ErrorMessage != "" {
		return nil, fmt.Errorf("swap order error: %s", order.ErrorMessage)
	}

	return &order, nil
}

type JupiterTokenInfo struct {
	Mint         string  `json:"mint"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	OrganicScore float64 `json:"organic_score"`
	OrganicLabel string  `json:"organic_label"`
	MCap         float64 `json:"mcap"`
	Price        float64 `json:"price"`
	Liquidity    float64 `json:"liquidity"`
	Holders      int     `json:"holders"`
	Launchpad    string  `json:"launchpad"`
	Graduated    bool    `json:"graduated"`
	GlobalFees   float64 `json:"global_fees_sol"`
}

type NarrativeResult struct {
	Mint      string `json:"mint"`
	Narrative string `json:"narrative"`
}

func FetchTokenInfo(query string) ([]JupiterTokenInfo, error) {
	apiKey := os.Getenv("JUPITER_API_KEY")
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	url := fmt.Sprintf("https://datapi.jup.ag/v1/assets/search?query=%s&limit=5", url.QueryEscape(query))
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jupiter search HTTP %d: %s", resp.StatusCode, truncate(body, 200))
	}
	var results []JupiterTokenInfo
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func FetchTokenNarrative(mint string) (*NarrativeResult, error) {
	apiKey := os.Getenv("JUPITER_API_KEY")
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	url := fmt.Sprintf("https://datapi.jup.ag/v1/chaininsight/narrative/%s", mint)
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jupiter narrative HTTP %d: %s", resp.StatusCode, truncate(body, 200))
	}
	var result NarrativeResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	result.Mint = mint
	return &result, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
