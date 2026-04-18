package main

import (
	"log"

	"pos-go/internal/config"
	"pos-go/internal/database"
	"pos-go/internal/handlers"
	"pos-go/internal/repository"
	"pos-go/internal/router"
	"pos-go/internal/services"
)

func main() {
	// Load config dari .env
	config.Load()

	// Koneksi ke PostgreSQL
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("gagal koneksi database: %v", err)
	}

	// Repository
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	merchantRepo := repository.NewMerchantRepository(db)
	productRepo := repository.NewProductRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	queueRepo := repository.NewQueueRepository(db)
	saleRepo := repository.NewSaleRepository(db)

	// Service
	authSvc := services.NewAuthService(userRepo, tokenRepo)

	// Handler
	authHandler := handlers.NewAuthHandler(authSvc)
	adminHandler := handlers.NewAdminHandler(userRepo, merchantRepo)
	merchantHandler := handlers.NewMerchantHandler(productRepo, customerRepo, queueRepo, saleRepo, userRepo)
	userPOSHandler := handlers.NewUserPOSHandler(productRepo, customerRepo, saleRepo, queueRepo)

	// Router
	r := router.Setup(authHandler, adminHandler, merchantHandler, userPOSHandler, authSvc)

	addr := ":" + config.App.Port
	log.Printf("Server berjalan di %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("gagal menjalankan server: %v", err)
	}
}
