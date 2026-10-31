package private

import (
	"net/http"

	"gin-auth/internal/asset"
	"gin-auth/internal/assetbroken"
	"gin-auth/internal/assetlending"
	"gin-auth/internal/auth"
	"gin-auth/internal/category"
	"gin-auth/internal/chatbot"
	"gin-auth/internal/complaint"
	"gin-auth/internal/dashboard"
	"gin-auth/internal/location"
	"gin-auth/internal/middleware"
	"gin-auth/internal/textanalysis"
	"gin-auth/internal/user"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(
	r *gin.Engine,
	requireAuth gin.HandlerFunc,
	categoryHandler *category.Handler,
	locationHandler *location.Handler,
	assetHandler *asset.Handler,
	dashboardHandler *dashboard.Handler,
	complaintHandler *complaint.Handler,
	assetBrokenHandler *assetbroken.Handler,
	assetLendingHandler *assetlending.Handler,
	userHandler *user.Handler,
	textAnalysisHandler *textanalysis.Handler,
	chatbotHandler *chatbot.Handler,
) {
	group := r.Group("/api/private")
	group.Use(requireAuth)

	// Keluhan (Admin, Teknisi, User terfilter)
	complaint.RegisterRoutes(group, complaintHandler)

	// Peminjaman (Admin dan User; Teknisi dicegah di handler)
	assetlending.RegisterRoutes(group, assetLendingHandler)

	// Chatbot dan Dashboard
	chatbot.RegisterRoutes(group, chatbotHandler)
	dashboard.RegisterRoutes(group, dashboardHandler)

	// Aset (GET terbuka, modifikasi hanya admin)
	asset.RegisterRoutes(group, assetHandler)

	// Master data Kategori & Lokasi (GET terbuka, modifikasi hanya admin)
	category.RegisterRoutes(group, categoryHandler)
	location.RegisterRoutes(group, locationHandler)

	// Fitur Khusus Admin & Teknisi
	assetbrokenGroup := group.Group("", auth.RequireRoles("admin", "teknisi"))
	assetbroken.RegisterRoutes(assetbrokenGroup, assetBrokenHandler)

	textAnalysisGroup := group.Group("", auth.RequireRoles("admin", "teknisi"))
	textanalysis.RegisterRoutes(textAnalysisGroup, textAnalysisHandler)

	// Manajemen User Khusus Admin
	userGroup := group.Group("", auth.RequireRole("admin"))
	user.RegisterRoutes(userGroup, userHandler)

	group.GET(
		"/profile",
		middleware.MockEndpoint("private.profile"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"data": c.MustGet("user"),
			})
		},
	)
}
