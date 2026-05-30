package cli

import (
	"github.com/spf13/cobra"
	"meridian-go-rewrite/internal/config"
)

var (
	cfgPath string
	dryRun  bool
)

var RootCmd = &cobra.Command{
	Use:   "meridian",
	Short: "Solana DLMM LP Agent CLI",
	Long:  "Autonomous DLMM liquidity agent with LLM-powered screening and management.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(cfgPath)
		if err != nil {
			panic(err)
		}
		if dryRun {
			cfg.DryRun = true
		}
		config.Set(cfg)
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "Path to config file")
	RootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Skip on-chain transactions")
}
