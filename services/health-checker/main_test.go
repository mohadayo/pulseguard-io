package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "healthy" {
		t.Errorf("expected healthy, got %s", resp.Status)
	}
	if resp.Service != "health-checker" {
		t.Errorf("expected health-checker, got %s", resp.Service)
	}
}

func TestCheckHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/check", nil)
	w := httptest.NewRecorder()
	checkHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestCheckHandlerMissingURL(t *testing.T) {
	body, _ := json.Marshal(CheckRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/check", bytes.NewReader(body))
	w := httptest.NewRecorder()
	checkHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCheckHandlerInvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/check", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	checkHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCheckHandlerSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer ts.Close()

	body, _ := json.Marshal(CheckRequest{URL: ts.URL, TimeoutSeconds: 5})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/check", bytes.NewReader(body))
	w := httptest.NewRecorder()
	checkHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result CheckResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result.Status != "healthy" {
		t.Errorf("expected healthy, got %s", result.Status)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected 200, got %d", result.StatusCode)
	}
}

func TestCheckHandlerUnhealthy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	body, _ := json.Marshal(CheckRequest{URL: ts.URL})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/check", bytes.NewReader(body))
	w := httptest.NewRecorder()
	checkHandler(w, req)

	var result CheckResult
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
}

func TestBatchCheckHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/check/batch", nil)
	w := httptest.NewRecorder()
	batchCheckHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestBatchCheckHandlerEmpty(t *testing.T) {
	body, _ := json.Marshal([]CheckRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/check/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()
	batchCheckHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestBatchCheckHandlerSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	requests := []CheckRequest{
		{URL: ts.URL, TimeoutSeconds: 5},
		{URL: ts.URL, TimeoutSeconds: 5},
	}
	body, _ := json.Marshal(requests)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/check/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()
	batchCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]CheckResult
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp["results"]) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp["results"]))
	}
}

func TestPerformCheckTimeout(t *testing.T) {
	result := performCheck("http://192.0.2.1:1", 1) // non-routable, will timeout quickly
	if result.Status != "unhealthy" {
		t.Errorf("expected unhealthy for unreachable host, got %s", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for unreachable host")
	}
}
