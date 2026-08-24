package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Global Configuration & Defaults
var (
	httpPort        = 8080
	wsPort          = 8765
	siteLat         = 45.186
	siteLon         = -78.863
	siteName        = "1296 Wren Lake Drive, Dorset, ON"
	arrayRatedWatts = 400.0
	cloudEndpoint   = "https://solaria-dashboard-952659886764.us-central1.run.app/api/v1/telemetry"
	cloudToken      = "solaria_cottage_secret_token_2026"
	storageMode     = "both" // "local", "bigquery" / "cloud", "both"

	mu             sync.Mutex
	rxBuffer       []byte
	cachedWeather  WeatherMetrics
	lastWxFetch    time.Time
	idTokenCache   string
	idTokenExpires time.Time

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all browser origins
		},
	}
)

type WeatherMetrics struct {
	TemperatureC        *float64 `json:"temperature_c"`
	CloudCoverPct       *int     `json:"cloud_cover_pct"`
	DirectRadiationWM2  float64  `json:"direct_radiation_w_m2"`
	DiffuseRadiationWM2 float64  `json:"diffuse_radiation_w_m2"`
	IsDay               bool     `json:"is_day"`
}

type Telemetry struct {
	PVPowerW                    int     `json:"pv_power_w"`
	PVVoltageV                  float64 `json:"pv_voltage_v"`
	PVCurrentA                  float64 `json:"pv_current_a"`
	BatterySOCPct               int     `json:"battery_soc_pct"`
	BatteryVoltageV             float64 `json:"battery_voltage_v"`
	BatteryCurrentA             float64 `json:"battery_current_a"`
	ControllerTempC             int     `json:"controller_temp_c"`
	BatteryTempC                int     `json:"battery_temp_c"`
	LoadPowerW                  int     `json:"load_power_w"`
	LoadVoltageV                float64 `json:"load_voltage_v"`
	LoadCurrentA                float64 `json:"load_current_a"`
	LoadStatus                  bool    `json:"load_status"`
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
	OperatingDays               int     `json:"operating_days"`
	TotalBatteryOverDischarge   int     `json:"total_battery_overdischarge_count"`
	TotalBatteryFullCharge      int     `json:"total_battery_fullcharge_count"`
	TotalChargingAh             int     `json:"total_charging_ah"`
	TotalDischargingAh          int     `json:"total_discharging_ah"`
	TotalGeneratedKWh           int     `json:"total_generated_kwh"`
	TotalConsumedKWh            int     `json:"total_consumed_kwh"`
	ChargingState               string  `json:"charging_state"`
	FaultBits                   int     `json:"fault_bits"`
	FaultFlags                  string  `json:"fault_flags"`
	ArrayCapacityW              int     `json:"array_capacity_w"`
	ArrayTopology               string  `json:"array_topology"`
	ArrayUtilizationPct         float64 `json:"array_utilization_pct"`
	PerformanceRatioPct         float64 `json:"performance_ratio_pct"`
}

type SolarRecord struct {
	Timestamp         string             `json:"timestamp"`
	Site              string             `json:"site"`
	Location          map[string]float64 `json:"location"`
	Telemetry         Telemetry          `json:"telemetry"`
	Weather           WeatherMetrics     `json:"weather"`
	SunClassification string             `json:"sun_classification"`
}

func loadEnv() {
	envPath := ".env"
	data, err := os.ReadFile(envPath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}

	if val := os.Getenv("SITE_NAME"); val != "" {
		siteName = val
	}
	if val := os.Getenv("SITE_LATITUDE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			siteLat = f
		}
	}
	if val := os.Getenv("SITE_LONGITUDE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			siteLon = f
		}
	}
	if val := os.Getenv("PANEL_RATED_WATTS"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			arrayRatedWatts = f
		}
	}
	if val := os.Getenv("SOLARIA_CLOUD_ENDPOINT"); val != "" {
		cloudEndpoint = val
	}
	if val := os.Getenv("SOLARIA_API_TOKEN"); val != "" {
		cloudToken = val
	}
	if val := os.Getenv("STORAGE_MODE"); val != "" {
		storageMode = strings.ToLower(val)
	}
}

