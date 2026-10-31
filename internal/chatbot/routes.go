package chatbot

import (
	"gin-auth/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(group *gin.RouterGroup, h *Handler) {
	group.POST(
		"/chatbot/ask",
		middleware.MockEndpoint("private.chatbot.ask"),
		h.Ask,
	)
	group.GET(
		"/chatbot/suggestions",
		middleware.MockEndpoint("private.chatbot.suggestions"),
		h.Suggestions,
	)
}
