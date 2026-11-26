package service

import "testing"

func TestComputeUnitBuyPrice(t *testing.T) {
	tests := []struct {
		name         string
		totalPrice   float64
		initialStock int64
		want         float64
	}{
		{
			name:         "zero stock returns zero",
			totalPrice:   500000,
			initialStock: 0,
			want:         0,
		},
		{
			name:         "exact division",
			totalPrice:   500000,
			initialStock: 20,
			want:         25000,
		},
		{
			name:         "rounded to two decimals",
			totalPrice:   100000,
			initialStock: 3,
			want:         33333.33,
		},
		{
			name:         "zero total price",
			totalPrice:   0,
			initialStock: 10,
			want:         0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeUnitBuyPrice(tc.totalPrice, tc.initialStock)
			if got != tc.want {
				t.Fatalf("ComputeUnitBuyPrice(%.2f, %d) = %.2f, want %.2f", tc.totalPrice, tc.initialStock, got, tc.want)
			}
		})
	}
}

func TestComputeSimpleAverageBuyPrice(t *testing.T) {
	tests := []struct {
		name   string
		prices []float64
		want   float64
	}{
		{
			name:   "no batches returns zero",
			prices: nil,
			want:   0,
		},
		{
			name:   "single batch",
			prices: []float64{22000},
			want:   22000,
		},
		{
			name:   "two batches even average",
			prices: []float64{20000, 25000},
			want:   22500,
		},
		{
			name:   "three batches rounded to two decimals",
			prices: []float64{20000, 25000, 26000},
			want:   23666.67,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeSimpleAverageBuyPrice(tc.prices)
			if got != tc.want {
				t.Fatalf("ComputeSimpleAverageBuyPrice(%v) = %.2f, want %.2f", tc.prices, got, tc.want)
			}
		})
	}
}

func TestComputeSaleFinancials(t *testing.T) {
	tests := []struct {
		name       string
		qty        int64
		sellPrice  float64
		totalModal float64
		wantBruto  float64
		wantNetto  float64
	}{
		{
			name:       "profitable sale",
			qty:        10,
			sellPrice:  30000,
			totalModal: 225000,
			wantBruto:  300000,
			wantNetto:  75000,
		},
		{
			name:       "break-even sale",
			qty:        5,
			sellPrice:  20000,
			totalModal: 100000,
			wantBruto:  100000,
			wantNetto:  0,
		},
		{
			name:       "loss sale",
			qty:        4,
			sellPrice:  18000,
			totalModal: 80000,
			wantBruto:  72000,
			wantNetto:  -8000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bruto, netto := ComputeSaleFinancials(tc.qty, tc.sellPrice, tc.totalModal)
			if bruto != tc.wantBruto || netto != tc.wantNetto {
				t.Fatalf("got (bruto=%.2f, netto=%.2f), want (bruto=%.2f, netto=%.2f)",
					bruto, netto, tc.wantBruto, tc.wantNetto)
			}
		})
	}
}
