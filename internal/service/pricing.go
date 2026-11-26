package service

import "math"

// RoundCurrency rounds a monetary value to 2 decimal places.
func RoundCurrency(val float64) float64 {
	return math.Round(val*100) / 100
}

// ComputeUnitBuyPrice calculates the per-unit buy price from total batch price and initial stock.
// Returns 0 when initialStock is <= 0.
func ComputeUnitBuyPrice(totalPrice float64, initialStock int64) float64 {
	if initialStock <= 0 {
		return 0
	}
	return RoundCurrency(totalPrice / float64(initialStock))
}

// ComputeSimpleAverageBuyPrice calculates the simple arithmetic mean of batch buy prices.
// Returns 0 when there are no batches.
func ComputeSimpleAverageBuyPrice(buyPrices []float64) float64 {
	if len(buyPrices) == 0 {
		return 0
	}
	var sum float64
	for _, price := range buyPrices {
		sum += price
	}
	return RoundCurrency(sum / float64(len(buyPrices)))
}

// ComputeSaleFinancials calculates bruto revenue and netto profit (bruto - modal).
func ComputeSaleFinancials(quantity int64, sellPrice, totalModal float64) (bruto float64, netto float64) {
	bruto = RoundCurrency(float64(quantity) * sellPrice)
	modal := RoundCurrency(totalModal)
	netto = RoundCurrency(bruto - modal)
	return bruto, netto
}
