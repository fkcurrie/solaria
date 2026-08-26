package main

import (
	"bytes"
	"encoding/json"
	"html/template"
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

	// 2. POST update hardware config without auth -> 401 Unauthorized
	newPayload := `{"controller_key":"RVR20","controller_name":"Renogy Rover 20A MPPT (RNG-CTRL-RVR20)","controller_rated_amps":20,"battery_key":"RENOGY_170_LFP","battery_name":"Renogy 12V 170Ah LiFePO4 (RBT170LFP12-BT)","battery_capacity_ah":170,"array_capacity_watts":400,"array_topology":"2S2P (4x100W)"}`
	reqPostUnauth := httptest.NewRequest(http.MethodPost, "/api/v1/hardware-config", strings.NewReader(newPayload))
	wPostUnauth := httptest.NewRecorder()
	handleHardwareConfig(wPostUnauth, reqPostUnauth)
	if wPostUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401 on unauthenticated POST /api/v1/hardware-config, got %d", wPostUnauth.Code)
	}

	// 3. POST update hardware config WITH valid API Token -> 200 OK
	reqPostAuth := httptest.NewRequest(http.MethodPost, "/api/v1/hardware-config", strings.NewReader(newPayload))
	reqPostAuth.Header.Set("Authorization", "Bearer "+apiToken)
	wPostAuth := httptest.NewRecorder()
	handleHardwareConfig(wPostAuth, reqPostAuth)
	if wPostAuth.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on authenticated POST /api/v1/hardware-config, got %d", wPostAuth.Code)
	}

	// 4. Verify updated latest ringBuffer record
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

func TestHandleSunTimes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sun-times", nil)
	w := httptest.NewRecorder()
	handleSunTimes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/sun-times, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode sun-times response: %v", err)
	}

	if res["site"] != "1296 Wren Lake Drive, Dorset, ON" {
		t.Errorf("Expected Dorset site, got %v", res["site"])
	}
	if res["next_event"] != "sunset" && res["next_event"] != "sunrise" {
		t.Errorf("Expected next_event to be sunset or sunrise, got %v", res["next_event"])
	}
	if res["countdown_text"] == nil || res["countdown_text"] == "" {
		t.Errorf("Expected non-empty countdown_text")
	}
}

func TestCalculateSunTimes(t *testing.T) {
	// Dorset ON (45.186 N, -78.863 W) at solar noon in August
	loc, _ := time.LoadLocation("America/Toronto")
	if loc == nil {
		loc = time.FixedZone("EDT", -4*3600)
	}
	// 2026-08-24 13:00:00 EDT (17:00 UTC) -> should be daytime
	testDateDay := time.Date(2026, 8, 24, 13, 0, 0, 0, loc)
	sunrise, sunset, solarNoon, isDay := CalculateSunTimes(testDateDay, 45.186, -78.863)

	if !isDay {
		t.Errorf("Expected 13:00 EDT to be daytime, got isDay=false (sunrise=%v, sunset=%v)", sunrise, sunset)
	}
	if sunrise.Hour() < 5 || sunrise.Hour() > 8 {
		t.Errorf("Expected realistic sunrise between 5am and 8am, got %v", sunrise)
	}
	if sunset.Hour() < 19 || sunset.Hour() > 22 {
		t.Errorf("Expected realistic sunset between 7pm and 10pm, got %v", sunset)
	}
	if solarNoon.Hour() < 12 || solarNoon.Hour() > 14 {
		t.Errorf("Expected realistic solar noon between 12pm and 2pm, got %v", solarNoon)
	}

	// 2026-08-24 02:00:00 EDT -> should be night
	testDateNight := time.Date(2026, 8, 24, 2, 0, 0, 0, loc)
	_, _, _, isDayNight := CalculateSunTimes(testDateNight, 45.186, -78.863)
	if isDayNight {
		t.Errorf("Expected 02:00 EDT to be night, got isDay=true")
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
	if !ok || len(steps) != 6 {
		t.Errorf("Expected 6 commissioning steps, got %v", res["steps"])
	}
}

func TestHandleArrayOrientation(t *testing.T) {
	// 1. GET orientation
	req := httptest.NewRequest(http.MethodGet, "/api/v1/array-orientation", nil)
	w := httptest.NewRecorder()
	handleArrayOrientation(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/array-orientation, got %d", w.Code)
	}

	var res ArrayOrientationConfig
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode array orientation response: %v", err)
	}

	if res.AzimuthDeg != 135.0 {
		t.Errorf("Expected default Azimuth 135.0, got %v", res.AzimuthDeg)
	}
	if res.TiltDeg != 45.0 {
		t.Errorf("Expected default Tilt 45.0, got %v", res.TiltDeg)
	}
	if !strings.Contains(res.DirectionCompass, "South-East") {
		t.Errorf("Expected South-East direction, got %v", res.DirectionCompass)
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

func TestHandleBluetoothSignal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bluetooth-signal", nil)
	w := httptest.NewRecorder()
	handleBluetoothSignal(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/bluetooth-signal, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode bluetooth signal response: %v", err)
	}

	if res["module_type"] == nil || res["rssi_dbm"] == nil {
		t.Errorf("Expected module_type and rssi_dbm, got %v", res)
	}
}

func TestHandleNetworkDiscovery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/network-discovery", nil)
	w := httptest.NewRecorder()
	handleNetworkDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/network-discovery, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode network discovery response: %v", err)
	}

	if res["mdns_domain"] != "solaria.local" {
		t.Errorf("Expected solaria.local, got %v", res["mdns_domain"])
	}
}

