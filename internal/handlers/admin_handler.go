package handlers

import (
	"fmt"
	"net/http"

	"pos-go/internal/models"
	"pos-go/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	userRepo     *repository.UserRepository
	merchantRepo *repository.MerchantRepository
}

func NewAdminHandler(userRepo *repository.UserRepository, merchantRepo *repository.MerchantRepository) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, merchantRepo: merchantRepo}
}

// GET /api/v1/admin/users?page=1&limit=20
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	var users []models.User
	var total int64
	var err error

	if status := c.Query("status"); status != "" {
		users, total, err = h.userRepo.ListByStatus(models.UserStatus(status), page, limit)
	} else {
		users, total, err = h.userRepo.List(page, limit)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar user"})
		return
	}

	// Hilangkan password hash dari response
	for i := range users {
		users[i].PasswordHash = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// PUT /api/v1/admin/users/:id/assign-merchant
// Body: { "merchant_id": "uuid", "role": "MERCHANT" }
func (h *AdminHandler) AssignMerchant(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID user tidak valid"})
		return
	}

	var body struct {
		MerchantID string `json:"merchant_id" binding:"required"`
		Role       string `json:"role"        binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merchantID, err := uuid.Parse(body.MerchantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "merchant_id tidak valid"})
		return
	}

	// Validasi role
	role := models.Role(body.Role)
	if role != models.RoleMerchant && role != models.RoleUser && role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	err = h.userRepo.UpdateFields(userID, map[string]interface{}{
		"merchant_id": merchantID,
		"role":        role,
		"status":      models.UserStatusActive,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal assign merchant"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil assign merchant ke user"})
}

// PUT /api/v1/admin/users/:id/role
// Body: { "role": "ADMIN" | "MERCHANT" | "USER", "status": "ACTIVE" | "INACTIVE" }
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID user tidak valid"})
		return
	}

	var body struct {
		Role   string `json:"role"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fields := map[string]interface{}{}

	if body.Role != "" {
		role := models.Role(body.Role)
		if role != models.RoleMerchant && role != models.RoleUser && role != models.RoleAdmin {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
			return
		}
		fields["role"] = role
	}

	if body.Status != "" {
		status := models.UserStatus(body.Status)
		if status != models.UserStatusActive && status != models.UserStatusInactive && status != models.UserStatusPending {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status tidak valid"})
			return
		}
		fields["status"] = status
	}

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada field yang diubah"})
		return
	}

	if err := h.userRepo.UpdateFields(userID, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil update user"})
}

// POST /api/v1/admin/merchants
// Body: { "name": "Toko ABC" }
func (h *AdminHandler) CreateMerchant(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	merchant := &models.Merchant{
		Name:   body.Name,
		Status: models.MerchantStatusActive,
	}

	if err := h.merchantRepo.Create(merchant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat merchant"})
		return
	}

	c.JSON(http.StatusCreated, merchant)
}

// GET /api/v1/admin/merchants
func (h *AdminHandler) ListMerchants(c *gin.Context) {
	merchants, err := h.merchantRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar merchant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": merchants})
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func intQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.DefaultQuery(key, "")
	if val == "" {
		return defaultVal
	}
	n := defaultVal
	_, _ = fmt.Sscan(val, &n)
	return n
}
