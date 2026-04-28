package handlers

import (
	"errors"
	"net/http"

	"pos-go/internal/middleware"
	"pos-go/internal/models"
	"pos-go/internal/repository"
	"pos-go/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type MerchantHandler struct {
	productRepo  *repository.ProductRepository
	customerRepo *repository.CustomerRepository
	queueRepo    *repository.QueueRepository
	saleRepo     *repository.SaleRepository
	userRepo     *repository.UserRepository
}

func NewMerchantHandler(
	productRepo *repository.ProductRepository,
	customerRepo *repository.CustomerRepository,
	queueRepo *repository.QueueRepository,
	saleRepo *repository.SaleRepository,
	userRepo *repository.UserRepository,
) *MerchantHandler {
	return &MerchantHandler{
		productRepo:  productRepo,
		customerRepo: customerRepo,
		queueRepo:    queueRepo,
		saleRepo:     saleRepo,
		userRepo:     userRepo,
	}
}

// merchantID extracts the merchant_id from the logged-in user context.
// Admins using the endpoint must pass merchant_id as a query param.
func (h *MerchantHandler) merchantID(c *gin.Context) (uuid.UUID, bool) {
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

// ─── Products ─────────────────────────────────────────────────────────────────

// GET /api/v1/merchant/products
func (h *MerchantHandler) ListProducts(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	products, err := h.productRepo.ListByMerchant(mid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil produk"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": products})
}

// POST /api/v1/merchant/products
func (h *MerchantHandler) CreateProduct(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	var body struct {
		Name      string               `json:"name"       binding:"required"`
		Category  string               `json:"category"`
		ImageURL  *string              `json:"image_url"`
		Barcode   *string              `json:"barcode"`
		Unit      models.ProductUnit   `json:"unit"       binding:"required"`
		PriceBase float64              `json:"price_base" binding:"required,gt=0"`
		PriceCost float64              `json:"price_cost"`
		Stock     float64              `json:"stock"`
		Status    models.ProductStatus `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Status == "" {
		body.Status = models.ProductActive
	}
	if body.Category == "" {
		body.Category = "Lainnya"
	}
	p := &models.Product{
		MerchantID: mid,
		Name:       body.Name,
		Category:   body.Category,
		ImageURL:   body.ImageURL,
		Barcode:    body.Barcode,
		Unit:       body.Unit,
		PriceBase:  body.PriceBase,
		PriceCost:  body.PriceCost,
		Stock:      body.Stock,
		Status:     body.Status,
	}
	if err := h.productRepo.Create(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat produk"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": p})
}

// PUT /api/v1/merchant/products/:id
func (h *MerchantHandler) UpdateProduct(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	p, err := h.productRepo.FindByID(id)
	if err != nil || p == nil || p.MerchantID != mid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.productRepo.UpdateFields(id, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui produk"})
		return
	}
	updated, _ := h.productRepo.FindByID(id)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

// DELETE /api/v1/merchant/products/:id
func (h *MerchantHandler) DeleteProduct(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	p, err := h.productRepo.FindByID(id)
	if err != nil || p == nil || p.MerchantID != mid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
		return
	}
	h.productRepo.Delete(id)
	c.JSON(http.StatusOK, gin.H{"message": "Produk dihapus"})
}

// ─── Bulk Tiers ───────────────────────────────────────────────────────────────

// GET /api/v1/merchant/products/:id/bulk-tiers
func (h *MerchantHandler) ListBulkTiers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	tiers, err := h.productRepo.ListBulkTiers(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil bulk tiers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tiers})
}

// POST /api/v1/merchant/products/:id/bulk-tiers
func (h *MerchantHandler) AddBulkTier(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	var body struct {
		MinQty      float64                `json:"min_qty"      binding:"required,gt=0"`
		PricingMode models.BulkPricingMode `json:"pricing_mode" binding:"required"`
		UnitPrice   *float64               `json:"unit_price"`
		BundleQty   *float64               `json:"bundle_qty"`
		BundleTotal *float64               `json:"bundle_total"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tier := &models.BulkTier{
		ProductID:   productID,
		MinQty:      body.MinQty,
		PricingMode: body.PricingMode,
		UnitPrice:   body.UnitPrice,
		BundleQty:   body.BundleQty,
		BundleTotal: body.BundleTotal,
	}
	if err := h.productRepo.CreateBulkTier(tier); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah bulk tier"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": tier})
}

// DELETE /api/v1/merchant/bulk-tiers/:id
func (h *MerchantHandler) DeleteBulkTier(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	h.productRepo.DeleteBulkTier(id)
	c.JSON(http.StatusOK, gin.H{"message": "Bulk tier dihapus"})
}

// ─── Customers ────────────────────────────────────────────────────────────────