func TestHandleGCPOnboarding(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gcp-onboarding", nil)
	w := httptest.NewRecorder()
	handleGCPOnboarding(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/v1/gcp-onboarding, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode GCP onboarding response: %v", err)
	}

	if res["setup_script"] != "./setup-gcp.sh" {
		t.Errorf("Expected ./setup-gcp.sh, got %v", res["setup_script"])
	}
}

func TestHandleSystemInfo_LiFePO4Parameters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system-info", nil)
	w := httptest.NewRecorder()
	handleSystemInfo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode system info: %v", err)
	}

	batt, ok := res["battery_bank"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing battery_bank map in response")
	}

	if batt["float_voltage_v"] != 13.6 {
		t.Errorf("Expected LiFePO4 float voltage 13.6V, got %v", batt["float_voltage_v"])
	}
	if batt["boost_voltage_v"] != 14.4 {
		t.Errorf("Expected LiFePO4 boost voltage 14.4V, got %v", batt["boost_voltage_v"])
	}
	if batt["low_voltage_disconnect_v"] != 10.6 {
		t.Errorf("Expected LiFePO4 LVD 10.6V, got %v", batt["low_voltage_disconnect_v"])
	}
	if batt["equalize_voltage_v"] != "NONE / Disabled (LiFePO4)" {
		t.Errorf("Expected LiFePO4 equalization disabled, got %v", batt["equalize_voltage_v"])
	}
}

func TestHandleSunTimes_SolarElevation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sun-times", nil)
	w := httptest.NewRecorder()
	handleSunTimes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode sun times: %v", err)
	}

	if res["latitude"] != 45.186 || res["longitude"] != -78.863 {
		t.Errorf("Expected Dorset coordinates 45.186, -78.863, got %v, %v", res["latitude"], res["longitude"])
	}
	if res["solar_elevation_deg"] == nil || res["solar_zenith_deg"] == nil {
		t.Errorf("Expected solar_elevation_deg and solar_zenith_deg in response")
	}
}

func TestRingBuffer_ClonedSliceIntegrity(t *testing.T) {
	rb := NewRingBuffer(10)

	rb.Push([]SolarRecord{
		{Timestamp: "item-1"},
		{Timestamp: "item-2"},
	})

	history := rb.GetHistory(2)
	if len(history) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(history))
	}

	// Mutate returned slice
	history[0].Timestamp = "MUTATED"

	// Ensure internal ring buffer was NOT mutated
	freshHistory := rb.GetHistory(2)
	if freshHistory[0].Timestamp == "MUTATED" {
		t.Errorf("Data race / aliasing bug: mutating returned slice modified internal ring buffer!")
	}
}

