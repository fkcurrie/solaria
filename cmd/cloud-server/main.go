package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
)

//go:embed templates/*
var templateFS embed.FS

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

	// Array Performance Metrics (4x100W 2S2P Array = 400Wp)
	ArrayCapacityW      int     `json:"array_capacity_w"`
	ArrayTopology       string  `json:"array_topology"`
	ArrayUtilizationPct float64 `json:"array_utilization_pct"`
	PerformanceRatioPct float64 `json:"performance_ratio_pct"`

	// Daily Min / Max & Lifetime Metrics
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

	OperatingDays             int `json:"operating_days"`
	TotalBatteryOverDischarge int `json:"total_battery_overdischarge_count"`
	TotalBatteryFullCharge    int `json:"total_battery_fullcharge_count"`
	TotalChargingAh           int `json:"total_charging_ah"`
	TotalDischargingAh        int `json:"total_discharging_ah"`
	TotalGeneratedKWh         int `json:"total_generated_kwh"`
	TotalConsumedKWh          int `json:"total_consumed_kwh"`

	FaultBits  int    `json:"fault_bits"`
	FaultFlags string `json:"fault_flags"`
}

type WeatherMetrics struct {
	TemperatureC        float64 `json:"temperature_c"`
	CloudCoverPct       int     `json:"cloud_cover_pct"`
	GHIWM2              float64 `json:"ghi_w_m2"`
	DNIWM2              float64 `json:"dni_w_m2"`
	DirectRadiationWM2  float64 `json:"direct_radiation_w_m2"`
	DiffuseRadiationWM2 float64 `json:"diffuse_radiation_w_m2"`
	IsDay               bool    `json:"is_day"`
}

type SolarRecord struct {
	Timestamp         string             `json:"timestamp"`
	Site              string             `json:"site"`
	Location          map[string]float64 `json:"location"`
	Telemetry         Telemetry          `json:"telemetry"`
	Weather           WeatherMetrics     `json:"weather"`
	SunClassification string             `json:"sun_classification"`
}

// BQRecord represents the complete flattened schema for BigQuery timeseries analytics
type BQRecord struct {
	Timestamp time.Time `bigquery:"timestamp"`
	Site      string    `bigquery:"site"`
	Latitude  float64   `bigquery:"latitude"`
	Longitude float64   `bigquery:"longitude"`

	// Solar Array Specifications (4x100W 2S2P = 400W)
	ArrayCapacityW      int64   `bigquery:"array_capacity_w"`
	ArrayTopology       string  `bigquery:"array_topology"`
	ArrayUtilizationPct float64 `bigquery:"array_utilization_pct"`
	PerformanceRatioPct float64 `bigquery:"performance_ratio_pct"`

	// Real-Time Telemetry
	PVPowerW        int64   `bigquery:"pv_power_w"`
	PVVoltageV      float64 `bigquery:"pv_voltage_v"`
	PVCurrentA      float64 `bigquery:"pv_current_a"`
	BatterySOCPct   int64   `bigquery:"battery_soc_pct"`
	BatteryVoltageV float64 `bigquery:"battery_voltage_v"`
	BatteryCurrentA float64 `bigquery:"battery_current_a"`
	ControllerTempC int64   `bigquery:"controller_temp_c"`
	BatteryTempC    int64   `bigquery:"battery_temp_c"`
	LoadPowerW      int64   `bigquery:"load_power_w"`
	LoadVoltageV    float64 `bigquery:"load_voltage_v"`
	LoadCurrentA    float64 `bigquery:"load_current_a"`
	LoadStatus      bool    `bigquery:"load_status"`
	ChargingState   string  `bigquery:"charging_state"`

	// Daily Min / Max & Cycle Counters
	DailyMinBatteryVoltageV     float64 `bigquery:"daily_min_battery_voltage_v"`
	DailyMaxBatteryVoltageV     float64 `bigquery:"daily_max_battery_voltage_v"`
	DailyMaxChargingCurrentA    float64 `bigquery:"daily_max_charging_current_a"`
	DailyMaxDischargingCurrentA float64 `bigquery:"daily_max_discharging_current_a"`
	DailyMaxPVWatts             int64   `bigquery:"daily_max_pv_w"`
	DailyMaxLoadWatts           int64   `bigquery:"daily_max_load_w"`
	DailyChargingAh             int64   `bigquery:"daily_charging_ah"`
	DailyDischargingAh          int64   `bigquery:"daily_discharging_ah"`
	DailyGeneratedWh            int64   `bigquery:"daily_generated_wh"`
	DailyConsumedWh             int64   `bigquery:"daily_consumed_wh"`

	// Lifetime Statistics & Health Counters
	OperatingDays             int64 `bigquery:"operating_days"`
	TotalBatteryOverDischarge int64 `bigquery:"total_battery_overdischarge_count"`
	TotalBatteryFullCharge    int64 `bigquery:"total_battery_fullcharge_count"`
	TotalChargingAh           int64 `bigquery:"total_charging_ah"`
	TotalDischargingAh        int64 `bigquery:"total_discharging_ah"`
	TotalGeneratedKWh         int64 `bigquery:"total_generated_kwh"`
	TotalConsumedKWh          int64 `bigquery:"total_consumed_kwh"`

	// Faults & Diagnostics
	FaultBits  int64  `bigquery:"fault_bits"`
	FaultFlags string `bigquery:"fault_flags"`

	// Atmospheric & Weather Enrichment
	WeatherTempC      float64 `bigquery:"weather_temp_c"`
	CloudCoverPct     int64   `bigquery:"weather_cloud_cover_pct"`
	DirectRadWM2      float64 `bigquery:"weather_direct_rad_w_m2"`
	DiffuseRadWM2     float64 `bigquery:"weather_diffuse_rad_w_m2"`
	IsDay             bool    `bigquery:"weather_is_day"`
	SunClassification string  `bigquery:"sun_classification"`
}

