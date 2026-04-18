package handlers

import (
	"net/http"

	"pos-backend/internal/middleware"
	"pos-backend/internal/models"
	"pos-backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserPOSHandler struct {
	productRepo  *repository.ProductRepository
	customerRepo *repository.CustomerRepository
	saleRepo     *repository.SaleRepository
	queueRepo    *repository.QueueRepository
}

func NewUserPOSHandler(
	productRepo *repository.ProductRepository,
	customerRepo *repository.CustomerRepository,
	saleRepo *repository.SaleRepository,
	queueRepo *repository.QueueRepository,
) *UserPOSHandler {
	return &UserPOSHandler{productRepo: productRepo, customerRepo: customerRepo, saleRepo: saleRepo, queueRepo: queueRepo}
}

func (h *UserPOSHandler) merchantID(c *gin.Context) (uuid.UUID, bool) {
	user, ok := middleware.GetUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return uuid.Nil, false
	}
	if user.MerchantID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun ini tidak terhubung ke merchant manapun"})
		return uuid.Nil, false
	}
	return *user.MerchantID, true
}

// GET /api/v1/user/products  — returns active products for the user's merchant
func (h *UserPOSHandler) ListProducts(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	products, err := h.productRepo.ListByMerchant(mid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil produk"})
		return
	}
	// Filter only active products
	var active []models.Product
	for _, p := range products {
		if p.Status == models.ProductActive {
			active = append(active, p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": active})
}

// GET /api/v1/user/customers — list active customers for the user's merchant
func (h *UserPOSHandler) ListCustomers(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	customers, err := h.customerRepo.ListByMerchant(mid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil pelanggan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customers})
}

// GET /api/v1/user/sales
func (h *UserPOSHandler) ListSales(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	user, _ := middleware.GetUser(c)
	status := c.Query("status")

	sales, err := h.saleRepo.ListByMerchant(mid, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat"})
		return
	}
	// Filter to sales created by this user (cashier_id would be ideal but not in schema —
	// for now return all merchant sales so the user can see store history)
	_ = user
	c.JSON(http.StatusOK, gin.H{"data": sales})
}

// POST /api/v1/user/sales
func (h *UserPOSHandler) CreateSale(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	h.createSale(c, mid)
}

// PUT /api/v1/user/sales/:id/pay
func (h *UserPOSHandler) PaySale(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	sid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	var body struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sale, err := h.saleRepo.FindByID(sid)
	if err != nil || sale.MerchantID != mid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
		return
	}
	sale.Paid += body.Amount
	if sale.Paid >= sale.Total {
		sale.Paid = sale.Total
		sale.Status = models.SalePaid
	} else {
		sale.Status = models.SalePartial
	}
	if err := h.saleRepo.Update(sale); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pembayaran"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sale})
}

// GET /api/v1/user/products/barcode?code=xxx
func (h *UserPOSHandler) FindProductByBarcode(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kode barcode wajib diisi"})
		return
	}
	product, err := h.productRepo.FindByBarcode(mid, code)
	if err != nil || product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

// POST /api/v1/user/customers
func (h *UserPOSHandler) CreateCustomer(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	var body struct {
		Name  string  `json:"name"  binding:"required"`
		Phone *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	customer := &models.Customer{
		MerchantID: mid,
		Name:       body.Name,
		Phone:      body.Phone,
		Status:     models.CustomerActive,
	}
	if err := h.customerRepo.Create(customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan pelanggan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": customer})
}

func (h *UserPOSHandler) createSale(c *gin.Context, merchantID uuid.UUID) {
	var body struct {
		CustomerID   *string           `json:"customer_id"`
		CustomerName *string           `json:"customer_name"`
		Status       models.SaleStatus `json:"status"  binding:"required"`
		IsQueue      bool              `json:"is_queue"`
		Total        float64           `json:"total"   binding:"required,gt=0"`
		Discount     float64           `json:"discount"`
		Paid         float64           `json:"paid"`
		Change       float64           `json:"change"`
		Notes        *string           `json:"notes"`
		Items        []struct {
			ProductID        string  `json:"product_id"         binding:"required"`
			ProductName      string  `json:"product_name"       binding:"required"`
			Unit             string  `json:"unit"`
			Qty              float64 `json:"qty"                binding:"required,gt=0"`
			UnitPriceApplied float64 `json:"unit_price_applied" binding:"required,gt=0"`
			LineTotal        float64 `json:"line_total"         binding:"required,gt=0"`
		} `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sale := &models.Sale{
		MerchantID:   merchantID,
		CustomerName: body.CustomerName,
		Status:       body.Status,
		IsQueue:      body.IsQueue,
		Total:        body.Total,
		Discount:     body.Discount,
		Paid:         body.Paid,
		Change:       body.Change,
		Notes:        body.Notes,
	}
	if body.CustomerID != nil {
		id, err := uuid.Parse(*body.CustomerID)
		if err == nil {
			sale.CustomerID = &id
		}
	}
	for _, it := range body.Items {
		pid, _ := uuid.Parse(it.ProductID)
		unit := it.Unit
		if unit == "" {
			unit = "PCS"
		}
		sale.Items = append(sale.Items, models.SaleItem{
			ProductID:        pid,
			ProductName:      it.ProductName,
			Unit:             unit,
			Qty:              it.Qty,
			UnitPriceApplied: it.UnitPriceApplied,
			LineTotal:        it.LineTotal,
		})
	}
	if err := h.saleRepo.Create(sale); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": sale})
}

// GET /api/v1/user/queues?status=PENDING
func (h *UserPOSHandler) ListQueues(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	status := c.Query("status")
	queues, err := h.queueRepo.ListByMerchant(mid, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil antrian"})
		return
	}
	var response []gin.H
	for _, q := range queues {
		itemNames := []string{}
		if q.Sale != nil && len(q.Sale.Items) > 0 {
			for _, item := range q.Sale.Items {
				itemNames = append(itemNames, item.ProductName)
			}
		}
		response = append(response, gin.H{
			"id":            q.ID,
			"merchant_id":   q.MerchantID,
			"sale_id":       q.SaleID,
			"customer_name": q.CustomerName,
			"status":        q.Status,
			"notes":         q.Notes,
			"item_names":    itemNames,
			"created_at":    q.CreatedAt,
			"updated_at":    q.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": response})
}

// POST /api/v1/user/queues
func (h *UserPOSHandler) AddQueue(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	var body struct {
		CustomerName *string `json:"customer_name"`
		Notes        *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	q := &models.Queue{
		MerchantID:   mid,
		CustomerName: body.CustomerName,
		Notes:        body.Notes,
		Status:       models.QueuePending,
	}
	if err := h.queueRepo.Create(q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan antrian"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": q})
}

// PUT /api/v1/user/queues/:id
func (h *UserPOSHandler) UpdateQueue(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	q, err := h.queueRepo.FindByID(id)
	if err != nil || q == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Antrian tidak ditemukan"})
		return
	}
	var body struct {
		Status models.QueueStatus `json:"status" binding:"required"`
		Notes  *string            `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	q.Status = body.Status
	if body.Notes != nil {
		q.Notes = body.Notes
	}
	if err := h.queueRepo.Update(q); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui antrian"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": q})
}
