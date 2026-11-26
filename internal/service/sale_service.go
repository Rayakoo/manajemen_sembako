package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ayam-batu-api/internal/models"
	"ayam-batu-api/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientStock = errors.New("insufficient remaining stock for this sale")
)

// SellPriceBelowMinimumError indicates the requested sell price is below the product's average buy price.
type SellPriceBelowMinimumError struct {
	SellPrice    float64
	MinSellPrice float64
}

func (e *SellPriceBelowMinimumError) Error() string {
	return fmt.Sprintf(
		"sell_price (%.2f) is below minimum sell price (%.2f) based on average batch buy price; set allow_below_min=true to override",
		e.SellPrice, e.MinSellPrice,
	)
}

// SaleService handles daily sales CRUD with FIFO batch stock allocation.
type SaleService struct {
	saleRepo    *repository.SaleRepository
	productRepo *repository.ProductRepository
}

func NewSaleService(saleRepo *repository.SaleRepository, productRepo *repository.ProductRepository) *SaleService {
	return &SaleService{
		saleRepo:    saleRepo,
		productRepo: productRepo,
	}
}

type CreateSaleInput struct {
	ProductID     uint     `json:"product_id" binding:"required,gt=0"`
	Quantity      int64    `json:"quantity" binding:"required,gt=0"`
	SellPrice     *float64 `json:"sell_price" binding:"required,gte=0"`
	SoldAt        string   `json:"sold_at"`
	Notes         string   `json:"notes"`
	AllowBelowMin bool     `json:"allow_below_min"`
}

type UpdateSaleInput struct {
	ProductID     *uint    `json:"product_id" binding:"omitempty,gt=0"`
	Quantity      *int64   `json:"quantity" binding:"omitempty,gt=0"`
	SellPrice     *float64 `json:"sell_price" binding:"omitempty,gte=0"`
	SoldAt        *string  `json:"sold_at"`
	Notes         *string  `json:"notes"`
	AllowBelowMin bool     `json:"allow_below_min"`
}

func (s *SaleService) Create(userID uint, input CreateSaleInput) (*models.Sale, error) {
	soldAt := currentDateUTC()
	if strings.TrimSpace(input.SoldAt) != "" {
		parsed, err := parseDateOnly(input.SoldAt)
		if err != nil {
			return nil, ErrInvalidDateFormat
		}
		soldAt = parsed
	}

	var createdID uint
	err := s.saleRepo.DB().Transaction(func(tx *gorm.DB) error {
		if err := ensureProductOwned(tx, userID, input.ProductID); err != nil {
			return err
		}

		sellPrice := RoundCurrency(*input.SellPrice)
		if err := validateMinSellPrice(tx, userID, input.ProductID, sellPrice, input.AllowBelowMin); err != nil {
			return err
		}

		allocations, totalModal, err := allocateBatchesFIFO(tx, userID, input.ProductID, input.Quantity)
		if err != nil {
			return err
		}

		bruto, netto := ComputeSaleFinancials(input.Quantity, sellPrice, totalModal)
		sale := &models.Sale{
			UserID:     userID,
			ProductID:  input.ProductID,
			Quantity:   input.Quantity,
			SellPrice:  sellPrice,
			TotalModal: totalModal,
			TotalBruto: bruto,
			TotalNetto: netto,
			SoldAt:     soldAt,
			Notes:      strings.TrimSpace(input.Notes),
		}
		if err := tx.Create(sale).Error; err != nil {
			return err
		}

		for i := range allocations {
			allocations[i].SaleID = sale.ID
		}
		if err := tx.Create(&allocations).Error; err != nil {
			return err
		}

		createdID = sale.ID
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.saleRepo.FindByID(userID, createdID)
}

func (s *SaleService) List(userID uint, filter repository.SaleFilter) ([]repository.SaleWithProduct, error) {
	return s.saleRepo.List(userID, filter)
}

func (s *SaleService) DailySummary(userID uint, filter repository.SaleFilter) ([]repository.DailySaleSummary, error) {
	return s.saleRepo.DailySummary(userID, filter)
}

func (s *SaleService) GetByID(userID, saleID uint) (*models.Sale, error) {
	return s.saleRepo.FindByID(userID, saleID)
}

func (s *SaleService) Update(userID, saleID uint, input UpdateSaleInput) (*models.Sale, error) {
	err := s.saleRepo.DB().Transaction(func(tx *gorm.DB) error {
		var sale models.Sale
		if err := tx.Preload("Allocations").Where("id = ? AND user_id = ?", saleID, userID).First(&sale).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrNotFound
			}
			return err
		}

		// Restore previously deducted batch stock before re-allocating.
		if err := restoreBatchAllocations(tx, sale.Allocations); err != nil {
			return err
		}
		if err := tx.Where("sale_id = ?", sale.ID).Delete(&models.SaleBatchAllocation{}).Error; err != nil {
			return err
		}

		if input.ProductID != nil {
			sale.ProductID = *input.ProductID
		}
		if input.Quantity != nil {
			sale.Quantity = *input.Quantity
		}
		if input.SellPrice != nil {
			sale.SellPrice = RoundCurrency(*input.SellPrice)
		}
		if input.SoldAt != nil && strings.TrimSpace(*input.SoldAt) != "" {
			parsed, err := parseDateOnly(*input.SoldAt)
			if err != nil {
				return ErrInvalidDateFormat
			}
			sale.SoldAt = parsed
		}
		if input.Notes != nil {
			sale.Notes = strings.TrimSpace(*input.Notes)
		}

		if err := ensureProductOwned(tx, userID, sale.ProductID); err != nil {
			return err
		}
		if err := validateMinSellPrice(tx, userID, sale.ProductID, sale.SellPrice, input.AllowBelowMin); err != nil {
			return err
		}

		allocations, totalModal, err := allocateBatchesFIFO(tx, userID, sale.ProductID, sale.Quantity)
		if err != nil {
			return err
		}

		sale.TotalModal = totalModal
		sale.TotalBruto, sale.TotalNetto = ComputeSaleFinancials(sale.Quantity, sale.SellPrice, totalModal)
		if err := tx.Save(&sale).Error; err != nil {
			return err
		}

		for i := range allocations {
			allocations[i].SaleID = sale.ID
		}
		return tx.Create(&allocations).Error
	})
	if err != nil {
		return nil, err
	}

	return s.saleRepo.FindByID(userID, saleID)
}

