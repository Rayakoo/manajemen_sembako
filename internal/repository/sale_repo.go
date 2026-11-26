package repository

import (
	"errors"

	"ayam-batu-api/internal/models"

	"gorm.io/gorm"
)

// SaleFilter filters sales list and daily summary queries.
type SaleFilter struct {
	ProductID *uint
	Date      string
	StartDate string
	EndDate   string
}

// DailySaleSummary represents aggregated sales for a single calendar date.
type DailySaleSummary struct {
	Date              string  `json:"date"`
	TotalTransactions int64   `json:"total_transactions"`
	TotalQuantitySold int64   `json:"total_quantity_sold"`
	TotalModal        float64 `json:"total_modal"`
	TotalBruto        float64 `json:"total_bruto"`
	TotalNetto        float64 `json:"total_netto"`
}

// SaleWithProduct enriches Sale with product name and unit.
type SaleWithProduct struct {
	models.Sale
	ProductName string `json:"product_name"`
	ProductUnit string `json:"product_unit"`
}

// SaleRepository handles persistence and querying for sales.
type SaleRepository struct {
	db *gorm.DB
}

func NewSaleRepository(db *gorm.DB) *SaleRepository {
	return &SaleRepository{db: db}
}

func (r *SaleRepository) DB() *gorm.DB {
	return r.db
}

func (r *SaleRepository) FindByID(userID, saleID uint) (*models.Sale, error) {
	var sale models.Sale
	err := r.db.Preload("Allocations").Where("id = ? AND user_id = ?", saleID, userID).First(&sale).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sale, nil
}

func (r *SaleRepository) List(userID uint, filter SaleFilter) ([]SaleWithProduct, error) {
	var rows []SaleWithProduct
	query := `
		SELECT
			s.*,
			p.name AS product_name,
			p.unit AS product_unit
		FROM sales s
		JOIN products p ON p.id = s.product_id
		WHERE s.user_id = ?
	`
	args := []any{userID}
	if filter.ProductID != nil {
		query += " AND s.product_id = ?"
		args = append(args, *filter.ProductID)
	}
	if filter.Date != "" {
		query += " AND s.sold_at = ?"
		args = append(args, filter.Date)
	}
	if filter.StartDate != "" {
		query += " AND s.sold_at >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		query += " AND s.sold_at <= ?"
		args = append(args, filter.EndDate)
	}
	query += " ORDER BY s.sold_at DESC, s.id DESC"

	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *SaleRepository) DailySummary(userID uint, filter SaleFilter) ([]DailySaleSummary, error) {
	var rows []DailySaleSummary
	query := `
		SELECT
			TO_CHAR(s.sold_at, 'YYYY-MM-DD') AS date,
			COUNT(s.id) AS total_transactions,
			COALESCE(SUM(s.quantity), 0) AS total_quantity_sold,
			COALESCE(SUM(s.total_modal), 0) AS total_modal,
			COALESCE(SUM(s.total_bruto), 0) AS total_bruto,
			COALESCE(SUM(s.total_netto), 0) AS total_netto
		FROM sales s
		WHERE s.user_id = ?
	`
	args := []any{userID}
	if filter.ProductID != nil {
		query += " AND s.product_id = ?"
		args = append(args, *filter.ProductID)
	}
	if filter.Date != "" {
		query += " AND s.sold_at = ?"
		args = append(args, filter.Date)
	}
	if filter.StartDate != "" {
		query += " AND s.sold_at >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		query += " AND s.sold_at <= ?"
		args = append(args, filter.EndDate)
	}
	query += " GROUP BY s.sold_at ORDER BY s.sold_at DESC"

	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
