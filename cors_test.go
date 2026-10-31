package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCORS(t *testing.T) {
	router := setupEngine(nil, []byte("test-secret"))
	for _, origin := range []string{"http://localhost:8081", "http://127.0.0.1:8081", "https://example.com"} {
		for _, method := range []string{http.MethodOptions, http.MethodGet} {
			request := httptest.NewRequest(method, "/health", nil)
			request.Header.Set("Origin", origin)
			request.Header.Set("Access-Control-Request-Method", "POST")
			request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			want := origin
			if origin != "http://localhost:8081" {
				want = ""
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != want {
				t.Fatalf("%s %s: origin = %q, want %q", method, origin, got, want)
			}
			if want != "" && method == http.MethodOptions && response.Code != http.StatusNoContent {
				t.Fatalf("preflight status = %d", response.Code)
			}
			if want != "" && method == http.MethodOptions && response.Header().Get("Access-Control-Allow-Headers") != "Origin,Content-Type,Authorization" {
				t.Fatal("header autentikasi dan JSON harus diizinkan")
			}
		}
	}
}

func TestCookieLogoutWithoutOrigin(t *testing.T) {
	router := setupEngine(nil, []byte("test-secret"))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Logout tanpa Origin: status %d, ingin %d", w.Code, http.StatusOK)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "access_token" || cookies[0].MaxAge != -1 {
		t.Fatal("Logout harus menghapus cookie")
	}
}

func TestStaticFrontendServing(t *testing.T) {
	router := setupEngine(nil, []byte("test-secret-at-least-32-bytes-long!"))

	routesToTest := []string{
		"/",
		"/index.html",
		"/login",
		"/login.html",
		"/register",
		"/register.html",
		"/client/admin/dashboard.html",
	}

	for _, path := range routesToTest {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if path == "/index.html" {
			if w.Code != http.StatusOK && w.Code != http.StatusMovedPermanently {
				t.Errorf("Path %s: got status %d, want 200 or 301", path, w.Code)
			}
			continue
		}

		if w.Code != http.StatusOK {
			t.Errorf("Path %s: got status %d, want %d", path, w.Code, http.StatusOK)
		}
	}
}

func TestChatbotRoutesInEngine(t *testing.T) {
	router := setupEngine(nil, []byte("test-secret-at-least-32-bytes-long!"))

	// Test public suggestions
	req := httptest.NewRequest(http.MethodGet, "/api/public/chatbot/suggestions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Chatbot public suggestions failed: status %d", w.Code)
	}

	// Test extra slash handling
	reqSlash := httptest.NewRequest(http.MethodGet, "//api/public/chatbot/suggestions", nil)
	wSlash := httptest.NewRecorder()
	router.ServeHTTP(wSlash, reqSlash)
	if wSlash.Code != http.StatusOK && wSlash.Code != http.StatusMovedPermanently {
		t.Fatalf("Chatbot extra slash request failed: status %d", wSlash.Code)
	}
}