func BenchmarkRingBuffer_PushAndHistory(b *testing.B) {
	rb := NewRingBuffer(1440)
	item := SolarRecord{
		Timestamp: "2026-08-25T12:00:00Z",
		Telemetry: Telemetry{PVPowerW: 350, BatteryVoltageV: 13.4},
		Weather:   WeatherMetrics{DirectRadiationWM2: 800},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Push([]SolarRecord{item})
		_ = rb.GetHistory(60)
	}
}

func TestIndexTemplate_RenderAndUXElements(t *testing.T) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		t.Fatalf("Failed to parse index.html template: %v", err)
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to execute index.html template: %v", err)
	}

	htmlContent := buf.String()

	// 1. Verify Tab Content Panes (All 10 Desktop/Mobile Tab Panes)
	expectedPanes := []string{
		`id="tab-live"`,
		`id="tab-day"`,
		`id="tab-week"`,
		`id="tab-month"`,
		`id="tab-year"`,
		`id="tab-advisor"`,
		`id="tab-specs"`,
		`id="tab-diagnostics"`,
		`id="tab-forecast"`,
		`id="tab-more"`,
	}
	for _, pane := range expectedPanes {
		if !strings.Contains(htmlContent, pane) {
			t.Errorf("Missing expected tab pane: %s", pane)
		}
	}

	// 2. Verify Badge and Button CSS classes
	expectedCSS := []string{
		".badge-online",
		".badge-retry",
		".badge-offline",
		".btn-solar",
		".btn-outline",
		".header-btn",
		"@media (max-width: 768px)",
	}
	for _, css := range expectedCSS {
		if !strings.Contains(htmlContent, css) {
			t.Errorf("Missing expected CSS definition: %s", css)
		}
	}

	// 3. Verify Appliance Budget Items (including Inverter Idle and 85% DoD floor)
	expectedAppliances := []string{
		"appInverter",
		"appStarlink",
		"appFridge",
		"appLights",
		"appPump",
		"appLaptop",
		"85% DoD floor",
	}
	for _, app := range expectedAppliances {
		if !strings.Contains(htmlContent, app) {
			t.Errorf("Missing expected appliance/budget element: %s", app)
		}
	}
}

func TestHandleShadingAnalysis_FourPatterns(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/shading-analysis", nil)
	w := httptest.NewRecorder()
	handleShadingAnalysis(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var res struct {
		ShadingPatterns []ShadingPattern `json:"shading_patterns"`
	}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode shading patterns: %v", err)
	}

	if len(res.ShadingPatterns) != 4 {
		t.Fatalf("Expected 4 cottage shading patterns, got %d", len(res.ShadingPatterns))
	}

	foundMidday := false
	foundAfternoon := false
	for _, p := range res.ShadingPatterns {
		if strings.Contains(p.ObstructionType, "Midday") {
			foundMidday = true
		}
		if strings.Contains(p.ObstructionType, "Afternoon") {
			foundAfternoon = true
		}
	}

	if !foundMidday || !foundAfternoon {
		t.Errorf("Expected Midday and Afternoon shading patterns to be present in Dorset model")
	}
}

