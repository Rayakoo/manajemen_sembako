package handler

import (
	"errors"
	"net/http"
	"strconv"

	"ayam-batu-api/internal/middleware"
	"ayam-batu-api/internal/repository"
	"ayam-batu-api/internal/response"
	"ayam-batu-api/internal/service"

	"github.com/gin-gonic/gin"
)

// ProductHandler exposes CRUD endpoints for products and their stock/price summary.
type ProductHandler struct {
	productService *service.ProductService
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

func (h *ProductHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.CreateProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.productService.Create(userID, input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to create product")
		return
	}

	response.JSON(c, http.StatusCreated, "product created", product)
}

func (h *ProductHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	products, err := h.productService.List(userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list products")
		return
	}

	response.JSON(c, http.StatusOK, "products retrieved", products)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	detail, err := h.productService.GetByID(userID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get product")
		return
	}

	response.JSON(c, http.StatusOK, "product retrieved", detail)
}

func (h *ProductHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	var input service.UpdateProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.productService.Update(userID, id, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update product")
		return
	}

	response.JSON(c, http.StatusOK, "product updated", product)
}

func (h *ProductHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.productService.Delete(userID, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to delete product")
		return
	}

	response.JSON(c, http.StatusOK, "product deleted", nil)
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || val == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(val), nil
}
