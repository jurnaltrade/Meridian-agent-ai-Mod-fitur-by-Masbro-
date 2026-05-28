package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(`
╔═══════════════════════════════════════════════╗
║        Meridian — Setup Wizard                ║
║        Autonomous Meteora DLMM LP Agent       ║
╚═══════════════════════════════════════════════╝

This wizard creates your .env and user-config.json.
Press Enter to keep the current/default value.
`)

		reader := bufio.NewReader(os.Stdin)

		ask := func(prompt, defaultVal string) string {
			hint := ""
			if defaultVal != "" {
				hint = fmt.Sprintf(" (default: %s)", defaultVal)
			}
			fmt.Printf("%s%s: ", prompt, hint)
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			if text == "" {
				return defaultVal
			}
			return text
		}

		askNum := func(prompt string, defaultVal float64) float64 {
			for {
				ans := ask(prompt, fmt.Sprintf("%v", defaultVal))
				val, err := strconv.ParseFloat(ans, 64)
				if err == nil {
					return val
				}
				fmt.Println("  ⚠ Please enter a number.")
			}
		}

		askBool := func(prompt string, defaultVal bool) bool {
			for {
				hint := "y/N"
				if defaultVal {
					hint = "Y/n"
				}
				ans := ask(fmt.Sprintf("%s [%s]", prompt, hint), "")
				if ans == "" {
					return defaultVal
				}
				ans = strings.ToLower(ans)
				if ans == "y" || ans == "yes" {
					return true
				}
				if ans == "n" || ans == "no" {
					return false
				}
				fmt.Println("  ⚠ Enter y or n.")
			}
		}

		// Simplified for 1:1 rewrite equivalent behaviour
		fmt.Println("── API Keys & Wallet ─────────────────────────────────────────")
		openrouterKey := ask("OpenRouter API key (sk-or-...)", "")
		walletKey := ask("Wallet private key (base58)", "")
		rpcUrl := ask("RPC URL", "https://api.mainnet-beta.solana.com")
		heliusKey := ask("Helius API key (for balance lookups, optional)", "")

		fmt.Println("\n── Telegram (optional — skip to disable) ─────────────────────")
		telegramToken := ask("Telegram bot token", "")
		telegramChatId := ask("Telegram chat ID", "")

		fmt.Println("\n── Deployment ────────────────────────────────────────────────")
		deployAmountSol := askNum("SOL to deploy per position", 0.3)
		maxPositions := int(askNum("Max concurrent positions", 3))
		minSolToOpen := askNum("Min SOL balance to open a new position", deployAmountSol+0.05)
		dryRun := askBool("Dry run mode? (no real transactions)", true)
		minBinsBelow := int(askNum("Minimum bins below active bin", 35))
		maxBinsBelow := int(askNum("Maximum bins below active bin", 69))
		defaultBinsBelow := int(askNum("Default bins below active bin", float64(maxBinsBelow)))

		fmt.Println("\n── Risk & Filters ────────────────────────────────────────────")
		timeframe := ask("Pool discovery timeframe (30m / 1h / 4h / 12h / 24h)", "4h")
		minOrganic := askNum("Min organic score (0–100)", 65)
		minHolders := int(askNum("Min token holders", 500))
		maxMcap := askNum("Max token market cap USD", 10000000)

		fmt.Println("\n── Exit Rules ────────────────────────────────────────────────")
		takeProfitFeePct := askNum("Take profit when fees earned >= X% of deployed capital", 5)
		stopLossPct := askNum("Stop loss at X% price drop (e.g. -15)", -15)
		outOfRangeWaitMinutes := int(askNum("Minutes out-of-range before closing", 30))
		repeatDeployCooldownEnabled := askBool("Cooldown token/pool after repeated fee-generating deploys?", true)
		repeatDeployCooldownTriggerCount := int(askNum("Repeat deploy cooldown trigger count", 3))
		repeatDeployCooldownHours := askNum("Repeat deploy cooldown hours", 12)
		repeatDeployCooldownScope := ask("Repeat deploy cooldown scope (pool/token/both)", "token")
		repeatDeployCooldownMinFeeEarnedPct := askNum("Repeat deploy min fee earned %", 0)

		fmt.Println("\n── Scheduling ────────────────────────────────────────────────")
		managementIntervalMin := int(askNum("Management cycle interval (minutes)", 10))
		screeningIntervalMin := int(askNum("Screening cycle interval (minutes)", 30))

		fmt.Println("\n── LLM Provider ──────────────────────────────────────────────")
		llmProvider := ask("LLM provider (openrouter/minimax/openai/local/custom)", "openrouter")
		llmBaseUrl := ask("Base URL", "https://openrouter.ai/api/v1")
		llmModel := ask("Model name", "nousresearch/hermes-3-llama-3.1-405b")

		envContent := ""
		if openrouterKey != "" {
			envContent += fmt.Sprintf("OPENROUTER_API_KEY=%s\n", openrouterKey)
		}
		if walletKey != "" {
			envContent += fmt.Sprintf("WALLET_PRIVATE_KEY=%s\n", walletKey)
		}
		if rpcUrl != "" {
			envContent += fmt.Sprintf("RPC_URL=%s\n", rpcUrl)
		}
		if heliusKey != "" {
			envContent += fmt.Sprintf("HELIUS_API_KEY=%s\n", heliusKey)
		}
		if telegramToken != "" {
			envContent += fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s\n", telegramToken)
		}
		if telegramChatId != "" {
			envContent += fmt.Sprintf("TELEGRAM_CHAT_ID=%s\n", telegramChatId)
		}
		if dryRun {
			envContent += "DRY_RUN=true\n"
		} else {
			envContent += "DRY_RUN=false\n"
		}

		os.WriteFile(".env", []byte(envContent), 0644)

		userConfig := map[string]any{
			"preset":                              "custom",
			"rpcUrl":                              rpcUrl,
			"deployAmountSol":                     deployAmountSol,
			"maxPositions":                        maxPositions,
			"minSolToOpen":                        minSolToOpen,
			"minBinsBelow":                        minBinsBelow,
			"maxBinsBelow":                        maxBinsBelow,
			"defaultBinsBelow":                    defaultBinsBelow,
			"timeframe":                           timeframe,
			"minOrganic":                          minOrganic,
			"minHolders":                          minHolders,
			"maxMcap":                             maxMcap,
			"takeProfitFeePct":                    takeProfitFeePct,
			"stopLossPct":                         stopLossPct,
			"outOfRangeWaitMinutes":               outOfRangeWaitMinutes,
			"repeatDeployCooldownEnabled":         repeatDeployCooldownEnabled,
			"repeatDeployCooldownTriggerCount":    repeatDeployCooldownTriggerCount,
			"repeatDeployCooldownHours":           repeatDeployCooldownHours,
			"repeatDeployCooldownScope":           repeatDeployCooldownScope,
			"repeatDeployCooldownMinFeeEarnedPct": repeatDeployCooldownMinFeeEarnedPct,
			"managementIntervalMin":               managementIntervalMin,
			"screeningIntervalMin":                screeningIntervalMin,
			"llmProvider":                         llmProvider,
			"llmBaseUrl":                          llmBaseUrl,
			"llmModel":                            llmModel,
			"telegramChatId":                      telegramChatId,
			"dryRun":                              dryRun,
		}

		b, _ := json.MarshalIndent(userConfig, "", "  ")
		os.WriteFile("user-config.json", b, 0644)

		fmt.Println("\nSetup Complete. Run 'go run cmd/meridian/main.go start' to launch the agent.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(setupCmd)
}
