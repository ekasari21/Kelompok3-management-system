package complaint

import (
	"gin-auth/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, h *Handler) {
	complaints := router.Group("/complaints")
	{
		complaints.POST(
			"",
			middleware.MockEndpoint("private.complaint.create"),
			h.Create,
		)

		complaints.GET(
			"",
			middleware.MockEndpoint("private.complaint.list"),
			h.FindAll,
		)

		complaints.GET(
			"/:id",
			middleware.MockEndpoint("private.complaint.findByID"),
			h.FindByID,
		)

		complaints.PUT(
			"/:id",
			middleware.MockEndpoint("private.complaint.update"),
			h.Update,
		)

		complaints.DELETE(
			"/:id",
			middleware.MockEndpoint("private.complaint.delete"),
			h.Delete,
		)

		// Endpoint khusus teknisi/admin untuk menandai aset rusak tidak bisa diperbaiki
		complaints.POST(
			"/:id/mark-unrepairable",
			middleware.MockEndpoint("private.complaint.markUnrepairable"),
			h.MarkUnrepairable,
		)
		complaints.PUT(
			"/:id/mark-unrepairable",
			middleware.MockEndpoint("private.complaint.markUnrepairable"),
			h.MarkUnrepairable,
		)
	}

	// Endpoint mengambil keluhan berdasarkan User ID
	router.GET(
		"/complaintsByUserID",
		middleware.MockEndpoint("private.complaint.findByUserID"),
		h.FindByUserID,
	)
}