func calcCRC16(data []byte) (byte, byte) {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 1) != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return byte(crc & 0xFF), byte((crc >> 8) & 0xFF)
}

func fetchWeather() WeatherMetrics {
	mu.Lock()
	defer mu.Unlock()

	if time.Since(lastWxFetch) < 5*time.Minute && cachedWeather.TemperatureC != nil {
		return cachedWeather
	}

	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,cloud_cover,direct_normal_irradiance,global_tilted_irradiance,diffuse_radiation,direct_radiation,is_day&timezone=auto", siteLat, siteLon)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return cachedWeather
	}
	req.Header.Set("User-Agent", "SolariaGoBridge/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return cachedWeather
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var omResp struct {
			Current struct {
				Temperature2M   *float64 `json:"temperature_2m"`
				CloudCover      *int     `json:"cloud_cover"`
				DirectRadiation *float64 `json:"direct_radiation"`
				DiffuseRad      *float64 `json:"diffuse_radiation"`
				IsDay           int      `json:"is_day"`
			} `json:"current"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&omResp); err == nil {
			var dirRad, diffRad float64
			if omResp.Current.DirectRadiation != nil {
				dirRad = *omResp.Current.DirectRadiation
			}
			if omResp.Current.DiffuseRad != nil {
				diffRad = *omResp.Current.DiffuseRad
			}
			cachedWeather = WeatherMetrics{
				TemperatureC:        omResp.Current.Temperature2M,
				CloudCoverPct:       omResp.Current.CloudCover,
				DirectRadiationWM2:  dirRad,
				DiffuseRadiationWM2: diffRad,
				IsDay:               omResp.Current.IsDay == 1,
			}
			lastWxFetch = time.Now()
		}
	}
	return cachedWeather
}

func classifySunCondition(telem Telemetry, wx WeatherMetrics) string {
	totalRad := wx.DirectRadiationWM2 + wx.DiffuseRadiationWM2
	cloudCover := 50
	if wx.CloudCoverPct != nil {
		cloudCover = *wx.CloudCoverPct
	}

	if !wx.IsDay {
		return "NIGHT"
	}
	if telem.BatterySOCPct >= 99 && (strings.Contains(telem.ChargingState, "Float") || strings.Contains(telem.ChargingState, "Boost")) {
		return "ABSORPTION_FLOAT_CLIPPED"
	}
	if cloudCover < 25 && (wx.DirectRadiationWM2 > 250 || totalRad > 300) {
		return "FULL_SUN"
	}
	if cloudCover > 70 || (wx.DiffuseRadiationWM2 > wx.DirectRadiationWM2) {
		return "DIFFUSE_OVERCAST"
	}
	if cloudCover >= 25 && cloudCover <= 70 {
		return "PARTIAL_SUN_OR_SHADE"
	}
	if totalRad < 20.0 {
		return "DAWN_LOW_LIGHT"
	}
	return "DIFFUSE_OVERCAST"
}

func decodeTelemetry(raw []byte) (Telemetry, error) {
	if len(raw) < 35 || raw[1] != 0x03 {
		return Telemetry{}, fmt.Errorf("invalid frame length: %d", len(raw))
	}
	data := raw[3 : len(raw)-2]
	if len(data) < 20 {
		return Telemetry{}, fmt.Errorf("insufficient register payload: %d", len(data))
	}

	u16 := func(offset int) int {
		if offset+2 <= len(data) {
			return int(uint16(data[offset])<<8 | uint16(data[offset+1]))
		}
		return 0
	}
	u32 := func(offset int) int {
		if offset+4 <= len(data) {
			return int(uint32(data[offset])<<24 | uint32(data[offset+1])<<16 | uint32(data[offset+2])<<8 | uint32(data[offset+3]))
		}
		return 0
	}
	s8 := func(offset int) int {
		if offset < len(data) {
			return int(int8(data[offset]))
		}
		return 0
	}

	battSOC := u16(0)
	battV := math.Round(float64(u16(2))*0.1*10) / 10
	battA := math.Round(float64(u16(4))*0.01*100) / 100

	ctrlTemp := s8(6)
	battTemp := s8(7)

	loadV := math.Round(float64(u16(8))*0.1*10) / 10
	loadA := math.Round(float64(u16(10))*0.01*100) / 100
	loadW := u16(12)

	pvV := math.Round(float64(u16(14))*0.1*10) / 10
	pvA := math.Round(float64(u16(16))*0.01*100) / 100
	pvW := u16(18)

	dailyMinBattV := battV
	if len(data) >= 24 {
		dailyMinBattV = math.Round(float64(u16(22))*0.1*10) / 10
	}
	dailyMaxBattV := battV
	if len(data) >= 26 {
		dailyMaxBattV = math.Round(float64(u16(24))*0.1*10) / 10
	}
	dailyMaxChgCurr := math.Round(float64(u16(26))*0.01*100) / 100
	dailyMaxDischgCurr := math.Round(float64(u16(28))*0.01*100) / 100
	dailyMaxPV := u16(30)
	dailyMaxLoad := u16(32)
	dailyChgAh := u16(34)
	dailyDischgAh := u16(36)
	dailyYieldWh := u16(38)
	dailyConsumedWh := u16(40)
	operatingDays := u16(42)
	totalOverDischg := u16(44)
	totalFullChg := u16(46)
	totalChgAh := u32(48)
	totalDischgAh := u32(52)
	totalYieldKWh := u32(56)
	totalConsumedKWh := u32(60)

	loadStatus := false
	if len(data) >= 65 {
		loadStatus = (data[64] & 0x80) != 0
	}

	stateCode := byte(0)
	if len(data) >= 66 {
		stateCode = data[65]
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
	chargingState := stateMap[stateCode]
	if chargingState == "" {
		chargingState = fmt.Sprintf("State 0x%02X", stateCode)
	}

	faultBits := u16(66)
	faultMap := map[int]string{
		0: "Battery Over-Discharge", 1: "Battery Over-Voltage", 2: "Battery Under-Voltage Warning",
		3: "Load Short-Circuit", 4: "Load Over-Current", 5: "Controller Over-Temp",
		6: "Battery Over-Temp", 7: "PV Array Over-Power", 8: "PV Array Short-Circuit",
		9: "PV Array Over-Voltage", 10: "PV Counter-Current", 11: "PV Reverse Polarity",
		12: "Battery Reverse Polarity", 13: "Battery Probe Disconnected",
	}
	var faults []string
	for bit, desc := range faultMap {
		if (faultBits & (1 << bit)) != 0 {
			faults = append(faults, desc)
		}
	}
	faultFlags := "NORMAL"
	if len(faults) > 0 {
		faultFlags = strings.Join(faults, ", ")
	}

	capW := int(arrayRatedWatts)
	utilPct := math.Round((float64(pvW)/float64(capW))*1000) / 10

	return Telemetry{
		PVPowerW:                    pvW,
		PVVoltageV:                  pvV,
		PVCurrentA:                  pvA,
		BatterySOCPct:               battSOC,
		BatteryVoltageV:             battV,
		BatteryCurrentA:             battA,
		ControllerTempC:             ctrlTemp,
		BatteryTempC:                battTemp,
		LoadPowerW:                  loadW,
		LoadVoltageV:                loadV,
		LoadCurrentA:                loadA,
		LoadStatus:                  loadStatus,
		DailyMinBatteryVoltageV:     dailyMinBattV,
		DailyMaxBatteryVoltageV:     dailyMaxBattV,
		DailyMaxChargingCurrentA:    dailyMaxChgCurr,
		DailyMaxDischargingCurrentA: dailyMaxDischgCurr,
		DailyMaxPVWatts:             dailyMaxPV,
		DailyMaxLoadWatts:           dailyMaxLoad,
		DailyChargingAh:             dailyChgAh,
		DailyDischargingAh:          dailyDischgAh,
		DailyGeneratedWh:            dailyYieldWh,
		DailyConsumedWh:             dailyConsumedWh,
		OperatingDays:               operatingDays,
		TotalBatteryOverDischarge:   totalOverDischg,
		TotalBatteryFullCharge:      totalFullChg,
		TotalChargingAh:             totalChgAh,
		TotalDischargingAh:          totalDischgAh,
		TotalGeneratedKWh:           totalYieldKWh,
		TotalConsumedKWh:            totalConsumedKWh,
		ChargingState:               chargingState,
		FaultBits:                   faultBits,
		FaultFlags:                  faultFlags,
		ArrayCapacityW:              capW,
		ArrayTopology:               "2S2P",
		ArrayUtilizationPct:         utilPct,
	}, nil
}

func logToCSV(telem Telemetry) {
	now := time.Now()
	logDir := "logs"
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, fmt.Sprintf("solar_telemetry_%s.csv", now.Format("2006-01-02")))

	isNew := false
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		isNew = true
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if isNew {
		_ = w.Write([]string{
			"timestamp", "pv_power_w", "pv_voltage_v", "pv_current_a",
			"battery_soc_pct", "battery_voltage_v", "battery_current_a",
			"charging_state", "controller_temp_c", "battery_temp_c",
			"load_power_w", "daily_generated_wh", "total_generated_kwh",
		})
	}

	_ = w.Write([]string{
		now.Format(time.RFC3339),
		strconv.Itoa(telem.PVPowerW),
		strconv.FormatFloat(telem.PVVoltageV, 'f', 1, 64),
		strconv.FormatFloat(telem.PVCurrentA, 'f', 2, 64),
		strconv.Itoa(telem.BatterySOCPct),
		strconv.FormatFloat(telem.BatteryVoltageV, 'f', 1, 64),
		strconv.FormatFloat(telem.BatteryCurrentA, 'f', 2, 64),
		telem.ChargingState,
		strconv.Itoa(telem.ControllerTempC),
		strconv.Itoa(telem.BatteryTempC),
		strconv.Itoa(telem.LoadPowerW),
		strconv.Itoa(telem.DailyGeneratedWh),
		strconv.Itoa(telem.TotalGeneratedKWh),
	})
}

func getIDToken() string {
	mu.Lock()
	if idTokenCache != "" && time.Now().Before(idTokenExpires) {
		token := idTokenCache
		mu.Unlock()
		return token
	}
	mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "auth", "print-identity-token")
	out, err := cmd.Output()
	if err == nil && len(bytes.TrimSpace(out)) > 0 {
		token := strings.TrimSpace(string(out))
		mu.Unlock()
		return token
	}
	return cloudToken
}

var (
	uploadMu        sync.Mutex
	lastCloudUpload time.Time
)

func uploadToCloud(record SolarRecord) {
	if storageMode == "local" || cloudEndpoint == "" {
		return
	}
	uploadMu.Lock()
	if time.Since(lastCloudUpload) < 8*time.Second {
		uploadMu.Unlock()
		return
	}
	lastCloudUpload = time.Now()
	uploadMu.Unlock()

	go func() {
		payload, err := json.Marshal(map[string]interface{}{
			"batch": []SolarRecord{record},
		})
		if err != nil {
			return
		}

		req, err := http.NewRequest("POST", cloudEndpoint, bytes.NewBuffer(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cloudToken)
		req.Header.Set("X-API-Key", cloudToken)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}()
}

var (
	frameMu          sync.Mutex
	lastFrameProcess time.Time
)

func processFrame(frame []byte) {
	if len(frame) < 5 || frame[1] != 0x03 {
		return
	}
	byteCount := frame[2]
	if byteCount < 60 && len(frame) < 65 {
		return
	}

	frameMu.Lock()
	if time.Since(lastFrameProcess) < 8*time.Second {
		frameMu.Unlock()
		return
	}
	lastFrameProcess = time.Now()
	frameMu.Unlock()

	nowStr := time.Now().Format("15:04:05.000")

	telem, err := decodeTelemetry(frame)
	if err != nil {
		return
	}

		wx := fetchWeather()
		sunState := classifySunCondition(telem, wx)

		expectedPower := (wx.DirectRadiationWM2 / 1000.0) * arrayRatedWatts
		prPct := 0.0
		if expectedPower > 5.0 {
			prPct = math.Round((float64(telem.PVPowerW)/expectedPower)*1000) / 10
		}
		telem.PerformanceRatioPct = prPct

		record := SolarRecord{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Site:      siteName,
			Location: map[string]float64{
				"latitude":  siteLat,
				"longitude": siteLon,
			},
			Telemetry:         telem,
			Weather:           wx,
			SunClassification: sunState,
		}

		if storageMode != "cloud" && storageMode != "bigquery" {
			logToCSV(telem)
		}
		if storageMode != "local" {
			uploadToCloud(record)
		}

		tempStr := "N/A"
		if wx.TemperatureC != nil {
			tempStr = fmt.Sprintf("%.1f", *wx.TemperatureC)
		}
		cloudStr := "N/A"
		if wx.CloudCoverPct != nil {
			cloudStr = strconv.Itoa(*wx.CloudCoverPct)
		}

		fmt.Printf("\n[\033[1;33m%s ☀️ RENOGY LIVE TELEMETRY | %s\033[0m]\n", nowStr, sunState)
		fmt.Printf("  ├─ Array (400W 2S2P): \033[1;33m%d W\033[0m (%.1fV @ %.2fA) | Util: %.1f%% | Peak: %dW\n",
			telem.PVPowerW, telem.PVVoltageV, telem.PVCurrentA, telem.ArrayUtilizationPct, telem.DailyMaxPVWatts)
		fmt.Printf("  ├─ Battery:           \033[1;32m%.1f V\033[0m | SOC: \033[1;36m%d%%\033[0m | Charge: %.2fA\n",
			telem.BatteryVoltageV, telem.BatterySOCPct, telem.BatteryCurrentA)
		fmt.Printf("  ├─ State:             \033[1;35m%s\033[0m | Health: \033[1;32m%s\033[0m\n",
			telem.ChargingState, telem.FaultFlags)
		fmt.Printf("  ├─ Dorset Wx:         %s°C | Clouds: %s%% | Rad: %.1f W/m² (PR: %.1f%%)\n",
			tempStr, cloudStr, wx.DirectRadiationWM2, prPct)
		fmt.Printf("  ├─ Temps:             Controller %d°C | Battery %d°C\n",
			telem.ControllerTempC, telem.BatteryTempC)
		fmt.Printf("  └─ Daily Yield:       \033[1;32m%d Wh\033[0m | Lifetime: %d kWh\n",
			telem.DailyGeneratedWh, telem.TotalGeneratedKWh)
}

type OutageEvent struct {
	ID           int    `json:"id"`
	Status       string `json:"status"` // "ACTIVE" or "RESOLVED"
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time,omitempty"`
	DurationSec  int    `json:"duration_sec"`
	Reason       string `json:"reason"`
	RecoveredVia string `json:"recovered_via,omitempty"`
}

type OutageStats struct {
	OutageCount      int     `json:"outage_count"`
	TotalDowntimeSec int     `json:"total_downtime_sec"`
	SessionUptimeSec int     `json:"session_uptime_sec"`
	AvailabilityPct  float64 `json:"availability_pct"`
	InOutage         bool    `json:"in_outage"`
	CurrentOutageSec int     `json:"current_outage_sec"`
}

type OutageTracker struct {
	mu            sync.Mutex
	sessionStart  time.Time
	hasSeenFrame  bool
	inOutage      bool
	outageCount   int
	outageStart   time.Time
	totalDowntime time.Duration
	history       []OutageEvent
}

var (
	clientsMu       sync.Mutex
	activeClients   = make(map[*websocket.Conn]string)
	lastFrameTime   = time.Now()
	lastHealthCheck time.Time
	tracker         = &OutageTracker{
		sessionStart: time.Now(),
		history:      make([]OutageEvent, 0),
	}
)

func (t *OutageTracker) GetStats() (OutageStats, []OutageEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	uptimeSec := int(now.Sub(t.sessionStart).Seconds())
	if uptimeSec < 1 {
		uptimeSec = 1
	}

	totalDown := t.totalDowntime
	curOutageSec := 0
	if t.inOutage {
		curOutageSec = int(now.Sub(t.outageStart).Seconds())
		totalDown += now.Sub(t.outageStart)
	}

	totalDownSec := int(totalDown.Seconds())
	if totalDownSec > uptimeSec {
		totalDownSec = uptimeSec
	}

	availPct := 100.0 * float64(uptimeSec-totalDownSec) / float64(uptimeSec)
	if availPct < 0 {
		availPct = 0
	}

	stats := OutageStats{
		OutageCount:      t.outageCount,
		TotalDowntimeSec: totalDownSec,
		SessionUptimeSec: uptimeSec,
		AvailabilityPct:  availPct,
		InOutage:         t.inOutage,
		CurrentOutageSec: curOutageSec,
	}

	histCopy := make([]OutageEvent, len(t.history))
	copy(histCopy, t.history)

	return stats, histCopy
}

func (t *OutageTracker) RecordOutageStart(reason string) (*OutageEvent, OutageStats) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.inOutage {
		return nil, OutageStats{}
	}

	t.inOutage = true
	t.outageCount++
	t.outageStart = time.Now()

	event := OutageEvent{
		ID:          t.outageCount,
		Status:      "ACTIVE",
		StartTime:   t.outageStart.Format("15:04:05"),
		DurationSec: 0,
		Reason:      reason,
	}

	t.history = append([]OutageEvent{event}, t.history...)
	if len(t.history) > 50 {
		t.history = t.history[:50]
	}

	now := time.Now()
	uptimeSec := int(now.Sub(t.sessionStart).Seconds())
	if uptimeSec < 1 {
		uptimeSec = 1
	}
	totalDownSec := int(t.totalDowntime.Seconds())
	availPct := 100.0 * float64(uptimeSec-totalDownSec) / float64(uptimeSec)
	if availPct < 0 {
		availPct = 0
	}

	stats := OutageStats{
		OutageCount:      t.outageCount,
		TotalDowntimeSec: totalDownSec,
		SessionUptimeSec: uptimeSec,
		AvailabilityPct:  availPct,
		InOutage:         true,
		CurrentOutageSec: 0,
	}

	return &event, stats
}