func (s *SaleService) Delete(userID, saleID uint) error {
	return s.saleRepo.DB().Transaction(func(tx *gorm.DB) error {
		var sale models.Sale
		if err := tx.Preload("Allocations").Where("id = ? AND user_id = ?", saleID, userID).First(&sale).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrNotFound
			}
			return err
		}

		if err := restoreBatchAllocations(tx, sale.Allocations); err != nil {
			return err
		}

		return tx.Delete(&sale).Error
	})
}

func ensureProductOwned(tx *gorm.DB, userID, productID uint) error {
	var count int64
	if err := tx.Model(&models.Product{}).Where("id = ? AND user_id = ?", productID, userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func validateMinSellPrice(tx *gorm.DB, userID, productID uint, sellPrice float64, allowBelowMin bool) error {
	if allowBelowMin {
		return nil
	}

	var buyPrices []float64
	if err := tx.Model(&models.StockBatch{}).
		Where("user_id = ? AND product_id = ?", userID, productID).
		Pluck("buy_price", &buyPrices).Error; err != nil {
		return err
	}

	minSellPrice := ComputeSimpleAverageBuyPrice(buyPrices)
	if minSellPrice > 0 && sellPrice < minSellPrice {
		return &SellPriceBelowMinimumError{
			SellPrice:    sellPrice,
			MinSellPrice: minSellPrice,
		}
	}
	return nil
}

func allocateBatchesFIFO(tx *gorm.DB, userID, productID uint, requiredQty int64) ([]models.SaleBatchAllocation, float64, error) {
	var batches []models.StockBatch
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND product_id = ? AND initial_stock > sold_stock", userID, productID).
		Order("purchased_at ASC, id ASC").
		Find(&batches).Error; err != nil {
		return nil, 0, err
	}

	var totalAvailable int64
	for _, b := range batches {
		totalAvailable += b.InitialStock - b.SoldStock
	}
	if totalAvailable < requiredQty {
		return nil, 0, ErrInsufficientStock
	}

	remainingToAllocate := requiredQty
	var allocations []models.SaleBatchAllocation
	var totalModal float64

	for i := range batches {
		if remainingToAllocate == 0 {
			break
		}
		available := batches[i].InitialStock - batches[i].SoldStock
		if available <= 0 {
			continue
		}

		take := available
		if remainingToAllocate < available {
			take = remainingToAllocate
		}

		batches[i].SoldStock += take
		if err := tx.Model(&batches[i]).Update("sold_stock", batches[i].SoldStock).Error; err != nil {
			return nil, 0, err
		}

		allocations = append(allocations, models.SaleBatchAllocation{
			BatchID:  batches[i].ID,
			Quantity: take,
			BuyPrice: batches[i].BuyPrice,
		})
		totalModal += float64(take) * batches[i].BuyPrice
		remainingToAllocate -= take
	}

	return allocations, RoundCurrency(totalModal), nil
}

func restoreBatchAllocations(tx *gorm.DB, allocations []models.SaleBatchAllocation) error {
	for _, alloc := range allocations {
		var batch models.StockBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&batch, alloc.BatchID).Error; err != nil {
			return err
		}
		batch.SoldStock -= alloc.Quantity
		if batch.SoldStock < 0 {
			batch.SoldStock = 0
		}
		if err := tx.Model(&batch).Update("sold_stock", batch.SoldStock).Error; err != nil {
			return err
		}
	}
	return nil
}

func parseDateOnly(value string) (time.Time, error) {
	t, err := parseFlexibleTime(value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
}

func currentDateUTC() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
