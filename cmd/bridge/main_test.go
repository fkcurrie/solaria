package main

import (
	"testing"
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

