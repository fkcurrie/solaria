package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestHandleSampleDay(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sample-day", nil)
	w := httptest.NewRecorder()

	handleSampleDay(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	var records []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		t.Fatalf("Failed to decode sample day json: %v", err)
	}
	if len(records) == 0 {
		t.Errorf("Expected non-empty sample day records")
	}
}

func TestPWAStaticFiles(t *testing.T) {
	manifest, err := staticFS.ReadFile("static/manifest.json")
	if err != nil || len(manifest) == 0 {
		t.Fatalf("Failed to read embedded manifest.json: %v", err)
	}

	sw, err := staticFS.ReadFile("static/sw.js")
	if err != nil || len(sw) == 0 {
		t.Fatalf("Failed to read embedded sw.js: %v", err)
	}

	logo, err := staticFS.ReadFile("static/assets/solaria-logo.svg")
	if err != nil || len(logo) == 0 {
		t.Fatalf("Failed to read embedded logo: %v", err)
	}
}

func TestStatsCache_TTLAndInvalidation(t *testing.T) {
	cache := &StatsCache{entries: make(map[string]CacheEntry)}

	// 1. Set and retrieve valid entry
	cache.Set("test_key", []byte(`{"hello":"world"}`), 200*time.Millisecond)
	val, ok := cache.Get("test_key")
	if !ok || string(val) != `{"hello":"world"}` {
		t.Errorf("Expected cache hit for active entry, got %v (%s)", ok, string(val))
	}

	// 2. Invalidate entry
	cache.Invalidate("test_key")
	_, okAfterInvalidate := cache.Get("test_key")
	if okAfterInvalidate {
		t.Errorf("Expected cache miss after explicit Invalidate()")
	}

	// 3. Expiration test
	cache.Set("expiring_key", []byte(`{"temp":1}`), 50*time.Millisecond)
	time.Sleep(70 * time.Millisecond)
	_, okExpired := cache.Get("expiring_key")
	if okExpired {
		t.Errorf("Expected cache miss for expired entry")
	}
}

func TestHandleDayStats_CacheHeaders(t *testing.T) {
	statsCache.InvalidateAll()

	// First request: Cache MISS
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/stats/day", nil)
	w1 := httptest.NewRecorder()
	handleDayStats(w1, req1)
	if w1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("Expected X-Cache: MISS on initial call, got %s", w1.Header().Get("X-Cache"))
	}

	// Second request: Cache HIT
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/stats/day", nil)
	w2 := httptest.NewRecorder()
	handleDayStats(w2, req2)
	if w2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("Expected X-Cache: HIT on subsequent call, got %s", w2.Header().Get("X-Cache"))
	}
}

func TestHandleHardwareConfig(t *testing.T) {
	// 1. GET initial hardware config
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/hardware-config", nil)
	wGet := httptest.NewRecorder()
	handleHardwareConfig(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/hardware-config, got %d", wGet.Code)
	}

	var cfg HardwareConfig
	if err := json.NewDecoder(wGet.Body).Decode(&cfg); err != nil {
		t.Fatalf("Failed to decode hardware config: %v", err)
	}
	if cfg.ControllerKey != "RVR20" {
		t.Errorf("Expected initial ControllerKey RVR20, got %s", cfg.ControllerKey)
	}

	// 2. POST update hardware config
	newPayload := `{"controller_key":"RVR20","controller_name":"Renogy Rover 20A MPPT (RNG-CTRL-RVR20)","controller_rated_amps":20,"battery_key":"RENOGY_170_LFP","battery_name":"Renogy 12V 170Ah LiFePO4 (RBT170LFP12-BT)","battery_capacity_ah":170,"array_capacity_watts":400,"array_topology":"2S2P (4x100W)"}`
	reqPost := httptest.NewRequest(http.MethodPost, "/api/v1/hardware-config", strings.NewReader(newPayload))
	wPost := httptest.NewRecorder()
	handleHardwareConfig(wPost, reqPost)
	if wPost.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on POST /api/v1/hardware-config, got %d", wPost.Code)
	}

	// 3. Verify updated latest ringBuffer record
	latest := ringBuf.GetLatest()
	if latest.Telemetry.ControllerModel != "Renogy Rover 20A MPPT (RNG-CTRL-RVR20)" {
		t.Errorf("Expected updated controller model in ring buffer, got %s", latest.Telemetry.ControllerModel)
	}
	if latest.Telemetry.BatteryType != "Renogy 12V 170Ah LiFePO4 (RBT170LFP12-BT)" {
		t.Errorf("Expected updated battery profile in ring buffer, got %s", latest.Telemetry.BatteryType)
	}
}

