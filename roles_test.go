package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gin-auth/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func createTestToken(id uint, email, role string, secret []byte) (string, error) {
	now := time.Now()
	claims := &auth.Claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "gin-auth",
			Subject:   strconv.FormatUint(uint64(id), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func TestRoleRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret-at-least-32-bytes-long!")

	t.Run("RequireRoles Middleware", func(t *testing.T) {
		r := gin.New()
		r.GET("/protected", func(c *gin.Context) {
			// Simulasikan inject user dari auth middleware
			role := c.Query("role")
			if role != "" {
				c.Set("user", auth.User{ID: 10, Role: role})
			}
			c.Next()
		}, auth.RequireRoles("admin", "teknisi"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// 1. Tanpa login -> 401 Unauthorized
		req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)
		if w1.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401 Unauthorized, got %d", w1.Code)
		}

		// 2. Role user -> 403 Forbidden
		req2 := httptest.NewRequest(http.MethodGet, "/protected?role=user", nil)
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)
		if w2.Code != http.StatusForbidden {
			t.Fatalf("Expected 403 Forbidden for role user, got %d", w2.Code)
		}

		// 3. Role teknisi -> 200 OK
		req3 := httptest.NewRequest(http.MethodGet, "/protected?role=teknisi", nil)
		w3 := httptest.NewRecorder()
		r.ServeHTTP(w3, req3)
		if w3.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for role teknisi, got %d", w3.Code)
		}

		// 4. Role admin -> 200 OK
		req4 := httptest.NewRequest(http.MethodGet, "/protected?role=admin", nil)
		w4 := httptest.NewRecorder()
		r.ServeHTTP(w4, req4)
		if w4.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK for role admin, got %d", w4.Code)
		}
	})

	t.Run("Teknisi Forbidden from Asset Lending", func(t *testing.T) {
		router := setupEngine(nil, secret)

		// Mock token generation
		token, err := createTestToken(1, "teknisi@example.com", "teknisi", secret)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// GET /api/private/asset-lending as teknisi -> must be 403
		req := httptest.NewRequest(http.MethodGet, "/api/private/asset-lending", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("Teknisi should receive 403 Forbidden on asset lending, got %d", w.Code)
		}

		// POST /api/private/asset-lending as teknisi -> must be 403
		reqPost := httptest.NewRequest(http.MethodPost, "/api/private/asset-lending", bytes.NewBufferString(`{}`))
		reqPost.Header.Set("Authorization", "Bearer "+token)
		reqPost.Header.Set("Content-Type", "application/json")
		wPost := httptest.NewRecorder()
		router.ServeHTTP(wPost, reqPost)

		if wPost.Code != http.StatusForbidden {
			t.Fatalf("Teknisi should receive 403 Forbidden on create lending, got %d", wPost.Code)
		}
	})

	t.Run("User Forbidden from Mark Complaint Unrepairable", func(t *testing.T) {
		router := setupEngine(nil, secret)

		userToken, err := createTestToken(2, "user@example.com", "user", secret)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// POST /api/private/complaints/:id/mark-unrepairable as user -> must be 403
		req := httptest.NewRequest(http.MethodPost, "/api/private/complaints/123/mark-unrepairable", bytes.NewBufferString(`{"note":"rusak"}`))
		req.Header.Set("Authorization", "Bearer "+userToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("User should receive 403 Forbidden when trying to mark unrepairable, got %d", w.Code)
		}
	})

	t.Run("User Scoped on ComplaintsByUserID", func(t *testing.T) {
		router := setupEngine(nil, secret)

		userToken, err := createTestToken(5, "user5@example.com", "user", secret)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// User 5 queries user 99 -> must be 403 Forbidden
		req := httptest.NewRequest(http.MethodGet, "/api/private/complaintsByUserID?userID=99", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("User should receive 403 Forbidden when accessing another user's complaints, got %d", w.Code)
		}
	})
}
