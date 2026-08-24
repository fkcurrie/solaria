package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if body["status"] != "healthy" {
		t.Errorf("Expected status healthy, got %v", body["status"])
	}
}

func TestVerifyAuth(t *testing.T) {
	// Set test token
	apiToken = "test_secret_token_123"

	// 1. Valid X-API-Key
	reqValidHeader := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	reqValidHeader.Header.Set("X-API-Key", "test_secret_token_123")
	if !verifyAuth(reqValidHeader) {
		t.Errorf("Expected valid auth for correct X-API-Key")
	}

	// 2. Valid Bearer Token
	reqValidBearer := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	reqValidBearer.Header.Set("Authorization", "Bearer test_secret_token_123")
	if !verifyAuth(reqValidBearer) {
		t.Errorf("Expected valid auth for correct Bearer token")
	}

	// 3. Invalid Token
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	reqInvalid.Header.Set("X-API-Key", "wrong_token")
	if verifyAuth(reqInvalid) {
		t.Errorf("Expected invalid auth for wrong token")
	}

	// 4. Missing Token
	reqMissing := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", nil)
	if verifyAuth(reqMissing) {
		t.Errorf("Expected invalid auth for missing token")
	}
}

func TestHandleIngest_Unauthorized(t *testing.T) {
	apiToken = "test_secret_token_123"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", strings.NewReader(`{"batch":[]}`))
	w := httptest.NewRecorder()

	handleIngest(w, req)
	if w.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", w.Result().StatusCode)
	}
}

func TestHandleIngest_Valid(t *testing.T) {
	apiToken = "test_secret_token_123"

	batch := IngestBatch{
		Batch: []SolarRecord{
			{
				Timestamp: "2026-08-24T12:00:00Z",
				Site:      "1296 Wren Lake Drive",
				Telemetry: Telemetry{
					PVPowerW:        280,
					PVVoltageV:      36.4,
					BatterySOCPct:   85,
					BatteryVoltageV: 13.3,
				},
			},
		},
	}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest", bytes.NewReader(body))
	req.Header.Set("X-API-Key", "test_secret_token_123")
	w := httptest.NewRecorder()

	handleIngest(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", w.Result().StatusCode)
	}

	latest := ringBuf.GetLatest()
	if latest.Telemetry.PVPowerW != 280 {
		t.Errorf("Expected latest PV power 280W, got %dW", latest.Telemetry.PVPowerW)
	}
}
