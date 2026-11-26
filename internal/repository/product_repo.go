package repository

import (
	"errors"

	"ayam-batu-api/internal/models"

	"gorm.io/gorm"
)

// ProductRepository manages product persistence and aggregated stock queries.
type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) FindByID(userID, productID uint) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("id = ? AND user_id = ?", productID, userID).First(&product).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) ListWithSummary(userID uint) ([]models.ProductWithSummary, error) {
	var items []models.ProductWithSummary
	query := `
		SELECT
			p.id, p.user_id, p.name, p.sku, p.unit, p.description, p.created_at, p.updated_at,
			COUNT(sb.id) AS total_batches,
			COALESCE(SUM(sb.initial_stock), 0) AS total_bought_stock,
			COALESCE(SUM(sb.sold_stock), 0) AS total_sold_stock,
			COALESCE(SUM(sb.initial_stock - sb.sold_stock), 0) AS remaining_stock,
			COALESCE(ROUND(AVG(sb.buy_price), 2), 0) AS average_buy_price,
			COALESCE(ROUND(AVG(sb.buy_price), 2), 0) AS min_sell_price
		FROM products p
		LEFT JOIN stock_batches sb ON sb.product_id = p.id AND sb.user_id = p.user_id
		WHERE p.user_id = ?
		GROUP BY p.id
		ORDER BY p.id DESC
	`
	if err := r.db.Raw(query, userID).Scan(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ProductRepository) GetSummaryByID(userID, productID uint) (*models.ProductWithSummary, error) {
	var item models.ProductWithSummary
	query := `
		SELECT
			p.id, p.user_id, p.name, p.sku, p.unit, p.description, p.created_at, p.updated_at,
			COUNT(sb.id) AS total_batches,
			COALESCE(SUM(sb.initial_stock), 0) AS total_bought_stock,
			COALESCE(SUM(sb.sold_stock), 0) AS total_sold_stock,
			COALESCE(SUM(sb.initial_stock - sb.sold_stock), 0) AS remaining_stock,
			COALESCE(ROUND(AVG(sb.buy_price), 2), 0) AS average_buy_price,
			COALESCE(ROUND(AVG(sb.buy_price), 2), 0) AS min_sell_price
		FROM products p
		LEFT JOIN stock_batches sb ON sb.product_id = p.id AND sb.user_id = p.user_id
		WHERE p.user_id = ? AND p.id = ?
		GROUP BY p.id
	`
	res := r.db.Raw(query, userID, productID).Scan(&item)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *ProductRepository) Update(product *models.Product) error {
	return r.db.Save(product).Error
}

func (r *ProductRepository) Delete(userID, productID uint) error {
	res := r.db.Where("id = ? AND user_id = ?", productID, userID).Delete(&models.Product{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
