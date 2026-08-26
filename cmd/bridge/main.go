package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Diagnostic Logging Structs & Subsystems
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`     // DEBUG, INFO, WARN, ERROR, FATAL
	Subsystem string                 `json:"subsystem"` // CONTROLLER_MODBUS, BLE_RADIO, DISK_SPOOLER, CLOUD_UPLOADER, WEATHER_CLIENT, HTTP_API, CONTROL_ENGINE, BATTERY_SAFETY
	Message   string                 `json:"message"`
	ErrorCode string                 `json:"error_code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type DiagnosticLogBuffer struct {
	mu         sync.RWMutex
	entries    []LogEntry
	maxEntries int
	errorCount int64
	warnCount  int64
	infoCount  int64
}

func NewDiagnosticLogBuffer(max int) *DiagnosticLogBuffer {
	return &DiagnosticLogBuffer{
		entries:    make([]LogEntry, 0, max),
		maxEntries: max,
	}
}

func (b *DiagnosticLogBuffer) Log(level, subsystem, message, errCode string, details map[string]interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch level {
	case "ERROR", "FATAL":
		b.errorCount++
	case "WARN":
		b.warnCount++
	case "INFO":
		b.infoCount++
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		Level:     level,
		Subsystem: subsystem,
		Message:   message,
		ErrorCode: errCode,
		Details:   details,
	}

	b.entries = append(b.entries, entry)
	if len(b.entries) > b.maxEntries {
		b.entries = b.entries[len(b.entries)-b.maxEntries:]
	}
}

func (b *DiagnosticLogBuffer) GetLogs(level, subsystem, search string, limit int) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var filtered []LogEntry
	level = strings.ToUpper(strings.TrimSpace(level))
	subsystem = strings.ToUpper(strings.TrimSpace(subsystem))
	search = strings.ToLower(strings.TrimSpace(search))

	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if level != "" && level != "ALL" && e.Level != level {
			continue
		}
		if subsystem != "" && subsystem != "ALL" && !strings.EqualFold(e.Subsystem, subsystem) {
			continue
		}
		if search != "" {
			match := strings.Contains(strings.ToLower(e.Message), search) ||
				strings.Contains(strings.ToLower(e.ErrorCode), search) ||
				strings.Contains(strings.ToLower(e.Subsystem), search)
			if !match {
				continue
			}
		}
		filtered = append(filtered, e)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered
}

func (b *DiagnosticLogBuffer) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return map[string]interface{}{
		"total_logged": len(b.entries),
		"error_count":  b.errorCount,
		"warn_count":   b.warnCount,
		"info_count":   b.infoCount,
	}
}

// Global Configuration & Defaults
var (
	httpPort        = 8080
	wsPort          = 8765
	siteLat         = 45.186
	siteLon         = -78.863
	siteName        = "1296 Wren Lake Drive, Dorset, ON"
	arrayRatedWatts       = 400.0
	cloudEndpoint         = "http://localhost:8081/api/v1/telemetry"
	fallbackCloudEndpoint = "http://localhost:8081/api/v1/ingest"
	cloudToken            = ""
	bridgeToken           = ""
	storageMode           = "both" // "local", "bigquery" / "cloud", "both"
	siteTZ                = "America/Toronto"
	siteLoc               = time.Local

	mu             sync.Mutex
	rxBuffer       []byte
	cachedWeather  WeatherMetrics
	lastWxFetch    time.Time
	idTokenCache   string
	idTokenExpires time.Time

	diskSpooler  *DiskSpooler
	bridgeLogger = NewDiagnosticLogBuffer(1000)

	// Rate limiter for control actions
	controlRateMu    sync.Mutex
	lastControlTimes = make(map[string]time.Time)

	upgrader = websocket.Upgrader{
		CheckOrigin: isAllowedOrigin,
	}
)

func isAllowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "solaria.local" || strings.HasSuffix(host, ".local") {
		return true
	}
	// Check if matches configured cloud endpoint hostname
	if cloudEndpoint != "" {
		if cu, err := url.Parse(cloudEndpoint); err == nil && cu.Hostname() != "" {
			if host == cu.Hostname() {
				return true
			}
		}
	}
	if strings.HasSuffix(host, ".run.app") {
		return true
	}
	// Check private LAN IP ranges (192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12)
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
		return true
	}
	return false
}

func initBridgeAuth() {
	token := os.Getenv("SOLARIA_BRIDGE_TOKEN")
	if token == "" {
		token = os.Getenv("SOLARIA_API_TOKEN")
	}
	if token == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		token = hex.EncodeToString(b)
	}
	bridgeToken = token
	cloudToken = token
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func verifyBridgeAuth(r *http.Request, payloadToken string) bool {
	if bridgeToken == "" {
		return true
	}
	if r != nil {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") && constantTimeEqual(strings.TrimPrefix(auth, "Bearer "), bridgeToken) {
			return true
		}
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && constantTimeEqual(apiKey, bridgeToken) {
			return true
		}
	}
	if payloadToken != "" && constantTimeEqual(payloadToken, bridgeToken) {
		return true
	}
	return false
}

func checkControlRateLimit(clientKey string, minInterval time.Duration) bool {
	controlRateMu.Lock()
	defer controlRateMu.Unlock()
	last, exists := lastControlTimes[clientKey]
	if exists && time.Since(last) < minInterval {
		return false
	}
	lastControlTimes[clientKey] = time.Now()
	return true
}

// DiskSpooler provides fault-tolerant disk-backed buffering for telemetry when network is lost.
type DiskSpooler struct {
	spoolPath   string
	mu          sync.Mutex
	cachedCount int64
}

func NewDiskSpooler(dir string) *DiskSpooler {
	_ = os.MkdirAll(dir, 0750)
	spoolPath := filepath.Join(dir, "telemetry_spool.jsonl")
	s := &DiskSpooler{
		spoolPath: spoolPath,
	}
	// Initial count on startup
	if f, err := os.Open(spoolPath); err == nil {
		count := int64(0)
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			if len(bytes.TrimSpace(scanner.Bytes())) > 0 {
				count++
			}
		}
		f.Close()
		s.cachedCount = count
	}
	return s
}

const MaxBridgeSpoolBytes int64 = 50 * 1024 * 1024 // 50MB quota

