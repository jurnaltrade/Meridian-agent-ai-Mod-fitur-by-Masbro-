package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"meridian-go-rewrite/internal/agent"
	"meridian-go-rewrite/internal/config"
	"meridian-go-rewrite/internal/solana"
	"meridian-go-rewrite/internal/solana/dlmm"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show wallet balances",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if cfg == nil {
			return fmt.Errorf("config not loaded")
		}
		walletAddr := cfg.WalletAddress()
		if walletAddr == "" {
			return fmt.Errorf("wallet address not configured — set WALLET_ADDRESS env var or add walletAddr to config")
		}
		balances, err := solana.GetHeliusBalances(walletAddr, cfg.Tokens.SOL, cfg.Tokens.USDC)
		if err != nil {
			OutputJSON(map[string]any{"error": err.Error()})
			return nil
		}
		OutputJSON(balances)
		return nil
	},
}

var positionsCmd = &cobra.Command{
	Use:   "positions",
	Short: "List open positions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if cfg == nil {
			return fmt.Errorf("config not loaded")
		}
		walletAddr := cfg.WalletAddress()
		if walletAddr == "" {
			return fmt.Errorf("wallet address not configured")
		}
		client := dlmm.NewClient(walletAddr, cfg.RPCURLOrDefault())
		positions, err := client.GetMyPositions(true)
		if err != nil {
			OutputJSON(map[string]any{"error": err.Error()})
			return nil
		}
		OutputJSON(positions)
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show runtime config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if cfg == nil {
			return fmt.Errorf("config not loaded")
		}
		out, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	},
}

var (
	deployPool      string
	deployAmount    float64
	deployBinsBelow int
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a position (dry-run preview by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if cfg == nil {
			return fmt.Errorf("config not loaded")
		}
		if deployPool == "" && len(args) > 0 {
			deployPool = args[0]
		}
		if deployPool == "" {
			return fmt.Errorf("pool address required")
		}
		if deployAmount <= 0 {
			deployAmount = cfg.Management.DeployAmountSol
		}
		if deployBinsBelow <= 0 {
			deployBinsBelow = cfg.Strategy.DefaultBinsBelow
		}

		input := dlmm.DeployInput{
			PoolAddress: deployPool,
			AmountY:     deployAmount,
			BinsBelow:   deployBinsBelow,
			Strategy:    cfg.Strategy.Strategy,
		}
		result, err := dlmm.DeployPosition(input, cfg)
		if err != nil {
			OutputJSON(map[string]any{"error": err.Error()})
			return nil
		}
		OutputJSON(result)
		return nil
	},
}

var closePosAddr string

var closePosCmd = &cobra.Command{
	Use:   "close",
	Short: "Close a position",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if cfg == nil {
			return fmt.Errorf("config not loaded")
		}
		if closePosAddr == "" && len(args) > 0 {
			closePosAddr = args[0]
		}
		if closePosAddr == "" {
			return fmt.Errorf("position address required")
		}
		result, err := dlmm.ClosePosition(closePosAddr, "manual", false, cfg)
		if err != nil {
			OutputJSON(map[string]any{"error": err.Error()})
			return nil
		}
		OutputJSON(result)
		return nil
	},
}

var claimPosAddr string

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim fees from a position",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if claimPosAddr == "" && len(args) > 0 {
			claimPosAddr = args[0]
		}
		result, err := dlmm.ClaimFees(claimPosAddr, cfg)
		if err != nil {
			OutputJSON(map[string]any{"error": err.Error()})
			return nil
		}
		OutputJSON(result)
		return nil
	},
}

var (
	swapInput  string
	swapOutput string
	swapAmount float64
)

var swapCmd = &cobra.Command{
	Use:   "swap",
	Short: "Swap tokens via Jupiter",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if swapInput == "" || swapOutput == "" || swapAmount <= 0 {
			return fmt.Errorf("--input-mint, --output-mint, and --amount required")
		}
		result := agent.ExecuteToolForCLI("swap_token", map[string]any{
			"input_mint":  swapInput,
			"output_mint": swapOutput,
			"amount":      swapAmount,
		}, cfg)
		OutputJSON(result)
		return nil
	},
}

var screenPool string

var screenCmd = &cobra.Command{
	Use:   "screen",
	Short: "Run screening cycle or check active bin",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if screenPool != "" {
			walletAddr := cfg.WalletAddress()
			client := dlmm.NewClient(walletAddr, cfg.RPCURLOrDefault())
			bin, err := client.GetActiveBin(screenPool)
			if err != nil {
				OutputJSON(map[string]any{"error": err.Error()})
				return nil
			}
			OutputJSON(bin)
			return nil
		}
		fmt.Println("Screening cycle — run 'meridian start' for autonomous mode")
		fmt.Println("Use --pool <address> to check active bin for a specific pool")
		return nil
	},
}

var managePool string

var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Run management cycle or get position PnL",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		if managePool != "" {
			walletAddr := cfg.WalletAddress()
			client := dlmm.NewClient(walletAddr, cfg.RPCURLOrDefault())
			pnl, err := client.GetPositionPnl(managePool, managePool)
			if err != nil {
				OutputJSON(map[string]any{"error": err.Error()})
				return nil
			}
			OutputJSON(pnl)
			return nil
		}
		fmt.Println("Management cycle — run 'meridian start' for autonomous mode")
		return nil
	},
}

func init() {
	deployCmd.Flags().StringVarP(&deployPool, "pool", "p", "", "Pool address")
	deployCmd.Flags().Float64VarP(&deployAmount, "amount", "a", 0, "SOL amount to deploy")
	deployCmd.Flags().IntVarP(&deployBinsBelow, "bins", "b", 0, "Bins below active bin")

	closePosCmd.Flags().StringVarP(&closePosAddr, "position", "p", "", "Position address")
	claimCmd.Flags().StringVarP(&claimPosAddr, "position", "p", "", "Position address")

	swapCmd.Flags().StringVar(&swapInput, "input-mint", "", "Input token mint")
	swapCmd.Flags().StringVar(&swapOutput, "output-mint", "", "Output token mint")
	swapCmd.Flags().Float64Var(&swapAmount, "amount", 0, "SOL amount")

	screenCmd.Flags().StringVarP(&screenPool, "pool", "p", "", "Pool address for active bin check")
	manageCmd.Flags().StringVarP(&managePool, "pool", "p", "", "Pool address for PnL check")

	RootCmd.AddCommand(balanceCmd)
	RootCmd.AddCommand(positionsCmd)
	RootCmd.AddCommand(configCmd)
	RootCmd.AddCommand(screenCmd)
	RootCmd.AddCommand(manageCmd)
	RootCmd.AddCommand(deployCmd)
	RootCmd.AddCommand(closePosCmd)
	RootCmd.AddCommand(claimCmd)
	RootCmd.AddCommand(swapCmd)
}

func OutputJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}
