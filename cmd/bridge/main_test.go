package main

import (
	"net/http"
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
		{"http://192.168.1.100:8080", true},
		{"http://10.0.0.45:8080", true},
		{"https://solaria-dashboard-952659886764.us-central1.run.app", true},
		{"chrome-extension://solaria-bridge-helper", true},
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

	// 1. Valid Query Param
	req1, _ := http.NewRequest("GET", "ws://localhost:8765?token=test_secret_bridge_token_123", nil)
	if !verifyBridgeAuth(req1, "") {
		t.Errorf("Expected auth with query token to pass")
	}

	// 2. Valid Authorization Header
	req2, _ := http.NewRequest("GET", "ws://localhost:8765", nil)
	req2.Header.Set("Authorization", "Bearer test_secret_bridge_token_123")
	if !verifyBridgeAuth(req2, "") {
		t.Errorf("Expected auth with Bearer header to pass")
	}

	// 3. Valid Payload Token
	if !verifyBridgeAuth(nil, "test_secret_bridge_token_123") {
		t.Errorf("Expected auth with payload token to pass")
	}

	// 4. Invalid Token
	reqInvalid, _ := http.NewRequest("GET", "ws://localhost:8765?token=wrong_token", nil)
	if verifyBridgeAuth(reqInvalid, "wrong_token") {
		t.Errorf("Expected invalid token to be rejected")
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
		t.Errorf("Expected SubZeroInhibitWarning to be true at -2C")
	}

	// Frame with String Imbalance: PV Volts = 18.0V (single string/bypass) while power is 120W
	frameImbalance := buildMockRTUFrame(120, 18.0, 13.5, 8.5, 80, 25, 20)
	telemImbalance, _ := decodeTelemetry(frameImbalance)
	if telemImbalance.StringHealthStatus != "POTENTIAL_SERIES_DIODE_BYPASS_OR_SINGLE_PANEL_FAULT" {
		t.Errorf("Expected POTENTIAL_SERIES_DIODE_BYPASS_OR_SINGLE_PANEL_FAULT, got %s", telemImbalance.StringHealthStatus)
	}
}