func (s *DiskSpooler) Spool(record SolarRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check file size and rotate safely on line boundary if exceeding 50MB quota
	if fi, err := os.Stat(s.spoolPath); err == nil && fi.Size() > MaxBridgeSpoolBytes {
		if data, rErr := os.ReadFile(s.spoolPath); rErr == nil {
			mid := len(data) / 2
			if idx := bytes.IndexByte(data[mid:], '\n'); idx != -1 {
				tmpPath := s.spoolPath + ".rotate.tmp"
				if err := os.WriteFile(tmpPath, data[mid+idx+1:], 0600); err == nil {
					_ = os.Rename(tmpPath, s.spoolPath)
				}
			}
		}
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.spoolPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	atomic.AddInt64(&s.cachedCount, 1)
	return f.Sync()
}

func (s *DiskSpooler) Count() int {
	if s == nil {
		return 0
	}
	return int(atomic.LoadInt64(&s.cachedCount))
}

func (s *DiskSpooler) Drain(uploader func(record SolarRecord) error) (int, error) {
	s.mu.Lock()
	if _, err := os.Stat(s.spoolPath); os.IsNotExist(err) {
		atomic.StoreInt64(&s.cachedCount, 0)
		s.mu.Unlock()
		return 0, nil
	}

	stagingPath := s.spoolPath + ".processing"
	if err := os.Rename(s.spoolPath, stagingPath); err != nil {
		s.mu.Unlock()
		return 0, err
	}
	atomic.StoreInt64(&s.cachedCount, 0)
	s.mu.Unlock()

	f, err := os.Open(stagingPath)
	if err != nil {
		return 0, err
	}

	var toUpload []SolarRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var r SolarRecord
		if err := json.Unmarshal(line, &r); err == nil {
			toUpload = append(toUpload, r)
		} else {
			// Quarantine corrupted record
			qf, qErr := os.OpenFile(s.spoolPath+".corrupt.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if qErr == nil {
				_, _ = qf.Write(append(line, '\n'))
				_ = qf.Close()
			}
		}
	}
	f.Close()

	if len(toUpload) == 0 {
		_ = os.Remove(stagingPath)
		return 0, nil
	}

	var remaining []SolarRecord
	drainedCount := 0

	for i, rec := range toUpload {
		if err := uploader(rec); err != nil {
			remaining = toUpload[i:]
			break
		}
		drainedCount++
		// Backpressure pause between records during recovery
		time.Sleep(10 * time.Millisecond)
	}

	_ = os.Remove(stagingPath)

	if len(remaining) > 0 {
		s.mu.Lock()
		// Write remaining records back to spool file
		tmpPath := s.spoolPath + ".requeue.tmp"
		tf, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err == nil {
			for _, r := range remaining {
				d, _ := json.Marshal(r)
				_, _ = tf.Write(append(d, '\n'))
			}
			// If existing spool arrived while we were draining, append it too
			if existingData, readErr := os.ReadFile(s.spoolPath); readErr == nil && len(existingData) > 0 {
				_, _ = tf.Write(existingData)
			}
			_ = tf.Sync()
			tf.Close()
			_ = os.Rename(tmpPath, s.spoolPath)
		}
		atomic.StoreInt64(&s.cachedCount, int64(len(remaining)))
		s.mu.Unlock()
	}

	return drainedCount, nil
}

// DrainBatch drains spooled records in batches for high-throughput recovery without exhausting HTTP connections.
func (s *DiskSpooler) DrainBatch(uploader func(records []SolarRecord) error, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 50
	}
	s.mu.Lock()
	if _, err := os.Stat(s.spoolPath); os.IsNotExist(err) {
		atomic.StoreInt64(&s.cachedCount, 0)
		s.mu.Unlock()
		return 0, nil
	}

	stagingPath := s.spoolPath + ".processing"
	if err := os.Rename(s.spoolPath, stagingPath); err != nil {
		s.mu.Unlock()
		return 0, err
	}
	atomic.StoreInt64(&s.cachedCount, 0)
	s.mu.Unlock()

	f, err := os.Open(stagingPath)
	if err != nil {
		return 0, err
	}

	var toUpload []SolarRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var r SolarRecord
		if err := json.Unmarshal(line, &r); err == nil {
			toUpload = append(toUpload, r)
		} else {
			// Quarantine corrupted record
			qf, qErr := os.OpenFile(s.spoolPath+".corrupt.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if qErr == nil {
				_, _ = qf.Write(append(line, '\n'))
				_ = qf.Close()
			}
		}
	}
	f.Close()

	if len(toUpload) == 0 {
		_ = os.Remove(stagingPath)
		return 0, nil
	}

	var remaining []SolarRecord
	drainedCount := 0

	for i := 0; i < len(toUpload); i += batchSize {
		end := i + batchSize
		if end > len(toUpload) {
			end = len(toUpload)
		}
		batchChunk := toUpload[i:end]
		if err := uploader(batchChunk); err != nil {
			remaining = toUpload[i:]
			break
		}
		drainedCount += len(batchChunk)
		time.Sleep(10 * time.Millisecond)
	}

	_ = os.Remove(stagingPath)

	if len(remaining) > 0 {
		s.mu.Lock()
		tmpPath := s.spoolPath + ".requeue.tmp"
		tf, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err == nil {
			for _, r := range remaining {
				d, _ := json.Marshal(r)
				_, _ = tf.Write(append(d, '\n'))
			}
			if existingData, readErr := os.ReadFile(s.spoolPath); readErr == nil && len(existingData) > 0 {
				_, _ = tf.Write(existingData)
			}
			_ = tf.Sync()
			tf.Close()
			_ = os.Rename(tmpPath, s.spoolPath)
		}
		atomic.StoreInt64(&s.cachedCount, int64(len(remaining)))
		s.mu.Unlock()
	}

	return drainedCount, nil
}

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
	MPPTEfficiencyPct           float64 `json:"mppt_efficiency_pct"`
	StringHealthStatus          string  `json:"string_health_status"`
	SubZeroInhibitWarning       bool    `json:"subzero_inhibit_warning"`
	SubZeroInhibitMessage       string  `json:"subzero_inhibit_message"`
	ColdDerateWarning           bool    `json:"cold_derate_warning"`
	ColdDerateMessage           string  `json:"cold_derate_message"`
	BatteryType                 string  `json:"battery_type"`
	ControllerModel             string  `json:"controller_model"`
	ControllerRatedCurrentA     int     `json:"controller_rated_current_a"`
	ControllerRatedVoltageV     int     `json:"controller_rated_voltage_v"`
}

type SolarRecord struct {
	Timestamp         string             `json:"timestamp"`
	Site              string             `json:"site"`
	Location          map[string]float64 `json:"location"`
	Telemetry         Telemetry          `json:"telemetry"`
	Weather           WeatherMetrics     `json:"weather"`
	SunClassification string             `json:"sun_classification"`
	IsMock            bool               `json:"is_mock"`
	DataSource        string             `json:"data_source"` // "HARDWARE_BLE", "BRIDGE_HEARTBEAT", "OFFLINE_OUTAGE"
	BLEConnected      bool               `json:"ble_connected"`
	OutageStatus      string             `json:"outage_status"` // "NOMINAL", "BLE_DISCONNECTED", "STREAM_SILENT"
	OutageDurationSec int                `json:"outage_duration_sec,omitempty"`
	OutageReason      string             `json:"outage_reason,omitempty"`
}

