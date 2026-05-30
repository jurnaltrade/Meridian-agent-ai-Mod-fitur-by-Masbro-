package screening

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"meridian-go-rewrite/internal/logger"
)

const (
	tokenBlacklistFile = "token-blacklist.json"
	devBlocklistFile   = "dev-blocklist.json"
)

type TokenBlacklistEntry struct {
	Symbol  string `json:"symbol"`
	Reason  string `json:"reason"`
	AddedAt string `json:"added_at"`
	AddedBy string `json:"added_by"`
}

type DevBlocklistEntry struct {
	Label   string `json:"label"`
	Reason  string `json:"reason"`
	AddedAt string `json:"added_at"`
}

var (
	tokenBlacklistMutex sync.RWMutex
	devBlocklistMutex   sync.RWMutex
)

func loadTokenBlacklist() map[string]TokenBlacklistEntry {
	tokenBlacklistMutex.RLock()
	defer tokenBlacklistMutex.RUnlock()

	data, err := os.ReadFile(tokenBlacklistFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]TokenBlacklistEntry)
		}
		logger.Error("blacklist_error", fmt.Errorf("invalid %s: %w", tokenBlacklistFile, err))
		return make(map[string]TokenBlacklistEntry)
	}

	var db map[string]TokenBlacklistEntry
	if err := json.Unmarshal(data, &db); err != nil {
		logger.Error("blacklist_error", fmt.Errorf("invalid %s format: %w", tokenBlacklistFile, err))
		return make(map[string]TokenBlacklistEntry)
	}
	return db
}

func saveTokenBlacklist(db map[string]TokenBlacklistEntry) {
	tokenBlacklistMutex.Lock()
	defer tokenBlacklistMutex.Unlock()

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		logger.Error("blacklist_error", fmt.Errorf("failed to encode %s: %w", tokenBlacklistFile, err))
		return
	}
	if err := os.WriteFile(tokenBlacklistFile, data, 0644); err != nil {
		logger.Error("blacklist_error", fmt.Errorf("failed to save %s: %w", tokenBlacklistFile, err))
	}
}

func IsTokenBlacklisted(mint string) bool {
	if mint == "" {
		return false
	}
	db := loadTokenBlacklist()
	_, exists := db[mint]
	return exists
}

func AddToTokenBlacklist(mint, symbol, reason string) map[string]interface{} {
	if mint == "" {
		return map[string]interface{}{"error": "mint required"}
	}

	db := loadTokenBlacklist()
	if entry, exists := db[mint]; exists {
		return map[string]interface{}{
			"already_blacklisted": true,
			"mint":                mint,
			"symbol":              entry.Symbol,
			"reason":              entry.Reason,
		}
	}

	if symbol == "" {
		symbol = "UNKNOWN"
	}
	if reason == "" {
		reason = "no reason provided"
	}

	db[mint] = TokenBlacklistEntry{
		Symbol:  symbol,
		Reason:  reason,
		AddedAt: time.Now().UTC().Format(time.RFC3339),
		AddedBy: "agent",
	}

	saveTokenBlacklist(db)
	logger.Log("blacklist", fmt.Sprintf("Blacklisted %s: %s", symbol, reason))

	return map[string]interface{}{
		"blacklisted": true,
		"mint":        mint,
		"symbol":      symbol,
		"reason":      reason,
	}
}

func RemoveFromTokenBlacklist(mint string) map[string]interface{} {
	if mint == "" {
		return map[string]interface{}{"error": "mint required"}
	}

	db := loadTokenBlacklist()
	entry, exists := db[mint]
	if !exists {
		return map[string]interface{}{"error": fmt.Sprintf("Mint %s not found on blacklist", mint)}
	}

	delete(db, mint)
	saveTokenBlacklist(db)
	logger.Log("blacklist", fmt.Sprintf("Removed %s from blacklist", entry.Symbol))

	return map[string]interface{}{
		"removed": true,
		"mint":    mint,
		"was":     entry,
	}
}

func ListTokenBlacklist() map[string]interface{} {
	db := loadTokenBlacklist()
	var entries []map[string]interface{}
	for mint, info := range db {
		entries = append(entries, map[string]interface{}{
			"mint":     mint,
			"symbol":   info.Symbol,
			"reason":   info.Reason,
			"added_at": info.AddedAt,
			"added_by": info.AddedBy,
		})
	}
	return map[string]interface{}{
		"count":     len(entries),
		"blacklist": entries,
	}
}

