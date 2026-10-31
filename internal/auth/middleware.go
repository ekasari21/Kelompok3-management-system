package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthReq memeriksa token dari cookie access_token atau header Bearer.
func (h *Handler) AuthReq() gin.HandlerFunc {
	return func(c *gin.Context) {
		unauthorized := func() {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token tidak valid atau kedaluwarsa",
			})
		}

		token, _ := c.Cookie("access_token")

		// Jika Authorization tersedia, gunakan token dari header.
		if header := c.GetHeader("Authorization"); header != "" {
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				unauthorized()
				return
			}
			token = parts[1]
		}

		if token == "" {
			unauthorized()
			return
		}

		// Authenticate mengembalikan User (value), bukan *User.
		user, err := h.service.Authenticate(c.Request.Context(), token)
		if errors.Is(err, ErrInvalidToken) {
			unauthorized()
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal membaca pengguna",
			})
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

// RequireRole hanya melanjutkan request jika role user diizinkan.
// Fungsi ini harus dipasang setelah AuthReq().
func RequireRole(allowedRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Silakan login terlebih dahulu",
			})
			return
		}

		user, ok := value.(User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Data autentikasi tidak valid",
			})
			return
		}

		if !strings.EqualFold(
			strings.TrimSpace(user.Role),
			strings.TrimSpace(allowedRole),
		) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Anda tidak memiliki akses ke halaman ini",
			})
			return
		}

		c.Next()
	}
}

// RequireRoles hanya melanjutkan request jika role user termasuk dalam allowedRoles.
func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Silakan login terlebih dahulu",
			})
			return
		}

		user, ok := value.(User)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Data autentikasi tidak valid",
			})
			return
		}

		userRole := strings.ToLower(strings.TrimSpace(user.Role))
		for _, r := range allowedRoles {
			if strings.EqualFold(userRole, strings.ToLower(strings.TrimSpace(r))) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Anda tidak memiliki akses ke fitur ini",
		})
	}
}
