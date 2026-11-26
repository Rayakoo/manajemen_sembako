package handler

import (
	"net/http"
	"strconv"

	"ayam-batu-api/internal/middleware"
	"ayam-batu-api/internal/repository"
	"ayam-batu-api/internal/response"
	"ayam-batu-api/internal/service"

	"github.com/gin-gonic/gin"
)

// ReportHandler exposes profit and financial summary endpoints.
type ReportHandler struct {
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) Financial(c *gin.Context) {
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

	filter := repository.FinancialReportFilter{
		ProductID: productID,
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}

	summary, err := h.reportService.FinancialSummary(userID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to generate financial report")
		return
	}

	response.JSON(c, http.StatusOK, "financial report retrieved", summary)
}

func (h *ReportHandler) Daily(c *gin.Context) {
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

	rows, err := h.reportService.DailyReport(userID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to generate daily report")
		return
	}

	response.JSON(c, http.StatusOK, "daily financial report retrieved", rows)
}
