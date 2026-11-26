package service

import "ayam-batu-api/internal/repository"

// ReportService produces financial and daily profit reports.
type ReportService struct {
	reportRepo *repository.ReportRepository
	saleRepo   *repository.SaleRepository
}

func NewReportService(reportRepo *repository.ReportRepository, saleRepo *repository.SaleRepository) *ReportService {
	return &ReportService{
		reportRepo: reportRepo,
		saleRepo:   saleRepo,
	}
}

func (s *ReportService) FinancialSummary(userID uint, filter repository.FinancialReportFilter) (*repository.FinancialSummary, error) {
	summary, err := s.reportRepo.GetFinancialSummary(userID, filter)
	if err != nil {
		return nil, err
	}
	summary.Modal = RoundCurrency(summary.Modal)
	summary.Bruto = RoundCurrency(summary.Bruto)
	summary.Netto = RoundCurrency(summary.Bruto - summary.Modal)
	summary.TotalPurchaseModal = RoundCurrency(summary.TotalPurchaseModal)
	return summary, nil
}

func (s *ReportService) DailyReport(userID uint, filter repository.SaleFilter) ([]repository.DailySaleSummary, error) {
	return s.saleRepo.DailySummary(userID, filter)
}
