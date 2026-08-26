package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDecodeTelemetry_Bridge(t *testing.T) {
	// 73-byte Modbus RTU telemetry frame: raw[0]=0xFF, raw[1]=0x03, raw[2]=0x44 (68 bytes) + 68 data bytes + 2 CRC bytes
	buf := make([]byte, 73)
	buf[0] = 0xFF
	buf[1] = 0x03
	buf[2] = 0x44

	// data[0:2] = raw[3:5]: Battery SOC = 92% (0x00, 0x5C)
	buf[3] = 0x00
	buf[4] = 92

	// data[2:4] = raw[5:7]: Battery Voltage = 13.4V (134 -> 0x0086)
	buf[5] = 0x00
	buf[6] = 0x86

	// data[4:6] = raw[7:9]: Battery Current = 15.20A (1520 -> 0x05F0)
	buf[7] = 0x05
	buf[8] = 0xF0

	// data[6] = raw[9]: Controller Temp = 28C
	buf[9] = 28

	// data[7] = raw[10]: Battery Temp = 22C
	buf[10] = 22

	// data[8:10] = raw[11:13]: Load Voltage = 12.0V
	buf[11] = 0x00
	buf[12] = 120

	// data[10:12] = raw[13:15]: Load Current = 0.0A
	buf[13] = 0x00
	buf[14] = 0x00

	// data[12:14] = raw[15:17]: Load Power = 0W
	buf[15] = 0x00
	buf[16] = 0x00

	// data[14:16] = raw[17:19]: PV Voltage = 35.8V (358 -> 0x0166)
	buf[17] = 0x01
	buf[18] = 0x66

	// data[16:18] = raw[19:21]: PV Current = 5.80A (580 -> 0x0244)
	buf[19] = 0x02
	buf[20] = 0x44

	// data[18:20] = raw[21:23]: PV Power = 208W (0x00D0)
	buf[21] = 0x00
	buf[22] = 0xD0

	telem, err := decodeTelemetry(buf)
	if err != nil {
		t.Fatalf("decodeTelemetry returned error: %v", err)
	}

	if telem.PVPowerW != 208 {
		t.Errorf("Expected PV Power 208W, got %dW", telem.PVPowerW)
	}

	if telem.BatterySOCPct != 92 {
		t.Errorf("Expected Battery SOC 92%%, got %d%%", telem.BatterySOCPct)
	}

	if telem.PVVoltageV < 35.7 || telem.PVVoltageV > 35.9 {
		t.Errorf("Expected PV Voltage ~35.8V, got %.1fV", telem.PVVoltageV)
	}

	if telem.StringHealthStatus != "NOMINAL_2S2P_ACTIVE" {
		t.Errorf("Expected NOMINAL_2S2P_ACTIVE, got %s", telem.StringHealthStatus)
	}

	if telem.MPPTEfficiencyPct <= 0 || telem.MPPTEfficiencyPct > 100 {
		t.Errorf("Expected MPPT efficiency between 0 and 100%%, got %.1f%%", telem.MPPTEfficiencyPct)
	}
}

func TestCalcCRC16_Bridge(t *testing.T) {
	data := []byte{0x01, 0x03, 0x00, 0x00, 0x00, 0x0A}
	crcLo, crcHi := calcCRC16(data)
	if crcLo == 0 && crcHi == 0 {
		t.Errorf("Expected non-zero CRC16, got 0")
	}
}

func TestBuildMockRTUFrame_Valid(t *testing.T) {
	frame := buildMockRTUFrame(350, 37.5, 13.8, 25.2, 94, 32, 24)
	if len(frame) != 73 {
		t.Fatalf("Expected 73 bytes, got %d", len(frame))
	}
	telem, err := decodeTelemetry(frame)
	if err != nil {
		t.Fatalf("Failed to decode mock frame: %v", err)
	}
	if telem.PVPowerW != 350 {
		t.Errorf("Expected PV Power 350W, got %dW", telem.PVPowerW)
	}
	if telem.BatterySOCPct != 94 {
		t.Errorf("Expected SOC 94%%, got %d%%", telem.BatterySOCPct)
	}
}

