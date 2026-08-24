package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Telemetry represents the full 34 registers and diagnostic bitfields from a Renogy Rover/Wanderer controller
type Telemetry struct {
	PVPowerW        int     `json:"pv_power_w"`
	PVVoltageV      float64 `json:"pv_voltage_v"`
	PVCurrentA      float64 `json:"pv_current_a"`
	BatterySOCPct   int     `json:"battery_soc_pct"`
	BatteryVoltageV float64 `json:"battery_voltage_v"`
	BatteryCurrentA float64 `json:"battery_current_a"`
	ControllerTempC int     `json:"controller_temp_c"`
	BatteryTempC    int     `json:"battery_temp_c"`
	LoadPowerW      int     `json:"load_power_w"`
	LoadVoltageV    float64 `json:"load_voltage_v"`
	LoadCurrentA    float64 `json:"load_current_a"`
	LoadStatus      bool    `json:"load_status"`
	ChargingState   string  `json:"charging_state"`

	// Daily Min/Max & Amp-Hour Counters
	DailyMinBatteryVoltageV     float64 `json:"daily_min_battery_voltage_v"`
	DailyMaxBatteryVoltageV     float64 `json:"daily_max_battery_voltage_v"`
	DailyMaxChargingCurrentA    float64 `json:"daily_max_charging_current_a"`
	DailyMaxDischargingCurrentA float64 `json:"daily_max_discharging_current_a"`
	DailyMaxPVWatts             int     `json:"daily_max_pv_w"`
	DailyMaxLoadWatts           int     `json:"daily_max_load_w"`
	DailyChargingAh             int     `json:"daily_charging_ah"`
	DailyDischargingAh          int     `json:"daily_discharging_ah"`
	DailyGeneratedWh            int     `json:"daily_generated_wh"`
	DailyConsumedWh             int     `json:"daily_consumed_wh"`

	// Lifetime Statistics & Health Counters
	OperatingDays             int `json:"operating_days"`
	TotalBatteryOverDischarge int `json:"total_battery_overdischarge_count"`
	TotalBatteryFullCharge    int `json:"total_battery_fullcharge_count"`
	TotalChargingAh           int `json:"total_charging_ah"`
	TotalDischargingAh        int `json:"total_discharging_ah"`
	TotalGeneratedKWh         int `json:"total_generated_kwh"`
	TotalConsumedKWh          int `json:"total_consumed_kwh"`

	// Faults & Diagnostics
	FaultBits  int    `json:"fault_bits"`
	FaultFlags string `json:"fault_flags"`
}

// CalcCrc16 computes the Modbus RTU 16-bit CRC with polynomial 0xA001
func CalcCrc16(data []byte) []byte {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return []byte{byte(crc & 0xFF), byte((crc >> 8) & 0xFF)}
}

// BuildReadRealtimeQuery creates the 8-byte Modbus RTU frame for registers 0x0100 - 0x0122
func BuildReadRealtimeQuery() []byte {
	raw := []byte{0xFF, 0x03, 0x01, 0x00, 0x00, 0x22}
	crc := CalcCrc16(raw)
	return append(raw, crc...)
}

