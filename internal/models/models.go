package models

import "time"

// User represents a registered account.
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:120;not null" json:"name"`
	Email        string    `gorm:"size:180;uniqueIndex;not null" json:"email"`
	PasswordHash *string   `gorm:"" json:"-"`
	AuthProvider string    `gorm:"size:30;not null;default:'local'" json:"auth_provider"`
	GoogleID     *string   `gorm:"size:255;uniqueIndex" json:"google_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Product represents a catalog item whose batches and sales are tracked.
type Product struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Name        string    `gorm:"size:160;not null" json:"name"`
	SKU         string    `gorm:"size:80;not null;default:''" json:"sku"`
	Unit        string    `gorm:"size:40;not null;default:'pcs'" json:"unit"`
	Description string    `gorm:"not null;default:''" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProductWithSummary enriches a Product with aggregated stock and pricing metrics.
type ProductWithSummary struct {
	Product
	TotalBatches     int64   `json:"total_batches"`
	TotalBoughtStock int64   `json:"total_bought_stock"`
	TotalSoldStock   int64   `json:"total_sold_stock"`
	RemainingStock   int64   `json:"remaining_stock"`
	AverageBuyPrice  float64 `json:"average_buy_price"`
	MinSellPrice     float64 `json:"min_sell_price"`
}

// StockBatch records a single purchase batch with its total batch price, derived unit buy price, and stock movement.
type StockBatch struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index" json:"user_id"`
	ProductID    uint      `gorm:"not null;index" json:"product_id"`
	BatchCode    string    `gorm:"size:100;not null" json:"batch_code"`
	TotalPrice   float64   `gorm:"type:numeric(15,2);not null" json:"total_price"`
	BuyPrice     float64   `gorm:"type:numeric(15,2);not null" json:"buy_price"`
	InitialStock int64     `gorm:"not null" json:"initial_stock"`
	SoldStock    int64     `gorm:"not null;default:0" json:"sold_stock"`
	PurchasedAt  time.Time `gorm:"not null" json:"purchased_at"`
	Notes        string    `gorm:"not null;default:''" json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StockBatchResponse includes the computed remaining stock field.
type StockBatchResponse struct {
	StockBatch
	RemainingStock int64  `json:"remaining_stock"`
	ProductName    string `json:"product_name,omitempty"`
}

// Sale records a daily product sale transaction and its financial totals.
type Sale struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	UserID      uint                  `gorm:"not null;index" json:"user_id"`
	ProductID   uint                  `gorm:"not null;index" json:"product_id"`
	Quantity    int64                 `gorm:"not null" json:"quantity"`
	SellPrice   float64               `gorm:"type:numeric(15,2);not null" json:"sell_price"`
	TotalModal  float64               `gorm:"type:numeric(15,2);not null;default:0" json:"total_modal"`
	TotalBruto  float64               `gorm:"type:numeric(15,2);not null;default:0" json:"total_bruto"`
	TotalNetto  float64               `gorm:"type:numeric(15,2);not null;default:0" json:"total_netto"`
	SoldAt      time.Time             `gorm:"type:date;not null" json:"sold_at"`
	Notes       string                `gorm:"not null;default:''" json:"notes"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Allocations []SaleBatchAllocation `gorm:"foreignKey:SaleID" json:"allocations,omitempty"`
}

// SaleBatchAllocation records how much stock was deducted from each batch for a sale (FIFO).
type SaleBatchAllocation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SaleID    uint      `gorm:"not null;index" json:"sale_id"`
	BatchID   uint      `gorm:"not null;index" json:"batch_id"`
	Quantity  int64     `gorm:"not null" json:"quantity"`
	BuyPrice  float64   `gorm:"type:numeric(15,2);not null" json:"buy_price"`
	CreatedAt time.Time `json:"created_at"`
}
