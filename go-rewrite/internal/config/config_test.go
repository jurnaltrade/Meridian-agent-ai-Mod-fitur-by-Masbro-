package config

import (
	"testing"
)

func TestComputeMinOpenBalance(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Management.MinSolToOpen = 0.55
	cfg.Management.DeployAmountSol = 0.5
	cfg.Management.GasReserve = 0.2
	got := ComputeMinOpenBalance(cfg)
	// floor = max(0.55, 0.5+0.2=0.7) = 0.7
	if got != 0.7 {
		t.Errorf("ComputeMinOpenBalance = %f, want 0.7", got)
	}

	cfg.Management.MinSolToOpen = 1.0
	cfg.Management.DeployAmountSol = 0.5
	cfg.Management.GasReserve = 0.2
	got = ComputeMinOpenBalance(cfg)
	// floor = max(1.0, 0.7) = 1.0
	if got != 1.0 {
		t.Errorf("ComputeMinOpenBalance = %f, want 1.0", got)
	}
}

func TestComputeDeployAmount(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Management.GasReserve = 0.2
	cfg.Management.PositionSizePct = 0.35
	cfg.Management.DeployAmountSol = 0.5
	cfg.Risk.MaxDeployAmount = 50

	tests := []struct {
		walletSol float64
		expected  float64
	}{
		{10, 3.43},
		{1, 0.5},
		{0.5, 0.5},
		{0.3, 0.5},
		{100, 34.93},
	}
	for _, tt := range tests {
		got := ComputeDeployAmount(tt.walletSol, cfg)
		if got != tt.expected {
			t.Errorf("ComputeDeployAmount(%f) = %f, want %f", tt.walletSol, got, tt.expected)
		}
	}
}

func TestDefaultConfig_HasRequiredFields(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Strategy.MinBinsBelow != MIN_SAFE_BINS_BELOW {
		t.Errorf("MinBinsBelow = %d, want %d", cfg.Strategy.MinBinsBelow, MIN_SAFE_BINS_BELOW)
	}
	if cfg.Strategy.DefaultBinsBelow != 69 {
		t.Errorf("DefaultBinsBelow = %d, want 69", cfg.Strategy.DefaultBinsBelow)
	}
	if cfg.Management.DeployAmountSol != 0.5 {
		t.Errorf("DeployAmountSol = %f, want 0.5", cfg.Management.DeployAmountSol)
	}
	if cfg.Management.GasReserve != 0.2 {
		t.Errorf("GasReserve = %f, want 0.2", cfg.Management.GasReserve)
	}
	if cfg.Screening.MinOrganic != 60 {
		t.Errorf("MinOrganic = %f, want 60", cfg.Screening.MinOrganic)
	}
}

func TestRPCURLOrDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RPCURLOrDefault() == "" {
		t.Error("RPCURLOrDefault returned empty")
	}

	cfg.RPCURL = "https://custom.rpc.com"
	if cfg.RPCURLOrDefault() != "https://custom.rpc.com" {
		t.Errorf("RPCURLOrDefault = %s, want custom rpc", cfg.RPCURLOrDefault())
	}
}

func TestWalletAddress(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WalletAddr = "TestWallet123"
	if cfg.WalletAddress() != "TestWallet123" {
		t.Errorf("WalletAddress = %s, want TestWallet123", cfg.WalletAddress())
	}
}