type IngestBatch struct {
	Batch []SolarRecord `json:"batch"`
}

type RingBuffer struct {
	mu      sync.RWMutex
	records []SolarRecord
	latest  SolarRecord
	maxCap  int
}

func NewRingBuffer(maxCap int) *RingBuffer {
	initial := SolarRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Site:      "1296 Wren Lake Drive, Dorset, ON",
		Telemetry: Telemetry{
			PVPowerW:            0,
			PVVoltageV:          13.0,
			BatterySOCPct:       80,
			BatteryVoltageV:     12.7,
			ControllerTempC:     27,
			ChargingState:       "MPPT Charging",
			ArrayCapacityW:      400,
			ArrayTopology:       "2S2P (4x100W)",
			ArrayUtilizationPct: 0.0,
			TotalGeneratedKWh:   8363,
			FaultFlags:          "NORMAL",
		},
		Weather: WeatherMetrics{
			TemperatureC:  18.0,
			CloudCoverPct: 15,
		},
		SunClassification: "NIGHT",
	}
	return &RingBuffer{
		records: []SolarRecord{initial},
		latest:  initial,
		maxCap:  maxCap,
	}
}

func (r *RingBuffer) Push(items []SolarRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, it := range items {
		// Populate array 400W 2S2P metadata if missing
		if it.Telemetry.ArrayCapacityW == 0 {
			it.Telemetry.ArrayCapacityW = 400
			it.Telemetry.ArrayTopology = "2S2P (4x100W)"
		}
		if it.Telemetry.ArrayCapacityW > 0 {
			it.Telemetry.ArrayUtilizationPct = (float64(it.Telemetry.PVPowerW) / float64(it.Telemetry.ArrayCapacityW)) * 100.0
		}
		totalRad := it.Weather.DirectRadiationWM2 + it.Weather.DiffuseRadiationWM2
		if totalRad > 20 && it.Telemetry.ArrayCapacityW > 0 {
			expectedW := (totalRad / 1000.0) * float64(it.Telemetry.ArrayCapacityW)
			it.Telemetry.PerformanceRatioPct = (float64(it.Telemetry.PVPowerW) / expectedW) * 100.0
		}
		r.records = append(r.records, it)
		r.latest = it
		if len(r.records) > r.maxCap {
			r.records = r.records[1:]
		}
	}
}

func (r *RingBuffer) GetLatest() SolarRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.latest
}

func (r *RingBuffer) GetHistory(limit int) []SolarRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := len(r.records)
	if limit > n {
		limit = n
	}
	return r.records[n-limit:]
}

var (
	ringBuf    = NewRingBuffer(1440)
	apiToken   = "solaria_cottage_secret_token_2026"
	gcpProject = "solaria-solar"
	bqClient   *bigquery.Client
	bqTable    *bigquery.Table
	tmpl       *template.Template
)

