package location

import (
	"gin-auth/internal/auth"
	"gin-auth/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(group *gin.RouterGroup, h *Handler) {
	group.POST("/location", middleware.MockEndpoint("private.location.create"), auth.RequireRole("admin"), h.Create)
	group.GET("/location", middleware.MockEndpoint("private.location.list"), h.GetAllLoc)
	group.GET("/location/:id", middleware.MockEndpoint("private.location.listByID"), h.GetLocByID)
	group.PUT("/location/:id", middleware.MockEndpoint("private.location.update"), auth.RequireRole("admin"), h.Update)
	group.DELETE("/location/:id", middleware.MockEndpoint("private.location.delete"), auth.RequireRole("admin"), h.Delete)
}
