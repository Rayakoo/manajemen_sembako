package repository

import (
	"errors"

	"ayam-batu-api/internal/models"

	"gorm.io/gorm"
)

// BatchRepository manages purchase batch persistence.
type BatchRepository struct {
	db *gorm.DB
}

func NewBatchRepository(db *gorm.DB) *BatchRepository {
	return &BatchRepository{db: db}
}

func (r *BatchRepository) DB() *gorm.DB {
	return r.db
}

func (r *BatchRepository) Create(batch *models.StockBatch) error {
	return r.db.Create(batch).Error
}

func (r *BatchRepository) FindByID(userID, batchID uint) (*models.StockBatch, error) {
	var batch models.StockBatch
	err := r.db.Where("id = ? AND user_id = ?", batchID, userID).First(&batch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (r *BatchRepository) List(userID uint, productID *uint) ([]models.StockBatchResponse, error) {
	var rows []models.StockBatchResponse
	query := `
		SELECT
			sb.*,
			(sb.initial_stock - sb.sold_stock) AS remaining_stock,
			p.name AS product_name
		FROM stock_batches sb
		JOIN products p ON p.id = sb.product_id
		WHERE sb.user_id = ?
	`
	args := []any{userID}
	if productID != nil {
		query += " AND sb.product_id = ?"
		args = append(args, *productID)
	}
	query += " ORDER BY sb.purchased_at DESC, sb.id DESC"

	if err := r.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *BatchRepository) Delete(userID, batchID uint) error {
	res := r.db.Where("id = ? AND user_id = ?", batchID, userID).Delete(&models.StockBatch{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