func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		origin  string
		allowed bool
	}{
		{"", true},
		{"http://localhost:8080", true},
		{"http://127.0.0.1:8080", true},
		{"http://solaria.local:8080", true},
		{"http://192.168.1.100:8080", true},
		{"http://10.0.0.45:8080", true},
		{"https://solaria-dashboard-952659886764.us-central1.run.app", true},
		{"https://evil-attacker.run.app", false},
		{"chrome-extension://solaria-bridge-helper", false},
		{"https://malicious-site.com", false},
		{"http://54.210.12.89", false},
	}

	for _, tc := range tests {
		req, _ := http.NewRequest("GET", "ws://localhost:8765", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		res := isAllowedOrigin(req)
		if res != tc.allowed {
			t.Errorf("Origin %q: expected %v, got %v", tc.origin, tc.allowed, res)
		}
	}
}

func TestVerifyBridgeAuth(t *testing.T) {
	bridgeToken = "test_secret_bridge_token_123"
	defer func() { bridgeToken = "" }()

	// 1. Valid Authorization Header
	req1, _ := http.NewRequest("GET", "ws://localhost:8765", nil)
	req1.Header.Set("Authorization", "Bearer test_secret_bridge_token_123")
	if !verifyBridgeAuth(req1, "") {
		t.Errorf("Expected auth with Bearer header to pass")
	}

	// 2. Valid X-API-Key Header
	req2, _ := http.NewRequest("GET", "ws://localhost:8765", nil)
	req2.Header.Set("X-API-Key", "test_secret_bridge_token_123")
	if !verifyBridgeAuth(req2, "") {
		t.Errorf("Expected auth with X-API-Key header to pass")
	}

	// 3. Valid Payload Token
	if !verifyBridgeAuth(nil, "test_secret_bridge_token_123") {
		t.Errorf("Expected auth with payload token to pass")
	}

	// 4. Invalid Token
	if verifyBridgeAuth(nil, "wrong_token_456") {
		t.Errorf("Expected auth with invalid token to fail")
	}
}

func TestCheckControlRateLimit(t *testing.T) {
	client := "test_client_ip:12345"
	if !checkControlRateLimit(client, 100*time.Millisecond) {
		t.Errorf("First rate limit check should succeed")
	}
	if checkControlRateLimit(client, 100*time.Millisecond) {
		t.Errorf("Immediate second check should be rate-limited")
	}
	time.Sleep(120 * time.Millisecond)
	if !checkControlRateLimit(client, 100*time.Millisecond) {
		t.Errorf("Check after delay should succeed")
	}
}

func TestDiskSpooler_SpoolAndDrain(t *testing.T) {
	tmpDir := t.TempDir()
	spooler := NewDiskSpooler(tmpDir)

	record1 := SolarRecord{
		Timestamp: "2026-08-24T12:00:00Z",
		Site:      "Test Site",
		Telemetry: Telemetry{PVPowerW: 300},
	}
	record2 := SolarRecord{
		Timestamp: "2026-08-24T12:01:00Z",
		Site:      "Test Site",
		Telemetry: Telemetry{PVPowerW: 310},
	}

	if err := spooler.Spool(record1); err != nil {
		t.Fatalf("Failed to spool record 1: %v", err)
	}
	if err := spooler.Spool(record2); err != nil {
		t.Fatalf("Failed to spool record 2: %v", err)
	}

	// Test successful drain
	var uploaded []SolarRecord
	drained, err := spooler.Drain(func(rec SolarRecord) error {
		uploaded = append(uploaded, rec)
		return nil
	})

	if err != nil {
		t.Fatalf("Drain returned error: %v", err)
	}
	if drained != 2 {
		t.Errorf("Expected 2 records drained, got %d", drained)
	}
	if len(uploaded) != 2 {
		t.Errorf("Expected 2 uploaded records, got %d", len(uploaded))
	}

	// Verify spool is empty after complete drain
	drainedAgain, _ := spooler.Drain(func(rec SolarRecord) error {
		return nil
	})
	if drainedAgain != 0 {
		t.Errorf("Expected 0 records on second drain, got %d", drainedAgain)
	}
}

