package database

import (
	"fmt"
	"log"

	"pos-backend/internal/config"
	"pos-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() (*gorm.DB, error) {
	cfg := config.App

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Jakarta",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	logLevel := logger.Silent
	if cfg.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&models.Merchant{},
		&models.User{},
		&models.RefreshToken{},
		&models.Product{},
		&models.BulkTier{},
		&models.Customer{},
		&models.Queue{},
		&models.Sale{},
		&models.SaleItem{},
		&models.Payment{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	DB = db
	log.Println("Database connected successfully")
	return db, nil
}

func Migrate() {
	err := DB.AutoMigrate(
		&models.Merchant{},
		&models.User{},
		&models.RefreshToken{},
		&models.Product{},
		&models.BulkTier{},
		&models.Customer{},
		&models.Queue{},
		&models.Sale{},
		&models.SaleItem{},
		&models.Payment{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")
}
