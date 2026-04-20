package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
		"X-Content-Type-Options":   "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
	}
	for header, expected := range tests {
		got := rec.Header().Get(header)
		if got != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, got)
		}
	}
}

func TestCORSAllowedOrigin(t *testing.T) {
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
	}
	handler := CORSMiddleware(allowedOrigins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://localhost:5173" {
		t.Errorf("expected allowed origin, got %q", origin)
	}
}

func TestCORSDisallowedOrigin(t *testing.T) {
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
	}
	handler := CORSMiddleware(allowedOrigins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "" {
		t.Errorf("expected no origin header for disallowed, got %q", origin)
	}
}

func TestCORSPreflight(t *testing.T) {
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
	}
	handler := CORSMiddleware(allowedOrigins, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", rec.Code)
	}
}