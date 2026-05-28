package solana

import (
	"math/big"
	"testing"
)

func TestGetPriceOfBinByBinId_BinZero(t *testing.T) {
	price := GetPriceOfBinByBinId(0, 100)
	if price.Cmp(PRICE_SCALE) != 0 {
		t.Errorf("Bin 0 price should be 2^128, got %s", price)
	}
}

func TestGetPriceOfBinByBinId_Monotonic(t *testing.T) {
	bps := 100
	var prev *big.Int
	for binID := -5; binID <= 5; binID++ {
		price := GetPriceOfBinByBinId(binID, bps)
		if prev != nil && price.Cmp(prev) <= 0 {
			t.Errorf("Price not monotonic: bin %d price %s <= bin %d price %s",
				binID, price, binID-1, prev)
		}
		prev = price
	}
}

func TestPriceRoundTrip(t *testing.T) {
	bps := 100
	for binID := -5; binID <= 5; binID++ {
		price := GetPriceOfBinByBinId(binID, bps)
		got := GetBinIdFromPrice(price, bps, true)
		if got != binID {
			t.Errorf("Round-trip for bin %d: got bin %d", binID, got)
		}
	}
}

func TestPriceBigToFloat(t *testing.T) {
	f := PriceBigToFloat(PRICE_SCALE)
	if f != 1.0 {
		t.Errorf("PriceBigToFloat(2^128) = %f, want 1.0", f)
	}
}

func TestRound(t *testing.T) {
	tests := []struct {
		val      float64
		decimals int
		expected float64
	}{
		{1.234567, 2, 1.23},
		{1.234567, 4, 1.2346},
		{5.0, 2, 5.0},
	}
	for _, tt := range tests {
		got := Round(tt.val, tt.decimals)
		if got != tt.expected {
			t.Errorf("Round(%f, %d) = %f, want %f", tt.val, tt.decimals, got, tt.expected)
		}
	}
}

func TestLamportsToSol(t *testing.T) {
	tests := []struct {
		lamports uint64
		expected float64
	}{
		{1_000_000_000, 1.0},
		{500_000_000, 0.5},
		{0, 0},
	}
	for _, tt := range tests {
		got := LamportsToSol(tt.lamports)
		if got != tt.expected {
			t.Errorf("LamportsToSol(%d) = %f, want %f", tt.lamports, got, tt.expected)
		}
	}
}

func TestSolToLamports(t *testing.T) {
	tests := []struct {
		sol      float64
		expected uint64
	}{
		{1.0, 1_000_000_000},
		{0.5, 500_000_000},
		{0, 0},
	}
	for _, tt := range tests {
		got := SolToLamports(tt.sol)
		if got != tt.expected {
			t.Errorf("SolToLamports(%f) = %d, want %d", tt.sol, got, tt.expected)
		}
	}
}