// DecodeModbusTelemetry decodes the complete 73-byte Modbus frame
func DecodeModbusTelemetry(raw []byte) (*Telemetry, error) {
	if len(raw) < 35 || raw[1] != 0x03 {
		return nil, fmt.Errorf("invalid Modbus response header or length: %d bytes", len(raw))
	}

	byteCount := int(raw[2])
	if len(raw) < 3+byteCount+2 {
		return nil, fmt.Errorf("truncated Modbus frame: expected %d bytes, got %d", 3+byteCount+2, len(raw))
	}

	data := raw[3 : 3+byteCount]

	battSOC := int(binary.BigEndian.Uint16(data[0:2]))
	battVolts := float64(binary.BigEndian.Uint16(data[2:4])) * 0.1
	battAmps := float64(binary.BigEndian.Uint16(data[4:6])) * 0.01

	ctrlTemp := int(int8(data[6]))
	battTemp := int(int8(data[7]))

	loadVolts := float64(binary.BigEndian.Uint16(data[8:10])) * 0.1
	loadAmps := float64(binary.BigEndian.Uint16(data[10:12])) * 0.01
	loadPower := int(binary.BigEndian.Uint16(data[12:14]))

	pvVolts := float64(binary.BigEndian.Uint16(data[14:16])) * 0.1
	pvAmps := float64(binary.BigEndian.Uint16(data[16:18])) * 0.01
	pvPower := int(binary.BigEndian.Uint16(data[18:20]))

	dailyMinBattV := battVolts
	dailyMaxBattV := battVolts
	dailyMaxChgCurr := 0.0
	dailyMaxDischgCurr := 0.0
	dailyMaxPV := pvPower
	dailyMaxLoadW := loadPower
	dailyChgAh := 0
	dailyDischgAh := 0
	dailyYieldWh := 0
	dailyConsumedWh := 0
	operatingDays := 0
	totalOverdischg := 0
	totalFullchg := 0
	totalChgAh := 0
	totalDischgAh := 0
	totalYieldKWh := 0
	totalConsumedKWh := 0

	if len(data) >= 24 {
		dailyMinBattV = float64(binary.BigEndian.Uint16(data[22:24])) * 0.1
	}
	if len(data) >= 26 {
		dailyMaxBattV = float64(binary.BigEndian.Uint16(data[24:26])) * 0.1
	}
	if len(data) >= 28 {
		dailyMaxChgCurr = float64(binary.BigEndian.Uint16(data[26:28])) * 0.01
	}
	if len(data) >= 30 {
		dailyMaxDischgCurr = float64(binary.BigEndian.Uint16(data[28:30])) * 0.01
	}
	if len(data) >= 32 {
		dailyMaxPV = int(binary.BigEndian.Uint16(data[30:32]))
	}
	if len(data) >= 34 {
		dailyMaxLoadW = int(binary.BigEndian.Uint16(data[32:34]))
	}
	if len(data) >= 36 {
		dailyChgAh = int(binary.BigEndian.Uint16(data[34:36]))
	}
	if len(data) >= 38 {
		dailyDischgAh = int(binary.BigEndian.Uint16(data[36:38]))
	}
	if len(data) >= 40 {
		dailyYieldWh = int(binary.BigEndian.Uint16(data[38:40]))
	}
	if len(data) >= 42 {
		dailyConsumedWh = int(binary.BigEndian.Uint16(data[40:42]))
	}
	if len(data) >= 44 {
		operatingDays = int(binary.BigEndian.Uint16(data[42:44]))
	}
	if len(data) >= 46 {
		totalOverdischg = int(binary.BigEndian.Uint16(data[44:46]))
	}
	if len(data) >= 48 {
		totalFullchg = int(binary.BigEndian.Uint16(data[46:48]))
	}
	if len(data) >= 52 {
		totalChgAh = int(binary.BigEndian.Uint32(data[48:52]))
	}
	if len(data) >= 56 {
		totalDischgAh = int(binary.BigEndian.Uint32(data[52:56]))
	}
	if len(data) >= 60 {
		totalYieldKWh = int(binary.BigEndian.Uint32(data[56:60]))
	}
	if len(data) >= 64 {
		totalConsumedKWh = int(binary.BigEndian.Uint32(data[60:64]))
	}

	loadStatus := false
	if len(data) >= 65 {
		loadStatus = (data[64] & 0x80) != 0
	}

	stateCode := byte(0)
	if len(data) >= 66 {
		stateCode = data[65]
	} else if len(data) > 33 {
		stateCode = data[33]
	}

	stateMap := map[byte]string{
		0x00: "Deactivated",
		0x01: "Activated",
		0x02: "MPPT Charging",
		0x03: "Equalizing Charging",
		0x04: "Boost Charging",
		0x05: "Floating Charging",
		0x06: "Current Limiting",
	}
	chargingState, ok := stateMap[stateCode]
	if !ok {
		chargingState = fmt.Sprintf("State 0x%02X", stateCode)
	}

	faultBits := 0
	if len(data) >= 68 {
		faultBits = int(binary.BigEndian.Uint16(data[66:68]))
	}

	faultMap := map[int]string{
		0:  "Battery Over-Discharge",
		1:  "Battery Over-Voltage",
		2:  "Battery Under-Voltage Warning",
		3:  "Load Short-Circuit",
		4:  "Load Over-Current",
		5:  "Controller Over-Temp",
		6:  "Battery Over-Temp",
		7:  "PV Array Over-Power",
		8:  "PV Array Short-Circuit",
		9:  "PV Array Over-Voltage",
		10: "PV Counter-Current",
		11: "PV Reverse Polarity",
		12: "Battery Reverse Polarity",
		13: "Battery Probe Disconnected",
	}

	var activeFaults []string
	for bitIdx, faultName := range faultMap {
		if (faultBits>>bitIdx)&1 != 0 {
			activeFaults = append(activeFaults, faultName)
		}
	}
	faultFlags := "NORMAL"
	if len(activeFaults) > 0 {
		faultFlags = strings.Join(activeFaults, ", ")
	}

	return &Telemetry{
		PVPowerW:                    pvPower,
		PVVoltageV:                  pvVolts,
		PVCurrentA:                  pvAmps,
		BatterySOCPct:               battSOC,
		BatteryVoltageV:             battVolts,
		BatteryCurrentA:             battAmps,
		ControllerTempC:             ctrlTemp,
		BatteryTempC:                battTemp,
		LoadPowerW:                  loadPower,
		LoadVoltageV:                loadVolts,
		LoadCurrentA:                loadAmps,
		LoadStatus:                  loadStatus,
		ChargingState:               chargingState,
		DailyMinBatteryVoltageV:     dailyMinBattV,
		DailyMaxBatteryVoltageV:     dailyMaxBattV,
		DailyMaxChargingCurrentA:    dailyMaxChgCurr,
		DailyMaxDischargingCurrentA: dailyMaxDischgCurr,
		DailyMaxPVWatts:             dailyMaxPV,
		DailyMaxLoadWatts:           dailyMaxLoadW,
		DailyChargingAh:             dailyChgAh,
		DailyDischargingAh:          dailyDischgAh,
		DailyGeneratedWh:            dailyYieldWh,
		DailyConsumedWh:             dailyConsumedWh,
		OperatingDays:               operatingDays,
		TotalBatteryOverDischarge:   totalOverdischg,
		TotalBatteryFullCharge:      totalFullchg,
		TotalChargingAh:             totalChgAh,
		TotalDischargingAh:          totalDischgAh,
		TotalGeneratedKWh:           totalYieldKWh,
		TotalConsumedKWh:            totalConsumedKWh,
		FaultBits:                   faultBits,
		FaultFlags:                  faultFlags,
	}, nil
}
