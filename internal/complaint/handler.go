package complaint

import (
	"fmt"
	"strings"
	"gin-auth/internal/auth"
	"gin-auth/internal/dashboard"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateComplaintRequest

	currentUser, hasUser := getContextUser(c)
	if hasUser && strings.EqualFold(currentUser.Role, "user") {
		req.UserID = currentUser.ID
	}

	// Mendukung JSON maupun multipart/form-data dengan field "image".
	imagePath, err := saveComplaintImage(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		removeComplaintImage(imagePath)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Format data tidak valid atau field wajib belum diisi",
		})
		return
	}

	req.ImagePath = imagePath

	result, err := h.service.Create(req)
	if err != nil {
		removeComplaintImage(imagePath)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Catat aktivitas setelah keluhan berhasil dibuat.
	activityDescription := fmt.Sprintf(
		"Membuat keluhan: %s. Deskripsi: %s",
		req.Title,
		req.Description,
	)

	if err := dashboard.AddActivity(
		"complaint_created",
		activityDescription,
	); err != nil {
		log.Printf("Gagal mencatat aktivitas keluhan: %v", err)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Keluhan berhasil dibuat",
		"data":    result,
	})
}

func (h *Handler) FindAll(c *gin.Context) {
	currentUser, hasUser := getContextUser(c)
	if hasUser && strings.EqualFold(currentUser.Role, "user") {
		// Role USER HANYA boleh melihat keluhan milik sendiri
		result, err := h.service.FindByUserID(currentUser.ID)
		if err != nil {
			log.Printf("ERROR FindByUserID complaints: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal mengambil data keluhan Anda",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
		return
	}

	result, err := h.service.FindAll()
	if err != nil {
		log.Printf("ERROR FindAll complaints: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Gagal mengambil data keluhan",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

func (h *Handler) FindByID(c *gin.Context) {
	result, err := h.service.FindByID(c.Param("id"))
	if err != nil {
		if IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Keluhan tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Jika role USER, pastikan data keluhan ini miliknya sendiri
	currentUser, hasUser := getContextUser(c)
	if hasUser && strings.EqualFold(currentUser.Role, "user") {
		if result.UserID != currentUser.ID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Anda hanya dapat melihat data keluhan milik sendiri",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateComplaintRequest

	currentUser, hasUser := getContextUser(c)
	if hasUser && strings.EqualFold(currentUser.Role, "user") {
		existing, err := h.service.FindByID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Keluhan tidak ditemukan"})
			return
		}
		if existing.UserID != currentUser.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat mengubah keluhan milik sendiri"})
			return
		}
		req.UserID = currentUser.ID
	}

	// Gambar baru bersifat opsional; jika tidak dikirim,
	// gambar lama dipertahankan sesuai logika service.
	imagePath, err := saveComplaintImage(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := c.ShouldBind(&req); err != nil {
		removeComplaintImage(imagePath)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Format data tidak valid atau field wajib belum diisi",
		})
		return
	}

	req.ImagePath = imagePath

	result, err := h.service.Update(c.Param("id"), req)
	if err != nil {
		removeComplaintImage(imagePath)

		if IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Keluhan tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Catat aktivitas setelah keluhan berhasil diperbarui.
	activityDescription := fmt.Sprintf(
		"Memperbarui keluhan: %s. Deskripsi: %s",
		req.Title,
		req.Description,
	)

	if err := dashboard.AddActivity(
		"complaint_updated",
		activityDescription,
	); err != nil {
		log.Printf("Gagal mencatat aktivitas keluhan: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Keluhan berhasil diperbarui",
		"data":    result,
	})
}

func (h *Handler) Delete(c *gin.Context) {
	currentUser, hasUser := getContextUser(c)
	if hasUser && strings.EqualFold(currentUser.Role, "user") {
		existing, err := h.service.FindByID(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Keluhan tidak ditemukan"})
			return
		}
		if existing.UserID != currentUser.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat menghapus keluhan milik sendiri"})
			return
		}
	}

	err := h.service.Delete(c.Param("id"))
	if err != nil {
		if IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Keluhan tidak ditemukan",
			})
			return
		}

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Keluhan tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Keluhan berhasil dihapus",
	})
}

func (h *Handler) FindByUserID(c *gin.Context) {
	currentUser, hasUser := getContextUser(c)
	userIDParam := c.Query("userID")

	var targetUserID uint
	if userIDParam == "" {
		if hasUser && currentUser.ID > 0 {
			targetUserID = currentUser.ID
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Parameter userID wajib diisi",
			})
			return
		}
	} else {
		parsedID, err := strconv.ParseUint(userIDParam, 10, 64)
		if err != nil || parsedID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "User ID harus berupa angka positif",
			})
			return
		}
		targetUserID = uint(parsedID)
	}

	if hasUser && strings.EqualFold(currentUser.Role, "user") && targetUserID != currentUser.ID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Anda hanya dapat melihat data keluhan milik sendiri",
		})
		return
	}

	result, err := h.service.FindByUserID(targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data keluhan berdasarkan User ID",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data keluhan berhasil diambil",
		"data":    result,
	})
}

func getContextUser(c *gin.Context) (auth.User, bool) {
	val, exists := c.Get("user")
	if !exists {
		return auth.User{}, false
	}
	u, ok := val.(auth.User)
	return u, ok
}

// MarkUnrepairable memungkinkan Teknisi (atau Admin) mengubah status aset yang dikeluhkan
// menjadi aset rusak yang tidak bisa diperbaiki (rusak berat/afkir).
func (h *Handler) MarkUnrepairable(c *gin.Context) {
	currentUser, hasUser := getContextUser(c)
	if !hasUser {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan login terlebih dahulu"})
		return
	}

	role := strings.ToLower(strings.TrimSpace(currentUser.Role))
	if role != "teknisi" && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Hanya teknisi atau admin yang dapat mengubah status aset menjadi rusak tidak bisa diperbaiki",
		})
		return
	}

	id := c.Param("id")
	var req struct {
		Note string `json:"note" form:"note"`
	}
	_ = c.ShouldBind(&req)

	result, err := h.service.MarkAsUnrepairable(id, req.Note)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Gagal mengubah status aset: %v", err),
		})
		return
	}

	// Catat aktivitas ke dashboard
	activityDescription := fmt.Sprintf(
		"Teknisi %s mengubah aset pada keluhan %s (%s) menjadi RUSAK TIDAK BISA DIPERBAIKI. Catatan: %s",
		currentUser.Name,
		result.Code,
		result.Title,
		req.Note,
	)
	_ = dashboard.AddActivity("complaint_unrepairable", activityDescription)

	c.JSON(http.StatusOK, gin.H{
		"message": "Aset berhasil diubah menjadi status rusak tidak bisa diperbaiki",
		"data":    result,
	})
}