func loadEnv() {
	envCandidates := []string{".env", "../.env", "../../.env", "/home/fcurrie/Projects/solar-testing/.env"}
	for _, envPath := range envCandidates {
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
			break
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
	if val := os.Getenv("SOLARIA_FALLBACK_ENDPOINT"); val != "" {
		fallbackCloudEndpoint = val
	}
	if val := os.Getenv("SOLARIA_API_TOKEN"); val != "" {
		cloudToken = val
	}
	if val := os.Getenv("STORAGE_MODE"); val != "" {
		storageMode = strings.ToLower(val)
	}
	if val := os.Getenv("TIMEZONE"); val != "" {
		siteTZ = val
	} else if val := os.Getenv("SITE_TIMEZONE"); val != "" {
		siteTZ = val
	}
	if loc, err := time.LoadLocation(siteTZ); err == nil {
		siteLoc = loc
	} else {
		siteLoc = time.Local
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
		bridgeLogger.Log("WARN", "WEATHER_CLIENT", fmt.Sprintf("Open-Meteo API query failed: %v (using cached/fallback weather)", err), "ERR_WEATHER_FETCH_FAIL", map[string]interface{}{"url": url})
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
			tempVal := 0.0
			if cachedWeather.TemperatureC != nil {
				tempVal = *cachedWeather.TemperatureC
			}
			cloudVal := 0
			if cachedWeather.CloudCoverPct != nil {
				cloudVal = *cachedWeather.CloudCoverPct
			}
			bridgeLogger.Log("DEBUG", "WEATHER_CLIENT", fmt.Sprintf("Weather synced from Open-Meteo: %.1f°C, %d%% clouds, %.0f W/m² direct, %.0f W/m² diffuse", tempVal, cloudVal, cachedWeather.DirectRadiationWM2, cachedWeather.DiffuseRadiationWM2), "WEATHER_SYNC_OK", nil)
		}
	} else {
		bridgeLogger.Log("WARN", "WEATHER_CLIENT", fmt.Sprintf("Open-Meteo API returned HTTP %d", resp.StatusCode), "ERR_WEATHER_HTTP_STATUS", map[string]interface{}{"status_code": resp.StatusCode})
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
		bridgeLogger.Log("ERROR", "CONTROLLER_MODBUS", "Malformed Modbus frame structure or unexpected function code", "ERR_MODBUS_INVALID_FRAME", map[string]interface{}{
			"frame_len": len(raw),
			"raw_hex":   hex.EncodeToString(raw),
		})
		return Telemetry{}, fmt.Errorf("invalid frame length: %d", len(raw))
	}

	// Verify Modbus CRC-16 Checksum if present
	if len(raw) >= 5 {
		actualLow, actualHigh := raw[len(raw)-2], raw[len(raw)-1]
		if actualLow != 0 || actualHigh != 0 {
			crcLow, crcHigh := calcCRC16(raw[:len(raw)-2])
			if crcLow != actualLow || crcHigh != actualHigh {
				bridgeLogger.Log("ERROR", "CONTROLLER_MODBUS", "Modbus CRC16 checksum mismatch detected on incoming frame", "ERR_MODBUS_CRC_MISMATCH", map[string]interface{}{
					"frame_len":    len(raw),
					"raw_hex":      hex.EncodeToString(raw),
					"expected_crc": fmt.Sprintf("0x%02X%02X", crcLow, crcHigh),
					"actual_crc":   fmt.Sprintf("0x%02X%02X", actualLow, actualHigh),
				})
				return Telemetry{}, fmt.Errorf("modbus CRC mismatch: expected 0x%02X%02X got 0x%02X%02X", crcLow, crcHigh, actualLow, actualHigh)
			}
		}
	}

	data := raw[3 : len(raw)-2]
	if len(data) < 20 {
		bridgeLogger.Log("ERROR", "CONTROLLER_MODBUS", "Insufficient Modbus register payload in frame", "ERR_MODBUS_SHORT_PAYLOAD", map[string]interface{}{
			"payload_len": len(data),
			"raw_hex":     hex.EncodeToString(raw),
		})
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
			b := data[offset]
			if b > 127 {
				return int(b) - 256
			}
			return int(b)
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

	// 1. MPPT DC-DC Buck Conversion Efficiency (Vbatt * Ibatt / Ppv)
	mpptEff := 0.0
	if pvW > 5 && battA > 0.05 && battV > 10.0 {
		outWatts := battV * battA
		eff := (outWatts / float64(pvW)) * 100.0
		if eff > 100.0 {
			eff = 99.1
		} else if eff < 50.0 {
			eff = 50.0
		}
		mpptEff = math.Round(eff*10) / 10
	}

	// 2. Sub-Zero Low-Temperature Lithium Inhibit & Cold Derate Protection Alert
	subZeroWarn := false
	subZeroMsg := "OK: Thermal probe within safe operating limits"
	coldDerateWarn := false
	coldDerateMsg := "OK: Thermal conditions optimal for full charging rate"

	// Check if external RTS probe is unconnected via Modbus fault register bit 13 (0x2000)
	probeDisconnected := (faultBits & (1 << 13)) != 0
	effectiveBattTemp := battTemp
	if probeDisconnected {
		effectiveBattTemp = ctrlTemp - 5
	}

	if effectiveBattTemp <= 0 {
		subZeroWarn = true
		if battA > 0.1 || pvW > 5 {
			subZeroMsg = fmt.Sprintf("CRITICAL: Battery temperature %d°C is sub-zero (<=0°C)! LiFePO4 charging must be strictly inhibited to prevent irreversible lithium dendrite plating.", effectiveBattTemp)
		} else {
			subZeroMsg = fmt.Sprintf("WARNING: Battery temperature is %d°C (Sub-Zero). LiFePO4 charge currently inhibited.", effectiveBattTemp)
		}
		bridgeLogger.Log("WARN", "BATTERY_SAFETY", subZeroMsg, "ERR_SUBZERO_INHIBIT", map[string]interface{}{
			"battery_temp_c": effectiveBattTemp,
			"battery_amps":   battA,
			"pv_watts":       pvW,
		})
	} else if effectiveBattTemp <= 5 {
		coldDerateWarn = true
		if battA > 15.0 {
			coldDerateMsg = fmt.Sprintf("ADVISORY: Low battery temperature (%d°C). High charge current (%.1fA) should be derated (< 0.1C / ~17A on 170Ah LiFePO4 bank) to prevent localized lithium plating.", effectiveBattTemp, battA)
		} else {
			coldDerateMsg = fmt.Sprintf("ADVISORY: Battery temperature is %d°C (Low Temp Transition Zone 1°C-5°C). Charging safely derated.", effectiveBattTemp)
		}
		bridgeLogger.Log("WARN", "BATTERY_SAFETY", coldDerateMsg, "ERR_COLD_DERATE", map[string]interface{}{
			"battery_temp_c": effectiveBattTemp,
			"battery_amps":   battA,
		})
	}

	if faultBits != 0 {
		bridgeLogger.Log("ERROR", "CONTROLLER_MODBUS", fmt.Sprintf("Controller hardware fault active: %s (Bits: 0x%04X)", faultFlags, faultBits), "ERR_HARDWARE_FAULT", map[string]interface{}{
			"fault_bits":  fmt.Sprintf("0x%04X", faultBits),
			"fault_flags": faultFlags,
		})
	}

	if battV < 11.0 && battV > 0 {
		bridgeLogger.Log("WARN", "BATTERY_SAFETY", fmt.Sprintf("Low battery voltage detected: %.1fV (LVD approaching at 10.6V)", battV), "ERR_LOW_BATTERY_VOLTAGE", map[string]interface{}{
			"battery_voltage_v": battV,
			"battery_soc_pct":   battSOC,
		})
	} else if battV > 15.0 {
		bridgeLogger.Log("WARN", "BATTERY_SAFETY", fmt.Sprintf("High battery voltage detected: %.1fV (OVD threshold at 16.0V)", battV), "ERR_HIGH_BATTERY_VOLTAGE", map[string]interface{}{
			"battery_voltage_v": battV,
		})
	}

	// 3. 2S2P String Balance & PV Fault Diagnostics
	stringStatus := "NOMINAL_2S2P"
	if pvV < 5.0 {
		stringStatus = "NIGHT_OR_INACTIVE"
	} else if pvV >= 10.0 && pvV < 26.0 {
		stringStatus = "POTENTIAL_SERIES_DIODE_BYPASS_OR_SINGLE_PANEL_FAULT"
		bridgeLogger.Log("WARN", "ARRAY_TOPOLOGY", fmt.Sprintf("Array voltage %.1fV is lower than nominal 2S Vmp (36-40V); possible bypass diode or single panel shading", pvV), "ERR_STRING_VOLTAGE_LOW", map[string]interface{}{
			"pv_voltage_v": pvV,
			"pv_watts":     pvW,
		})
	} else if pvV >= 26.0 && pvW > 0 {
		stringStatus = "NOMINAL_2S2P_ACTIVE"
	} else if pvV >= 26.0 && pvW == 0 {
		stringStatus = "DIFFUSE_OVERCAST_OPEN_CIRCUIT"
	}

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
		MPPTEfficiencyPct:           mpptEff,
		StringHealthStatus:          stringStatus,
		SubZeroInhibitWarning:       subZeroWarn,
		SubZeroInhibitMessage:       subZeroMsg,
		ColdDerateWarning:           coldDerateWarn,
		ColdDerateMessage:           coldDerateMsg,
		BatteryType:                 "LiFePO4 12V 170Ah (B07Q8DQ6TR)",
		ControllerModel:             "Renogy Rover 20A MPPT (RNG-CTRL-RVR20)",
		ControllerRatedCurrentA:     20,
		ControllerRatedVoltageV:     12,
	}, nil
}

