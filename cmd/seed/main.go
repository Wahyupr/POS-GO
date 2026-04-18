package main

import (
	"fmt"
	"log"

	"pos-go/internal/config"
	"pos-go/internal/database"
	"pos-go/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func ptr[T any](v T) *T { return &v }

func main() {
	config.Load()

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("DB connect error: %v", err)
	}

	// ── Merchant (sudah ada) ────────────────────────────────────────────────
	var merchant models.Merchant
	if err := db.First(&merchant).Error; err != nil {
		log.Fatalf("Merchant tidak ditemukan: %v", err)
	}
	log.Printf("Merchant: %s (%s)", merchant.Name, merchant.ID)

	// ── Password hash (sama untuk semua akun demo) ──────────────────────────
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	hashStr := string(hash)

	// ── Update akun admin yang sudah ada ────────────────────────────────────
	if res := db.Model(&models.User{}).
		Where("email = ?", "admin@pos.com").
		Updates(map[string]interface{}{
			"role":   models.RoleAdmin,
			"status": models.UserStatusActive,
		}); res.Error != nil {
		log.Printf("WARN update admin: %v", res.Error)
	} else {
		log.Println("✓ admin@pos.com → role=ADMIN")
	}

	// ── Buat / upsert akun merchant ─────────────────────────────────────────
	merchantUser := models.User{
		Email:        "merchant@pos.com",
		Name:         "Pemilik Toko",
		Role:         models.RoleMerchant,
		MerchantID:   &merchant.ID,
		Status:       models.UserStatusActive,
		PasswordHash: &hashStr,
	}
	res := db.Where("email = ?", merchantUser.Email).First(&models.User{})
	if res.Error != nil {
		merchantUser.ID = uuid.New()
		if err := db.Create(&merchantUser).Error; err != nil {
			log.Fatalf("Gagal buat merchant user: %v", err)
		}
		log.Println("✓ merchant@pos.com dibuat")
	} else {
		db.Model(&models.User{}).Where("email = ?", merchantUser.Email).
			Updates(map[string]interface{}{
				"role":        models.RoleMerchant,
				"merchant_id": merchant.ID,
				"status":      models.UserStatusActive,
			})
		log.Println("✓ merchant@pos.com diupdate")
	}

	// ── Buat / upsert akun kasir (USER) ─────────────────────────────────────
	kasirUser := models.User{
		Email:        "kasir@pos.com",
		Name:         "Kasir Toko",
		Role:         models.RoleUser,
		MerchantID:   &merchant.ID,
		Status:       models.UserStatusActive,
		PasswordHash: &hashStr,
	}
	res2 := db.Where("email = ?", kasirUser.Email).First(&models.User{})
	if res2.Error != nil {
		kasirUser.ID = uuid.New()
		if err := db.Create(&kasirUser).Error; err != nil {
			log.Fatalf("Gagal buat kasir user: %v", err)
		}
		log.Println("✓ kasir@pos.com dibuat")
	} else {
		db.Model(&models.User{}).Where("email = ?", kasirUser.Email).
			Updates(map[string]interface{}{
				"role":        models.RoleUser,
				"merchant_id": merchant.ID,
				"status":      models.UserStatusActive,
			})
		log.Println("✓ kasir@pos.com diupdate")
	}

	// ── Hapus produk lama milik merchant ini ────────────────────────────────
	db.Where("merchant_id = ?", merchant.ID).Delete(&models.Product{})
	log.Println("✓ produk lama dihapus")

	// ── Seed produk + bulk tiers ─────────────────────────────────────────────
	type tierDef struct {
		minQty      float64
		pricingMode models.BulkPricingMode
		unitPrice   *float64
		bundleQty   *float64
		bundleTotal *float64
	}
	type productDef struct {
		name      string
		unit      models.ProductUnit
		priceBase float64
		stock     float64
		imageURL  *string
		tiers     []tierDef
	}

	products := []productDef{
		// ── Minuman ────────────────────────────────────────────────────────
		{
			name: "Kopi Hitam", unit: models.UnitPCS, priceBase: 8000, stock: 100,
			imageURL: ptr("https://images.unsplash.com/photo-1509042239860-f550ce710b93?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingUnitPrice, unitPrice: ptr(7000.0)},
				{minQty: 10, pricingMode: models.PricingBundleTotal, bundleQty: ptr(10.0), bundleTotal: ptr(65000.0)},
			},
		},
		{
			name: "Kopi Susu", unit: models.UnitPCS, priceBase: 12000, stock: 100,
			imageURL: ptr("https://images.unsplash.com/photo-1561047029-3000c68339ca?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingUnitPrice, unitPrice: ptr(10000.0)},
				{minQty: 10, pricingMode: models.PricingUnitPrice, unitPrice: ptr(9000.0)},
			},
		},
		{
			name: "Teh Manis", unit: models.UnitPCS, priceBase: 6000, stock: 100,
			imageURL: ptr("https://images.unsplash.com/photo-1556679343-c7306c1976bc?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingBundleTotal, bundleQty: ptr(5.0), bundleTotal: ptr(25000.0)},
			},
		},
		{
			name: "Es Jeruk", unit: models.UnitPCS, priceBase: 10000, stock: 80,
			imageURL: ptr("https://images.unsplash.com/photo-1621506289937-a8e4df240d0b?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingUnitPrice, unitPrice: ptr(8500.0)},
			},
		},
		{
			name: "Jus Alpukat", unit: models.UnitPCS, priceBase: 15000, stock: 60,
			imageURL: ptr("https://images.unsplash.com/photo-1610970881699-44a5587cabec?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 3, pricingMode: models.PricingUnitPrice, unitPrice: ptr(13000.0)},
			},
		},

		// ── Makanan ───────────────────────────────────────────────────────
		{
			name: "Nasi Goreng", unit: models.UnitPCS, priceBase: 20000, stock: 50,
			imageURL: ptr("https://images.unsplash.com/photo-1512058564366-18510be2db19?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingUnitPrice, unitPrice: ptr(18000.0)},
				{minQty: 10, pricingMode: models.PricingUnitPrice, unitPrice: ptr(16000.0)},
			},
		},
		{
			name: "Mie Goreng", unit: models.UnitPCS, priceBase: 18000, stock: 50,
			imageURL: ptr("https://images.unsplash.com/photo-1569718212165-3a8278d5f624?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingBundleTotal, bundleQty: ptr(5.0), bundleTotal: ptr(80000.0)},
			},
		},
		{
			name: "Roti Bakar", unit: models.UnitPCS, priceBase: 15000, stock: 40,
			imageURL: ptr("https://images.unsplash.com/photo-1484723091739-30a097e8f929?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 3, pricingMode: models.PricingUnitPrice, unitPrice: ptr(13000.0)},
			},
		},
		{
			name: "Nasi Uduk", unit: models.UnitPCS, priceBase: 17000, stock: 40,
			imageURL: ptr("https://images.unsplash.com/photo-1536304993881-ff86e0c9b4cb?w=400&auto=format"),
			tiers:    []tierDef{},
		},
		{
			name: "Sandwich", unit: models.UnitPCS, priceBase: 22000, stock: 30,
			imageURL: ptr("https://images.unsplash.com/photo-1567234669003-dce7a7a88821?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingUnitPrice, unitPrice: ptr(19000.0)},
			},
		},

		// ── Snack ─────────────────────────────────────────────────────────
		{
			name: "Pisang Goreng", unit: models.UnitPCS, priceBase: 3000, stock: 100,
			imageURL: ptr("https://images.unsplash.com/photo-1571771894821-ce9b6c11b08e?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingBundleTotal, bundleQty: ptr(5.0), bundleTotal: ptr(12000.0)},
				{minQty: 10, pricingMode: models.PricingBundleTotal, bundleQty: ptr(10.0), bundleTotal: ptr(22000.0)},
			},
		},
		{
			name: "Cireng", unit: models.UnitPCS, priceBase: 2000, stock: 150,
			imageURL: ptr("https://images.unsplash.com/photo-1604329760661-e71dc83f8f26?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 10, pricingMode: models.PricingBundleTotal, bundleQty: ptr(10.0), bundleTotal: ptr(15000.0)},
			},
		},
		{
			name: "Kacang Goreng", unit: models.UnitONS, priceBase: 7000, stock: 50,
			imageURL: ptr("https://images.unsplash.com/photo-1567306226416-28f0efdc88ce?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingUnitPrice, unitPrice: ptr(6000.0)},
			},
		},
		{
			name: "Keripik Singkong", unit: models.UnitONS, priceBase: 5000, stock: 80,
			imageURL: ptr("https://images.unsplash.com/photo-1621939514649-280e2ee25f60?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 5, pricingMode: models.PricingBundleTotal, bundleQty: ptr(5.0), bundleTotal: ptr(20000.0)},
			},
		},
		{
			name: "Donat", unit: models.UnitPCS, priceBase: 5000, stock: 60,
			imageURL: ptr("https://images.unsplash.com/photo-1551024601-bec78aea704b?w=400&auto=format"),
			tiers: []tierDef{
				{minQty: 6, pricingMode: models.PricingBundleTotal, bundleQty: ptr(6.0), bundleTotal: ptr(25000.0)},
				{minQty: 12, pricingMode: models.PricingBundleTotal, bundleQty: ptr(12.0), bundleTotal: ptr(48000.0)},
			},
		},
	}

	for _, pd := range products {
		p := models.Product{
			ID:         uuid.New(),
			MerchantID: merchant.ID,
			Name:       pd.name,
			ImageURL:   pd.imageURL,
			Unit:       pd.unit,
			PriceBase:  pd.priceBase,
			Stock:      pd.stock,
			Status:     models.ProductActive,
		}
		if err := db.Create(&p).Error; err != nil {
			log.Printf("WARN gagal buat produk %s: %v", pd.name, err)
			continue
		}
		for _, td := range pd.tiers {
			tier := models.BulkTier{
				ID:          uuid.New(),
				ProductID:   p.ID,
				MinQty:      td.minQty,
				PricingMode: td.pricingMode,
				UnitPrice:   td.unitPrice,
				BundleQty:   td.bundleQty,
				BundleTotal: td.bundleTotal,
			}
			db.Create(&tier)
		}
		log.Printf("✓ %s (%d tier)", pd.name, len(pd.tiers))
	}

	fmt.Println("\n=== SEED SELESAI ===")
	fmt.Println("Akun tersedia:")
	fmt.Println("  admin@pos.com    | password123 | ADMIN")
	fmt.Println("  merchant@pos.com | password123 | MERCHANT → Toko Berkah")
	fmt.Println("  kasir@pos.com    | password123 | USER    → Toko Berkah")
	fmt.Printf("  %d produk + bulk tiers di-seed ke merchant '%s'\n", len(products), merchant.Name)
}
