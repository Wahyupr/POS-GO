package models

import (
	"time"

	"github.com/google/uuid"
)

// ─── Product ──────────────────────────────────────────────────────────────────

type ProductUnit string
type ProductStatus string
type BulkPricingMode string

const (
	UnitPCS ProductUnit = "PCS"
	UnitKG  ProductUnit = "KG"
	UnitONS ProductUnit = "ONS"
	UnitDUS ProductUnit = "DUS"
)

const (
	ProductActive   ProductStatus = "ACTIVE"
	ProductInactive ProductStatus = "INACTIVE"
)

const (
	PricingUnitPrice   BulkPricingMode = "UNIT_PRICE"
	PricingBundleTotal BulkPricingMode = "BUNDLE_TOTAL"
)

type Product struct {
	ID         uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MerchantID uuid.UUID     `gorm:"type:uuid;not null;index"                       json:"merchant_id"`
	Name       string        `gorm:"not null"                                       json:"name"`
	Category   string        `gorm:"type:varchar(50);not null;default:'Lainnya'"    json:"category"`
	ImageURL   *string       `gorm:"column:image_url"                               json:"image_url,omitempty"`
	Barcode    *string       `gorm:"column:barcode;uniqueIndex"                     json:"barcode,omitempty"`
	Unit       ProductUnit   `gorm:"type:varchar(10);not null;default:'PCS'"        json:"unit"`
	PriceBase  float64       `gorm:"type:decimal(12,2);not null"                    json:"price_base"`
	PriceCost  float64       `gorm:"type:decimal(12,2);not null;default:0"          json:"price_cost"`
	Stock      float64       `gorm:"type:decimal(12,3);not null;default:0"          json:"stock"`
	Status     ProductStatus `gorm:"type:varchar(10);not null;default:'ACTIVE'"     json:"status"`
	BulkTiers  []BulkTier    `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"bulk_tiers,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type BulkTier struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID   uuid.UUID       `gorm:"type:uuid;not null;index"                       json:"product_id"`
	MinQty      float64         `gorm:"type:decimal(12,3);not null"                    json:"min_qty"`
	PricingMode BulkPricingMode `gorm:"type:varchar(20);not null"                      json:"pricing_mode"`
	UnitPrice   *float64        `gorm:"type:decimal(12,2)"                             json:"unit_price,omitempty"`
	BundleQty   *float64        `gorm:"type:decimal(12,3)"                             json:"bundle_qty,omitempty"`
	BundleTotal *float64        `gorm:"type:decimal(12,2)"                             json:"bundle_total,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ─── Customer ─────────────────────────────────────────────────────────────────

type CustomerStatus string

const (
	CustomerActive   CustomerStatus = "ACTIVE"
	CustomerInactive CustomerStatus = "INACTIVE"
)

type Customer struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MerchantID uuid.UUID      `gorm:"type:uuid;not null;index"                       json:"merchant_id"`
	Name       string         `gorm:"not null"                                       json:"name"`
	Phone      *string        `json:"phone,omitempty"`
	Status     CustomerStatus `gorm:"type:varchar(10);not null;default:'ACTIVE'"     json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// ─── Queue ────────────────────────────────────────────────────────────────────

type QueueStatus string

const (
	QueuePending QueueStatus = "PENDING"
	QueueProcess QueueStatus = "PROCESS"
	QueueDone    QueueStatus = "DONE"
)

type Queue struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MerchantID   uuid.UUID   `gorm:"type:uuid;not null;index"                       json:"merchant_id"`
	SaleID       *uuid.UUID  `gorm:"type:uuid"                                      json:"sale_id,omitempty"`
	CustomerName *string     `gorm:"column:customer_name"                           json:"customer_name,omitempty"`
	Status       QueueStatus `gorm:"type:varchar(10);not null;default:'PENDING'"    json:"status"`
	Notes        *string     `json:"notes,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Sale         *Sale       `gorm:"foreignKey:SaleID;references:ID" json:"-"`
}

// ─── Sale ─────────────────────────────────────────────────────────────────────

type SaleStatus string
type PaymentMethod string

const (
	SalePaid    SaleStatus = "PAID"
	SalePartial SaleStatus = "PARTIAL"
	SaleDebt    SaleStatus = "DEBT"
)

const (
	PaymentCash PaymentMethod = "CASH"
)

type Sale struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MerchantID   uuid.UUID  `gorm:"type:uuid;not null;index"                       json:"merchant_id"`
	CustomerID   *uuid.UUID `gorm:"type:uuid"                                      json:"customer_id,omitempty"`
	CustomerName *string    `gorm:"column:customer_name"                           json:"customer_name,omitempty"`
	Status       SaleStatus `gorm:"type:varchar(10);not null"                      json:"status"`
	QueueID      *uuid.UUID `gorm:"type:uuid"                                      json:"queue_id,omitempty"`
	IsQueue      bool       `gorm:"column:is_queue;not null;default:false"         json:"is_queue"`
	Total        float64    `gorm:"type:decimal(12,2);not null"                    json:"total"`
	Discount     float64    `gorm:"type:decimal(12,2);not null;default:0"          json:"discount"`
	Paid         float64    `gorm:"type:decimal(12,2);not null;default:0"          json:"paid"`
	Change       float64    `gorm:"type:decimal(12,2);not null;default:0"          json:"change"`
	Notes        *string    `json:"notes,omitempty"`
	Items        []SaleItem `gorm:"foreignKey:SaleID;constraint:OnDelete:CASCADE"  json:"items,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type SaleItem struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SaleID           uuid.UUID `gorm:"type:uuid;not null;index"                       json:"sale_id"`
	ProductID        uuid.UUID `gorm:"type:uuid;not null"                             json:"product_id"`
	ProductName      string    `gorm:"column:product_name;not null"                   json:"product_name"`
	Unit             string    `gorm:"not null;default:'PCS'"                         json:"unit"`
	Qty              float64   `gorm:"type:decimal(12,3);not null"                    json:"qty"`
	UnitPriceApplied float64   `gorm:"type:decimal(12,2);not null"                    json:"unit_price_applied"`
	LineTotal        float64   `gorm:"type:decimal(12,2);not null"                    json:"line_total"`
}

type Payment struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SaleID    uuid.UUID     `gorm:"type:uuid;not null;index"                       json:"sale_id"`
	Amount    float64       `gorm:"type:decimal(12,2);not null"                    json:"amount"`
	Method    PaymentMethod `gorm:"type:varchar(10);not null;default:'CASH'"       json:"method"`
	CreatedAt time.Time     `json:"created_at"`
}