func TestDiagnosticLogBuffer_Cloud(t *testing.T) {
	buf := NewDiagnosticLogBuffer(10)

	buf.Log("INFO", "INGEST_PIPELINE", "Ingested 1 record", "INGEST_OK", nil)
	buf.Log("WARN", "AUTH_GATEWAY", "Unauthorized access attempt", "ERR_AUTH_FAIL", map[string]interface{}{"ip": "1.2.3.4"})
	buf.Log("ERROR", "BIGQUERY_STREAMER", "Streaming insert timeout", "ERR_BQ_TIMEOUT", nil)

	stats := buf.GetStats()
	if stats["total_logged"].(int) != 3 {
		t.Errorf("Expected total_logged 3, got %v", stats["total_logged"])
	}
	if stats["error_count"].(int64) != 1 {
		t.Errorf("Expected error_count 1, got %v", stats["error_count"])
	}
	if stats["warn_count"].(int64) != 1 {
		t.Errorf("Expected warn_count 1, got %v", stats["warn_count"])
	}

	// Subsystem filter
	ingestLogs := buf.GetLogs("", "INGEST_PIPELINE", "", 10)
	if len(ingestLogs) != 1 || ingestLogs[0].ErrorCode != "INGEST_OK" {
		t.Errorf("Expected 1 INGEST_PIPELINE log, got %v", ingestLogs)
	}

	// Search filter
	searchLogs := buf.GetLogs("", "", "timeout", 10)
	if len(searchLogs) != 1 || searchLogs[0].ErrorCode != "ERR_BQ_TIMEOUT" {
		t.Errorf("Expected 1 search match, got %v", searchLogs)
	}
}

