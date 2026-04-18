package services

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pos-go/internal/config"
	"pos-go/internal/models"
	"pos-go/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ─── Errors ──────────────────────────────────────────────────────────────────

var (
	ErrInvalidCredentials = errors.New("invalid email/username or password")
	ErrUserInactive       = errors.New("account is inactive")
	ErrUserPending        = errors.New("account pending admin approval")
	ErrEmailExists        = errors.New("email already registered")
	ErrUsernameExists     = errors.New("username already taken")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrGoogleTokenInvalid = errors.New("invalid google token")
)

// ─── Claims ──────────────────────────────────────────────────────────────────

type JWTClaims struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	MerchantID string `json:"merchant_id,omitempty"`
	jwt.RegisteredClaims
}

// ─── DTOs ────────────────────────────────────────────────────────────────────

type RegisterInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"     binding:"required,min=2"`
}

type LoginInput struct {
	Identifier string `json:"identifier" binding:"required"` // email or username
	Password   string `json:"password"   binding:"required"`
}

type GoogleLoginInput struct {
	IDToken string `json:"id_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

// ─── Google Token Info ────────────────────────────────────────────────────────

type googleTokenInfo struct {
	Sub           string `json:"sub"` // Google user ID
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Aud           string `json:"aud"`
	Error         string `json:"error"`
}

// ─── Service ─────────────────────────────────────────────────────────────────

type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
}

func NewAuthService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository) *AuthService {
	return &AuthService{userRepo: userRepo, tokenRepo: tokenRepo}
}

// Register mendaftarkan user baru dengan email & password
func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
	// Cek email duplikat
	existing, err := s.userRepo.FindByEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailExists
	}

	// Cek username duplikat
	existingUser, err := s.userRepo.FindByUsername(input.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUsernameExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	username := input.Username
	passwordHash := string(hash)

	user := &models.User{
		Email:        input.Email,
		Username:     &username,
		PasswordHash: &passwordHash,
		Name:         input.Name,
		Role:         models.RoleNone,
		Status:       models.UserStatusPending,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// Tidak return token — user harus tunggu assign dari merchant/admin
	return nil, nil
}

// Login dengan email/username + password
func (s *AuthService) Login(input LoginInput) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmailOrUsername(input.Identifier)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	return s.generateAuthResponse(user)
}

// GoogleLogin verifikasi Google ID token
func (s *AuthService) GoogleLogin(input GoogleLoginInput) (*AuthResponse, error) {
	info, err := s.verifyGoogleToken(input.IDToken)
	if err != nil {
		return nil, ErrGoogleTokenInvalid
	}

	// Cek apakah Google Client ID cocok
	if config.App.GoogleClientID != "" && info.Aud != config.App.GoogleClientID {
		return nil, ErrGoogleTokenInvalid
	}

	// Cari user by Google ID
	user, err := s.userRepo.FindByGoogleID(info.Sub)
	if err != nil {
		return nil, err
	}

	if user == nil {
		// Cek apakah email sudah ada (link akun)
		user, err = s.userRepo.FindByEmail(info.Email)
		if err != nil {
			return nil, err
		}

		if user == nil {
			// Buat user baru dengan status PENDING
			googleID := info.Sub
			avatar := info.Picture
			user = &models.User{
				Email:        info.Email,
				Name:         info.Name,
				GoogleID:     &googleID,
				GoogleAvatar: &avatar,
				Role:         models.RoleNone,
				Status:       models.UserStatusPending,
			}
			if err := s.userRepo.Create(user); err != nil {
				return nil, err
			}
		} else {
			// Link Google ID ke akun yang sudah ada
			googleID := info.Sub
			user.GoogleID = &googleID
			if err := s.userRepo.Update(user); err != nil {
				return nil, err
			}
		}
	}

	// User PENDING tidak bisa login (butuh assign dari admin)
	if user.Status == models.UserStatusPending {
		return nil, ErrUserPending
	}

	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	return s.generateAuthResponse(user)
}

// RefreshToken memperbarui access token menggunakan refresh token
func (s *AuthService) RefreshToken(rawRefreshToken string) (*AuthResponse, error) {
	rt, err := s.tokenRepo.FindByToken(rawRefreshToken)
	if err != nil || rt == nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(rt.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidToken
	}

	if err := s.checkUserStatus(user); err != nil {
		return nil, err
	}

	// Hapus refresh token lama
	_ = s.tokenRepo.DeleteByToken(rawRefreshToken)

	return s.generateAuthResponse(user)
}

// Logout invalidasi refresh token
func (s *AuthService) Logout(rawRefreshToken string) error {
	return s.tokenRepo.DeleteByToken(rawRefreshToken)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *AuthService) checkUserStatus(user *models.User) error {
	switch user.Status {
	case models.UserStatusInactive:
		return ErrUserInactive
	case models.UserStatusPending:
		return ErrUserPending
	}
	return nil
}

func (s *AuthService) generateAuthResponse(user *models.User) (*AuthResponse, error) {
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	expDays := config.App.JWTRefreshExpDays
	expiresAt := time.Now().Add(time.Duration(expDays) * 24 * time.Hour)
	if err := s.tokenRepo.Save(user.ID, rawRefresh, expiresAt); err != nil {
		return nil, err
	}

	// Hilangkan password dari response
	user.PasswordHash = nil

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		User:         user,
	}, nil
}

func (s *AuthService) generateAccessToken(user *models.User) (string, error) {
	cfg := config.App
	exp := time.Duration(cfg.JWTAccessExpMinutes) * time.Minute

	merchantID := ""
	if user.MerchantID != nil {
		merchantID = user.MerchantID.String()
	}

	claims := JWTClaims{
		UserID:     user.ID.String(),
		Email:      user.Email,
		Name:       user.Name,
		Role:       string(user.Role),
		MerchantID: merchantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(exp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

func ParseAccessToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(config.App.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) verifyGoogleToken(idToken string) (*googleTokenInfo, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var info googleTokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	if info.Error != "" || info.Sub == "" {
		return nil, fmt.Errorf("google token error: %s", info.Error)
	}

	if !strings.EqualFold(info.EmailVerified, "true") {
		return nil, fmt.Errorf("google email not verified")
	}

	return &info, nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetUserByID digunakan middleware untuk load user dari claims
func (s *AuthService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(id)
}
