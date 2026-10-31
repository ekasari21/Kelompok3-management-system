package asset

import (
	"gin-auth/internal/auth"
	"gin-auth/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(group *gin.RouterGroup, h *Handler) {
	group.POST("/assets", middleware.MockEndpoint("private.asset.create"), auth.RequireRole("admin"), h.Create)
	group.GET("/assets", middleware.MockEndpoint("private.asset.list"), h.List)
	group.GET("/assets/:id", middleware.MockEndpoint("private.asset.list"), h.GetByID)
	group.PUT("/assets/:id", middleware.MockEndpoint("private.asset.update"), auth.RequireRole("admin"), h.Update)
	group.DELETE("/assets/:id", middleware.MockEndpoint("private.asset.delete"), auth.RequireRole("admin"), h.Delete)
}
