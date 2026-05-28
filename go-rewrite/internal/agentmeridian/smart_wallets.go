package agentmeridian

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"meridian-go-rewrite/internal/config"
	"meridian-go-rewrite/internal/logger"
	"meridian-go-rewrite/internal/solana/dlmm"
)

var solanaPubKeyRe = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)

type SmartWallet struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Category string `json:"category"`
	Type     string `json:"type"`
	AddedAt  string `json:"addedAt"`
}

type SmartWalletsData struct {
	Wallets []SmartWallet `json:"wallets"`
}

var (
	smartWalletsMutex sync.RWMutex
	smartWalletsFile  string
)

func initSmartWalletsPath() {
	if smartWalletsFile == "" {
		cfg := config.Get()
		if cfg != nil {
			smartWalletsFile = cfg.DataPath("smart-wallets.json")
		} else {
			smartWalletsFile = "smart-wallets.json"
		}
	}
}

func loadWallets() SmartWalletsData {
	initSmartWalletsPath()
	smartWalletsMutex.RLock()
	defer smartWalletsMutex.RUnlock()

	data, err := os.ReadFile(smartWalletsFile)
	if err != nil {
		return SmartWalletsData{Wallets: []SmartWallet{}}
	}

	var ld SmartWalletsData
	if err := json.Unmarshal(data, &ld); err != nil {
		return SmartWalletsData{Wallets: []SmartWallet{}}
	}
	return ld
}

func saveWallets(data SmartWalletsData) {
	initSmartWalletsPath()
	smartWalletsMutex.Lock()
	defer smartWalletsMutex.Unlock()

	if dir := filepath.Dir(smartWalletsFile); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	bytes, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(smartWalletsFile, bytes, 0644)
}

func AddSmartWallet(name, address, category, typ string) map[string]interface{} {
	if !solanaPubKeyRe.MatchString(address) {
		return map[string]interface{}{"success": false, "error": "Invalid Solana address format"}
	}
	if category == "" {
		category = "alpha"
	}
	if typ == "" {
		typ = "lp"
	}

	data := loadWallets()
	for _, w := range data.Wallets {
		if w.Address == address {
			return map[string]interface{}{"success": false, "error": fmt.Sprintf("Already tracked as %q", w.Name)}
		}
	}

	wallet := SmartWallet{
		Name:     name,
		Address:  address,
		Category: category,
		Type:     typ,
		AddedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	data.Wallets = append(data.Wallets, wallet)
	saveWallets(data)

	logger.Log("smart_wallets", fmt.Sprintf("Added wallet: %s (%s, type=%s)", name, category, typ))
	return map[string]interface{}{"success": true, "wallet": wallet}
}

func RemoveSmartWallet(address string) map[string]interface{} {
	data := loadWallets()
	idx := -1
	var name string
	for i, w := range data.Wallets {
		if w.Address == address {
			idx = i
			name = w.Name
			break
		}
	}
	if idx == -1 {
		return map[string]interface{}{"success": false, "error": "Wallet not found"}
	}

	data.Wallets = append(data.Wallets[:idx], data.Wallets[idx+1:]...)
	saveWallets(data)
	logger.Log("smart_wallets", fmt.Sprintf("Removed wallet: %s", name))
	return map[string]interface{}{"success": true, "removed": name}
}

func ListSmartWallets() map[string]interface{} {
	data := loadWallets()
	return map[string]interface{}{
		"total":   len(data.Wallets),
		"wallets": data.Wallets,
	}
}

type cachedPositions struct {
	Positions []dlmm.Position
	FetchedAt time.Time
}

var (
	walletCache      = make(map[string]cachedPositions)
	walletCacheMutex sync.RWMutex
	cacheTTL         = 5 * time.Minute
)

func CheckSmartWalletsOnPool(poolAddress string) map[string]interface{} {
	data := loadWallets()
	var lpWallets []SmartWallet
	for _, w := range data.Wallets {
		if w.Type == "" || w.Type == "lp" {
			lpWallets = append(lpWallets, w)
		}
	}

	if len(lpWallets) == 0 {
		return map[string]interface{}{
			"pool":             poolAddress,
			"tracked_wallets":  0,
			"in_pool":          []map[string]interface{}{},
			"confidence_boost": false,
			"signal":           "No smart wallets tracked yet — neutral signal",
		}
	}

	cfg := config.Get()
	var rpc string
	if cfg != nil {
		rpc = cfg.RPCURLOrDefault()
	} else {
		rpc = "https://api.mainnet-beta.solana.com"
	}

	var inPool []map[string]interface{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, w := range lpWallets {
		wg.Add(1)
		go func(wallet SmartWallet) {
			defer wg.Done()
			var positions []dlmm.Position

			walletCacheMutex.RLock()
			cached, ok := walletCache[wallet.Address]
			walletCacheMutex.RUnlock()

			if ok && time.Since(cached.FetchedAt) < cacheTTL {
				positions = cached.Positions
			} else {
				client := dlmm.NewClient(wallet.Address, rpc)
				posList, err := client.GetMyPositions(false) // just standard positions
				if err == nil {
					positions = posList.Positions
				}
				walletCacheMutex.Lock()
				walletCache[wallet.Address] = cachedPositions{
					Positions: positions,
					FetchedAt: time.Now(),
				}
				walletCacheMutex.Unlock()
			}

			for _, p := range positions {
				if p.Pool == poolAddress {
					mu.Lock()
					inPool = append(inPool, map[string]interface{}{
						"name":     wallet.Name,
						"category": wallet.Category,
						"address":  wallet.Address,
					})
					mu.Unlock()
					break
				}
			}
		}(w)
	}

	wg.Wait()

	if inPool == nil {
		inPool = make([]map[string]interface{}, 0)
	}

	signal := fmt.Sprintf("0/%d smart wallets in this pool — neutral, rely on fundamentals", len(lpWallets))
	if len(inPool) > 0 {
		var names []string
		for _, w := range inPool {
			if name, ok := w["name"].(string); ok {
				names = append(names, name)
			}
		}
		signal = fmt.Sprintf("%d/%d smart wallet(s) are in this pool: %s — STRONG signal", len(inPool), len(lpWallets), strings.Join(names, ", "))
	}

	return map[string]interface{}{
		"pool":             poolAddress,
		"tracked_wallets":  len(lpWallets),
		"in_pool":          inPool,
		"confidence_boost": len(inPool) > 0,
		"signal":           signal,
	}
}