func loadDevBlocklist() map[string]DevBlocklistEntry {
	devBlocklistMutex.RLock()
	defer devBlocklistMutex.RUnlock()

	data, err := os.ReadFile(devBlocklistFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]DevBlocklistEntry)
		}
		logger.Error("dev_blocklist_error", fmt.Errorf("invalid %s: %w", devBlocklistFile, err))
		return make(map[string]DevBlocklistEntry)
	}

	var db map[string]DevBlocklistEntry
	if err := json.Unmarshal(data, &db); err != nil {
		logger.Error("dev_blocklist_error", fmt.Errorf("invalid %s format: %w", devBlocklistFile, err))
		return make(map[string]DevBlocklistEntry)
	}
	return db
}

func saveDevBlocklist(db map[string]DevBlocklistEntry) {
	devBlocklistMutex.Lock()
	defer devBlocklistMutex.Unlock()

	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		logger.Error("dev_blocklist_error", fmt.Errorf("failed to encode %s: %w", devBlocklistFile, err))
		return
	}
	if err := os.WriteFile(devBlocklistFile, data, 0644); err != nil {
		logger.Error("dev_blocklist_error", fmt.Errorf("failed to save %s: %w", devBlocklistFile, err))
	}
}

func loadDeployerBlacklist() map[string]bool {
	data, err := os.ReadFile("deployer-blacklist.json")
	if err != nil {
		return make(map[string]bool)
	}
	var db struct {
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(data, &db); err != nil {
		return make(map[string]bool)
	}
	m := make(map[string]bool)
	for _, addr := range db.Addresses {
		m[addr] = true
	}
	return m
}

func IsDevBlocked(devWallet string) bool {
	if devWallet == "" {
		return false
	}
	db := loadDevBlocklist()
	if _, exists := db[devWallet]; exists {
		return true
	}
	deployerDB := loadDeployerBlacklist()
	if deployerDB[devWallet] {
		return true
	}
	return false
}

func BlockDev(wallet, reason, label string) map[string]interface{} {
	if wallet == "" {
		return map[string]interface{}{"error": "wallet required"}
	}

	db := loadDevBlocklist()
	if entry, exists := db[wallet]; exists {
		return map[string]interface{}{
			"already_blocked": true,
			"wallet":          wallet,
			"label":           entry.Label,
			"reason":          entry.Reason,
		}
	}

	if label == "" {
		label = "unknown"
	}
	if reason == "" {
		reason = "no reason provided"
	}

	db[wallet] = DevBlocklistEntry{
		Label:   label,
		Reason:  reason,
		AddedAt: time.Now().UTC().Format(time.RFC3339),
	}

	saveDevBlocklist(db)
	logger.Log("dev_blocklist", fmt.Sprintf("Blocked deployer %s: %s", label, reason))

	return map[string]interface{}{
		"blocked": true,
		"wallet":  wallet,
		"label":   label,
		"reason":  reason,
	}
}

func UnblockDev(wallet string) map[string]interface{} {
	if wallet == "" {
		return map[string]interface{}{"error": "wallet required"}
	}

	db := loadDevBlocklist()
	entry, exists := db[wallet]
	if !exists {
		return map[string]interface{}{"error": fmt.Sprintf("Wallet %s not on dev blocklist", wallet)}
	}

	delete(db, wallet)
	saveDevBlocklist(db)
	logger.Log("dev_blocklist", fmt.Sprintf("Removed deployer %s from blocklist", entry.Label))

	return map[string]interface{}{
		"unblocked": true,
		"wallet":    wallet,
		"was":       entry,
	}
}

func ListBlockedDevs() map[string]interface{} {
	db := loadDevBlocklist()
	var entries []map[string]interface{}
	for wallet, info := range db {
		entries = append(entries, map[string]interface{}{
			"wallet":   wallet,
			"label":    info.Label,
			"reason":   info.Reason,
			"added_at": info.AddedAt,
		})
	}
	return map[string]interface{}{
		"count":        len(entries),
		"blocked_devs": entries,
	}
}
