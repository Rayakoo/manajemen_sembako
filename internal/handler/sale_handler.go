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

// SaleHandler exposes CRUD and daily summary endpoints for product sales.
type SaleHandler struct {
	saleService *service.SaleService
}

func NewSaleHandler(saleService *service.SaleService) *SaleHandler {
	return &SaleHandler{saleService: saleService}
}

func (h *SaleHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.CreateSaleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	sale, err := h.saleService.Create(userID, input)
	if err != nil {
		h.handleSaleError(c, err, "failed to create sale")
		return
	}

	response.JSON(c, http.StatusCreated, "sale recorded", sale)
}

func (h *SaleHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter, err := parseSaleFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	sales, err := h.saleService.List(userID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list sales")
		return
	}

	response.JSON(c, http.StatusOK, "sales retrieved", sales)
}

func (h *SaleHandler) DailySummary(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	filter, err := parseSaleFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	summary, err := h.saleService.DailySummary(userID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to get daily sales summary")
		return
	}

	response.JSON(c, http.StatusOK, "daily sales retrieved", summary)
}

func (h *SaleHandler) GetByID(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sale id")
		return
	}

	sale, err := h.saleService.GetByID(userID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "sale not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get sale")
		return
	}

	response.JSON(c, http.StatusOK, "sale retrieved", sale)
}

func (h *SaleHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sale id")
		return
	}

	var input service.UpdateSaleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	sale, err := h.saleService.Update(userID, id, input)
	if err != nil {
		h.handleSaleError(c, err, "failed to update sale")
		return
	}

	response.JSON(c, http.StatusOK, "sale updated", sale)
}

func (h *SaleHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid sale id")
		return
	}

	if err := h.saleService.Delete(userID, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "sale not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to delete sale")
		return
	}

	response.JSON(c, http.StatusOK, "sale deleted", nil)
}

func (h *SaleHandler) handleSaleError(c *gin.Context, err error, fallback string) {
	var minPriceErr *service.SellPriceBelowMinimumError
	switch {
	case errors.Is(err, repository.ErrNotFound):
		response.Error(c, http.StatusNotFound, "sale or product not found")
	case errors.Is(err, service.ErrInsufficientStock):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidDateFormat):
		response.Error(c, http.StatusBadRequest, err.Error())
	case errors.As(err, &minPriceErr):
		response.Error(c, http.StatusUnprocessableEntity, minPriceErr.Error())
	default:
		response.Error(c, http.StatusInternalServerError, fallback)
	}
}

func parseSaleFilter(c *gin.Context) (repository.SaleFilter, error) {
	var filter repository.SaleFilter
	if pidStr := c.Query("product_id"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 64)
		if err != nil || pid == 0 {
			return filter, errors.New("invalid product_id query parameter")
		}
		u := uint(pid)
		filter.ProductID = &u
	}
	filter.Date = c.Query("date")
	filter.StartDate = c.Query("start_date")
	filter.EndDate = c.Query("end_date")
	return filter, nil
}