func TestDecodeTelemetry_ColdDeratingAndStringImbalance(t *testing.T) {
	// Frame with Battery Temp = 4C (Cold derate zone 0-5C)
	frame := buildMockRTUFrame(300, 36.5, 13.5, 20.0, 80, 25, 4)
	telem, err := decodeTelemetry(frame)
	if err != nil {
		t.Fatalf("decodeTelemetry failed: %v", err)
	}
	if !telem.ColdDerateWarning {
		t.Errorf("Expected ColdDerateWarning to be true at 4C")
	}

	// Frame with Battery Temp = -2C (Sub-zero inhibit)
	frameSubZero := buildMockRTUFrame(300, 36.5, 13.5, 20.0, 80, 25, -2)
	telemSubZero, _ := decodeTelemetry(frameSubZero)
	if !telemSubZero.SubZeroInhibitWarning {
		t.Errorf("Expected SubZeroInhibitWarning to be true at -2°C")
	}

	// Frame with Battery Temp = 0C (Genuine freezing point -> Sub-zero inhibit MUST trigger)
	frameFreezing := buildMockRTUFrame(300, 36.5, 13.5, 20.0, 80, 25, 0)
	telemFreezing, _ := decodeTelemetry(frameFreezing)
	if !telemFreezing.SubZeroInhibitWarning {
		t.Errorf("Expected SubZeroInhibitWarning to be true at 0°C freezing point")
	}

	// Frame with Unconnected RTS probe (bit 13 = 0x2000 in fault register at raw[69:71]) with 25°C controller temp
	frameNoProbe := buildMockRTUFrame(300, 36.5, 13.5, 20.0, 80, 25, 0)
	frameNoProbe[69] = 0x20 // Bit 13 (0x2000 in uint16 big endian)
	telemNoProbe, _ := decodeTelemetry(frameNoProbe)
	if telemNoProbe.SubZeroInhibitWarning {
		t.Errorf("Expected SubZeroInhibitWarning to be false when probe disconnected with 25°C controller temp (effective temp = 20°C)")
	}

	// Frame with String Imbalance: PV Volts = 18.0V (single string/bypass) while power is 120W
	frameImbalance := buildMockRTUFrame(120, 18.0, 13.5, 8.5, 80, 25, 20)
	telemImbalance, _ := decodeTelemetry(frameImbalance)
	if telemImbalance.StringHealthStatus != "POTENTIAL_SERIES_DIODE_BYPASS_OR_SINGLE_PANEL_FAULT" {
		t.Errorf("Expected POTENTIAL_SERIES_DIODE_BYPASS_OR_SINGLE_PANEL_FAULT, got %s", telemImbalance.StringHealthStatus)
	}
}

func TestHandleNetworkDiscovery(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/network-discovery", nil)
	w := httptest.NewRecorder()
	handleNetworkDiscovery(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var info NetworkDiscoveryInfo
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("Failed to decode discovery info: %v", err)
	}

	if info.MDNSDomain != "solaria.local" {
		t.Errorf("Expected solaria.local, got %s", info.MDNSDomain)
	}
	if len(info.LocalIPs) == 0 {
		t.Errorf("Expected at least one local IP address")
	}
}

func TestDiskSpooler_Count(t *testing.T) {
	tempDir := t.TempDir()
	spooler := NewDiskSpooler(tempDir)

	if count := spooler.Count(); count != 0 {
		t.Fatalf("Expected empty spool count 0, got %d", count)
	}

	rec := SolarRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Site:      "Test Site",
		Telemetry: Telemetry{PVPowerW: 250, BatterySOCPct: 90},
	}
	_ = spooler.Spool(rec)
	_ = spooler.Spool(rec)

	if count := spooler.Count(); count != 2 {
		t.Fatalf("Expected spool count 2, got %d", count)
	}
}

func TestHandleBridgeStatus(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/bridge-status", nil)
	w := httptest.NewRecorder()
	handleBridgeStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to parse bridge status JSON: %v", err)
	}

	if resp["site"] == nil || resp["site"] == "" {
		t.Errorf("Expected non-empty site name in status response")
	}
	if _, ok := resp["spool_count"]; !ok {
		t.Errorf("Expected spool_count in response")
	}
	if _, ok := resp["total_successful_uploads"]; !ok {
		t.Errorf("Expected total_successful_uploads in response")
	}
}

