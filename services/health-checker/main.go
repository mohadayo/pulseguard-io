package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type HealthResponse struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Timestamp     string `json:"timestamp"`
}

type CheckRequest struct {
	URL            string `json:"url"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type CheckResult struct {
	URL            string `json:"url"`
	Status         string `json:"status"`
	StatusCode     int    `json:"status_code,omitempty"`
	ResponseTimeMs int64  `json:"response_time_ms"`
	Error          string `json:"error,omitempty"`
	CheckedAt      string `json:"checked_at"`
}

var startTime = time.Now()

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:        "healthy",
		Service:       "health-checker",
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, `{"error":"url is required"}`, http.StatusBadRequest)
		return
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	result := performCheck(req.URL, timeout)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func performCheck(url string, timeout time.Duration) CheckResult {
	client := &http.Client{Timeout: timeout}
	start := time.Now()

	resp, err := client.Get(url)
	elapsed := time.Since(start).Milliseconds()

	result := CheckResult{
		URL:            url,
		ResponseTimeMs: elapsed,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	if err != nil {
		log.Printf("Check failed for %s: %v", url, err)
		result.Status = "unhealthy"
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "healthy"
	} else {
		result.Status = "unhealthy"
	}

	log.Printf("Check completed for %s: status=%s code=%d time=%dms",
		url, result.Status, result.StatusCode, elapsed)

	return result
}

func batchCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var requests []CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(requests) == 0 {
		http.Error(w, `{"error":"at least one check request is required"}`, http.StatusBadRequest)
		return
	}

	type indexedResult struct {
		index  int
		result CheckResult
	}

	ch := make(chan indexedResult, len(requests))
	for i, req := range requests {
		go func(idx int, r CheckRequest) {
			timeout := time.Duration(r.TimeoutSeconds) * time.Second
			if timeout <= 0 {
				timeout = 10 * time.Second
			}
			ch <- indexedResult{index: idx, result: performCheck(r.URL, timeout)}
		}(i, req)
	}

	results := make([]CheckResult, len(requests))
	for range requests {
		ir := <-ch
		results[ir.index] = ir.result
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}

func main() {
	port := getEnv("CHECKER_PORT", "8002")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/v1/check", checkHandler)
	mux.HandleFunc("/api/v1/check/batch", batchCheckHandler)

	log.Printf("Starting Health Checker on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