func (t *OutageTracker) RecordOutageEnd(recoveredVia string) (*OutageEvent, OutageStats) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.hasSeenFrame = true
	if !t.inOutage {
		return nil, OutageStats{}
	}

	now := time.Now()
	duration := now.Sub(t.outageStart)
	t.totalDowntime += duration
	t.inOutage = false
	durSec := int(duration.Seconds())

	event := OutageEvent{
		ID:           t.outageCount,
		Status:       "RESOLVED",
		StartTime:    t.outageStart.Format("15:04:05"),
		EndTime:      now.Format("15:04:05"),
		DurationSec:  durSec,
		Reason:       "Telemetry stream interrupted",
		RecoveredVia: recoveredVia,
	}

	if len(t.history) > 0 {
		t.history[0].Status = "RESOLVED"
		t.history[0].EndTime = now.Format("15:04:05")
		t.history[0].DurationSec = durSec
		t.history[0].RecoveredVia = recoveredVia
	}

	uptimeSec := int(now.Sub(t.sessionStart).Seconds())
	if uptimeSec < 1 {
		uptimeSec = 1
	}
	totalDownSec := int(t.totalDowntime.Seconds())
	availPct := 100.0 * float64(uptimeSec-totalDownSec) / float64(uptimeSec)
	if availPct < 0 {
		availPct = 0
	}

	stats := OutageStats{
		OutageCount:      t.outageCount,
		TotalDowntimeSec: totalDownSec,
		SessionUptimeSec: uptimeSec,
		AvailabilityPct:  availPct,
		InOutage:         false,
		CurrentOutageSec: 0,
	}

	return &event, stats
}