func logToCSV(telem Telemetry) {
	now := time.Now()
	logDir := "logs"
	_ = os.MkdirAll(logDir, 0750)
	logFile := filepath.Join(logDir, fmt.Sprintf("solar_telemetry_%s.csv", now.Format("2006-01-02")))

	isNew := false
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		isNew = true
	}

	f, err := os.OpenFile(filepath.Clean(logFile), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
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
	if val := os.Getenv("SOLARIA_IDENTITY_TOKEN"); val != "" {
		return val
	}
	if cloudToken != "" {
		return cloudToken
	}
	return ""
}

var (
	cloudHTTPClient = &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	uploadMu            sync.Mutex
	lastCloudUpload     time.Time
	lastSuccessUpload   time.Time
	totalSuccessUploads int64
)

func getCloudUploadStats() (time.Time, int64, int) {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	spoolCount := 0
	if diskSpooler != nil {
		spoolCount = diskSpooler.Count()
	}
	return lastSuccessUpload, totalSuccessUploads, spoolCount
}

func uploadSingleRecord(record SolarRecord) error {
	endpoints := make([]string, 0, 2)
	if cloudEndpoint != "" {
		endpoints = append(endpoints, cloudEndpoint)
	}
	if fallbackCloudEndpoint != "" && fallbackCloudEndpoint != cloudEndpoint {
		endpoints = append(endpoints, fallbackCloudEndpoint)
	}
	if len(endpoints) == 0 {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"batch": []SolarRecord{record},
	})
	if err != nil {
		bridgeLogger.Log("ERROR", "CLOUD_UPLOADER", fmt.Sprintf("Failed to marshal single record: %v", err), "ERR_UPLOAD_MARSHAL", nil)
		return err
	}

	token := getIDToken()
	var uploadErrors []string
	successCount := 0

	for _, ep := range endpoints {
		start := time.Now()
		req, err := http.NewRequest("POST", ep, bytes.NewBuffer(payload))
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %v", ep, err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-API-Key", token)
		} else if cloudToken != "" {
			req.Header.Set("Authorization", "Bearer "+cloudToken)
			req.Header.Set("X-API-Key", cloudToken)
		}

		resp, err := cloudHTTPClient.Do(req)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %v", ep, err))
			bridgeLogger.Log("WARN", "CLOUD_UPLOADER", fmt.Sprintf("Upload to %s failed after %dms: %v", ep, latency, err), "ERR_CLOUD_POST", map[string]interface{}{
				"endpoint": ep,
				"error":    err.Error(),
			})
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: HTTP %d", ep, resp.StatusCode))
			bridgeLogger.Log("WARN", "CLOUD_UPLOADER", fmt.Sprintf("Upload to %s returned HTTP %d", ep, resp.StatusCode), "ERR_CLOUD_HTTP", map[string]interface{}{
				"status_code": resp.StatusCode,
				"endpoint":    ep,
			})
			continue
		}

		successCount++
		bridgeLogger.Log("DEBUG", "CLOUD_UPLOADER", fmt.Sprintf("Cloud upload to %s succeeded in %dms (HTTP 200)", ep, latency), "CLOUD_UPLOAD_OK", map[string]interface{}{
			"endpoint":   ep,
			"latency_ms": latency,
		})
	}

	if successCount == 0 && len(uploadErrors) > 0 {
		return fmt.Errorf("all upload endpoints failed: %s", strings.Join(uploadErrors, "; "))
	}
	return nil
}

func uploadBatchRecords(records []SolarRecord) error {
	if len(records) == 0 {
		return nil
	}
	endpoints := make([]string, 0, 2)
	if cloudEndpoint != "" {
		endpoints = append(endpoints, cloudEndpoint)
	}
	if fallbackCloudEndpoint != "" && fallbackCloudEndpoint != cloudEndpoint {
		endpoints = append(endpoints, fallbackCloudEndpoint)
	}
	if len(endpoints) == 0 {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"batch": records,
	})
	if err != nil {
		bridgeLogger.Log("ERROR", "CLOUD_UPLOADER", fmt.Sprintf("Failed to marshal batch of %d records: %v", len(records), err), "ERR_BATCH_MARSHAL", nil)
		return err
	}

	token := getIDToken()
	var uploadErrors []string
	successCount := 0

	for _, ep := range endpoints {
		start := time.Now()
		req, err := http.NewRequest("POST", ep, bytes.NewBuffer(payload))
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %v", ep, err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-API-Key", token)
		} else if cloudToken != "" {
			req.Header.Set("Authorization", "Bearer "+cloudToken)
			req.Header.Set("X-API-Key", cloudToken)
		}

		resp, err := cloudHTTPClient.Do(req)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: %v", ep, err))
			bridgeLogger.Log("WARN", "CLOUD_UPLOADER", fmt.Sprintf("Batch upload to %s failed after %dms: %v", ep, latency, err), "ERR_BATCH_POST", map[string]interface{}{
				"count":    len(records),
				"endpoint": ep,
				"error":    err.Error(),
			})
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			uploadErrors = append(uploadErrors, fmt.Sprintf("%s: HTTP %d", ep, resp.StatusCode))
			bridgeLogger.Log("WARN", "CLOUD_UPLOADER", fmt.Sprintf("Batch upload to %s returned HTTP %d", ep, resp.StatusCode), "ERR_BATCH_HTTP", map[string]interface{}{
				"count":       len(records),
				"status_code": resp.StatusCode,
				"endpoint":    ep,
			})
			continue
		}

		successCount++
		bridgeLogger.Log("INFO", "CLOUD_UPLOADER", fmt.Sprintf("Batch upload of %d records succeeded to %s in %dms (HTTP 200)", len(records), ep, latency), "CLOUD_BATCH_OK", map[string]interface{}{
			"count":      len(records),
			"endpoint":   ep,
			"latency_ms": latency,
		})
	}

	if successCount == 0 && len(uploadErrors) > 0 {
		return fmt.Errorf("all batch upload endpoints failed: %s", strings.Join(uploadErrors, "; "))
	}
	return nil
}

var (
	lastTelemetryMu sync.RWMutex
	lastSeenTelem   Telemetry
)

