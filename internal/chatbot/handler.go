package chatbot

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Ask menerima pesan pengguna dan mengembalikan respons chatbot khusus data inventaris.
func (h *Handler) Ask(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Pesan pertanyaan tidak boleh kosong",
		})
		return
	}

	resp, err := h.service.ProcessMessage(c.Request.Context(), req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Gagal memproses pertanyaan: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Suggestions mengembalikan daftar saran pertanyaan awal inventaris.
func (h *Handler) Suggestions(c *gin.Context) {
	suggestions := h.service.GetInitialSuggestions()
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"suggestions": suggestions,
	})
}