func broadcastControlMsg(msg map[string]interface{}) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	for conn := range activeClients {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

func checkAndHealBluetoothSubsystem() {
	now := time.Now()
	if time.Since(lastHealthCheck) < 20*time.Second {
		return
	}
	lastHealthCheck = now

	// 1. Check systemctl status for bluetooth daemon
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "bluetooth")
	out, err := cmd.Output()
	status := strings.TrimSpace(string(out))

	if err != nil || status != "active" {
		fmt.Printf("\n[\033[1;31mWATCHDOG ALERT\033[0m] Bluetooth daemon is %s. Attempting auto-recovery...\n", status)
		// Attempt restart
		restartCmd := exec.Command("sudo", "systemctl", "restart", "bluetooth")
		if rErr := restartCmd.Run(); rErr != nil {
			// Try non-sudo systemctl start if sudo not configured
			_ = exec.Command("systemctl", "start", "bluetooth").Run()
		}
		time.Sleep(1 * time.Second)
	}

	// 2. Ensure bluetooth power is ON
	_ = exec.Command("bluetoothctl", "power", "on").Run()
}

func startBluetoothWatchdog(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Printf("[\033[1;32mSUPERVISOR\033[0m] Autonomous Bluetooth, Outage Logger & Watchdog active.\n")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			frameMu.Lock()
			elapsed := time.Since(lastFrameTime)
			frameMu.Unlock()

			clientsMu.Lock()
			clientCount := len(activeClients)
			clientsMu.Unlock()

			// Check stream freshness
			if elapsed > 30*time.Second {
				// 1. Detect Outage Start
				if !tracker.inOutage && time.Since(tracker.sessionStart) > 20*time.Second {
					ev, stats := tracker.RecordOutageStart("Stream silent > 30s (BLE connection lost or stalled)")
					if ev != nil {
						fmt.Printf("\n[\033[1;31m🚨 OUTAGE DETECTED #%d\033[0m] Interrupted at %s (Quiet for %.0fs). Self-healing active...\n",
							ev.ID, ev.StartTime, elapsed.Seconds())
						_, hist := tracker.GetStats()
						broadcastControlMsg(map[string]interface{}{
							"type":    "outage_event",
							"event":   "outage_start",
							"outage":  ev,
							"stats":   stats,
							"history": hist,
						})
					}
				} else if tracker.inOutage {
					// Tick while outage continues
					stats, hist := tracker.GetStats()
					broadcastControlMsg(map[string]interface{}{
						"type":    "outage_tick",
						"stats":   stats,
						"history": hist,
					})
				}

				fmt.Printf("[\033[1;33mWATCHDOG\033[0m] Telemetry silent for %.0fs (Clients connected: %d). Verifying host BLE health...\n",
					elapsed.Seconds(), clientCount)

				// Self-heal Linux bluetooth subsystem
				checkAndHealBluetoothSubsystem()

				if clientCount > 0 {
					// Instruct browser client to force-reconnect GATT session
					broadcastControlMsg(map[string]interface{}{
						"type":                      "watchdog_reconnect",
						"reason":                    "stalled_telemetry",
						"seconds_since_last_packet": int(elapsed.Seconds()),
					})
				} else {
					fmt.Printf("[\033[1;31mWATCHDOG\033[0m] No browser WebSocket connected to ws://localhost:%d. Open http://localhost:%d in Chrome.\n", wsPort, httpPort)
				}
			} else {
				// Send lightweight heartbeat ping with uptime stats
				stats, _ := tracker.GetStats()
				broadcastControlMsg(map[string]interface{}{
					"type":      "ping",
					"stats":     stats,
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
			}
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer func() {
		clientsMu.Lock()
		delete(activeClients, conn)
		clientsMu.Unlock()
		conn.Close()
	}()

	clientAddr := conn.RemoteAddr().String()
	clientsMu.Lock()
	activeClients[conn] = clientAddr
	clientsMu.Unlock()

	fmt.Printf("[\033[92mWS\033[0m] Browser connected: %s (Active clients: %d)\n", clientAddr, len(activeClients))

	// Send initial outage & uptime state synchronization
	initStats, initHistory := tracker.GetStats()
	if syncData, err := json.Marshal(map[string]interface{}{
		"type":    "outage_sync",
		"stats":   initStats,
		"history": initHistory,
	}); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, syncData)
	}

	var localRx []byte

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var payload struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			ID    string `json:"id"`
			Bytes []byte `json:"bytes"`
		}
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		nowStr := time.Now().Format("15:04:05.000")

		switch payload.Type {
		case "device_selected":
			name := payload.Name
			if name == "" {
				name = "Renogy Device"
			}
			fmt.Printf("\n[\033[1;33m%s ☀️ RENOGY BLE SELECTED\033[0m] Name: \033[1m%s\033[0m | ID: %s\n", nowStr, name, payload.ID)

		case "gatt_connected":
			name := payload.Name
			if name == "" {
				name = "Renogy Device"
			}
			localRx = localRx[:0]
			fmt.Printf("[\033[92m%s GATT READY\033[0m] Connected to \033[1m%s\033[0m (Channels: FFD1 TX, FFF1 RX)\n", nowStr, name)

		case "notification", "characteristic_value":
			localRx = append(localRx, payload.Bytes...)
			if len(localRx) >= 3 && localRx[1] == 0x03 {
				expectedLen := 3 + int(localRx[2]) + 2
				if len(localRx) >= expectedLen {
					frame := make([]byte, expectedLen)
					copy(frame, localRx[:expectedLen])
					localRx = localRx[expectedLen:]

					// Verify CRC16
					crcl, crch := calcCRC16(frame[:len(frame)-2])
					if frame[len(frame)-2] == crcl && frame[len(frame)-1] == crch {
						frameMu.Lock()
						lastFrameTime = time.Now()
						frameMu.Unlock()

						// Record outage restoration if previously in outage
						ev, stats := tracker.RecordOutageEnd("Auto-healed / GATT Re-synced")
						if ev != nil {
							fmt.Printf("\n[\033[1;32m✅ STREAM RESTORED\033[0m] Telemetry resumed at %s! Outage #%d resolved (Duration: %ds | Session Uptime: %.1f%%)\n\n",
								ev.EndTime, ev.ID, ev.DurationSec, stats.AvailabilityPct)
							_, hist := tracker.GetStats()
							broadcastControlMsg(map[string]interface{}{
								"type":    "outage_event",
								"event":   "outage_end",
								"outage":  ev,
								"stats":   stats,
								"history": hist,
							})
						}

						processFrame(frame)
					}
				}
			}
			if len(localRx) > 512 {
				localRx = localRx[:0]
			}

		case "get_outages":
			s, h := tracker.GetStats()
			if d, err := json.Marshal(map[string]interface{}{
				"type":    "outage_sync",
				"stats":   s,
				"history": h,
			}); err == nil {
				_ = conn.WriteMessage(websocket.TextMessage, d)
			}
		}
	}
}