func uploadToCloud(record SolarRecord) {
	if storageMode == "local" && cloudEndpoint == "" {
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
		if err := uploadSingleRecord(record); err != nil {
			spoolCount := 0
			if diskSpooler != nil {
				_ = diskSpooler.Spool(record)
				spoolCount = diskSpooler.Count()
				fmt.Printf("[\033[33mSPOOL\033[0m] Cloud upload error (%v). Telemetry safely spooled to disk (Queue: %d).\n", err, spoolCount)
			}
			broadcastControlMsg(map[string]interface{}{
				"type":        "cloud_upload",
				"status":      "error",
				"error":       err.Error(),
				"spooled":     true,
				"spool_count": spoolCount,
			})
		} else {
			uploadMu.Lock()
			lastSuccessUpload = time.Now()
			totalSuccessUploads++
			succTime := lastSuccessUpload
			succCount := totalSuccessUploads
			uploadMu.Unlock()

			spoolCount := 0
			if diskSpooler != nil {
				spoolCount = diskSpooler.Count()
			}

			broadcastControlMsg(map[string]interface{}{
				"type":          "cloud_upload",
				"status":        "success",
				"timestamp":     succTime.Format(time.RFC3339),
				"total_uploads": succCount,
				"spool_count":   spoolCount,
			})
			tracker.save()
		}
	}()
}

func startSpoolDrainer(ctx context.Context) {
	if diskSpooler == nil || cloudEndpoint == "" {
		return
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if diskSpooler.Count() == 0 {
				continue
			}
			drained, err := diskSpooler.DrainBatch(uploadBatchRecords, 50)
			if err == nil && drained > 0 {
				uploadMu.Lock()
				lastSuccessUpload = time.Now()
				totalSuccessUploads += int64(drained)
				succTime := lastSuccessUpload
				succCount := totalSuccessUploads
				uploadMu.Unlock()

				spoolCount := diskSpooler.Count()
				fmt.Printf("[\033[32mSPOOL DRAINED\033[0m] Successfully restored network uplink: uploaded %d spooled records to Cloud Run!\n", drained)
				broadcastControlMsg(map[string]interface{}{
					"type":          "cloud_upload",
					"status":        "success",
					"timestamp":     succTime.Format(time.RFC3339),
					"total_uploads": succCount,
					"spool_count":   spoolCount,
					"drained":       drained,
				})
			}
		}
	}
}

// startHeartbeatWorker guarantees that the main dashboard receives fresh telemetry at least every 30s
// even when BLE hardware is disconnected, night standby occurs, or during peripheral controller sleep.
func startHeartbeatWorker(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			uploadMu.Lock()
			timeSinceUpload := time.Since(lastCloudUpload)
			uploadMu.Unlock()

			if timeSinceUpload >= 30*time.Second {
				lastTelemetryMu.RLock()
				telem := lastSeenTelem
				lastTelemetryMu.RUnlock()

				if telem.BatteryVoltageV == 0 {
					telem.BatteryVoltageV = 13.2
					telem.BatterySOCPct = 80
					telem.ControllerTempC = 22
					telem.BatteryTempC = 20
					telem.ArrayCapacityW = int(arrayRatedWatts)
					telem.ArrayTopology = "2S2P (4x100W)"
				}
				telem.PVPowerW = 0
				telem.PVCurrentA = 0
				if telem.ChargingState == "" {
					telem.ChargingState = "STANDBY"
				}

				wx := fetchWeather()

				frameMu.Lock()
				elapsedSinceFrame := time.Since(lastFrameTime)
				frameMu.Unlock()

				hbRecord := SolarRecord{
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Site:      siteName,
					Location: map[string]float64{
						"latitude":  siteLat,
						"longitude": siteLon,
					},
					Telemetry:         telem,
					Weather:           wx,
					SunClassification: "STANDBY_OFFLINE",
					IsMock:            false,
					DataSource:        "BRIDGE_HEARTBEAT",
					BLEConnected:      false,
					OutageStatus:      "BLE_DISCONNECTED",
					OutageDurationSec: int(elapsedSinceFrame.Seconds()),
					OutageReason:      "Renogy BT-1 Bluetooth LE hardware disconnected or silent (> 30s)",
				}

				uploadToCloud(hbRecord)
			}
		}
	}
}

var (
	frameMu              sync.Mutex
	lastFrameProcess     time.Time
	totalFramesProcessed int64
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
	lastFrameTime = time.Now()
	lastFrameProcess = time.Now()
	totalFramesProcessed++
	frameMu.Unlock()
	tracker.save()

	// 1. Decode Raw Modbus Registers
	telem, err := decodeTelemetry(frame)
	if err != nil {
		bridgeLogger.Log("ERROR", "CONTROLLER_MODBUS", fmt.Sprintf("Failed to decode Modbus frame: %v", err), "ERR_MODBUS_DECODE", map[string]interface{}{
			"frame_len": len(frame),
		})
		return
	}

	bridgeLogger.Log("INFO", "CONTROLLER_MODBUS", fmt.Sprintf("Modbus telemetry decoded: %dW PV (%.1fV @ %.2fA), %d%% SOC (%.1fV), State: %s", telem.PVPowerW, telem.PVVoltageV, telem.PVCurrentA, telem.BatterySOCPct, telem.BatteryVoltageV, telem.ChargingState), "MODBUS_FRAME_OK", map[string]interface{}{
		"pv_w":      telem.PVPowerW,
		"soc":       telem.BatterySOCPct,
		"batt_v":    telem.BatteryVoltageV,
		"chg_state": telem.ChargingState,
	})

	wx := fetchWeather()
	sunState := classifySunCondition(telem, wx)

	totalIrradiance := wx.DirectRadiationWM2 + wx.DiffuseRadiationWM2
	ambTemp := 25.0
	if wx.TemperatureC != nil {
		ambTemp = *wx.TemperatureC
	}
	// NOCT cell temperature estimation: T_cell = T_amb + ((NOCT-20)/800)*G
	cellTemp := ambTemp + ((45.0-20.0)/800.0)*totalIrradiance
	// Monocrystalline silicon temp coefficient: -0.4%/°C above 25°C STC
	tempFactor := 1.0 - (0.004 * (cellTemp - 25.0))
	if tempFactor < 0.70 {
		tempFactor = 0.70
	} else if tempFactor > 1.15 {
		tempFactor = 1.15
	}
	expectedPower := (totalIrradiance / 1000.0) * arrayRatedWatts * tempFactor
	prPct := 0.0
	if expectedPower > 5.0 {
		prPct = math.Round((float64(telem.PVPowerW)/expectedPower)*1000) / 10
	}
	telem.PerformanceRatioPct = prPct

	// Passive Diagnostic Hazard Check: Reverse-Order Wiring (PV > 18V without Battery < 9V)
	if telem.PVVoltageV > 18.0 && telem.BatteryVoltageV < 9.0 {
		fmt.Printf("\n[\033[1;31m⚠️ HAZARD_REVERSE_WIRING\033[0m] High PV voltage (%.1fV) detected while battery voltage (%.1fV) is missing or disconnected! Ensure battery is connected before PV.\n",
			telem.PVVoltageV, telem.BatteryVoltageV)
		bridgeLogger.Log("ERROR", "BATTERY_SAFETY",
			fmt.Sprintf("HAZARD_REVERSE_WIRING: PV voltage %.1fV active while battery is %.1fV. Ensure battery is connected before solar array to protect MPPT buck converter.", telem.PVVoltageV, telem.BatteryVoltageV),
			"ERR_HAZARD_REVERSE_WIRING", map[string]interface{}{"pv_voltage": telem.PVVoltageV, "batt_voltage": telem.BatteryVoltageV})
	}

	lastTelemetryMu.Lock()
	lastSeenTelem = telem
	lastTelemetryMu.Unlock()

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
		IsMock:            false,
		DataSource:        "HARDWARE_BLE",
		BLEConnected:      true,
		OutageStatus:      "NOMINAL",
	}

	if storageMode != "cloud" && storageMode != "bigquery" {
		logToCSV(telem)
	}
	if storageMode != "local" {
		uploadToCloud(record)
	}

	broadcastControlMsg(map[string]interface{}{
		"type":   "telemetry_frame",
		"record": record,
	})

	tempStr := "N/A"
	if wx.TemperatureC != nil {
		tempStr = fmt.Sprintf("%.1f", *wx.TemperatureC)
	}
	cloudStr := "N/A"
	if wx.CloudCoverPct != nil {
		cloudStr = strconv.Itoa(*wx.CloudCoverPct)
	}

	nowStr := time.Now().Format("15:04:05.000")
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

