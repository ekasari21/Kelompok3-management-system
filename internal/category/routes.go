package category

import (
	"gin-auth/internal/auth"
	"gin-auth/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(group *gin.RouterGroup, h *Handler) {
	group.POST("/categories", middleware.MockEndpoint("private.category.create"), auth.RequireRole("admin"), h.Create)
	group.GET("/categories", middleware.MockEndpoint("private.category.list"), h.ListCategory)
	group.GET("/categories/:id", middleware.MockEndpoint("private.category.ListByID"), h.GetByID)
	group.PUT("/categories/:id", middleware.MockEndpoint("private.category.update"), auth.RequireRole("admin"), h.Update)
	group.DELETE("/categories/:id", middleware.MockEndpoint("private.category.delete"), auth.RequireRole("admin"), h.Delete)
}