func TestHandlePowerBudget(t *testing.T) {
	// Test default / 75W load
	req := httptest.NewRequest(http.MethodGet, "/api/v1/power-budget?watts=75", nil)
	w := httptest.NewRecorder()
	handlePowerBudget(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/power-budget, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode power budget response: %v", err)
	}

	runtimeHours, ok := res["runtime_hours"].(float64)
	if !ok || runtimeHours <= 0 {
		t.Errorf("Expected positive runtime_hours, got %v", res["runtime_hours"])
	}

	usableWh, ok := res["usable_wh"].(float64)
	if !ok || usableWh <= 0 {
		t.Errorf("Expected positive usable_wh, got %v", res["usable_wh"])
	}

	// Test high load (e.g. 1000W -> critical warning)
	reqCrit := httptest.NewRequest(http.MethodGet, "/api/v1/power-budget?watts=1000", nil)
	wCrit := httptest.NewRecorder()
	handlePowerBudget(wCrit, reqCrit)
	var resCrit map[string]interface{}
	if err := json.NewDecoder(wCrit.Body).Decode(&resCrit); err != nil {
		t.Fatalf("Failed to decode critical power budget: %v", err)
	}
	if resCrit["status"] != "CRITICAL" {
		t.Errorf("Expected status CRITICAL for 1000W load, got %v", resCrit["status"])
	}
}

func TestHandleWinterizeStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/winterize-status", nil)
	w := httptest.NewRecorder()
	handleWinterizeStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/winterize-status, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode winterize status response: %v", err)
	}

	if res["site"] != "1296 Wren Lake Drive, Dorset, ON" {
		t.Errorf("Expected Dorset site, got %v", res["site"])
	}

	checklist, ok := res["departure_checklist"].([]interface{})
	if !ok || len(checklist) != 5 {
		t.Errorf("Expected 5-step departure checklist, got %v", res["departure_checklist"])
	}

	recs, ok := res["winter_recommendations"].([]interface{})
	if !ok || len(recs) == 0 {
		t.Errorf("Expected winter recommendations, got %v", res["winter_recommendations"])
	}
}

func TestHandleSunsetDigest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sunset-digest", nil)
	w := httptest.NewRecorder()
	handleSunsetDigest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/sunset-digest, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode sunset digest response: %v", err)
	}

	if res["site"] != "1296 Wren Lake Drive, Dorset, ON" {
		t.Errorf("Expected Dorset site, got %v", res["site"])
	}

	if res["today_generated_kwh"] == nil || res["evening_battery_soc_pct"] == nil {
		t.Errorf("Expected today_generated_kwh and evening_battery_soc_pct, got %v", res)
	}

	if res["tomorrow_peak_window"] != "11:30 AM - 02:30 PM" {
		t.Errorf("Expected tomorrow_peak_window 11:30 AM - 02:30 PM, got %v", res["tomorrow_peak_window"])
	}
}

func TestHandleShadingAnalysis(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shading-analysis", nil)
	w := httptest.NewRecorder()
	handleShadingAnalysis(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/shading-analysis, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode shading analysis response: %v", err)
	}

	if res["site"] != "1296 Wren Lake Drive, Dorset, ON" {
		t.Errorf("Expected Dorset site, got %v", res["site"])
	}

	patterns, ok := res["shading_patterns"].([]interface{})
	if !ok || len(patterns) == 0 {
		t.Errorf("Expected non-empty shading_patterns, got %v", res["shading_patterns"])
	}

	if res["total_shading_loss_kwh_day"] == nil {
		t.Errorf("Expected total_shading_loss_kwh_day in response")
	}
}

func TestHandleCommissioningWizard(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/commissioning-wizard", nil)
	w := httptest.NewRecorder()
	handleCommissioningWizard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/commissioning-wizard, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode commissioning wizard response: %v", err)
	}

	if res["site"] != "1296 Wren Lake Drive, Dorset, ON" {
		t.Errorf("Expected Dorset site, got %v", res["site"])
	}

	steps, ok := res["steps"].([]interface{})
	if !ok || len(steps) != 5 {
		t.Errorf("Expected 5 commissioning steps, got %v", res["steps"])
	}
}

func TestHandleArrayTopology(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/array-topology", nil)
	w := httptest.NewRecorder()
	handleArrayTopology(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/array-topology, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode array topology response: %v", err)
	}

	if res["status"] != "OPTIMAL" {
		t.Errorf("Expected OPTIMAL status for nominal 37.4V, got %v", res["status"])
	}

	// Test 4S overvoltage classification
	over := classifyTopology(78.0, 5.1)
	if over.Status != "WARNING_OVERVOLTAGE" {
		t.Errorf("Expected WARNING_OVERVOLTAGE for 78V, got %v", over.Status)
	}

	// Test 4P high current classification
	parallel := classifyTopology(19.0, 19.5)
	if parallel.Status != "SUBOPTIMAL_HIGH_CURRENT" {
		t.Errorf("Expected SUBOPTIMAL_HIGH_CURRENT for 19V, got %v", parallel.Status)
	}
}