func TestTracker_PersistAndRestoreUploadAndFrames(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := outageFilePath
	outageFilePath = tempDir + "/test_outages.json"
	defer func() { outageFilePath = originalPath }()

	testTime := time.Date(2026, 8, 24, 21, 0, 0, 0, time.UTC)
	uploadMu.Lock()
	lastSuccessUpload = testTime
	totalSuccessUploads = 42
	uploadMu.Unlock()

	frameMu.Lock()
	lastFrameTime = testTime
	totalFramesProcessed = 150
	frameMu.Unlock()

	testTracker := &OutageTracker{
		sessionStart: time.Now(),
		firstStart:   time.Now(),
		history:      make([]OutageEvent, 0),
	}
	testTracker.save()

	// Reset in-memory values
	uploadMu.Lock()
	lastSuccessUpload = time.Time{}
	totalSuccessUploads = 0
	uploadMu.Unlock()

	frameMu.Lock()
	lastFrameTime = time.Time{}
	totalFramesProcessed = 0
	frameMu.Unlock()

	// Load back
	newTracker := &OutageTracker{}
	newTracker.load()

	uploadMu.Lock()
	loadedUploads := totalSuccessUploads
	loadedTime := lastSuccessUpload
	uploadMu.Unlock()

	frameMu.Lock()
	loadedFrames := totalFramesProcessed
	frameMu.Unlock()

	if loadedUploads != 42 {
		t.Errorf("Expected restored totalSuccessUploads 42, got %d", loadedUploads)
	}
	if !loadedTime.Equal(testTime) {
		t.Errorf("Expected restored lastSuccessUpload %v, got %v", testTime, loadedTime)
	}
	if loadedFrames != 150 {
		t.Errorf("Expected restored totalFramesProcessed 150, got %d", loadedFrames)
	}
}

func TestDiskSpooler_AtomicCountPerformance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bridge-spool-perf-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spooler := NewDiskSpooler(tmpDir)
	if count := spooler.Count(); count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	for i := 0; i < 10; i++ {
		_ = spooler.Spool(SolarRecord{
			Timestamp: "2026-08-25T12:00:00Z",
			Site:      "Dorset",
		})
	}

	if count := spooler.Count(); count != 10 {
		t.Errorf("Expected atomic count 10, got %d", count)
	}
}

func TestDiskSpooler_DrainBatch(t *testing.T) {
	tmpDir := t.TempDir()
	spooler := NewDiskSpooler(tmpDir)

	for i := 0; i < 25; i++ {
		_ = spooler.Spool(SolarRecord{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Site:      "Test Site",
			Telemetry: Telemetry{PVPowerW: 100 + i},
		})
	}

	if count := spooler.Count(); count != 25 {
		t.Fatalf("Expected spool count 25, got %d", count)
	}

	var totalBatches int
	var totalRecords int
	drained, err := spooler.DrainBatch(func(batch []SolarRecord) error {
		totalBatches++
		totalRecords += len(batch)
		if len(batch) > 10 {
			t.Errorf("Batch chunk exceeded max batch size 10: got %d", len(batch))
		}
		return nil
	}, 10)

	if err != nil {
		t.Fatalf("DrainBatch returned error: %v", err)
	}
	if drained != 25 {
		t.Errorf("Expected 25 records drained, got %d", drained)
	}
	if totalBatches != 3 { // 10, 10, 5
		t.Errorf("Expected 3 batches, got %d", totalBatches)
	}
	if spooler.Count() != 0 {
		t.Errorf("Expected spool count 0 after drain, got %d", spooler.Count())
	}
}

func TestHandleHealth(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if resp["status"] != "healthy" && resp["status"] != "degraded" {
		t.Errorf("Unexpected health status: %v", resp["status"])
	}
	if _, ok := resp["uptime_seconds"]; !ok {
		t.Errorf("Expected uptime_seconds in health response")
	}
}

func TestHandleReload_Auth(t *testing.T) {
	bridgeToken = "secure_reload_token_999"
	defer func() { bridgeToken = "" }()

	// 1. Unauthorized reload
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/reload", nil)
	w1 := httptest.NewRecorder()
	handleReload(w1, req1)
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthenticated reload, got %d", w1.Code)
	}

	// 2. Authorized reload
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/reload", nil)
	req2.Header.Set("Authorization", "Bearer secure_reload_token_999")
	w2 := httptest.NewRecorder()
	handleReload(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for authorized reload, got %d", w2.Code)
	}
}