func init() {
	if envTok := os.Getenv("SOLARIA_API_TOKEN"); envTok != "" {
		apiToken = envTok
	}
	if envProj := os.Getenv("GCP_PROJECT"); envProj != "" {
		gcpProject = envProj
	}
	t, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		log.Printf("Template parse note: %v", err)
	}
	tmpl = t

	// Initialize BigQuery Client & Dataset / Table Schema
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, gcpProject)
	if err == nil {
		bqClient = client
		dataset := bqClient.Dataset("solaria")
		if err := dataset.Create(ctx, &bigquery.DatasetMetadata{Location: "US"}); err != nil && !strings.Contains(err.Error(), "Already Exists") {
			log.Printf("BigQuery dataset note: %v", err)
		}

		bqTable = dataset.Table("telemetry")
		schema, err := bigquery.InferSchema(BQRecord{})
		if err == nil {
			tableMetadata := &bigquery.TableMetadata{
				Schema: schema,
				TimePartitioning: &bigquery.TimePartitioning{
					Field: "timestamp",
					Type:  bigquery.DayPartitioningType,
				},
				Clustering: &bigquery.Clustering{
					Fields: []string{"site", "sun_classification"},
				},
			}
			if err := bqTable.Create(ctx, tableMetadata); err != nil && !strings.Contains(err.Error(), "Already Exists") {
				log.Printf("BigQuery table create note: %v", err)
			}
		}
		log.Printf("✅ Google Cloud BigQuery client initialized for dataset 'solaria.telemetry' in project: %s", gcpProject)
	} else {
		log.Printf("BigQuery note: %v", err)
	}
}

