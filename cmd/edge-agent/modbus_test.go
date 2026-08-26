package main

import (
	"testing"
)

func TestCalcCrc16(t *testing.T) {
	// Standard Modbus RTU telemetry query frame: [0xFF, 0x03, 0x01, 0x00, 0x00, 0x22]
	queryFrame := []byte{0xFF, 0x03, 0x01, 0x00, 0x00, 0x22}
	crc := CalcCrc16(queryFrame)
	if len(crc) != 2 {
		t.Fatalf("Expected 2-byte CRC16 slice, got %d", len(crc))
	}

	// Verify append CRC and check
	frameWithCRC := append(queryFrame, crc...)
	if len(frameWithCRC) != 8 {
		t.Fatalf("Expected 8-byte frame with CRC, got %d", len(frameWithCRC))
	}
}

func TestDecodeModbusTelemetry_SubZeroInhibit(t *testing.T) {
	// Construct simulated Modbus response frame (73 bytes):
	// Address 0xFF, Function 0x03, ByteCount 0x44 (68 bytes) + 68 data bytes + 2 CRC bytes
	buf := make([]byte, 73)
	buf[0] = 0xFF
	buf[1] = 0x03
	buf[2] = 0x44

	// data[0:2] = buf[3:5]: Battery SOC = 85%
	buf[3] = 0x00
	buf[4] = 85

	// data[2:4] = buf[5:7]: Battery Voltage = 13.2V (132 -> 0x0084)
	buf[5] = 0x00
	buf[6] = 0x84

	// data[4:6] = buf[7:9]: Battery Current = 5.00A (500 -> 0x01F4)
	buf[7] = 0x01
	buf[8] = 0xF4

	// data[6] = buf[9]: Controller Temp = 25C
	buf[9] = 25
	// data[7] = buf[10]: Battery Temp = -5C (signed int8: 251 - 256 = -5C)
	buf[10] = 251

	// data[14:16] = buf[17:19]: PV Voltage = 36.2V (362 -> 0x016A)
	buf[17] = 0x01
	buf[18] = 0x6A

	// data[16:18] = buf[19:21]: PV Current = 2.00A (200 -> 0x00C8)
	buf[19] = 0x00
	buf[20] = 0xC8

	// data[18:20] = buf[21:23]: PV Power = 72W (0x0048)
	buf[21] = 0x00
	buf[22] = 0x48

	telem, err := DecodeModbusTelemetry(buf)
	if err != nil {
		t.Fatalf("Unexpected error decoding telemetry: %v", err)
	}

	if telem.PVVoltageV < 36.0 || telem.PVVoltageV > 36.4 {
		t.Errorf("Expected PV Voltage ~36.2V, got %.1fV", telem.PVVoltageV)
	}

	if telem.BatteryVoltageV < 13.0 || telem.BatteryVoltageV > 13.5 {
		t.Errorf("Expected Battery Voltage ~13.2V, got %.1fV", telem.BatteryVoltageV)
	}

	if telem.BatteryTempC > 0 {
		t.Errorf("Expected sub-zero battery temp, got %dC", telem.BatteryTempC)
	}

	if !telem.SubZeroInhibitWarning {
		t.Errorf("Expected SubZeroInhibitWarning to be true for sub-zero temperature, got false")
	}

	if telem.StringHealthStatus != "NOMINAL_2S2P_ACTIVE" {
		t.Errorf("Expected NOMINAL_2S2P_ACTIVE for 36.2V PV, got %s", telem.StringHealthStatus)
	}
}

func TestDecodeModbusTelemetry_HalfStringDiodeFault(t *testing.T) {
	buf := make([]byte, 73)
	buf[0] = 0xFF
	buf[1] = 0x03
	buf[2] = 0x44

	// Battery Voltage: 12.5V (125) -> data[2:4] = buf[5:7]
	buf[5] = 0x00
	buf[6] = 0x7D

	// PV Voltage: 18.0V (180 -> 0x00B4) -> data[14:16] = buf[17:19]
	buf[17] = 0x00
	buf[18] = 0xB4

	// PV Power: 50W -> data[18:20] = buf[21:23]
	buf[21] = 0x00
	buf[22] = 0x32

	telem, err := DecodeModbusTelemetry(buf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if telem.StringHealthStatus != "POTENTIAL_SERIES_DIODE_BYPASS_OR_SINGLE_PANEL_FAULT" {
		t.Errorf("Expected bypass diode fault warning for 18.0V, got %s", telem.StringHealthStatus)
	}
}

func TestDecodeModbusTelemetry_UnconnectedRTSProbe(t *testing.T) {
	buf := make([]byte, 73)
	buf[0] = 0xFF
	buf[1] = 0x03
	buf[2] = 0x44

	// Battery Voltage: 13.2V, Current: 5A
	buf[5] = 0x00
	buf[6] = 0x84
	buf[7] = 0x01
	buf[8] = 0xF4

	// Controller Temp = 25C (warm), Battery Temp = 0C (no probe)
	buf[9] = 25
	buf[10] = 0

	// PV Voltage: 36.2V, Power: 72W
	buf[17] = 0x01
	buf[18] = 0x6A
	buf[21] = 0x00
	buf[22] = 0x48

	// Fault register 0x0121 at data[66:68] (raw[69:71]): bit 13 set (0x2000 = probe disconnected)
	buf[69] = 0x20

	telem, err := DecodeModbusTelemetry(buf)
	if err != nil {
		t.Fatalf("Unexpected error decoding telemetry: %v", err)
	}

	if telem.SubZeroInhibitWarning {
		t.Errorf("Expected SubZeroInhibitWarning to be false for unconnected RTS probe (effective temp = 20C)")
	}
}

func TestDecodeModbusTelemetry_FreezingZeroCutoff(t *testing.T) {
	buf := make([]byte, 73)
	buf[0] = 0xFF
	buf[1] = 0x03
	buf[2] = 0x44

	// Battery Voltage: 13.2V, Current: 5A
	buf[5] = 0x00
	buf[6] = 0x84
	buf[7] = 0x01
	buf[8] = 0xF4

	// Genuine 0°C battery with connected probe (no fault bits)
	buf[9] = 5
	buf[10] = 0 // 0°C freezing

	// PV Voltage: 36.2V, Power: 72W
	buf[17] = 0x01
	buf[18] = 0x6A
	buf[21] = 0x00
	buf[22] = 0x48

	telem, err := DecodeModbusTelemetry(buf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !telem.SubZeroInhibitWarning {
		t.Errorf("Expected SubZeroInhibitWarning to be true at 0°C freezing point")
	}
}
