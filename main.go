package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gin-auth/internal/asset"
	"gin-auth/internal/assetbroken"
	"gin-auth/internal/assetlending"
	"gin-auth/internal/auth"
	"gin-auth/internal/category"
	"gin-auth/internal/chatbot"
	"gin-auth/internal/complaint"
	"gin-auth/internal/dashboard"
	"gin-auth/internal/database"
	"gin-auth/internal/location"
	"gin-auth/internal/private"
	"gin-auth/internal/public"
	"gin-auth/internal/textanalysis"
	"gin-auth/internal/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

// =====================================================
// SETUP GIN ENGINE
// =====================================================

func setupEngine(db *gorm.DB, secret []byte) *gin.Engine {
	r := gin.Default()

	r.MaxMultipartMemory = 8 << 20

	// Sajikan gambar hasil upload.
	r.Static("/uploads", "uploads")

	// CORS untuk frontend lokal dan frontend produksi.
	allowedOrigins := []string{
		"http://localhost:8081",
	}

	if customOrigin := os.Getenv("ALLOWED_ORIGIN"); customOrigin != "" {
		for _, origin := range strings.Split(customOrigin, ",") {
			origin = strings.TrimSpace(origin)

			if origin != "" {
				allowedOrigins = append(allowedOrigins, origin)
			}
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
		},
		ExposeHeaders: []string{
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Batasi ukuran request.
	r.Use(func(c *gin.Context) {
		limit := int64(1 << 20) // 1 MB

		if strings.HasPrefix(
			strings.ToLower(c.ContentType()),
			"multipart/form-data",
		) {
			limit = 6 << 20 // 6 MB
		}

		c.Request.Body = http.MaxBytesReader(
			c.Writer,
			c.Request.Body,
			limit,
		)

		c.Next()
	})

	// =================================================
	// AUTH
	// =================================================

	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, secret)
	authHandler := auth.NewHandler(authService)

	auth.RegisterRoutes(r, authHandler)

	// =================================================
	// PUBLIC ROUTES
	// =================================================

	public.RegisterRoutes(r)

	// Repository dan service autentikasi untuk middleware.
	repo := auth.NewRepository(db)
	service := auth.NewService(repo, secret)
	handler := auth.NewHandler(service)

	// =================================================
	// CATEGORY
	// =================================================

	categoryRepo := category.NewRepository(db)
	categoryService := category.NewService(categoryRepo)
	categoryHandler := category.NewHandler(categoryService)

	// =================================================
	// LOCATION
	// =================================================

	locationRepo := location.NewRepository(db)
	locationService := location.NewService(locationRepo)
	locationHandler := location.NewHandler(locationService)

	// =================================================
	// ASSET
	// =================================================

	assetRepo := asset.NewRepository(db)
	assetService := asset.NewService(
		assetRepo,
		categoryRepo,
		locationRepo,
	)
	assetHandler := asset.NewHandler(assetService)

	// =================================================
	// DASHBOARD
	// =================================================

	dashboardRepo := dashboard.NewRepository(db)
	dashboardService := dashboard.NewService(dashboardRepo)
	dashboardHandler := dashboard.NewHandler(dashboardService)

	// =================================================
	// COMPLAINT
	// =================================================

	complaintRepo := complaint.NewRepository(db)
	complaintService := complaint.NewService(complaintRepo)
	complaintHandler := complaint.NewHandler(complaintService)

	// =================================================
	// ASSET BROKEN
	// =================================================

	assetBrokenRepo := assetbroken.NewRepository(db)
	assetBrokenService := assetbroken.NewService(
		assetBrokenRepo,
		assetRepo,
	)
	assetBrokenHandler := assetbroken.NewHandler(
		assetBrokenService,
	)

	// =================================================
	// ASSET LENDING
	// =================================================

	assetLendingRepo := assetlending.NewRepository(db)
	assetLendingService := assetlending.NewService(
		assetLendingRepo,
		assetRepo,
		repo,
	)
	assetLendingHandler := assetlending.NewHandler(
		assetLendingService,
	)

	// =================================================
	// USER
	// =================================================

	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// =================================================
	// TEXT ANALYSIS: TF-IDF + K-MEANS
	// =================================================

	textAnalysisRepo := textanalysis.NewRepository(db)
	textAnalysisService := textanalysis.NewService(
		textAnalysisRepo,
	)
	textAnalysisHandler := textanalysis.NewHandler(
		textAnalysisService,
	)

	// =================================================
 // =================================================
 // CHATBOT
 // =================================================

 chatbotRepo := chatbot.NewRepository(db)
 chatbotService := chatbot.NewService(chatbotRepo)
 chatbotHandler := chatbot.NewHandler(chatbotService)

	// PRIVATE ROUTES
	// =================================================

	private.RegisterRoutes(
		r,
		handler.AuthReq(),
		categoryHandler,
		locationHandler,
		assetHandler,
		dashboardHandler,
		complaintHandler,
		assetBrokenHandler,
		assetLendingHandler,
		userHandler,
		textAnalysisHandler,
 chatbotHandler,
	)

	// =================================================
	// STATIC FRONTEND
	// =================================================

	r.Static("/client", "./frontend/client")

	r.StaticFile("/", "./frontend/index.html")
	r.StaticFile("/index.html", "./frontend/index.html")
	r.StaticFile("/login", "./frontend/login.html")
	r.StaticFile("/login.html", "./frontend/login.html")
	r.StaticFile("/register", "./frontend/register.html")
	r.StaticFile("/register.html", "./frontend/register.html")
	r.StaticFile("/contoh.html", "./frontend/contoh.html")

	// Fallback file statis dan respons 404.
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Clean(c.Request.URL.Path)

		// Hindari membaca path di luar direktori frontend.
		if filepath.IsAbs(path) || path == "." || path == ".." ||
			strings.HasPrefix(path, ".."+string(os.PathSeparator)) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Halaman tidak ditemukan",
			})
			return
		}

		target := filepath.Join("frontend", path)

		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			c.File(target)
			return
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Halaman tidak ditemukan",
		})
	})

	return r
}

