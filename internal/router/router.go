package router

import (
	"pos-go/internal/handlers"
	"pos-go/internal/middleware"
	"pos-go/internal/models"
	"pos-go/internal/services"

	"github.com/gin-gonic/gin"
)

func Setup(
	authHandler *handlers.AuthHandler,
	adminHandler *handlers.AdminHandler,
	merchantHandler *handlers.MerchantHandler,
	userPOSHandler *handlers.UserPOSHandler,
	authSvc *services.AuthService,
) *gin.Engine {
	r := gin.Default()

	// CORS sederhana
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api/v1")

	// ─── Image proxy (public, no auth) ────────────────────────────────────────
	api.GET("/proxy/image", handlers.ImageProxy)

	// ─── Auth (public) ────────────────────────────────────────────────────────
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register) // pendaftaran mandiri (merchant/calon user, status PENDING)
		auth.POST("/login", authHandler.Login)
		auth.POST("/google", authHandler.GoogleLogin)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)

		// Protected
		auth.GET("/me", middleware.Auth(authSvc), authHandler.Me)
	}

	// ─── Admin (admin only) ───────────────────────────────────────────────────
	admin := api.Group("/admin",
		middleware.Auth(authSvc),
		middleware.RequireRole(models.RoleAdmin),
	)
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.PUT("/users/:id/assign-merchant", adminHandler.AssignMerchant)
		admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		admin.POST("/merchants", adminHandler.CreateMerchant)
		admin.GET("/merchants", adminHandler.ListMerchants)
	}

	// ─── Merchant (merchant only) ─────────────────────────────────────────────
	merchant := api.Group("/merchant",
		middleware.Auth(authSvc),
		middleware.RequireRole(models.RoleMerchant),
	)
	{
		merchant.GET("/products", merchantHandler.ListProducts)
		merchant.POST("/products", merchantHandler.CreateProduct)
		merchant.PUT("/products/:id", merchantHandler.UpdateProduct)
		merchant.DELETE("/products/:id", merchantHandler.DeleteProduct)
		merchant.GET("/products/:id/bulk-tiers", merchantHandler.ListBulkTiers)
		merchant.POST("/products/:id/bulk-tiers", merchantHandler.AddBulkTier)
		merchant.DELETE("/bulk-tiers/:id", merchantHandler.DeleteBulkTier)
		merchant.GET("/customers", merchantHandler.ListCustomers)
		merchant.POST("/customers", merchantHandler.CreateCustomer)
		merchant.PUT("/customers/:id", merchantHandler.UpdateCustomer)
		merchant.GET("/queues", merchantHandler.ListQueues)
		merchant.PUT("/queues/:id", merchantHandler.UpdateQueue)
		merchant.GET("/sales", merchantHandler.ListSales)
		merchant.POST("/sales", merchantHandler.CreateSale)
		merchant.PUT("/sales/:id/pay", merchantHandler.PaySale)
		merchant.POST("/users/register", merchantHandler.RegisterUser)
		merchant.GET("/kasir", merchantHandler.ListKasir)
		merchant.PUT("/kasir/:id", merchantHandler.UpdateKasir)
		merchant.DELETE("/kasir/:id", merchantHandler.DeleteKasir)
	}

	// ─── User / POS (user + merchant) ────────────────────────────────────────
	userPos := api.Group("/user",
		middleware.Auth(authSvc),
		middleware.RequireRole(models.RoleUser, models.RoleMerchant),
	)
	{
		userPos.GET("/products", userPOSHandler.ListProducts)
		userPos.GET("/products/barcode", userPOSHandler.FindProductByBarcode)
		userPos.GET("/customers", userPOSHandler.ListCustomers)
		userPos.POST("/customers", userPOSHandler.CreateCustomer)
		userPos.GET("/queues", userPOSHandler.ListQueues)
		userPos.POST("/queues", userPOSHandler.AddQueue)
		userPos.PUT("/queues/:id", userPOSHandler.UpdateQueue)
		userPos.GET("/sales", userPOSHandler.ListSales)
		userPos.POST("/sales", userPOSHandler.CreateSale)
		userPos.PUT("/sales/:id/pay", userPOSHandler.PaySale)
	}

	return r
}
