package service

import "math"

// RoundCurrency rounds a monetary value to 2 decimal places.
func RoundCurrency(val float64) float64 {
	return math.Round(val*100) / 100
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
