package main

import (
	"log"

	"ayam-batu-api/internal/config"
	"ayam-batu-api/internal/database"
	"ayam-batu-api/internal/handler"
	"ayam-batu-api/internal/repository"
	"ayam-batu-api/internal/router"
	"ayam-batu-api/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	batchRepo := repository.NewBatchRepository(db)
	saleRepo := repository.NewSaleRepository(db)
	reportRepo := repository.NewReportRepository(db)

	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiresHours, service.GoogleOAuthOptions{
		ClientID:            cfg.GoogleClientID,
		ClientSecret:        cfg.GoogleClientSecret,
		RedirectURL:         cfg.GoogleRedirectURL,
		FrontendRedirectURL: cfg.GoogleFrontendRedirectURL,
	})
	productSvc := service.NewProductService(productRepo, batchRepo)
	batchSvc := service.NewBatchService(batchRepo, productRepo)
	saleSvc := service.NewSaleService(saleRepo, productRepo)
	reportSvc := service.NewReportService(reportRepo, saleRepo)

	engine := router.Setup(cfg.GinMode, router.Dependencies{
		JWTSecret:      cfg.JWTSecret,
		AuthHandler:    handler.NewAuthHandler(authSvc),
		ProductHandler: handler.NewProductHandler(productSvc),
		BatchHandler:   handler.NewBatchHandler(batchSvc),
		SaleHandler:    handler.NewSaleHandler(saleSvc),
		ReportHandler:  handler.NewReportHandler(reportSvc),
	})

	addr := ":" + cfg.AppPort
	log.Printf("Ayam Batu API listening on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