func TestHandleLogs_Cloud(t *testing.T) {
	cloudLogger.Log("INFO", "INGEST_PIPELINE", "Unit test ingest log", "TEST_INGEST", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=50", nil)
	w := httptest.NewRecorder()
	handleLogs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp struct {
		Status string     `json:"status"`
		Count  int        `json:"count"`
		Logs   []LogEntry `json:"logs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Status != "ok" || resp.Count == 0 {
		t.Errorf("Expected status ok and count > 0, got %+v", resp)
	}
}

func TestHandleDiagnostics_Cloud(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/diagnostics", nil)
	w := httptest.NewRecorder()
	handleDiagnostics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode diagnostics: %v", err)
	}
	if resp["service"] != "solaria-cloud-server" {
		t.Errorf("Expected solaria-cloud-server, got %v", resp["service"])
	}
	if resp["health"] == nil || resp["runtime"] == nil {
		t.Errorf("Expected health and runtime keys, got %+v", resp)
	}
}

func TestHandleDiagnosticBundle_Cloud(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/diagnostic-bundle?download=true", nil)
	w := httptest.NewRecorder()
	handleDiagnosticBundle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	disp := w.Header().Get("Content-Disposition")
	if !strings.Contains(disp, "attachment; filename=\"solaria-diagnostics-") {
		t.Errorf("Expected attachment Content-Disposition header, got %s", disp)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode diagnostic bundle: %v", err)
	}
	if resp["cloud_server"] == nil || resp["edge_bridge"] == nil {
		t.Errorf("Expected cloud_server and edge_bridge keys in bundle, got %+v", resp)
	}
}

func TestHandleLive_StaleAndOutageDetection(t *testing.T) {
	rb := NewRingBuffer(10)
	// Push a stale record from 2 minutes ago
	staleTime := time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339)
	rb.Push([]SolarRecord{
		{
			Timestamp: staleTime,
			Telemetry: Telemetry{
				PVPowerW:        250,
				PVVoltageV:      36.0,
				PVCurrentA:      6.9,
				BatteryVoltageV: 13.5,
				ChargingState:   "MPPT_BULK",
			},
			BLEConnected: true,
			OutageStatus: "NOMINAL",
		},
	})

	// Temporarily swap global ringBuf
	oldBuf := ringBuf
	ringBuf = rb
	defer func() { ringBuf = oldBuf }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/live", nil)
	w := httptest.NewRecorder()
	handleLive(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var rec SolarRecord
	if err := json.NewDecoder(w.Body).Decode(&rec); err != nil {
		t.Fatalf("Failed to decode live record: %v", err)
	}

	if rec.BLEConnected != false {
		t.Errorf("Expected BLEConnected to be false for stale record, got %v", rec.BLEConnected)
	}
	if rec.OutageStatus != "STREAM_STALE" {
		t.Errorf("Expected OutageStatus 'STREAM_STALE', got %s", rec.OutageStatus)
	}
	if rec.Telemetry.PVPowerW != 0 {
		t.Errorf("Expected PVPowerW zeroed out (0), got %d", rec.Telemetry.PVPowerW)
	}
	if rec.Telemetry.ChargingState != "OFFLINE" {
		t.Errorf("Expected ChargingState 'OFFLINE', got %s", rec.Telemetry.ChargingState)
	}
}

func TestHandleBatteryControllerDiagnostics(t *testing.T) {
	rb := NewRingBuffer(10)
	nowTime := time.Now().UTC().Format(time.RFC3339)
	rb.Push([]SolarRecord{
		{
			Timestamp: nowTime,
			Telemetry: Telemetry{
				PVPowerW:        320,
				PVVoltageV:      36.5,
				PVCurrentA:      8.76,
				BatteryVoltageV: 13.5,
				BatteryCurrentA: 23.5,
				BatterySOCPct:   85,
				ControllerTempC: 32,
				BatteryTempC:    22,
				ChargingState:   "MPPT Charging",
				OperatingDays:   45,
			},
			Weather: WeatherMetrics{
				IsDay:                true,
				DirectRadiationWM2:   450.0,
				DiffuseRadiationWM2:  80.0,
			},
			BLEConnected: true,
			OutageStatus: "NOMINAL",
		},
	})

	oldBuf := ringBuf
	ringBuf = rb
	defer func() { ringBuf = oldBuf }()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/battery-controller-diagnostics", nil)
	w := httptest.NewRecorder()
	handleBatteryControllerDiagnostics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var rep BatteryControllerDiagnosticReport
	if err := json.NewDecoder(w.Body).Decode(&rep); err != nil {
		t.Fatalf("Failed to decode diagnostic report: %v", err)
	}

	if rep.HardwareProfile["charge_controller"] == nil || rep.HardwareProfile["battery_bank"] == nil {
		t.Errorf("Expected hardware profile fields, got %+v", rep.HardwareProfile)
	}

	if rep.BatteryHealth["voltage_zone"] == nil {
		t.Errorf("Expected voltage_zone in battery health, got %+v", rep.BatteryHealth)
	}

	if len(rep.ActiveAnomalies) == 0 {
		t.Errorf("Expected active anomaly categories in report")
	}

	if rep.NighttimeAnalysis["phantom_power_detected"] == nil {
		t.Errorf("Expected phantom_power_detected field in nighttime analysis")
	}
}

func TestHandlePeakGenerationForecast(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/peak-generation-forecast", nil)
	w := httptest.NewRecorder()
	handlePeakGenerationForecast(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp PeakForecastResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode peak forecast response: %v", err)
	}

	if resp.ArrayCapacityW != 400 {
		t.Errorf("Expected ArrayCapacityW 400, got %d", resp.ArrayCapacityW)
	}

	if resp.ArrayTiltDeg != 45.0 {
		t.Errorf("Expected ArrayTiltDeg 45.0, got %f", resp.ArrayTiltDeg)
	}

	if resp.ArrayAzimuthDeg != 135.0 {
		t.Errorf("Expected ArrayAzimuthDeg 135.0, got %f", resp.ArrayAzimuthDeg)
	}

	if len(resp.HourlyCurve) != 24 {
		t.Errorf("Expected 24 hourly curve points, got %d", len(resp.HourlyCurve))
	}

	if len(resp.MonthlyForecast) != 12 {
		t.Errorf("Expected 12 monthly forecast entries, got %d", len(resp.MonthlyForecast))
	}

	if resp.SolsticeAnalysis["summer_solstice"] == nil || resp.SolsticeAnalysis["winter_solstice"] == nil {
		t.Errorf("Expected summer and winter solstice analysis, got %+v", resp.SolsticeAnalysis)
	}

	if len(resp.ApplianceGuidance) == 0 {
		t.Errorf("Expected appliance guidance entries")
	}

	if resp.LearnedModel["accuracy_score_pct"] == nil {
		t.Errorf("Expected accuracy_score_pct in LearnedModel, got %+v", resp.LearnedModel)
	}
}

func TestSolarModelLearner(t *testing.T) {
	learner := NewSolarModelLearner("")

	// 1. Test theoretical solar calculation
	loc, _ := time.LoadLocation("America/Toronto")
	if loc == nil {
		loc = time.FixedZone("EDT", -4*3600)
	}
	middayTime := time.Date(2026, 8, 25, 11, 30, 0, 0, loc)
	theoMidday := computeTheoreticalWatts(middayTime, 45.186, -78.863, 45.0, 135.0, 400.0)
	if theoMidday < 200 || theoMidday > 400 {
		t.Errorf("Expected midday theoretical watts between 200W and 400W, got %d", theoMidday)
	}

	nightTime := time.Date(2026, 8, 25, 1, 0, 0, 0, loc)
	theoNight := computeTheoreticalWatts(nightTime, 45.186, -78.863, 45.0, 135.0, 400.0)
	if theoNight != 0 {
		t.Errorf("Expected nighttime theoretical watts 0, got %d", theoNight)
	}

	// 2. Test EMA training update
	// Train with actual = 80% of theoretical (e.g. tree shade)
	rec := SolarRecord{
		Timestamp: middayTime.Format(time.RFC3339),
		Telemetry: Telemetry{
			PVPowerW: int(float64(theoMidday) * 0.8),
		},
	}
	learner.TrainRecord(rec)

	hour := middayTime.Hour()
	if learner.HourlyMultipliers[hour] >= 1.0 {
		t.Errorf("Expected hourly multiplier for hour %d to decrease after lower actuals, got %f", hour, learner.HourlyMultipliers[hour])
	}

	// 3. Test batch training
	var batch []SolarRecord
	for h := 8; h <= 17; h++ {
		tSample := time.Date(2026, 8, 25, h, 15, 0, 0, loc)
		theo := computeTheoreticalWatts(tSample, 45.186, -78.863, 45.0, 135.0, 400.0)
		batch = append(batch, SolarRecord{
			Timestamp: tSample.Format(time.RFC3339),
			Telemetry: Telemetry{
				PVPowerW: int(float64(theo) * 0.95),
			},
		})
	}
	learner.TrainBatch(batch)

	summary := learner.GetSummary()
	if summary["training_samples"].(int64) == 0 {
		t.Errorf("Expected non-zero training samples")
	}
	if summary["accuracy_score_pct"].(float64) < 80.0 {
		t.Errorf("Expected high accuracy score, got %v", summary["accuracy_score_pct"])
	}

	// 4. Test Persistence Save/Load
	tempFile := t.TempDir() + "/solar_learned_test.json"
	learner.filePath = tempFile
	if err := learner.Save(); err != nil {
		t.Fatalf("Failed to save learned model: %v", err)
	}

	learner2 := NewSolarModelLearner(tempFile)
	if learner2.TrainingSamples != learner.TrainingSamples {
		t.Errorf("Expected loaded model to have %d samples, got %d", learner.TrainingSamples, learner2.TrainingSamples)
	}
}

func TestHandleModelRetrain(t *testing.T) {
	rb := NewRingBuffer(10)
	nowTime := time.Now().UTC().Format(time.RFC3339)
	rb.Push([]SolarRecord{
		{
			Timestamp: nowTime,
			Telemetry: Telemetry{
				PVPowerW:        340,
				PVVoltageV:      36.8,
				PVCurrentA:      9.2,
				BatteryVoltageV: 13.6,
			},
		},
	})

	oldBuf := ringBuf
	ringBuf = rb
	defer func() { ringBuf = oldBuf }()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/model-retrain", nil)
	w := httptest.NewRecorder()
	handleModelRetrain(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on /api/v1/model-retrain, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode model-retrain response: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("Expected status success, got %v", resp["status"])
	}
	if resp["samples_trained"].(float64) != 2 {
		t.Errorf("Expected 2 samples trained (initial + pushed), got %v", resp["samples_trained"])
	}
}

func TestHandleE2EAudit(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/e2e-audit", nil)
	w := httptest.NewRecorder()
	handleE2EAudit(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on /api/v1/e2e-audit, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode e2e-audit response: %v", err)
	}

	if resp["total_probes"].(float64) < 3 {
		t.Errorf("Expected at least 3 probes in scorecard, got %v", resp["total_probes"])
	}
}
