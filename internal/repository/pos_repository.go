package repository

import (
	"pos-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Product ──────────────────────────────────────────────────────────────────

type ProductRepository struct{ db *gorm.DB }

func NewProductRepository(db *gorm.DB) *ProductRepository { return &ProductRepository{db: db} }

func (r *ProductRepository) ListByMerchant(merchantID uuid.UUID) ([]models.Product, error) {
	var p []models.Product
	err := r.db.Preload("BulkTiers").Where("merchant_id = ?", merchantID).Order("name").Find(&p).Error
	return p, err
}

func (r *ProductRepository) Create(p *models.Product) error { return r.db.Create(p).Error }

func (r *ProductRepository) FindByID(id uuid.UUID) (*models.Product, error) {
	var p models.Product
	err := r.db.Preload("BulkTiers").First(&p, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &p, err
}

func (r *ProductRepository) FindByBarcode(merchantID uuid.UUID, barcode string) (*models.Product, error) {
	var p models.Product
	err := r.db.Preload("BulkTiers").Where("merchant_id = ? AND barcode = ? AND status = 'ACTIVE'", merchantID, barcode).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &p, err
}

func (r *ProductRepository) Update(p *models.Product) error { return r.db.Save(p).Error }

func (r *ProductRepository) UpdateFields(id uuid.UUID, fields map[string]interface{}) error {
	return r.db.Model(&models.Product{}).Where("id = ?", id).Updates(fields).Error
}

func (r *ProductRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Product{}, "id = ?", id).Error
}

// BulkTier sub-operations
func (r *ProductRepository) ListBulkTiers(productID uuid.UUID) ([]models.BulkTier, error) {
	var tiers []models.BulkTier
	err := r.db.Where("product_id = ?", productID).Order("min_qty").Find(&tiers).Error
	return tiers, err
}

func (r *ProductRepository) CreateBulkTier(t *models.BulkTier) error { return r.db.Create(t).Error }

func (r *ProductRepository) DeleteBulkTier(id uuid.UUID) error {
	return r.db.Delete(&models.BulkTier{}, "id = ?", id).Error
}

// ─── Customer ─────────────────────────────────────────────────────────────────

type CustomerRepository struct{ db *gorm.DB }

func NewCustomerRepository(db *gorm.DB) *CustomerRepository { return &CustomerRepository{db: db} }

func (r *CustomerRepository) ListByMerchant(merchantID uuid.UUID) ([]models.Customer, error) {
	var c []models.Customer
	err := r.db.Where("merchant_id = ?", merchantID).Order("name").Find(&c).Error
	return c, err
}

func (r *CustomerRepository) Create(c *models.Customer) error { return r.db.Create(c).Error }

func (r *CustomerRepository) FindByID(id uuid.UUID) (*models.Customer, error) {
	var c models.Customer
	err := r.db.First(&c, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &c, err
}

func (r *CustomerRepository) Update(c *models.Customer) error { return r.db.Save(c).Error }

// ─── Queue ────────────────────────────────────────────────────────────────────

type QueueRepository struct{ db *gorm.DB }

func NewQueueRepository(db *gorm.DB) *QueueRepository { return &QueueRepository{db: db} }

func (r *QueueRepository) ListByMerchant(merchantID uuid.UUID, status string) ([]models.Queue, error) {
	var q []models.Queue
	query := r.db.Where("merchant_id = ?", merchantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Preload("Sale", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Items")
	}).Order("created_at desc").Find(&q).Error
	return q, err
}

func (r *QueueRepository) Create(q *models.Queue) error { return r.db.Create(q).Error }

func (r *QueueRepository) FindByID(id uuid.UUID) (*models.Queue, error) {
	var q models.Queue
	err := r.db.First(&q, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &q, err
}

func (r *QueueRepository) Update(q *models.Queue) error { return r.db.Save(q).Error }

// ─── Sale ─────────────────────────────────────────────────────────────────────

type SaleRepository struct{ db *gorm.DB }

func NewSaleRepository(db *gorm.DB) *SaleRepository { return &SaleRepository{db: db} }

func (r *SaleRepository) ListByMerchant(merchantID uuid.UUID, status string) ([]models.Sale, error) {
	var s []models.Sale
	query := r.db.Preload("Items").Where("merchant_id = ?", merchantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("created_at desc").Find(&s).Error
	return s, err
}

func (r *SaleRepository) Create(s *models.Sale) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(s).Error; err != nil {
			return err
		}
		for _, item := range s.Items {
			if err := tx.Model(&models.Product{}).
				Where("id = ?", item.ProductID).
				UpdateColumn("stock", gorm.Expr("stock - ?", item.Qty)).Error; err != nil {
				return err
			}
		}
		// If this is a queue order, create a Queue record
		if s.IsQueue {
			queue := &models.Queue{
				MerchantID:   s.MerchantID,
				SaleID:       &s.ID,
				CustomerName: s.CustomerName,
				Status:       models.QueuePending,
				Notes:        s.Notes,
			}
			if err := tx.Create(queue).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *SaleRepository) FindByID(id uuid.UUID) (*models.Sale, error) {
	var s models.Sale
	err := r.db.Preload("Items").First(&s, "id = ?", id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &s, err
}

func (r *SaleRepository) Update(s *models.Sale) error { return r.db.Save(s).Error }
