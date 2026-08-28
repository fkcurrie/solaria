package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type SolarRecord struct {
	Timestamp         string                 `json:"timestamp"`
	Site              string                 `json:"site"`
	Location          map[string]float64     `json:"location"`
	Telemetry         map[string]interface{} `json:"telemetry"`
	Weather           map[string]interface{} `json:"weather"`
	SunClassification string                 `json:"sun_classification"`
}

func main() {
	dataFile := flag.String("file", "testdata/sample_day.json", "Path to JSON telemetry sample dataset")
	endpoint := flag.String("endpoint", "http://localhost:8081/api/v1/telemetry", "Target telemetry ingestion endpoint")
	token := flag.String("token", "solaria_cottage_secret_token_2026", "Authorization token")
	intervalMs := flag.Int("interval-ms", 200, "Interval between ingested frames in milliseconds")
	repeat := flag.Bool("repeat", false, "Continuously loop telemetry replay")
	flag.Parse()

	data, err := os.ReadFile(*dataFile)
	if err != nil {
		fmt.Printf("Error opening %s: %v\n", *dataFile, err)
		os.Exit(1)
	}

	var rawRecords []map[string]interface{}
	if err := json.Unmarshal(data, &rawRecords); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("========================================================\n")
	fmt.Printf("SOLARIA TELEMETRY REPLAY UTILITY\n")
	fmt.Printf("========================================================\n")
	fmt.Printf("  • File:     %s (%d records)\n", *dataFile, len(rawRecords))
	fmt.Printf("  • Target:   %s\n", *endpoint)
	fmt.Printf("  • Interval: %d ms / frame\n", *intervalMs)
	fmt.Printf("========================================================\n\n")

	client := &http.Client{Timeout: 5 * time.Second}

	for {
		for i, rec := range rawRecords {
			// Format as SolarRecord expected by cloud-server
			payload := map[string]interface{}{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"site":      "1296 Wren Lake Drive, Dorset, ON",
				"location": map[string]float64{
					"latitude":  45.186,
					"longitude": -78.863,
				},
				"telemetry":          rec,
				"weather":            rec,
				"sun_classification": rec["sun_condition"],
			}

			batchPayload := map[string]interface{}{
				"batch": []interface{}{payload},
			}

			body, _ := json.Marshal(batchPayload)
			req, err := http.NewRequest("POST", *endpoint, bytes.NewReader(body))
			if err != nil {
				fmt.Printf("[%d/%d] Request creation error: %v\n", i+1, len(rawRecords), err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+*token)
			req.Header.Set("X-API-Key", *token)

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("[%d/%d] Ingest POST failed: %v\n", i+1, len(rawRecords), err)
			} else {
				_ = resp.Body.Close()
				pvW := rec["pv_power_w"]
				soc := rec["battery_soc_pct"]
				vBatt := rec["battery_voltage_v"]
				fmt.Printf("\r[%d/%d] Sent frame | PV: %v W | Battery: %vV (%v%%) | Status: %d OK ",
					i+1, len(rawRecords), pvW, vBatt, soc, resp.StatusCode)
			}

			time.Sleep(time.Duration(*intervalMs) * time.Millisecond)
		}
		if !*repeat {
			break
		}
	}
	fmt.Printf("\nReplay finished successfully.\n")
}
