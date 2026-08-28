package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

type SolarRecord struct {
	Timestamp           time.Time `json:"timestamp"`
	PVPowerW            int       `json:"pv_power_w"`
	PVVoltageV          float64   `json:"pv_voltage_v"`
	PVCurrentA          float64   `json:"pv_current_a"`
	BatterySOCPct       int       `json:"battery_soc_pct"`
	BatteryVoltageV     float64   `json:"battery_voltage_v"`
	BatteryCurrentA     float64   `json:"battery_current_a"`
	ControllerTempC     int       `json:"controller_temp_c"`
	BatteryTempC        int       `json:"battery_temp_c"`
	LoadPowerW          int       `json:"load_power_w"`
	LoadVoltageV        float64   `json:"load_voltage_v"`
	LoadCurrentA        float64   `json:"load_current_a"`
	TemperatureC        float64   `json:"temperature_c"`
	CloudCoverPct       int       `json:"cloud_cover_pct"`
	DirectRadiationWM2  float64   `json:"direct_radiation_w_m2"`
	DiffuseRadiationWM2 float64   `json:"diffuse_radiation_w_m2"`
	SunCondition        string    `json:"sun_condition"`
	ChargeState         string    `json:"charge_state"`
	FaultText           string    `json:"fault_text"`
}

