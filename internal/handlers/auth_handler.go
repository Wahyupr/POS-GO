package handlers

import (
	"errors"
	"net/http"

	"pos-go/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authSvc *services.AuthService
}

func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var input services.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authSvc.Register(input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar"})
		case errors.Is(err, services.ErrUsernameExists):
			c.JSON(http.StatusConflict, gin.H{"error": "Username sudah digunakan"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan akun"})
		}
		return
	}
	_ = resp // resp selalu nil sekarang
	c.JSON(http.StatusCreated, gin.H{"message": "Akun berhasil dibuat, menunggu persetujuan dari merchant atau admin"})
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var input services.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authSvc.Login(input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email/username atau password salah"})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun tidak aktif"})
		case errors.Is(err, services.ErrUserPending):
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun menunggu persetujuan admin"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal login"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/google
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var input services.GoogleLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authSvc.GoogleLogin(input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGoogleTokenInvalid):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Google token tidak valid"})
		case errors.Is(err, services.ErrUserPending):
			c.JSON(http.StatusForbidden, gin.H{
				"error":  "Akun menunggu persetujuan admin",
				"status": "PENDING",
			})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun tidak aktif"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal login dengan Google"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authSvc.RefreshToken(body.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidToken):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token tidak valid atau sudah kedaluwarsa"})
		case errors.Is(err, services.ErrUserPending):
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun menunggu persetujuan admin"})
		case errors.Is(err, services.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun tidak aktif"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui token"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.authSvc.Logout(body.RefreshToken) // selalu sukses

	c.JSON(http.StatusOK, gin.H{"message": "Berhasil logout"})
}

// GET /api/v1/auth/me  (protected — JWT middleware set context key "user")
func (h *AuthHandler) Me(c *gin.Context) {
	user, _ := c.Get("user")
	c.JSON(http.StatusOK, gin.H{"user": user})
}