var outageFilePath = "logs/outages.json"

type OutageEvent struct {
	ID           int    `json:"id"`
	Status       string `json:"status"` // "ACTIVE" or "RESOLVED"
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time,omitempty"`
	StartISO     string `json:"start_iso"`
	EndISO       string `json:"end_iso,omitempty"`
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

type OutagePersistedState struct {
	FirstStartedAt       time.Time     `json:"first_started_at"`
	LastSeenAt           time.Time     `json:"last_seen_at"`
	OutageCount          int           `json:"outage_count"`
	TotalDowntimeSec     int           `json:"total_downtime_sec"`
	TotalUptimeSec       int           `json:"total_uptime_sec"`
	LastSuccessfulUpload time.Time     `json:"last_successful_upload,omitempty"`
	TotalSuccessUploads  int64         `json:"total_successful_uploads,omitempty"`
	LastBlePacketTime    time.Time     `json:"last_ble_packet_time,omitempty"`
	TotalBleFrames       int64         `json:"total_ble_frames,omitempty"`
	History              []OutageEvent `json:"history"`
}

type OutageTracker struct {
	mu            sync.Mutex
	sessionStart  time.Time
	firstStart    time.Time
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
		firstStart:   time.Now(),
		history:      make([]OutageEvent, 0),
	}
)

func (t *OutageTracker) load() {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(outageFilePath)
	if err != nil {
		return
	}
	var state OutagePersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	t.history = state.History
	t.outageCount = state.OutageCount
	t.totalDowntime = time.Duration(state.TotalDowntimeSec) * time.Second
	if !state.FirstStartedAt.IsZero() {
		t.firstStart = state.FirstStartedAt
	}

	if !state.LastSuccessfulUpload.IsZero() {
		uploadMu.Lock()
		lastSuccessUpload = state.LastSuccessfulUpload
		totalSuccessUploads = state.TotalSuccessUploads
		uploadMu.Unlock()
	}
	if !state.LastBlePacketTime.IsZero() {
		frameMu.Lock()
		lastFrameTime = state.LastBlePacketTime
		totalFramesProcessed = state.TotalBleFrames
		frameMu.Unlock()
	} else if !state.LastSeenAt.IsZero() {
		frameMu.Lock()
		lastFrameTime = state.LastSeenAt
		frameMu.Unlock()
	}

	// If there was an active outage when previous process exited, close it
	if len(t.history) > 0 && t.history[0].Status == "ACTIVE" {
		now := time.Now()
		t.history[0].Status = "RESOLVED"
		t.history[0].EndTime = now.In(siteLoc).Format("15:04:05")
		t.history[0].EndISO = now.UTC().Format(time.RFC3339)
		if t.history[0].DurationSec <= 0 {
			if parsedStart, pErr := time.Parse(time.RFC3339, t.history[0].StartISO); pErr == nil {
				dur := int(now.Sub(parsedStart).Seconds())
				if dur < 0 {
					dur = 0
				}
				t.history[0].DurationSec = dur
				t.totalDowntime += time.Duration(dur) * time.Second
			}
		}
		t.history[0].RecoveredVia = "Bridge restarted / stream restored"
	}
}

func (t *OutageTracker) save() {
	_ = os.MkdirAll("logs", 0750)
	now := time.Now()
	uptimeSec := int(now.Sub(t.firstStart).Seconds())
	downSec := int(t.totalDowntime.Seconds())
	if t.inOutage {
		downSec += int(now.Sub(t.outageStart).Seconds())
	}

	uploadMu.Lock()
	lSucc := lastSuccessUpload
	totSucc := totalSuccessUploads
	uploadMu.Unlock()

	frameMu.Lock()
	lFrame := lastFrameTime
	totFrames := totalFramesProcessed
	frameMu.Unlock()

	state := OutagePersistedState{
		FirstStartedAt:       t.firstStart,
		LastSeenAt:           now,
		OutageCount:          t.outageCount,
		TotalDowntimeSec:     downSec,
		TotalUptimeSec:       uptimeSec,
		LastSuccessfulUpload: lSucc,
		TotalSuccessUploads:  totSucc,
		LastBlePacketTime:    lFrame,
		TotalBleFrames:       totFrames,
		History:              t.history,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		tmpPath := outageFilePath + ".tmp"
		if wErr := os.WriteFile(tmpPath, data, 0600); wErr == nil {
			_ = os.Rename(tmpPath, outageFilePath)
		}
	}
}

func (t *OutageTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sessionStart = time.Now()
	t.firstStart = time.Now()
	t.outageCount = 0
	t.totalDowntime = 0
	t.inOutage = false
	t.history = make([]OutageEvent, 0)
	_ = os.Remove(outageFilePath)
}

func (t *OutageTracker) GetStats() (OutageStats, []OutageEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	uptimeSec := int(now.Sub(t.firstStart).Seconds())
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
		StartTime:   t.outageStart.In(siteLoc).Format("15:04:05"),
		StartISO:    t.outageStart.UTC().Format(time.RFC3339),
		DurationSec: 0,
		Reason:      reason,
	}

	t.history = append([]OutageEvent{event}, t.history...)
	if len(t.history) > 100 {
		t.history = t.history[:100]
	}

	now := time.Now()
	uptimeSec := int(now.Sub(t.firstStart).Seconds())
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

	t.save()
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
		StartTime:    t.outageStart.In(siteLoc).Format("15:04:05"),
		EndTime:      now.In(siteLoc).Format("15:04:05"),
		StartISO:     t.outageStart.UTC().Format(time.RFC3339),
		EndISO:       now.UTC().Format(time.RFC3339),
		DurationSec:  durSec,
		Reason:       "Telemetry stream interrupted",
		RecoveredVia: recoveredVia,
	}

	if len(t.history) > 0 {
		t.history[0].Status = "RESOLVED"
		t.history[0].EndTime = now.In(siteLoc).Format("15:04:05")
		t.history[0].EndISO = now.UTC().Format(time.RFC3339)
		t.history[0].DurationSec = durSec
		t.history[0].RecoveredVia = recoveredVia
	}

	uptimeSec := int(now.Sub(t.firstStart).Seconds())
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

	t.save()
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

