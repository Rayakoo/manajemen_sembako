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

// BatchHandler exposes CRUD endpoints for purchase stock batches.
type BatchHandler struct {
	batchService *service.BatchService
}

func NewBatchHandler(batchService *service.BatchService) *BatchHandler {
	return &BatchHandler{batchService: batchService}
}

func (h *BatchHandler) Create(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.CreateBatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	batch, err := h.batchService.Create(userID, input)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "product not found")
		case errors.Is(err, service.ErrInvalidDateFormat):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to create batch")
		}
		return
	}

	response.JSON(c, http.StatusCreated, "batch created", batch)
}

func (h *BatchHandler) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var productID *uint
	if pidStr := c.Query("product_id"); pidStr != "" {
		pid, err := strconv.ParseUint(pidStr, 10, 64)
		if err != nil || pid == 0 {
			response.Error(c, http.StatusBadRequest, "invalid product_id query parameter")
			return
		}
		u := uint(pid)
		productID = &u
	}

	batches, err := h.batchService.List(userID, productID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to list batches")
		return
	}

	response.JSON(c, http.StatusOK, "batches retrieved", batches)
}

func (h *BatchHandler) GetByID(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid batch id")
		return
	}

	batch, err := h.batchService.GetByID(userID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.Error(c, http.StatusNotFound, "batch not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to get batch")
		return
	}

	response.JSON(c, http.StatusOK, "batch retrieved", batch)
}

func (h *BatchHandler) Update(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid batch id")
		return
	}

	var input service.UpdateBatchInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	batch, err := h.batchService.Update(userID, id, input)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "batch not found")
		case errors.Is(err, service.ErrInitialStockBelowSold):
			response.Error(c, http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrInvalidDateFormat):
			response.Error(c, http.StatusBadRequest, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to update batch")
		}
		return
	}

	response.JSON(c, http.StatusOK, "batch updated", batch)
}

func (h *BatchHandler) Delete(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid batch id")
		return
	}

	if err := h.batchService.Delete(userID, id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "batch not found")
		case errors.Is(err, service.ErrBatchHasSoldStock):
			response.Error(c, http.StatusConflict, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to delete batch")
		}
		return
	}

	response.JSON(c, http.StatusOK, "batch deleted", nil)
}