// GET /api/v1/merchant/customers
func (h *MerchantHandler) ListCustomers(c *gin.Context) {
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

// POST /api/v1/merchant/customers
func (h *MerchantHandler) CreateCustomer(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	var body struct {
		Name   string                `json:"name"   binding:"required"`
		Phone  *string               `json:"phone"`
		Status models.CustomerStatus `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Status == "" {
		body.Status = models.CustomerActive
	}
	cust := &models.Customer{
		MerchantID: mid,
		Name:       body.Name,
		Phone:      body.Phone,
		Status:     body.Status,
	}
	if err := h.customerRepo.Create(cust); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat pelanggan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": cust})
}

// PUT /api/v1/merchant/customers/:id
func (h *MerchantHandler) UpdateCustomer(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	cust, err := h.customerRepo.FindByID(id)
	if err != nil || cust == nil || cust.MerchantID != mid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelanggan tidak ditemukan"})
		return
	}
	var body struct {
		Name   string                `json:"name"`
		Phone  *string               `json:"phone"`
		Status models.CustomerStatus `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name != "" {
		cust.Name = body.Name
	}
	cust.Phone = body.Phone
	if body.Status != "" {
		cust.Status = body.Status
	}
	if err := h.customerRepo.Update(cust); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui pelanggan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cust})
}

// ─── Queues ───────────────────────────────────────────────────────────────────

// GET /api/v1/merchant/queues?status=PENDING
func (h *MerchantHandler) ListQueues(c *gin.Context) {
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

// PUT /api/v1/merchant/queues/:id
func (h *MerchantHandler) UpdateQueue(c *gin.Context) {
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

// ─── Sales ────────────────────────────────────────────────────────────────────

// GET /api/v1/merchant/sales?status=PAID
func (h *MerchantHandler) ListSales(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	status := c.Query("status")
	sales, err := h.saleRepo.ListByMerchant(mid, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil transaksi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sales})
}

// POST /api/v1/merchant/sales
func (h *MerchantHandler) CreateSale(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	h.createSaleForMerchant(c, mid)
}

// createSaleForMerchant is shared between merchant and user handlers.
func (h *MerchantHandler) createSaleForMerchant(c *gin.Context, merchantID uuid.UUID) {
	var body struct {
		CustomerID   *string           `json:"customer_id"`
		CustomerName *string           `json:"customer_name"`
		Status       models.SaleStatus `json:"status" binding:"required"`
		Total        float64           `json:"total"  binding:"required,gt=0"`
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

// PUT /api/v1/merchant/sales/:id/pay
func (h *MerchantHandler) PaySale(c *gin.Context) {
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

// ─── User Registration by Merchant ───────────────────────────────────────────

// POST /api/v1/merchant/users/register
// Merchant mendaftarkan user baru (kasir/staff) — status PENDING, menunggu aktivasi admin
func (h *MerchantHandler) RegisterUser(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}

	var body struct {
		Name     string `json:"name"     binding:"required,min=2"`
		Email    string `json:"email"    binding:"required,email"`
		Username string `json:"username" binding:"required,min=3,max=30"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek duplikat email
	existing, err := h.userRepo.FindByEmail(body.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa email"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar"})
		return
	}

	// Cek duplikat username
	existingUser, err := h.userRepo.FindByUsername(body.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa username"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username sudah digunakan"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}

	username := body.Username
	passwordHash := string(hash)
	user := &models.User{
		Name:         body.Name,
		Email:        body.Email,
		Username:     &username,
		PasswordHash: &passwordHash,
		Role:         models.RoleUser,
		Status:       models.UserStatusActive,
		MerchantID:   &mid,
	}

	if err := h.userRepo.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan user"})
		return
	}

	if err := errors.New(""); err != nil && false {
		// placeholder to keep errors import
	}
	_ = services.ErrEmailExists // keep services import used elsewhere

	c.JSON(http.StatusCreated, gin.H{"message": "Akun kasir berhasil dibuat"})
}

// GET /api/v1/merchant/kasir
func (h *MerchantHandler) ListKasir(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	users, err := h.userRepo.ListByMerchant(mid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kasir"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// PUT /api/v1/merchant/kasir/:id
func (h *MerchantHandler) UpdateKasir(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	user, err := h.userRepo.FindByID(id)
	if err != nil || user == nil || user.MerchantID == nil || *user.MerchantID != mid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kasir tidak ditemukan"})
		return
	}
	var body struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name != nil {
		user.Name = *body.Name
	}
	if body.Status != nil {
		user.Status = models.UserStatus(*body.Status)
	}
	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui kasir"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// DELETE /api/v1/merchant/kasir/:id
func (h *MerchantHandler) DeleteKasir(c *gin.Context) {
	mid, ok := h.merchantID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	user, err := h.userRepo.FindByID(id)
	if err != nil || user == nil || user.MerchantID == nil || *user.MerchantID != mid {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kasir tidak ditemukan"})
		return
	}
	if err := h.userRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kasir"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Kasir berhasil dihapus"})
}