func main() {
	loadEnv()

	banner := `===========================================================================
RENOGY BT-1 / BT-2 SOLAR & BLE GOLANG GATEWAY
===========================================================================
  • Site: %s (%.3f°N, %.3f°W)
  • Service UUID 0xFFD0 (Write Commands to FFD1)
  • Service UUID 0xFFF0 (Receive Telemetry from FFF1)
  • Automatic Stream Chunk Reassembly & CRC16 Engine
  • High-Performance Golang Runtime
---------------------------------------------------------------------------
Open Dashboard: http://localhost:%d in Chrome on your device
===========================================================================`

	fmt.Printf(banner+"\n", siteName, siteLat, siteLon, httpPort)

	// 1. Start WebSocket Server on 8765
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/", handleWebSocket)
	go func() {
		wsAddr := fmt.Sprintf(":%d", wsPort)
		fmt.Printf("[WS] WebSocket Gateway listening on: ws://localhost:%d\n", wsPort)
		if err := http.ListenAndServe(wsAddr, wsMux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WebSocket listener failed: %v", err)
		}
	}()

	// 2. Start HTTP Dashboard Server on 8080 (No-cache for instant UI updates)
	httpMux := http.NewServeMux()
	fs := http.FileServer(http.Dir("static"))
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fs.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", httpPort),
		Handler: httpMux,
	}

	go func() {
		fmt.Printf("[HTTP] Solar Web Dashboard: \033[1;32mhttp://localhost:%d\033[0m\n", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 3. Start Autonomous Bluetooth & Telemetry Watchdog Supervisor
	watchdogCtx, watchdogCancel := context.WithCancel(context.Background())
	defer watchdogCancel()
	go startBluetoothWatchdog(watchdogCtx)

	// Wait for terminate signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down Solaria Golang Gateway...")
	watchdogCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