func main() {
	outDir := "testdata"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("Error creating dir: %v\n", err)
		return
	}

	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		loc = time.UTC
	}

	// Generate 1 full 24-hour day (August 24, 2026) at 1-minute intervals (1440 points)
	startTime := time.Date(2026, 8, 24, 0, 0, 0, 0, loc)
	var records []SolarRecord

	// System constants (400W 2S2P, 12V LiFePO4, Rover 20A)
	batterySOC := 62.0

	for m := 0; m < 1440; m++ {
		t := startTime.Add(time.Duration(m) * time.Minute)
		hourFloat := float64(t.Hour()) + float64(t.Minute())/60.0

		var pvWatts int
		var pvV, pvA float64
		var directRad, diffuseRad float64
		var cloudCover int
		var tempC float64
		var sunCond string
		var chargeState string

		// Diurnal ambient temperature curve (14°C at dawn to 26°C at 15:00)
		tempC = math.Round((20.0-6.0*math.Cos((hourFloat-5.0)*math.Pi/12.0))*10) / 10

		// Solar generation physics
		// Sunrise ~ 06:15 (6.25), Solar Noon ~ 13:15 (13.25), Sunset ~ 20:15 (20.25)
		if hourFloat > 6.25 && hourFloat < 20.25 {
			// Sun elevation factor (sine bell curve)
			solarAngle := (hourFloat - 6.25) / (20.25 - 6.25) * math.Pi
			sunIntensity := math.Sin(solarAngle)

			// Clouds simulation: passing scattered cumulus between 14:00 and 15:30
			if hourFloat >= 14.0 && hourFloat <= 15.5 {
				cloudCover = 65
				directRad = math.Max(0, sunIntensity*820.0*0.35)
				diffuseRad = 220.0
				sunCond = "PARTIAL_SUN_OR_SHADE"
			} else {
				cloudCover = 12
				directRad = math.Max(0, sunIntensity*880.0)
				diffuseRad = 95.0
				sunCond = "FULL_SUN"
			}

			// 400W 2S2P Array Peak ~ 380W under clear sky
			rawPvW := float64(385.0) * (directRad/900.0 + (diffuseRad/400.0)*0.3)
			if rawPvW < 5.0 {
				rawPvW = 5.0
			}
			pvWatts = int(math.Round(rawPvW))

			// String Vmp ~ 36.5V - 39.5V
			pvV = math.Round((36.5+2.5*(1.0-sunIntensity))*10) / 10
			if pvV > 44.0 {
				pvV = 44.0
			}
			pvA = math.Round((float64(pvWatts)/pvV)*100) / 100

			// LiFePO4 Charging Profile
			if batterySOC < 98.0 {
				batterySOC += (float64(pvWatts) / 1200.0) * (1.0 / 60.0) * 100.0
				if batterySOC > 100.0 {
					batterySOC = 100.0
				}
				chargeState = "MPPT"
			} else {
				// LiFePO4 Absorption Saturation
				chargeState = "BOOST_ABSORPTION"
				sunCond = "ABSORPTION_FLOAT_CLIPPED"
				// Controller throttles PV current down
				pvWatts = int(math.Min(float64(pvWatts), 45.0))
				pvA = math.Round((float64(pvWatts)/pvV)*100) / 100
				batterySOC = 100.0
			}
		} else {
			// Night
			pvWatts = 0
			pvV = 0.2
			pvA = 0.0
			directRad = 0.0
			diffuseRad = 0.0
			cloudCover = 10
			sunCond = "NIGHT"
			chargeState = "DEACTIVATED"

			// Night baseline battery discharge (standby + small cottage load)
			batterySOC -= (0.4 / 60.0)
			if batterySOC < 55.0 {
				batterySOC = 55.0
			}
		}

		// Battery Voltage derivation from SOC & charging current
		var battV float64
		if chargeState == "BOOST_ABSORPTION" {
			battV = 14.2
		} else if pvWatts > 20 {
			// Bulk charging voltage ramp (13.3V to 14.1V)
			battV = math.Round((13.25+(batterySOC/100.0)*0.85)*10) / 10
		} else {
			// Resting LiFePO4 voltage (13.15V to 13.35V)
			battV = math.Round((13.10+(batterySOC/100.0)*0.22)*10) / 10
		}

		// Battery current (charging or discharging)
		var battA float64
		loadW := 18
		if hourFloat >= 19.0 && hourFloat <= 23.0 {
			loadW = 65 // Evening cabin lights / laptop
		}
		loadV := battV
		loadA := math.Round((float64(loadW)/loadV)*100) / 100

		if pvWatts > loadW {
			battA = math.Round((float64(pvWatts-loadW)/battV)*100) / 100
		} else {
			battA = -math.Round((float64(loadW-pvWatts)/battV)*100) / 100
		}

		ctrlTemp := int(math.Round(tempC + float64(pvWatts)/400.0*8.0))
		battTemp := int(math.Round(tempC + 2.0))

		records = append(records, SolarRecord{
			Timestamp:           t.UTC(),
			PVPowerW:            pvWatts,
			PVVoltageV:          pvV,
			PVCurrentA:          pvA,
			BatterySOCPct:       int(math.Round(batterySOC)),
			BatteryVoltageV:     battV,
			BatteryCurrentA:     battA,
			ControllerTempC:     ctrlTemp,
			BatteryTempC:        battTemp,
			LoadPowerW:          loadW,
			LoadVoltageV:        loadV,
			LoadCurrentA:        loadA,
			TemperatureC:        tempC,
			CloudCoverPct:       cloudCover,
			DirectRadiationWM2:  directRad,
			DiffuseRadiationWM2: diffuseRad,
			SunCondition:        sunCond,
			ChargeState:         chargeState,
			FaultText:           "NONE",
		})
	}

	// Write JSON fixture
	jsonPath := filepath.Join(outDir, "sample_day.json")
	jsonData, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling json: %v\n", err)
		return
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		fmt.Printf("Error writing json: %v\n", err)
		return
	}
	fmt.Printf("Successfully generated %s (%d records)\n", jsonPath, len(records))

	// Write CSV fixture
	csvPath := filepath.Join(outDir, "sample_day.csv")
	csvFile, err := os.Create(csvPath)
	if err != nil {
		fmt.Printf("Error creating csv: %v\n", err)
		return
	}
	defer func() { _ = csvFile.Close() }()

	w := csv.NewWriter(csvFile)
	defer w.Flush()

	// Header
	_ = w.Write([]string{
		"timestamp", "pv_power_w", "pv_voltage_v", "pv_current_a",
		"battery_soc_pct", "battery_voltage_v", "battery_current_a",
		"controller_temp_c", "battery_temp_c", "load_power_w", "load_voltage_v", "load_current_a",
		"temperature_c", "cloud_cover_pct", "direct_radiation_w_m2", "diffuse_radiation_w_m2",
		"sun_condition", "charge_state", "fault_text",
	})

	for _, r := range records {
		_ = w.Write([]string{
			r.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%d", r.PVPowerW),
			fmt.Sprintf("%.1f", r.PVVoltageV),
			fmt.Sprintf("%.2f", r.PVCurrentA),
			fmt.Sprintf("%d", r.BatterySOCPct),
			fmt.Sprintf("%.1f", r.BatteryVoltageV),
			fmt.Sprintf("%.2f", r.BatteryCurrentA),
			fmt.Sprintf("%d", r.ControllerTempC),
			fmt.Sprintf("%d", r.BatteryTempC),
			fmt.Sprintf("%d", r.LoadPowerW),
			fmt.Sprintf("%.1f", r.LoadVoltageV),
			fmt.Sprintf("%.2f", r.LoadCurrentA),
			fmt.Sprintf("%.1f", r.TemperatureC),
			fmt.Sprintf("%d", r.CloudCoverPct),
			fmt.Sprintf("%.1f", r.DirectRadiationWM2),
			fmt.Sprintf("%.1f", r.DiffuseRadiationWM2),
			r.SunCondition,
			r.ChargeState,
			r.FaultText,
		})
	}
	fmt.Printf("Successfully generated %s\n", csvPath)
}