func checkAndHealBluetoothSubsystem(silenceDuration time.Duration) {
	now := time.Now()
	if time.Since(lastHealthCheck) < 15*time.Second {
		return
	}
	lastHealthCheck = now

	// Tier 1 (Soft): Check systemctl status for bluetooth daemon
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "is-active", "bluetooth")
	out, err := cmd.Output()
	status := strings.TrimSpace(string(out))

	if err != nil || status != "active" {
		fmt.Printf("\n[\033[1;31mWATCHDOG ALERT\033[0m] Bluetooth daemon is %s. Attempting auto-recovery...\n", status)
		restartCmd := exec.Command("sudo", "systemctl", "restart", "bluetooth")
		if rErr := restartCmd.Run(); rErr != nil {
			_ = exec.Command("systemctl", "start", "bluetooth").Run()
		}
		time.Sleep(1 * time.Second)
	}

	_ = exec.Command("bluetoothctl", "power", "on").Run()

	// Tier 2 (Radio Unblock): If silent > 60s, unblock rfkill
	if silenceDuration > 60*time.Second {
		fmt.Printf("[\033[1;33mWATCHDOG TIER-2\033[0m] Executing rfkill unblock bluetooth (Silence: %.0fs)...\n", silenceDuration.Seconds())
		_ = exec.Command("rfkill", "unblock", "bluetooth").Run()
	}

	// Tier 3 (Hardware Subsystem Reset): If silent > 180s, reset HCI adapter
	if silenceDuration > 180*time.Second {
		fmt.Printf("[\033[1;31mWATCHDOG TIER-3\033[0m] Executing hciconfig hci0 reset (Silence: %.0fs)...\n", silenceDuration.Seconds())
		_ = exec.Command("hciconfig", "hci0", "reset").Run()
		_ = exec.Command("bluetoothctl", "power", "off").Run()
		time.Sleep(500 * time.Millisecond)
		_ = exec.Command("bluetoothctl", "power", "on").Run()
	}
}

func startBluetoothWatchdog(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Printf("[\033[1;32mSUPERVISOR\033[0m] Autonomous Bluetooth, Outage Logger & 3-Tier Hardware Watchdog active.\n")

	watchdogAttempt := 0
	var nextReconnectAttempt time.Time

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

				// Self-heal Linux bluetooth subsystem with 3-tier escalation
				checkAndHealBluetoothSubsystem(elapsed)

				now := time.Now()
				if clientCount > 0 && (nextReconnectAttempt.IsZero() || now.After(nextReconnectAttempt)) {
					// Calculate exponential backoff with jitter (base 2s, factor 1.8, max 60s)
					watchdogAttempt++
					backoffSec := math.Min(60.0, 2.0*math.Pow(1.8, float64(watchdogAttempt)))
					jitterSec := float64(time.Now().UnixNano()%1000) / 1000.0 * 2.0
					nextReconnectAttempt = now.Add(time.Duration(backoffSec+jitterSec) * time.Second)

					// Instruct browser client to force-reconnect GATT session
					broadcastControlMsg(map[string]interface{}{
						"type":                      "watchdog_reconnect",
						"reason":                    "stalled_telemetry",
						"seconds_since_last_packet": int(elapsed.Seconds()),
						"reconnect_attempt":         watchdogAttempt,
						"next_backoff_sec":          int(backoffSec + jitterSec),
					})
				} else if clientCount == 0 {
					fmt.Printf("[\033[1;31mWATCHDOG\033[0m] No browser WebSocket connected to ws://localhost:%d. Open http://localhost:%d in Chrome.\n", wsPort, httpPort)
				}
			} else {
				// Reset backoff state when fresh telemetry is flowing
				watchdogAttempt = 0
				nextReconnectAttempt = time.Time{}

				// Send lightweight heartbeat ping with uptime stats
				stats, _ := tracker.GetStats()
				broadcastControlMsg(map[string]interface{}{
					"type":      "ping",
					"stats":     stats,
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				})
				tracker.save()
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

	// Send initial cloud upload status & spool count
	lastSucc, totalSucc, spoolCount := getCloudUploadStats()
	var lastSuccStr string
	if !lastSucc.IsZero() {
		lastSuccStr = lastSucc.Format(time.RFC3339)
	}
	if cloudSyncData, err := json.Marshal(map[string]interface{}{
		"type":                     "cloud_sync",
		"cloud_endpoint":           cloudEndpoint,
		"last_successful_upload":   lastSuccStr,
		"total_successful_uploads": totalSucc,
		"spool_count":              spoolCount,
	}); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, cloudSyncData)
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
			Token string `json:"token"`
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

		case "clear_outages":
			if !checkControlRateLimit(clientAddr, 500*time.Millisecond) {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"rate limit exceeded: slow down control commands"}`))
				continue
			}
			if !verifyBridgeAuth(r, payload.Token) {
				fmt.Printf("[\033[1;31mSECURITY\033[0m] Unauthorized clear_outages attempt from %s\n", clientAddr)
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"unauthorized: bridge token required for control actions"}`))
				continue
			}
			tracker.Clear()
			s, h := tracker.GetStats()
			broadcastControlMsg(map[string]interface{}{
				"type":    "outage_sync",
				"stats":   s,
				"history": h,
			})

		case "flash_profile", "write_register":
			if !checkControlRateLimit(clientAddr, 500*time.Millisecond) {
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"rate limit exceeded: slow down control commands"}`))
				continue
			}
			if !verifyBridgeAuth(r, payload.Token) {
				fmt.Printf("[\033[1;31mSECURITY\033[0m] Unauthorized %s attempt from %s\n", payload.Type, clientAddr)
				_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","error":"unauthorized: bridge token required for control actions"}`))
				continue
			}
			fmt.Printf("[\033[1;32mCONTROL\033[0m] Authenticated %s executed successfully from %s\n", payload.Type, clientAddr)
			_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"control_ack","status":"success"}`))
		}
	}
}

func main() {
	flag.Parse()

	loadEnv()
	initBridgeAuth()
	diskSpooler = NewDiskSpooler("spool")
	tracker.load()

	banner := `===========================================================================
SOLARIA: RENOGY BT-1 / BT-2 SOLAR & BLE GOLANG GATEWAY
===========================================================================
  • Site: %s (%.3f°N, %.3f°W)
  • Service UUID 0xFFD0 (Write Commands to FFD1)
  • Service UUID 0xFFF0 (Receive Telemetry from FFF1)
  • Automatic Stream Chunk Reassembly & CRC16 Engine
  • Disk Spooling & Resilience Engine: ACTIVE (spool/telemetry_spool.jsonl)
  • Bridge Session Security: ACTIVE (Token: %s...)
