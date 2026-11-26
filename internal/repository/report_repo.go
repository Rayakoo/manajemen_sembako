package repository

import "gorm.io/gorm"

// FinancialReportFilter scopes financial reports by date range and/or product.
type FinancialReportFilter struct {
	ProductID *uint
	StartDate string
	EndDate   string
}

// ProductFinancialRow holds stock, pricing, and profit figures for one product.
type ProductFinancialRow struct {
	ProductID          uint    `json:"product_id"`
	ProductName        string  `json:"product_name"`
	Unit               string  `json:"unit"`
	BoughtStock        int64   `json:"bought_stock"`
	SoldStock          int64   `json:"sold_stock"`
	RemainingStock     int64   `json:"remaining_stock"`
	MinSellPrice       float64 `json:"min_sell_price"`
	TotalPurchaseModal float64 `json:"total_purchase_modal"`
	Modal              float64 `json:"modal"`
	Bruto              float64 `json:"bruto"`
	Netto              float64 `json:"netto"`
}

// FinancialSummary holds top-level modal, bruto, netto, and stock totals.
type FinancialSummary struct {
	Modal              float64               `json:"modal"`
	Bruto              float64               `json:"bruto"`
	Netto              float64               `json:"netto"`
	TotalPurchaseModal float64               `json:"total_purchase_modal"`
	TotalTransactions  int64                 `json:"total_transactions"`
	TotalBoughtStock   int64                 `json:"total_bought_stock"`
	TotalSoldStock     int64                 `json:"total_sold_stock"`
	RemainingStock     int64                 `json:"remaining_stock"`
	Products           []ProductFinancialRow `json:"products"`
}

// ReportRepository aggregates sales and batch data for financial reporting.
type ReportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) GetFinancialSummary(userID uint, filter FinancialReportFilter) (*FinancialSummary, error) {
	salesJoinCond := "s.product_id = p.id AND s.user_id = p.user_id"
	args := []any{}
	if filter.StartDate != "" {
		salesJoinCond += " AND s.sold_at >= ?"
		args = append(args, filter.StartDate)
	}
	if filter.EndDate != "" {
		salesJoinCond += " AND s.sold_at <= ?"
		args = append(args, filter.EndDate)
	}

	batchJoinCond := "sb.product_id = p.id AND sb.user_id = p.user_id"

	query := `
		WITH batch_agg AS (
			SELECT
				p.id AS product_id,
				COALESCE(SUM(sb.initial_stock), 0) AS bought_stock,
				COALESCE(SUM(sb.sold_stock), 0) AS current_sold_stock,
				COALESCE(SUM(sb.initial_stock - sb.sold_stock), 0) AS remaining_stock,
				COALESCE(ROUND(AVG(sb.buy_price), 2), 0) AS min_sell_price,
				COALESCE(SUM(sb.initial_stock * sb.buy_price), 0) AS total_purchase_modal
			FROM products p
			LEFT JOIN stock_batches sb ON ` + batchJoinCond + `
			WHERE p.user_id = ?
			GROUP BY p.id
		),
		sales_agg AS (
			SELECT
				p.id AS product_id,
				COUNT(s.id) AS total_transactions,
				COALESCE(SUM(s.quantity), 0) AS sold_stock,
				COALESCE(SUM(s.total_modal), 0) AS modal,
				COALESCE(SUM(s.total_bruto), 0) AS bruto,
				COALESCE(SUM(s.total_netto), 0) AS netto
			FROM products p
			LEFT JOIN sales s ON ` + salesJoinCond + `
			WHERE p.user_id = ?
			GROUP BY p.id
		)
		SELECT
			p.id AS product_id,
			p.name AS product_name,
			p.unit AS unit,
			ba.bought_stock,
			sa.sold_stock,
			ba.remaining_stock,
			ba.min_sell_price,
			ba.total_purchase_modal,
			sa.modal,
			sa.bruto,
			sa.netto,
			sa.total_transactions
		FROM products p
		JOIN batch_agg ba ON ba.product_id = p.id
		JOIN sales_agg sa ON sa.product_id = p.id
		WHERE p.user_id = ?
	`

	// Placeholder order: userID (batch_agg), salesJoinCond args, userID (sales_agg), userID (outer WHERE)
	orderedArgs := make([]any, 0, len(args)+4)
	orderedArgs = append(orderedArgs, userID)
	orderedArgs = append(orderedArgs, args...)
	orderedArgs = append(orderedArgs, userID, userID)

	if filter.ProductID != nil {
		query += " AND p.id = ?"
		orderedArgs = append(orderedArgs, *filter.ProductID)
	}
	query += " ORDER BY p.id DESC"

	type rowWithTx struct {
		ProductFinancialRow
		TotalTransactions int64 `json:"-"`
	}
	var rows []rowWithTx
	if err := r.db.Raw(query, orderedArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	summary := &FinancialSummary{
		Products: make([]ProductFinancialRow, 0, len(rows)),
	}
	for _, row := range rows {
		summary.Modal += row.Modal
		summary.Bruto += row.Bruto
		summary.Netto += row.Netto
		summary.TotalPurchaseModal += row.TotalPurchaseModal
		summary.TotalTransactions += row.TotalTransactions
		summary.TotalBoughtStock += row.BoughtStock
		summary.TotalSoldStock += row.SoldStock
		summary.RemainingStock += row.RemainingStock
		summary.Products = append(summary.Products, row.ProductFinancialRow)
	}

	return summary, nil
}