// =====================================================
// RUN APPLICATION
// =====================================================

func run() error {
	// Memuat .env untuk pengembangan lokal.
	// Pada Render, gunakan Environment Variables.
	if err := godotenv.Load(); err != nil &&
		!errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load .env: %w", err)
	}

	// Validasi JWT secret.
	secret := []byte(os.Getenv("JWT_SECRET"))

	if len(secret) < 32 {
		return fmt.Errorf(
			"JWT_SECRET wajib diisi minimal 32 byte",
		)
	}

	// =================================================
	// DATABASE
	// =================================================

	db, err := database.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("buka database: %w", err)
	}

	conn, err := db.DB()
	if err != nil {
		return fmt.Errorf("akses koneksi database: %w", err)
	}
	defer conn.Close()

	// =================================================
	// DATABASE MIGRATIONS
	// =================================================

	migrations := []struct {
		name  string
		model interface{}
	}{
		{"users", &auth.User{}},
		{"locations", &location.Location{}},
		{"categories", &category.Category{}},
		{"assets", &asset.Asset{}},
		{"complaints", &complaint.Complaint{}},
		{"asset_rusak", &assetbroken.AssetRusak{}},
		{"asset_lending", &assetlending.LendingAsset{}},
	}

	for _, migration := range migrations {
		log.Printf(
			"Memulai migrasi model: %s",
			migration.name,
		)

		if err := db.AutoMigrate(migration.model); err != nil {
			return fmt.Errorf(
				"migrasi model %s gagal: %w",
				migration.name,
				err,
			)
		}

		log.Printf(
			"Migrasi model %s berhasil",
			migration.name,
		)
	}

	// =================================================
	// GIN ENGINE
	// =================================================

	router := setupEngine(db, secret)

	// =================================================
	// SERVER CONFIGURATION
	// =================================================

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,

		// Beri waktu untuk analisis Python yang mungkin
		// memerlukan waktu lebih lama.
		WriteTimeout: 2 * time.Minute,

		IdleTimeout: 60 * time.Second,
	}

	log.Printf("Server berjalan pada port %s", port)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server gagal berjalan: %w", err)
	}

	return nil
}

// =====================================================
// MAIN
// =====================================================

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
