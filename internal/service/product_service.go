package service

import (
	"strings"

	"ayam-batu-api/internal/models"
	"ayam-batu-api/internal/repository"
)

// ProductService manages product CRUD operations.
type ProductService struct {
	productRepo *repository.ProductRepository
	batchRepo   *repository.BatchRepository
}

func NewProductService(productRepo *repository.ProductRepository, batchRepo *repository.BatchRepository) *ProductService {
	return &ProductService{
		productRepo: productRepo,
		batchRepo:   batchRepo,
	}
}

type CreateProductInput struct {
	Name        string `json:"name" binding:"required,min=1,max=160"`
	SKU         string `json:"sku" binding:"max=80"`
	Unit        string `json:"unit" binding:"max=40"`
	Description string `json:"description"`
}

type UpdateProductInput struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=160"`
	SKU         *string `json:"sku" binding:"omitempty,max=80"`
	Unit        *string `json:"unit" binding:"omitempty,max=40"`
	Description *string `json:"description"`
}

type ProductDetail struct {
	models.ProductWithSummary
	Batches []models.StockBatchResponse `json:"batches"`
}

func (s *ProductService) Create(userID uint, input CreateProductInput) (*models.ProductWithSummary, error) {
	unit := strings.TrimSpace(input.Unit)
	if unit == "" {
		unit = "pcs"
	}

	product := &models.Product{
		UserID:      userID,
		Name:        strings.TrimSpace(input.Name),
		SKU:         strings.TrimSpace(input.SKU),
		Unit:        unit,
		Description: strings.TrimSpace(input.Description),
	}
	if err := s.productRepo.Create(product); err != nil {
		return nil, err
	}

	return s.productRepo.GetSummaryByID(userID, product.ID)
}

func (s *ProductService) List(userID uint) ([]models.ProductWithSummary, error) {
	return s.productRepo.ListWithSummary(userID)
}

func (s *ProductService) GetByID(userID, productID uint) (*ProductDetail, error) {
	summary, err := s.productRepo.GetSummaryByID(userID, productID)
	if err != nil {
		return nil, err
	}

	batches, err := s.batchRepo.List(userID, &productID)
	if err != nil {
		return nil, err
	}

	return &ProductDetail{
		ProductWithSummary: *summary,
		Batches:            batches,
	}, nil
}

func (s *ProductService) Update(userID, productID uint, input UpdateProductInput) (*models.ProductWithSummary, error) {
	product, err := s.productRepo.FindByID(userID, productID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		product.Name = strings.TrimSpace(*input.Name)
	}
	if input.SKU != nil {
		product.SKU = strings.TrimSpace(*input.SKU)
	}
	if input.Unit != nil {
		unit := strings.TrimSpace(*input.Unit)
		if unit != "" {
			product.Unit = unit
		}
	}
	if input.Description != nil {
		product.Description = strings.TrimSpace(*input.Description)
	}

	if err := s.productRepo.Update(product); err != nil {
		return nil, err
	}
	return s.productRepo.GetSummaryByID(userID, product.ID)
}

func (s *ProductService) Delete(userID, productID uint) error {
	return s.productRepo.Delete(userID, productID)
}
