package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ayam-batu-api/internal/models"
	"ayam-batu-api/internal/repository"

	"gorm.io/gorm"
)

var (
	ErrInitialStockBelowSold = errors.New("initial_stock cannot be less than sold_stock")
	ErrBatchHasSoldStock     = errors.New("cannot delete batch that already has sold stock")
	ErrInvalidDateFormat     = errors.New("invalid date format; expected YYYY-MM-DD or RFC3339")
)

// BatchService manages stock batch CRUD and stock/price consistency.
type BatchService struct {
	batchRepo   *repository.BatchRepository
	productRepo *repository.ProductRepository
}

func NewBatchService(batchRepo *repository.BatchRepository, productRepo *repository.ProductRepository) *BatchService {
	return &BatchService{
		batchRepo:   batchRepo,
		productRepo: productRepo,
	}
}

type CreateBatchInput struct {
	ProductID    uint     `json:"product_id" binding:"required,gt=0"`
	BatchCode    string   `json:"batch_code" binding:"max=100"`
	TotalPrice   *float64 `json:"total_price" binding:"required,gte=0"`
	InitialStock int64    `json:"initial_stock" binding:"required,gt=0"`
	PurchasedAt  string   `json:"purchased_at"`
	Notes        string   `json:"notes"`
}

type UpdateBatchInput struct {
	BatchCode    *string  `json:"batch_code" binding:"omitempty,min=1,max=100"`
	TotalPrice   *float64 `json:"total_price" binding:"omitempty,gte=0"`
	InitialStock *int64   `json:"initial_stock" binding:"omitempty,gt=0"`
	PurchasedAt  *string  `json:"purchased_at"`
	Notes        *string  `json:"notes"`
}

func (s *BatchService) Create(userID uint, input CreateBatchInput) (*models.StockBatchResponse, error) {
	if _, err := s.productRepo.FindByID(userID, input.ProductID); err != nil {
		return nil, err
	}

	purchasedAt := time.Now()
	if strings.TrimSpace(input.PurchasedAt) != "" {
		parsed, err := parseFlexibleTime(input.PurchasedAt)
		if err != nil {
			return nil, ErrInvalidDateFormat
		}
		purchasedAt = parsed
	}

	batchCode := strings.TrimSpace(input.BatchCode)
	if batchCode == "" {
		batchCode = fmt.Sprintf("BATCH-%s", time.Now().Format("20060102-150405"))
	}

	totalPrice := RoundCurrency(*input.TotalPrice)
	unitBuyPrice := ComputeUnitBuyPrice(totalPrice, input.InitialStock)

	batch := &models.StockBatch{
		UserID:       userID,
		ProductID:    input.ProductID,
		BatchCode:    batchCode,
		TotalPrice:   totalPrice,
		BuyPrice:     unitBuyPrice,
		InitialStock: input.InitialStock,
		SoldStock:    0,
		PurchasedAt:  purchasedAt,
		Notes:        strings.TrimSpace(input.Notes),
	}
	if err := s.batchRepo.Create(batch); err != nil {
		return nil, err
	}

	return s.GetByID(userID, batch.ID)
}

func (s *BatchService) List(userID uint, productID *uint) ([]models.StockBatchResponse, error) {
	return s.batchRepo.List(userID, productID)
}

func (s *BatchService) GetByID(userID, batchID uint) (*models.StockBatchResponse, error) {
	batch, err := s.batchRepo.FindByID(userID, batchID)
	if err != nil {
		return nil, err
	}
	product, err := s.productRepo.FindByID(userID, batch.ProductID)
	if err != nil {
		return nil, err
	}

	return &models.StockBatchResponse{
		StockBatch:     *batch,
		RemainingStock: batch.InitialStock - batch.SoldStock,
		ProductName:    product.Name,
	}, nil
}

func (s *BatchService) Update(userID, batchID uint, input UpdateBatchInput) (*models.StockBatchResponse, error) {
	err := s.batchRepo.DB().Transaction(func(tx *gorm.DB) error {
		var batch models.StockBatch
		if err := tx.Where("id = ? AND user_id = ?", batchID, userID).First(&batch).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrNotFound
			}
			return err
		}

		pricingInputChanged := false
		if input.InitialStock != nil {
			if *input.InitialStock < batch.SoldStock {
				return ErrInitialStockBelowSold
			}
			if *input.InitialStock != batch.InitialStock {
				batch.InitialStock = *input.InitialStock
				pricingInputChanged = true
			}
		}
		if input.BatchCode != nil {
			batch.BatchCode = strings.TrimSpace(*input.BatchCode)
		}
		if input.PurchasedAt != nil && strings.TrimSpace(*input.PurchasedAt) != "" {
			parsed, err := parseFlexibleTime(*input.PurchasedAt)
			if err != nil {
				return ErrInvalidDateFormat
			}
			batch.PurchasedAt = parsed
		}
		if input.Notes != nil {
			batch.Notes = strings.TrimSpace(*input.Notes)
		}

		if input.TotalPrice != nil {
			rounded := RoundCurrency(*input.TotalPrice)
			if rounded != batch.TotalPrice {
				batch.TotalPrice = rounded
				pricingInputChanged = true
			}
		}

		buyPriceChanged := false
		if pricingInputChanged {
			newBuyPrice := ComputeUnitBuyPrice(batch.TotalPrice, batch.InitialStock)
			if newBuyPrice != batch.BuyPrice {
				batch.BuyPrice = newBuyPrice
				buyPriceChanged = true
			}
		}

		if err := tx.Save(&batch).Error; err != nil {
			return err
		}

		// Keep sale allocation cost and sale modal/netto consistent if derived unit buy_price changed.
		if buyPriceChanged && batch.SoldStock > 0 {
			if err := recalculateSalesForBatch(tx, batch.ID, batch.BuyPrice); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetByID(userID, batchID)
}

func (s *BatchService) Delete(userID, batchID uint) error {
	batch, err := s.batchRepo.FindByID(userID, batchID)
	if err != nil {
		return err
	}
	if batch.SoldStock > 0 {
		return ErrBatchHasSoldStock
	}
	return s.batchRepo.Delete(userID, batchID)
}

func recalculateSalesForBatch(tx *gorm.DB, batchID uint, newBuyPrice float64) error {
	if err := tx.Model(&models.SaleBatchAllocation{}).
		Where("batch_id = ?", batchID).
		Update("buy_price", newBuyPrice).Error; err != nil {
		return err
	}

	var affectedSaleIDs []uint
	if err := tx.Model(&models.SaleBatchAllocation{}).
		Where("batch_id = ?", batchID).
		Distinct("sale_id").
		Pluck("sale_id", &affectedSaleIDs).Error; err != nil {
		return err
	}

	for _, saleID := range affectedSaleIDs {
		var sale models.Sale
		if err := tx.First(&sale, saleID).Error; err != nil {
			return err
		}
		var newModal float64
		if err := tx.Model(&models.SaleBatchAllocation{}).
			Where("sale_id = ?", saleID).
			Select("COALESCE(SUM(quantity * buy_price), 0)").
			Scan(&newModal).Error; err != nil {
			return err
		}
		sale.TotalModal = RoundCurrency(newModal)
		sale.TotalBruto, sale.TotalNetto = ComputeSaleFinancials(sale.Quantity, sale.SellPrice, sale.TotalModal)
		if err := tx.Save(&sale).Error; err != nil {
			return err
		}
	}

	return nil
}

func parseFlexibleTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", trimmed); err == nil {
		return t, nil
	}
	return time.Time{}, ErrInvalidDateFormat
}