func TestUploadBatchRecords(t *testing.T) {
	var receivedBatches int
	var receivedAuthHeader string
	var receivedKeyHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		receivedKeyHeader = r.Header.Get("X-API-Key")

		var body map[string][]SolarRecord
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			receivedBatches += len(body["batch"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	origEndpoint := cloudEndpoint
	origToken := cloudToken
	cloudEndpoint = ts.URL
	cloudToken = "test_batch_token_777"
	defer func() {
		cloudEndpoint = origEndpoint
		cloudToken = origToken
	}()

	records := []SolarRecord{
		{Timestamp: "2026-08-25T12:00:00Z", Site: "Site A"},
		{Timestamp: "2026-08-25T12:01:00Z", Site: "Site A"},
	}

	if err := uploadBatchRecords(records); err != nil {
		t.Fatalf("uploadBatchRecords failed: %v", err)
	}

	if receivedBatches != 2 {
		t.Errorf("Expected 2 received records, got %d", receivedBatches)
	}
	if receivedKeyHeader != "test_batch_token_777" {
		t.Errorf("Expected X-API-Key 'test_batch_token_777', got %q", receivedKeyHeader)
	}
	if !strings.HasPrefix(receivedAuthHeader, "Bearer ") {
		t.Errorf("Expected Authorization starting with 'Bearer ', got %q", receivedAuthHeader)
	}
}

func TestDiagnosticLogBuffer_Bridge(t *testing.T) {
	buf := NewDiagnosticLogBuffer(5)

	buf.Log("INFO", "CONTROLLER_MODBUS", "Modbus frame received", "MODBUS_OK", nil)
	buf.Log("WARN", "BATTERY_SAFETY", "Low battery voltage warning", "ERR_LOW_VOLTAGE", map[string]interface{}{"voltage": 11.2})
	buf.Log("ERROR", "CLOUD_UPLOADER", "Upload failed: HTTP 500", "ERR_HTTP_500", nil)

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

	// Filter by level
	errorLogs := buf.GetLogs("ERROR", "", "", 10)
	if len(errorLogs) != 1 || errorLogs[0].ErrorCode != "ERR_HTTP_500" {
		t.Errorf("Expected 1 error log with code ERR_HTTP_500, got %v", errorLogs)
	}

	// Filter by subsystem
	modbusLogs := buf.GetLogs("", "CONTROLLER_MODBUS", "", 10)
	if len(modbusLogs) != 1 || modbusLogs[0].Subsystem != "CONTROLLER_MODBUS" {
		t.Errorf("Expected 1 modbus log, got %v", modbusLogs)
	}

	// Search filter
	searchLogs := buf.GetLogs("", "", "voltage", 10)
	if len(searchLogs) != 1 || searchLogs[0].ErrorCode != "ERR_LOW_VOLTAGE" {
		t.Errorf("Expected 1 search matched log, got %v", searchLogs)
	}

	// Ring buffer rollover
	for i := 0; i < 10; i++ {
		buf.Log("DEBUG", "TEST", fmt.Sprintf("Message %d", i), "", nil)
	}
	allLogs := buf.GetLogs("", "", "", 50)
	if len(allLogs) > 5 {
		t.Errorf("Expected buffer size capped at 5, got %d", len(allLogs))
	}
}

func TestHandleLogs_Bridge(t *testing.T) {
	bridgeLogger.Log("ERROR", "CONTROLLER_MODBUS", "CRC failure test", "ERR_TEST_CRC", nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/logs?level=ERROR&search=CRC", nil)
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
		t.Errorf("Expected status ok and non-zero count, got %+v", resp)
	}
}

func TestHandleDiagnostics_Bridge(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/diagnostics", nil)
	w := httptest.NewRecorder()
	handleDiagnostics(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode diagnostics: %v", err)
	}
	if resp["service"] != "solaria-bridge" {
		t.Errorf("Expected service 'solaria-bridge', got %v", resp["service"])
	}
	if resp["health"] == nil || resp["runtime"] == nil {
		t.Errorf("Expected health and runtime keys in diagnostics, got %+v", resp)
	}
}

func buildMockRTUFrame(pvWatts int, pvV float64, battV float64, battA float64, soc int, ctrlTemp int, battTemp int) []byte {
	raw := make([]byte, 73)
	raw[0] = 0xFF
	raw[1] = 0x03
	raw[2] = 0x44 // 68 bytes payload

	data := raw[3:71]

	// 0x0100: SOC
	binary.BigEndian.PutUint16(data[0:2], uint16(soc))
	// 0x0101: Battery Voltage (0.1V)
	binary.BigEndian.PutUint16(data[2:4], uint16(battV*10))
	// 0x0102: Charging Current (0.01A)
	if battA > 0 {
		binary.BigEndian.PutUint16(data[4:6], uint16(battA*100))
	} else {
		binary.BigEndian.PutUint16(data[4:6], 0)
	}
	// 0x0103: Controller Temp & Battery Temp
	data[6] = byte(int8(ctrlTemp))
	data[7] = byte(int8(battTemp))

	// 0x0104: Load Voltage (0.1V)
	binary.BigEndian.PutUint16(data[8:10], uint16(battV*10))
	// 0x0105: Load Current (0.01A)
	loadW := 15
	if battA < 0 {
		loadW = int(-battA * battV)
	}
	loadA := float64(loadW) / battV
	binary.BigEndian.PutUint16(data[10:12], uint16(loadA*100))
	// 0x0106: Load Power
	binary.BigEndian.PutUint16(data[12:14], uint16(loadW))

	// 0x0107: PV Voltage (0.1V)
	binary.BigEndian.PutUint16(data[14:16], uint16(pvV*10))
	// 0x0108: PV Current (0.01A)
	var pvA float64
	if pvV > 0 {
		pvA = float64(pvWatts) / pvV
	}
	binary.BigEndian.PutUint16(data[16:18], uint16(pvA*100))
	// 0x0109: PV Power
	binary.BigEndian.PutUint16(data[18:20], uint16(pvWatts))

	// 0x010B: Daily Min Battery Voltage (12.8V)
	binary.BigEndian.PutUint16(data[22:24], 128)
	// 0x010C: Daily Max Battery Voltage (14.2V)
	binary.BigEndian.PutUint16(data[24:26], 142)
	// 0x010D: Daily Max Charging Current (20.0A)
	binary.BigEndian.PutUint16(data[26:28], 2000)
	// 0x010F: Daily Max PV Power (385W)
	binary.BigEndian.PutUint16(data[30:32], 385)
	// 0x0113: Daily Generated Wh (1450 Wh)
	binary.BigEndian.PutUint16(data[38:40], 1450)
	// 0x0114: Daily Consumed Wh (380 Wh)
	binary.BigEndian.PutUint16(data[40:42], 380)
	// 0x0115: Operating Days (128)
	binary.BigEndian.PutUint16(data[42:44], 128)
	// 0x011C: Total Generated kWh (412 kWh)
	binary.BigEndian.PutUint32(data[56:60], 412)

	// 0x0120: Charging State
	chgCode := byte(0x02) // MPPT
	if pvWatts < 5 {
		chgCode = 0x00 // Deactivated
	} else if battV >= 14.1 {
		chgCode = 0x04 // Boost/Absorption
	}
	data[65] = chgCode
	// 0x0121: Fault Register (0x0000 = No faults)
	data[66] = 0x00
	data[67] = 0x00

	// Modbus CRC-16
	crcLow, crcHigh := calcCRC16(raw[:71])
	raw[71] = crcLow
	raw[72] = crcHigh

	return raw
}

func TestSolarRecord_ProcessingAndOutageFields(t *testing.T) {
	frame := buildMockRTUFrame(300, 36.5, 13.5, 20.0, 80, 25, 20)
	processFrame(frame)

	lastTelemetryMu.RLock()
	telem := lastSeenTelem
	lastTelemetryMu.RUnlock()

	if telem.PVPowerW != 300 {
		t.Errorf("Expected PVPowerW 300, got %d", telem.PVPowerW)
	}
}
