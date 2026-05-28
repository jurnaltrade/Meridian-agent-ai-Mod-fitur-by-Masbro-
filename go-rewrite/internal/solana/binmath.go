package solana

import (
	"math"
	"math/big"
)

var BASIS_POINT_MAX = big.NewInt(10_000)
var PRICE_SCALE = new(big.Int).Lsh(big.NewInt(1), 128)

func GetBinIdFromPrice(price *big.Int, binStep int, roundDown bool) int {
	bps := new(big.Int).Add(BASIS_POINT_MAX, big.NewInt(int64(binStep)))

	if price.Cmp(PRICE_SCALE) >= 0 {
		binID := 0
		threshold := new(big.Int).Set(PRICE_SCALE)
		for binID < 100_000 {
			next := new(big.Int).Mul(threshold, bps)
			next.Div(next, BASIS_POINT_MAX)
			cmp := price.Cmp(next)
			if cmp < 0 {
				if !roundDown && price.Cmp(threshold) > 0 {
					return binID + 1
				}
				return binID
			}
			if cmp == 0 {
				return binID + 1
			}
			threshold = next
			binID++
		}
		return binID
	}

	binID := 0
	threshold := new(big.Int).Set(PRICE_SCALE)
	for binID > -100_000 {
		next := new(big.Int).Mul(threshold, BASIS_POINT_MAX)
		next.Div(next, bps)
		binID--
		cmp := price.Cmp(next)
		if cmp > 0 {
			if !roundDown {
				return binID + 1
			}
			return binID
		}
		if cmp == 0 {
			return binID
		}
		threshold = next
	}
	return binID
}

func GetPriceOfBinByBinId(binId int, binStep int) *big.Int {
	bps := new(big.Int).Add(BASIS_POINT_MAX, big.NewInt(int64(binStep)))
	result := new(big.Int).Set(PRICE_SCALE)

	if binId >= 0 {
		absBin := big.NewInt(int64(binId))
		factor := new(big.Int).Exp(bps, absBin, nil)
		denom := new(big.Int).Exp(BASIS_POINT_MAX, absBin, nil)
		result.Mul(result, factor)
		result.Div(result, denom)
		return result
	}

	absBin := big.NewInt(int64(-binId))
	factor := new(big.Int).Exp(bps, absBin, nil)
	denom := new(big.Int).Exp(BASIS_POINT_MAX, absBin, nil)
	result.Mul(result, denom)
	result.Div(result, factor)
	return result
}

func PriceBigToFloat(price *big.Int) float64 {
	f := new(big.Float).SetInt(price)
	scaleF := new(big.Float).SetInt(PRICE_SCALE)
	f = f.Quo(f, scaleF)
	result, _ := f.Float64()
	return result
}

func Round(val float64, decimals int) float64 {
	pow := math.Pow10(decimals)
	return math.Round(val*pow) / pow
}
