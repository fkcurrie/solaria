package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Telemetry struct {
	PVPowerW          int     `json:"pv_power_w"`
	PVVoltageV        float64 `json:"pv_voltage_v"`
	PVCurrentA        float64 `json:"pv_current_a"`
	BatterySOCPct     int     `json:"battery_soc_pct"`
	BatteryVoltageV   float64 `json:"battery_voltage_v"`
	BatteryCurrentA   float64 `json:"battery_current_a"`
	ControllerTempC   int     `json:"controller_temp_c"`
	BatteryTempC      int     `json:"battery_temp_c"`
	LoadPowerW        int     `json:"load_power_w"`
	ChargingState     string  `json:"charging_state"`
	DailyGeneratedWh  int     `json:"daily_generated_wh"`
	TotalGeneratedKWh int     `json:"total_generated_kwh"`
}

type SolarRecord struct {
	Timestamp         string             `json:"timestamp"`
	Site              string             `json:"site"`
	Location          map[string]float64 `json:"location"`
	Telemetry         Telemetry          `json:"telemetry"`
	SunClassification string             `json:"sun_classification"`
	IsMock            bool               `json:"is_mock"`
	DataSource        string             `json:"data_source"`
	BLEConnected      bool               `json:"ble_connected"`
	OutageStatus      string             `json:"outage_status"`
}

type IngestBatch struct {
	Batch []SolarRecord `json:"batch"`
}

func main() {
	targetURL := flag.String("url", "https://solaria-dashboard-952659886764.us-central1.run.app/api/v1/ingest", "Target Cloud Run ingestion URL")
	token := flag.String("token", "solaria_cottage_secret_token_2026", "Bearer auth token")
	logPattern := flag.String("logs", "logs/*.csv", "Glob pattern for local telemetry CSV logs")
	batchSize := flag.Int("batch", 100, "Ingestion batch size")
	flag.Parse()

	files, err := filepath.Glob(*logPattern)
	if err != nil || len(files) == 0 {
		log.Fatalf("No CSV log files found matching pattern: %s", *logPattern)
	}

	fmt.Printf("🔍 Found %d CSV telemetry log files for synchronization:\n", len(files))
	for _, f := range files {
		fmt.Printf("   • %s\n", f)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var totalRecords int
	var totalSynced int

	for _, filename := range files {
		f, err := os.Open(filename)
		if err != nil {
			log.Printf("⚠️ Could not open %s: %v", filename, err)
			continue
		}

		r := csv.NewReader(f)
		header, err := r.Read()
		if err != nil {
			f.Close()
			continue
		}

		// Map header column names to indices
		colMap := make(map[string]int)
		for i, h := range header {
			colMap[strings.TrimSpace(h)] = i
		}

		var batch []SolarRecord

		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			totalRecords++
			rec := parseRow(row, colMap)
			if rec != nil {
				batch = append(batch, *rec)
			}

			if len(batch) >= *batchSize {
				if sendBatch(client, *targetURL, *token, batch) {
					totalSynced += len(batch)
					fmt.Printf("\r🚀 Synced %d / %d records to BigQuery (%s)...", totalSynced, totalRecords, filename)
				}
				batch = batch[:0]
			}
		}
		f.Close()

		if len(batch) > 0 {
			if sendBatch(client, *targetURL, *token, batch) {
				totalSynced += len(batch)
			}
			batch = batch[:0]
		}
	}

	fmt.Printf("\n\n✅ Historical Synchronization Complete!\n")
	fmt.Printf("   • Total Telemetry Rows Processed: %d\n", totalRecords)
	fmt.Printf("   • Total Records Successfully Streamed to Cloud Run & BigQuery: %d\n", totalSynced)
}

func parseRow(row []string, colMap map[string]int) *SolarRecord {
	get := func(name string) string {
		if idx, ok := colMap[name]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	tsStr := get("timestamp")
	if tsStr == "" {
		return nil
	}

	// Normalize timestamp format
	parsedTime, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		parsedTime, err = time.Parse(time.RFC3339, tsStr)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02T15:04:05.999999", tsStr)
			if err != nil {
				return nil
			}
		}
	}
	tsRFC3339 := parsedTime.UTC().Format(time.RFC3339)

	pvPower, _ := strconv.Atoi(get("pv_power_w"))
	pvVolts, _ := strconv.ParseFloat(get("pv_voltage_v"), 64)
	pvAmps, _ := strconv.ParseFloat(get("pv_current_a"), 64)
	battSoc, _ := strconv.Atoi(get("battery_soc_pct"))
	battVolts, _ := strconv.ParseFloat(get("battery_voltage_v"), 64)
	battAmps, _ := strconv.ParseFloat(get("battery_current_a"), 64)
	state := get("charging_state")
	if state == "" {
		state = "MPPT Charging"
	}
	ctrlTemp, _ := strconv.Atoi(get("controller_temp_c"))
	battTemp, _ := strconv.Atoi(get("battery_temp_c"))
	loadPower, _ := strconv.Atoi(get("load_power_w"))
	dailyWh, _ := strconv.Atoi(get("daily_generated_wh"))
	totalKWh, _ := strconv.Atoi(get("total_generated_kwh"))

	return &SolarRecord{
		Timestamp: tsRFC3339,
		Site:      "1296 Wren Lake Drive, Dorset, ON",
		Location: map[string]float64{
			"latitude":  45.186,
			"longitude": -78.863,
		},
		Telemetry: Telemetry{
			PVPowerW:          pvPower,
			PVVoltageV:        pvVolts,
			PVCurrentA:        pvAmps,
			BatterySOCPct:     battSoc,
			BatteryVoltageV:   battVolts,
			BatteryCurrentA:   battAmps,
			ControllerTempC:   ctrlTemp,
			BatteryTempC:      battTemp,
			LoadPowerW:        loadPower,
			ChargingState:     state,
			DailyGeneratedWh:  dailyWh,
			TotalGeneratedKWh: totalKWh,
		},
		SunClassification: "PARTLY_CLOUDY",
		IsMock:            false,
		DataSource:        "HARDWARE_BLE",
		BLEConnected:      true,
		OutageStatus:      "NOMINAL",
	}
}

func sendBatch(client *http.Client, url, token string, batch []SolarRecord) bool {
	payload, err := json.Marshal(IngestBatch{Batch: batch})
	if err != nil {
		return false
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("⚠️ Ingestion request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("⚠️ Ingestion rejected (HTTP %d): %s", resp.StatusCode, string(body))
		return false
	}

	return true
}