---------------------------------------------------------------------------
Open Dashboard: http://localhost:%d in Chrome on your device
===========================================================================`

	maskedToken := bridgeToken
	if len(maskedToken) > 8 {
		maskedToken = maskedToken[:8]
	}
	fmt.Printf(banner+"\n", siteName, siteLat, siteLon, maskedToken, httpPort)

	// 1. Start WebSocket Server on 8765
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/", handleWebSocket)
	wsServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", wsPort),
		Handler:           wsMux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		fmt.Printf("[WS] WebSocket Gateway listening on: ws://localhost:%d\n", wsPort)
		if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("WebSocket listener failed: %v", err)
		}
	}()

	// 2. Start HTTP Dashboard Server on 8080 (No-cache for instant UI updates)
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/api/v1/health", handleHealth)
	httpMux.HandleFunc("/api/v1/bridge-status", handleBridgeStatus)
	httpMux.HandleFunc("/api/v1/reload", handleReload)
	httpMux.HandleFunc("/api/v1/network-discovery", handleNetworkDiscovery)
	httpMux.HandleFunc("/api/v1/logs", handleLogs)
	httpMux.HandleFunc("/api/v1/diagnostics", handleDiagnostics)
	fs := http.FileServer(http.Dir("static"))
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fs.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", httpPort),
		Handler:           httpMux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		fmt.Printf("[HTTP] Solar Web Dashboard: \033[1;32mhttp://localhost:%d\033[0m\n", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 3. Start Bluetooth LE Watchdog, Spool Drainer, and Heartbeat Keepalive
	watchdogCtx, watchdogCancel := context.WithCancel(context.Background())
	defer watchdogCancel()

	go startSpoolDrainer(watchdogCtx)
	go startHeartbeatWorker(watchdogCtx)
	go startBluetoothWatchdog(watchdogCtx)

	// Listen for terminate and hot-reload signals
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			fmt.Println("\n[\033[1;35mHOT RELOAD\033[0m] SIGHUP signal received. Hot-reloading .env and authentication tokens...")
			loadEnv()
			initBridgeAuth()
			continue
		}
		break
	}

	fmt.Println("\nShutting down Solaria Golang Gateway...")
	watchdogCancel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

// NetworkDiscoveryInfo contains mDNS zero-config network discovery details
type NetworkDiscoveryInfo struct {
	Hostname        string   `json:"hostname"`
	MDNSDomain      string   `json:"mdns_domain"`
	MDNSURL         string   `json:"mdns_url"`
	ServiceType     string   `json:"service_type"`
	Port            int      `json:"port"`
	AvahiService    string   `json:"avahi_service"`
	BroadcastStatus string   `json:"broadcast_status"`
	LocalIPs        []string `json:"local_ips"`
}

func getLocalIPAddresses() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ips = append(ips, ipnet.IP.String())
				}
			}
		}
	}
	if len(ips) == 0 {
		ips = append(ips, "127.0.0.1")
	}
	return ips
}

func handleNetworkDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "solaria"
	}

	info := NetworkDiscoveryInfo{
		Hostname:        hostname,
		MDNSDomain:      "solaria.local",
		MDNSURL:         fmt.Sprintf("http://solaria.local:%d", httpPort),
		ServiceType:     "_http._tcp",
		Port:            httpPort,
		AvahiService:    "/etc/avahi/services/solaria.service",
		BroadcastStatus: "ACTIVE (Multicast DNS / Bonjour / Avahi Daemon)",
		LocalIPs:        getLocalIPAddresses(),
	}
	_ = json.NewEncoder(w).Encode(info)
}

func handleBridgeStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lastSucc, totalSucc, spoolCount := getCloudUploadStats()
	var lastSuccStr string
	if !lastSucc.IsZero() {
		lastSuccStr = lastSucc.Format(time.RFC3339)
	}

	frameMu.Lock()
	lastFrame := lastFrameTime
	totalFrames := totalFramesProcessed
	frameMu.Unlock()

	var lastFrameStr string
	if !lastFrame.IsZero() {
		lastFrameStr = lastFrame.Format(time.RFC3339)
	}

	stats, hist := tracker.GetStats()

	resp := map[string]interface{}{
		"site": siteName,
		"location": map[string]interface{}{
			"latitude":  siteLat,
			"longitude": siteLon,
		},
		"cloud_endpoint":           cloudEndpoint,
		"last_successful_upload":   lastSuccStr,
		"total_successful_uploads": totalSucc,
		"last_ble_packet_time":     lastFrameStr,
		"total_ble_frames":         totalFrames,
		"spool_count":              spoolCount,
		"outage_stats":             stats,
		"outage_history":           hist,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lastSucc, totalSucc, spoolCount := getCloudUploadStats()
	frameMu.Lock()
	elapsedFrame := time.Since(lastFrameTime).Seconds()
	totalFrames := totalFramesProcessed
	frameMu.Unlock()

	status := "healthy"
	if spoolCount > 100 {
		status = "degraded"
	}

	resp := map[string]interface{}{
		"status":                    status,
		"uptime_seconds":            int(time.Since(tracker.sessionStart).Seconds()),
		"spool_count":               spoolCount,
		"total_uploads":             totalSucc,
		"last_upload_seconds_ago":   int(time.Since(lastSucc).Seconds()),
		"total_frames":              totalFrames,
		"last_frame_seconds_ago":    int(elapsedFrame),
		"cloud_endpoint_configured": cloudEndpoint != "",
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !verifyBridgeAuth(r, "") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	loadEnv()
	initBridgeAuth()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Configuration and authentication credentials reloaded successfully",
	})
}

func handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	level := r.URL.Query().Get("level")
	subsystem := r.URL.Query().Get("subsystem")
	search := r.URL.Query().Get("search")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 1000 {
			limit = val
		}
	}

	logs := bridgeLogger.GetLogs(level, subsystem, search, limit)
	stats := bridgeLogger.GetStats()

	resp := map[string]interface{}{
		"status":    "ok",
		"service":   "solaria-bridge",
		"stats":     stats,
		"count":     len(logs),
		"logs":      logs,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	lastSucc, totalSucc, spoolCount := getCloudUploadStats()
	frameMu.Lock()
	elapsedFrame := time.Since(lastFrameTime).Seconds()
	totalFrames := totalFramesProcessed
	frameMu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	recentErrors := bridgeLogger.GetLogs("ERROR", "", "", 20)
	stats, hist := tracker.GetStats()

	var lastSuccStr string
	if !lastSucc.IsZero() {
		lastSuccStr = lastSucc.Format(time.RFC3339)
	}

	diag := map[string]interface{}{
		"service":        "solaria-bridge",
		"version":        "2.0-rover-400w",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"uptime_seconds": int(time.Since(tracker.sessionStart).Seconds()),
		"health": map[string]interface{}{
			"overall": func() string {
				if spoolCount > 100 || (elapsedFrame > 120 && totalFrames > 0) {
					return "DEGRADED"
				}
				return "HEALTHY"
			}(),
			"modbus_link": func() string {
				if elapsedFrame > 60 && totalFrames > 0 {
					return "SILENT"
				}
				return "NOMINAL"
			}(),
			"cloud_uplink": func() string {
				if time.Since(lastSucc) > 60*time.Second && totalSucc > 0 {
					return "DEGRADED"
				}
				return "NOMINAL"
			}(),
			"spooler": func() string {
				if spoolCount > 0 {
					return fmt.Sprintf("BUFFERING (%d queued)", spoolCount)
				}
				return "NOMINAL (0 queued)"
			}(),
		},
		"modbus": map[string]interface{}{
			"total_frames_decoded":   totalFrames,
			"last_frame_seconds_ago": int(elapsedFrame),
			"last_frame_timestamp":   lastFrameTime.Format(time.RFC3339),
		},
		"cloud_uploader": map[string]interface{}{
			"endpoint":                cloudEndpoint,
			"total_uploads":           totalSucc,
			"last_upload_seconds_ago": int(time.Since(lastSucc).Seconds()),
			"last_upload_timestamp":   lastSuccStr,
		},
		"spooler": map[string]interface{}{
			"spool_count": spoolCount,
			"spool_file":  "spool/telemetry_spool.jsonl",
		},
		"outages": map[string]interface{}{
			"stats":   stats,
			"history": hist,
		},
		"runtime": map[string]interface{}{
			"alloc_mb":       float64(m.Alloc) / 1024 / 1024,
			"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
			"sys_mb":         float64(m.Sys) / 1024 / 1024,
			"num_gc":         m.NumGC,
			"goroutines":     runtime.NumGoroutine(),
		},
		"log_stats":     bridgeLogger.GetStats(),
		"recent_errors": recentErrors,
	}

	_ = json.NewEncoder(w).Encode(diag)
}
