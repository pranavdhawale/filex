package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFileInitBadSize(t *testing.T) {
	h := &FileHandler{}
	body, _ := json.Marshal(InitUploadRequest{
		Filename:     "test.pdf",
		Size:         -1,
		TTLSeconds:   3600,
		EncryptedFEK: "abc",
		Salt:         "def",
	})
	req := httptest.NewRequest("POST", "/api/v1/files/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleInit(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative size, got %d", rec.Code)
	}
}

func TestFileInitMissingFilename(t *testing.T) {
	h := &FileHandler{}
	body, _ := json.Marshal(InitUploadRequest{
		Filename:     "",
		Size:         1024,
		TTLSeconds:   3600,
		EncryptedFEK: "abc",
		Salt:         "def",
	})
	req := httptest.NewRequest("POST", "/api/v1/files/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleInit(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty filename, got %d", rec.Code)
	}
}

func TestFileInitMissingFEK(t *testing.T) {
	h := &FileHandler{}
	body, _ := json.Marshal(InitUploadRequest{
		Filename:     "test.pdf",
		Size:         1024,
		TTLSeconds:   3600,
		EncryptedFEK: "",
		Salt:         "def",
	})
	req := httptest.NewRequest("POST", "/api/v1/files/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleInit(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing encrypted_fek, got %d", rec.Code)
	}
}

func TestFileInitInvalidTTL(t *testing.T) {
	h := &FileHandler{}
	body, _ := json.Marshal(InitUploadRequest{
		Filename:     "test.pdf",
		Size:         1024,
		TTLSeconds:   999,
		EncryptedFEK: "abc",
		Salt:         "def",
	})
	req := httptest.NewRequest("POST", "/api/v1/files/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleInit(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid TTL, got %d", rec.Code)
	}
}

func TestFileInitSizeTooLarge(t *testing.T) {
	h := &FileHandler{}
	body, _ := json.Marshal(InitUploadRequest{
		Filename:     "test.pdf",
		Size:         6 * 1024 * 1024 * 1024, // 6GB
		TTLSeconds:   3600,
		EncryptedFEK: "abc",
		Salt:         "def",
	})
	req := httptest.NewRequest("POST", "/api/v1/files/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleInit(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for size too large, got %d", rec.Code)
	}
}