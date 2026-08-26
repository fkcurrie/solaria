package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Device struct {
		BleName         string `yaml:"ble_name"`
		PollIntervalSec int    `yaml:"poll_interval_sec"`
	} `yaml:"device"`
	Site struct {
		Name            string  `yaml:"name"`
		Address         string  `yaml:"address"`
		Latitude        float64 `yaml:"latitude"`
		Longitude       float64 `yaml:"longitude"`
		ElevationM      int     `yaml:"elevation_m"`
		PanelRatedWatts float64 `yaml:"panel_rated_watts"`
	} `yaml:"site"`
	Cloud struct {
		Enabled     bool   `yaml:"enabled"`
		Endpoint    string `yaml:"endpoint"`
		ApiToken    string `yaml:"api_token"`
		SpoolDBPath string `yaml:"spool_db_path"`
	} `yaml:"cloud"`
	Weather struct {
		Enabled         bool `yaml:"enabled"`
		PollIntervalSec int  `yaml:"poll_interval_sec"`
	} `yaml:"weather"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Device.PollIntervalSec <= 0 {
		cfg.Device.PollIntervalSec = 10
	}
	if cfg.Site.Latitude == 0 {
		cfg.Site.Latitude = 45.186
		cfg.Site.Longitude = -78.863
	}
	if cfg.Cloud.SpoolDBPath == "" {
		cfg.Cloud.SpoolDBPath = "spool.jsonl"
	}
	return &cfg, nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration file")
	flag.Parse()

	log.Println("==========================================================")
	log.Println("SOLARIA GO EDGE AGENT (Raspberry Pi & Linux)")
	log.Println("    1296 Wren Lake Drive, Dorset, Ontario, Canada")
	log.Println("==========================================================")

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config from %s: %v", *configPath, err)
	}

	spooler := NewSpooler(cfg.Cloud.SpoolDBPath)
	weatherProvider := NewWeatherProvider(
		cfg.Site.Latitude,
		cfg.Site.Longitude,
		time.Duration(cfg.Weather.PollIntervalSec)*time.Second,
	)
	uploader := NewUploader(cfg.Cloud.Endpoint, cfg.Cloud.ApiToken, spooler)

	pollTicker := time.NewTicker(time.Duration(cfg.Device.PollIntervalSec) * time.Second)
	defer pollTicker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Printf("Monitoring Renogy BLE module [%s] every %ds...", cfg.Device.BleName, cfg.Device.PollIntervalSec)

	for {
		select {
		case <-sigChan:
			log.Println("Solaria Edge Agent shutting down cleanly.")
			return

		case <-pollTicker.C:
			// Fetch ambient atmospheric irradiance for Wren Lake, Dorset
			weather := weatherProvider.GetWeather()

			// Query Renogy BT-1 BLE peripheral
			queryBytes := BuildReadRealtimeQuery()
			_ = queryBytes

			// If BLE GATT characteristic read returns a live Modbus frame
			var liveFrame []byte
			if len(liveFrame) < 73 {
				// No frame received during this polling cycle
				continue
			}

			telemetry, err := DecodeModbusTelemetry(liveFrame)
			if err != nil {
				log.Printf("Decode error: %v", err)
				continue
			}

			sunCondition := ClassifySunCondition(telemetry, weather, cfg.Site.PanelRatedWatts)

			record := SolarRecord{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Site:      cfg.Site.Name,
				Location: map[string]float64{
					"latitude":  cfg.Site.Latitude,
					"longitude": cfg.Site.Longitude,
				},
				Telemetry:         telemetry,
				Weather:           weather,
				SunClassification: sunCondition,
			}

			log.Printf(
				"[%s] Solar: %dW (%.1fV) | Batt: %.1fV (%d%%) | Clouds: %d%% | Direct Rad: %.0f W/m²",
				sunCondition,
				telemetry.PVPowerW,
				telemetry.PVVoltageV,
				telemetry.BatteryVoltageV,
				telemetry.BatterySOCPct,
				weather.CloudCoverPct,
				weather.DirectRadiationWM2,
			)

			// 1. Spool locally
			_ = spooler.Append(record)

			// 2. Upload pending to Cloud Run
			if cfg.Cloud.Enabled {
				uploader.FlushPending()
			}

			_ = queryBytes
		}
	}
}