func verifyAuth(r *http.Request) bool {
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" && apiKey == apiToken {
		return true
	}
	auth := r.Header.Get("Authorization")
	if auth != "" {
		token := strings.TrimPrefix(auth, "Bearer ")
		return token == apiToken
	}
	return false
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if tmpl != nil {
		_ = tmpl.Execute(w, nil)
	} else {
		fmt.Fprintf(w, "Solaria Dashboard Running")
	}
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if !verifyAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var batch IngestBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	if len(batch.Batch) > 0 {
		ringBuf.Push(batch.Batch)

		// Stream to BigQuery asynchronously
		if bqTable != nil {
			go func(items []SolarRecord) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				var bqRows []*BQRecord
				for _, it := range items {
					ts, err := time.Parse(time.RFC3339, it.Timestamp)
					if err != nil {
						ts = time.Now().UTC()
					}
					lat := 45.186
					lon := -78.863
					if it.Location != nil {
						if v, ok := it.Location["latitude"]; ok {
							lat = v
						}
						if v, ok := it.Location["longitude"]; ok {
							lon = v
						}
					}
					arrCap := int64(it.Telemetry.ArrayCapacityW)
					if arrCap == 0 {
						arrCap = 400
					}
					arrTop := it.Telemetry.ArrayTopology
					if arrTop == "" {
						arrTop = "2S2P (4x100W)"
					}
					arrUtil := (float64(it.Telemetry.PVPowerW) / float64(arrCap)) * 100.0
					perfRatio := 0.0
					totalRad := it.Weather.DirectRadiationWM2 + it.Weather.DiffuseRadiationWM2
					if totalRad > 20 {
						expectedW := (totalRad / 1000.0) * float64(arrCap)
						perfRatio = (float64(it.Telemetry.PVPowerW) / expectedW) * 100.0
					}

					row := &BQRecord{
						Timestamp:                   ts,
						Site:                        it.Site,
						Latitude:                    lat,
						Longitude:                   lon,
						ArrayCapacityW:              arrCap,
						ArrayTopology:               arrTop,
						ArrayUtilizationPct:         arrUtil,
						PerformanceRatioPct:         perfRatio,
						PVPowerW:                    int64(it.Telemetry.PVPowerW),
						PVVoltageV:                  it.Telemetry.PVVoltageV,
						PVCurrentA:                  it.Telemetry.PVCurrentA,
						BatterySOCPct:               int64(it.Telemetry.BatterySOCPct),
						BatteryVoltageV:             it.Telemetry.BatteryVoltageV,
						BatteryCurrentA:             it.Telemetry.BatteryCurrentA,
						ControllerTempC:             int64(it.Telemetry.ControllerTempC),
						BatteryTempC:                int64(it.Telemetry.BatteryTempC),
						LoadPowerW:                  int64(it.Telemetry.LoadPowerW),
						LoadVoltageV:                it.Telemetry.LoadVoltageV,
						LoadCurrentA:                it.Telemetry.LoadCurrentA,
						LoadStatus:                  it.Telemetry.LoadStatus,
						ChargingState:               it.Telemetry.ChargingState,
						DailyMinBatteryVoltageV:     it.Telemetry.DailyMinBatteryVoltageV,
						DailyMaxBatteryVoltageV:     it.Telemetry.DailyMaxBatteryVoltageV,
						DailyMaxChargingCurrentA:    it.Telemetry.DailyMaxChargingCurrentA,
						DailyMaxDischargingCurrentA: it.Telemetry.DailyMaxDischargingCurrentA,
						DailyMaxPVWatts:             int64(it.Telemetry.DailyMaxPVWatts),
						DailyMaxLoadWatts:           int64(it.Telemetry.DailyMaxLoadWatts),
						DailyChargingAh:             int64(it.Telemetry.DailyChargingAh),
						DailyDischargingAh:          int64(it.Telemetry.DailyDischargingAh),
						DailyGeneratedWh:            int64(it.Telemetry.DailyGeneratedWh),
						DailyConsumedWh:             int64(it.Telemetry.DailyConsumedWh),
						OperatingDays:               int64(it.Telemetry.OperatingDays),
						TotalBatteryOverDischarge:   int64(it.Telemetry.TotalBatteryOverDischarge),
						TotalBatteryFullCharge:      int64(it.Telemetry.TotalBatteryFullCharge),
						TotalChargingAh:             int64(it.Telemetry.TotalChargingAh),
						TotalDischargingAh:          int64(it.Telemetry.TotalDischargingAh),
						TotalGeneratedKWh:           int64(it.Telemetry.TotalGeneratedKWh),
						TotalConsumedKWh:            int64(it.Telemetry.TotalConsumedKWh),
						FaultBits:                   int64(it.Telemetry.FaultBits),
						FaultFlags:                  it.Telemetry.FaultFlags,
						WeatherTempC:                it.Weather.TemperatureC,
						CloudCoverPct:               int64(it.Weather.CloudCoverPct),
						DirectRadWM2:                it.Weather.DirectRadiationWM2,
						DiffuseRadWM2:               it.Weather.DiffuseRadiationWM2,
						IsDay:                       it.Weather.IsDay,
						SunClassification:           it.SunClassification,
					}
					bqRows = append(bqRows, row)
				}
				inserter := bqTable.Inserter()
				if err := inserter.Put(ctx, bqRows); err != nil {
					log.Printf("[BigQuery] Streaming insert error: %v", err)
				}
			}(batch.Batch)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"ingested": len(batch.Batch),
	})
}

func handleLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(ringBuf.GetLatest())
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	limit := 60
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(ringBuf.GetHistory(limit))
}

func handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	latest := ringBuf.GetLatest()
	telem := latest.Telemetry

	info := map[string]interface{}{
		"device": map[string]interface{}{
			"model":                   "Renogy Rover 20A MPPT (RNG-CTRL-RVR20)",
			"bt_module":               "Renogy BT-1 (BT-TH-66F984D6)",
			"communication":           "BLE GATT 0xFFD0 / RS232 Modbus RTU",
			"software_version":        "v1.0.4",
			"hardware_version":        "v1.0.0",
			"rated_pv_voltage":        "100V DC Max",
			"rated_charge_current":    "20A Max",
			"rated_discharge_current": "20A Max",
		},
		"solar_array": map[string]interface{}{
			"peak_capacity_w":   400,
			"topology":          "2S2P (4 x 100W Monocrystalline Panels)",
			"series_strings":    2,
			"panels_per_string": 2,
			"string_vmp":        "36.0V - 40.8V",
			"array_voc":         "43.2V - 48.6V",
			"array_imp":         "9.8A - 11.0A",
			"overpaneling_pct":  138.0,
			"site_name":         "1296 Wren Lake Drive",
			"location":          "Dorset, Ontario, Canada (45.186° N, -78.863° W)",
			"elevation_m":       350,
		},
		"battery_bank": map[string]interface{}{
			"system_voltage":           "12V DC Nominal",
			"battery_type":             "Deep-Cycle / AGM / Lithium",
			"boost_voltage_v":          14.4,
			"float_voltage_v":          13.8,
			"equalize_voltage_v":       14.6,
			"overvoltage_disconnect_v": 16.0,
			"low_voltage_disconnect_v": 11.1,
		},
		"lifetime_statistics": map[string]interface{}{
			"operating_days":                    telem.OperatingDays,
			"total_generated_kwh":               telem.TotalGeneratedKWh,
			"total_charging_ah":                 telem.TotalChargingAh,
			"total_battery_fullcharge_count":    telem.TotalBatteryFullCharge,
			"total_battery_overdischarge_count": telem.TotalBatteryOverDischarge,
			"total_discharging_ah":              telem.TotalDischargingAh,
			"total_consumed_kwh":                telem.TotalConsumedKWh,
			"fault_register":                    fmt.Sprintf("0x%04X (%s)", telem.FaultBits, telem.FaultFlags),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(info)
}

func classifyWeather(cloudPct float64, tempC float64, isDay bool, avgIrr float64, sunClass string) (string, string) {
	if !isDay || (avgIrr < 5.0 && (sunClass == "NIGHT" || sunClass == "")) {
		return "🌙", "Night / Dark"
	}
	if strings.Contains(sunClass, "RAIN") || strings.Contains(sunClass, "STORM") {
		return "🌧️", fmt.Sprintf("Rain (%.0f%% clouds, %.1f°C)", cloudPct, tempC)
	}
	if cloudPct <= 20 {
		return "☀️", fmt.Sprintf("Sunny / Clear (%.0f%% clouds, %.1f°C)", cloudPct, tempC)
	}
	if cloudPct <= 60 {
		return "⛅", fmt.Sprintf("Partly Cloudy (%.0f%% clouds, %.1f°C)", cloudPct, tempC)
	}
	if cloudPct <= 85 {
		return "🌥️", fmt.Sprintf("Mostly Cloudy (%.0f%% clouds, %.1f°C)", cloudPct, tempC)
	}
	return "☁️", fmt.Sprintf("Overcast (%.0f%% clouds, %.1f°C)", cloudPct, tempC)
}

func handleDayStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	loc, err := time.LoadLocation("America/Toronto")
	if err != nil {
		loc = time.FixedZone("EDT", -4*3600)
	}
	nowLocal := time.Now().In(loc)

	hours := make([]string, 24)
	genWh := make([]interface{}, 24)
	irradiance := make([]interface{}, 24)
	battSOC := make([]interface{}, 24)
	weatherIcons := make([]interface{}, 24)
	weatherConds := make([]interface{}, 24)
	cloudPct := make([]interface{}, 24)
	tempC := make([]interface{}, 24)
	for i := 0; i < 24; i++ {
		hours[i] = fmt.Sprintf("%02d:00", i)
		genWh[i] = nil
		irradiance[i] = nil
		battSOC[i] = nil
		weatherIcons[i] = nil
		weatherConds[i] = nil
		cloudPct[i] = nil
		tempC[i] = nil
	}

	recordsCount := 0
	peakWatts := 0
	totalWh := 0.0

	if bqClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		q := bqClient.Query(fmt.Sprintf(`
			SELECT 
				EXTRACT(HOUR FROM timestamp AT TIME ZONE "America/Toronto") as hour,
				AVG(pv_power_w) as avg_pv_w,
				MAX(pv_power_w) as max_pv_w,
				AVG(weather_direct_rad_w_m2 + weather_diffuse_rad_w_m2) as avg_irr,
				AVG(battery_soc_pct) as avg_soc,
				AVG(weather_cloud_cover_pct) as avg_cloud,
				AVG(weather_temp_c) as avg_temp,
				LOGICAL_OR(weather_is_day) as is_day,
				ARRAY_AGG(sun_classification ORDER BY timestamp DESC LIMIT 1)[OFFSET(0)] as sun_class,
				SUM(pv_power_w * (5.0 / 3600.0)) as est_wh,
				COUNT(*) as samples
			FROM `+"`%s.solaria.telemetry`"+`
			WHERE DATE(timestamp, "America/Toronto") = CURRENT_DATE("America/Toronto")
			GROUP BY hour
			ORDER BY hour
		`, gcpProject))

		it, err := q.Read(ctx)
		if err == nil {
			for {
				var row struct {
					Hour     int64   `bigquery:"hour"`
					AvgPvW   float64 `bigquery:"avg_pv_w"`
					MaxPvW   int64   `bigquery:"max_pv_w"`
					AvgIrr   float64 `bigquery:"avg_irr"`
					AvgSOC   float64 `bigquery:"avg_soc"`
					AvgCloud float64 `bigquery:"avg_cloud"`
					AvgTemp  float64 `bigquery:"avg_temp"`
					IsDay    bool    `bigquery:"is_day"`
					SunClass string  `bigquery:"sun_class"`
					EstWh    float64 `bigquery:"est_wh"`
					Samples  int64   `bigquery:"samples"`
				}
				err := it.Next(&row)
				if err != nil {
					break
				}
				h := int(row.Hour)
				if h >= 0 && h < 24 {
					genWh[h] = mathRound(row.EstWh, 1)
					irradiance[h] = mathRound(row.AvgIrr, 1)
					battSOC[h] = int(row.AvgSOC + 0.5)
					cloudPct[h] = int(row.AvgCloud + 0.5)
					tempC[h] = mathRound(row.AvgTemp, 1)

					icon, cond := classifyWeather(row.AvgCloud, row.AvgTemp, row.IsDay, row.AvgIrr, row.SunClass)
					weatherIcons[h] = icon
					weatherConds[h] = cond

					recordsCount += int(row.Samples)
					if int(row.MaxPvW) > peakWatts {
						peakWatts = int(row.MaxPvW)
					}
					totalWh += row.EstWh
				}
			}
		}
	}

	// If BigQuery had no data (e.g. streaming just began), check ring buffer for today's points
	if recordsCount == 0 {
		history := ringBuf.GetHistory(1440)
		for _, item := range history {
			t, err := time.Parse(time.RFC3339, item.Timestamp)
			if err == nil && t.In(loc).Format("2006-01-02") == nowLocal.Format("2006-01-02") {
				h := t.In(loc).Hour()
				if h >= 0 && h < 24 {
					genWh[h] = float64(item.Telemetry.DailyGeneratedWh)
					irradiance[h] = item.Weather.DirectRadiationWM2 + item.Weather.DiffuseRadiationWM2
					battSOC[h] = item.Telemetry.BatterySOCPct
					cloudPct[h] = item.Weather.CloudCoverPct
					tempC[h] = item.Weather.TemperatureC

					icon, cond := classifyWeather(float64(item.Weather.CloudCoverPct), item.Weather.TemperatureC, item.Weather.IsDay, item.Weather.DirectRadiationWM2+item.Weather.DiffuseRadiationWM2, item.SunClassification)
					weatherIcons[h] = icon
					weatherConds[h] = cond

					recordsCount++
					if item.Telemetry.PVPowerW > peakWatts {
						peakWatts = item.Telemetry.PVPowerW
					}
				}
			}
		}
	}

	resp := map[string]interface{}{
		"date":               nowLocal.Format("Monday, Jan 02, 2006"),
		"hours":              hours,
		"generation_wh":      genWh,
		"irradiance_w_m2":    irradiance,
		"battery_soc_pct":    battSOC,
		"weather_icons":      weatherIcons,
		"weather_conditions": weatherConds,
		"cloud_cover_pct":    cloudPct,
		"temperature_c":      tempC,
		"total_yield_wh":     mathRound(totalWh, 1),
		"peak_watts":         peakWatts,
		"data_available":     recordsCount > 0,
		"sample_count":       recordsCount,
		"status_message":     fmt.Sprintf("Streaming live data: %d samples recorded today.", recordsCount),
	}
	json.NewEncoder(w).Encode(resp)
}

func handleWeekStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	now := time.Now()
	daysMap := make(map[string]map[string]interface{})
	daysList := []string{}
	for i := 6; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format("2006-01-02")
		daysList = append(daysList, d)
		daysMap[d] = map[string]interface{}{
			"label":      now.AddDate(0, 0, -i).Format("Mon 01/02"),
			"yield_kwh":  nil,
			"peak_watts": nil,
			"min_batt_v": nil,
			"max_batt_v": nil,
			"samples":    0,
		}
	}

	totalDaysWithData := 0
	if bqClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		q := bqClient.Query(fmt.Sprintf(`
			SELECT 
				CAST(DATE(timestamp, "America/Toronto") AS STRING) as log_date,
				MAX(daily_generated_wh) / 1000.0 as yield_kwh,
				MAX(pv_power_w) as peak_w,
				MIN(battery_voltage_v) as min_batt_v,
				MAX(battery_voltage_v) as max_batt_v,
				COUNT(*) as samples
			FROM `+"`%s.solaria.telemetry`"+`
			WHERE DATE(timestamp, "America/Toronto") >= DATE_SUB(CURRENT_DATE("America/Toronto"), INTERVAL 7 DAY)
			GROUP BY log_date
			ORDER BY log_date
		`, gcpProject))

		it, err := q.Read(ctx)
		if err == nil {
			for {
				var row struct {
					LogDate  string  `bigquery:"log_date"`
					YieldKWh float64 `bigquery:"yield_kwh"`
					PeakW    int64   `bigquery:"peak_w"`
					MinBattV float64 `bigquery:"min_batt_v"`
					MaxBattV float64 `bigquery:"max_batt_v"`
					Samples  int64   `bigquery:"samples"`
				}
				err := it.Next(&row)
				if err != nil {
					break
				}
				if entry, ok := daysMap[row.LogDate]; ok {
					entry["yield_kwh"] = mathRound(row.YieldKWh, 2)
					entry["peak_watts"] = row.PeakW
					entry["min_batt_v"] = mathRound(row.MinBattV, 1)
					entry["max_batt_v"] = mathRound(row.MaxBattV, 1)
					entry["samples"] = row.Samples
					totalDaysWithData++
				}
			}
		}
	}

	labels := []string{}
	yieldKWh := []interface{}{}
	peakWatts := []interface{}{}
	minBattV := []interface{}{}
	maxBattV := []interface{}{}

	for _, d := range daysList {
		entry := daysMap[d]
		labels = append(labels, entry["label"].(string))
		yieldKWh = append(yieldKWh, entry["yield_kwh"])
		peakWatts = append(peakWatts, entry["peak_watts"])
		minBattV = append(minBattV, entry["min_batt_v"])
		maxBattV = append(maxBattV, entry["max_batt_v"])
	}

	resp := map[string]interface{}{
		"days":           labels,
		"yield_kwh":      yieldKWh,
		"peak_watts":     peakWatts,
		"min_batt_v":     minBattV,
		"max_batt_v":     maxBattV,
		"data_available": totalDaysWithData > 0,
		"days_with_data": totalDaysWithData,
		"status_message": fmt.Sprintf("%d of 7 days logged in BigQuery since setup on %s.", totalDaysWithData, now.Format("Jan 02, 2006")),
	}
	json.NewEncoder(w).Encode(resp)
}

func handleMonthStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	now := time.Now()
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysList := []string{}
	dailyKWh := make([]interface{}, daysInMonth)
	cumulativeKWh := make([]interface{}, daysInMonth)
	perfRatio := make([]interface{}, daysInMonth)

	for d := 1; d <= daysInMonth; d++ {
		daysList = append(daysList, fmt.Sprintf("%s %02d", now.Format("Jan"), d))
		dailyKWh[d-1] = nil
		cumulativeKWh[d-1] = nil
		perfRatio[d-1] = nil
	}

	totalDaysWithData := 0
	if bqClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		q := bqClient.Query(fmt.Sprintf(`
			SELECT 
				EXTRACT(DAY FROM timestamp AT TIME ZONE "America/Toronto") as day_num,
				MAX(daily_generated_wh) / 1000.0 as yield_kwh,
				AVG(performance_ratio_pct) as avg_pr,
				COUNT(*) as samples
			FROM `+"`%s.solaria.telemetry`"+`
			WHERE EXTRACT(MONTH FROM timestamp AT TIME ZONE "America/Toronto") = EXTRACT(MONTH FROM CURRENT_DATE("America/Toronto"))
			  AND EXTRACT(YEAR FROM timestamp AT TIME ZONE "America/Toronto") = EXTRACT(YEAR FROM CURRENT_DATE("America/Toronto"))
			GROUP BY day_num
			ORDER BY day_num
		`, gcpProject))

		it, err := q.Read(ctx)
		if err == nil {
			cumul := 0.0
			for {
				var row struct {
					DayNum   int64   `bigquery:"day_num"`
					YieldKWh float64 `bigquery:"yield_kwh"`
					AvgPR    float64 `bigquery:"avg_pr"`
					Samples  int64   `bigquery:"samples"`
				}
				err := it.Next(&row)
				if err != nil {
					break
				}
				idx := int(row.DayNum) - 1
				if idx >= 0 && idx < daysInMonth {
					cumul += row.YieldKWh
					dailyKWh[idx] = mathRound(row.YieldKWh, 2)
					cumulativeKWh[idx] = mathRound(cumul, 2)
					perfRatio[idx] = mathRound(row.AvgPR, 1)
					totalDaysWithData++
				}
			}
		}
	}

	resp := map[string]interface{}{
		"month":          now.Format("January 2006"),
		"days":           daysList,
		"daily_kwh":      dailyKWh,
		"cumulative_kwh": cumulativeKWh,
		"perf_ratio_pct": perfRatio,
		"data_available": totalDaysWithData > 0,
		"days_with_data": totalDaysWithData,
		"status_message": fmt.Sprintf("%d of %d days logged for %s.", totalDaysWithData, daysInMonth, now.Format("January 2006")),
	}
	json.NewEncoder(w).Encode(resp)
}

func handleYearStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	now := time.Now()
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	monthlyKWh := make([]interface{}, 12)
	cumulMWh := make([]interface{}, 12)
	for i := 0; i < 12; i++ {
		monthlyKWh[i] = nil
		cumulMWh[i] = nil
	}

	totalMonthsWithData := 0
	if bqClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		q := bqClient.Query(fmt.Sprintf(`
			SELECT 
				EXTRACT(MONTH FROM timestamp AT TIME ZONE "America/Toronto") as month_num,
				MAX(total_generated_kwh) - MIN(total_generated_kwh) as month_kwh,
				COUNT(*) as samples
			FROM `+"`%s.solaria.telemetry`"+`
			WHERE EXTRACT(YEAR FROM timestamp AT TIME ZONE "America/Toronto") = EXTRACT(YEAR FROM CURRENT_DATE("America/Toronto"))
			GROUP BY month_num
			ORDER BY month_num
		`, gcpProject))

		it, err := q.Read(ctx)
		if err == nil {
			runningMWh := 0.0
			for {
				var row struct {
					MonthNum int64   `bigquery:"month_num"`
					MonthKWh float64 `bigquery:"month_kwh"`
					Samples  int64   `bigquery:"samples"`
				}
				err := it.Next(&row)
				if err != nil {
					break
				}
				idx := int(row.MonthNum) - 1
				if idx >= 0 && idx < 12 {
					monthlyKWh[idx] = mathRound(row.MonthKWh, 1)
					runningMWh += (row.MonthKWh / 1000.0)
					cumulMWh[idx] = mathRound(runningMWh, 3)
					totalMonthsWithData++
				}
			}
		}
	}

	resp := map[string]interface{}{
		"year":             now.Year(),
		"months":           months,
		"monthly_kwh":      monthlyKWh,
		"cumulative_mwh":   cumulMWh,
		"data_available":   totalMonthsWithData > 0,
		"months_with_data": totalMonthsWithData,
		"status_message":   fmt.Sprintf("%d of 12 months logged in %d.", totalMonthsWithData, now.Year()),
	}
	json.NewEncoder(w).Encode(resp)
}

func mathSinFactor(h int) float64 {
	// Peak at solar noon (h=13)
	val := float64(h-6) / 14.0 * 3.14159265
	s := float64(0.0)
	if val > 0 && val < 3.14159265 {
		// Taylor expansion approximation for sin(x)
		x := val
		s = x - (x*x*x)/6.0 + (x*x*x*x*x)/120.0
		if s < 0 {
			s = 0
		}
	}
	return s
}

func mathRound(val float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10.0
	}
	return float64(int(val*pow+0.5)) / pow
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/api/v1/telemetry", handleIngest)
	http.HandleFunc("/api/v1/live", handleLive)
	http.HandleFunc("/api/v1/history", handleHistory)
	http.HandleFunc("/api/v1/stats/day", handleDayStats)
	http.HandleFunc("/api/v1/stats/week", handleWeekStats)
	http.HandleFunc("/api/v1/stats/month", handleMonthStats)
	http.HandleFunc("/api/v1/stats/year", handleYearStats)
	http.HandleFunc("/api/v1/system-info", handleSystemInfo)
	http.HandleFunc("/healthz", handleHealthz)

	log.Printf("☀️ Solaria Cloud Run Service listening on port %s...", port)
	log.Printf("   Site: 1296 Wren Lake Drive, Dorset, ON (GCP: %s)", gcpProject)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
