package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WeatherMetrics holds the environmental data for Dorset, ON
type WeatherMetrics struct {
	TemperatureC        float64 `json:"temperature_c"`
	CloudCoverPct       int     `json:"cloud_cover_pct"`
	GHIIrradianceWM2    float64 `json:"ghi_w_m2"`
	DNIIrradianceWM2    float64 `json:"dni_w_m2"`
	DirectRadiationWM2  float64 `json:"direct_radiation_w_m2"`
	DiffuseRadiationWM2 float64 `json:"diffuse_radiation_w_m2"`
	IsDay               bool    `json:"is_day"`
}

type openMeteoResponse struct {
	Current struct {
		Temperature2M          float64 `json:"temperature_2m"`
		CloudCover             int     `json:"cloud_cover"`
		DirectNormalIrradiance float64 `json:"direct_normal_irradiance"`
		GlobalTiltedIrradiance float64 `json:"global_tilted_irradiance"`
		DiffuseRadiation       float64 `json:"diffuse_radiation"`
		DirectRadiation        float64 `json:"direct_radiation"`
		IsDay                  int     `json:"is_day"`
	} `json:"current"`
}

// WeatherProvider handles periodic fetching and caching of high-resolution weather data
type WeatherProvider struct {
	lat          float64
	lon          float64
	pollInterval time.Duration
	client       *http.Client
	mu           sync.RWMutex
	cached       WeatherMetrics
	lastFetch    time.Time
}

func NewWeatherProvider(lat, lon float64, interval time.Duration) *WeatherProvider {
	return &WeatherProvider{
		lat:          lat,
		lon:          lon,
		pollInterval: interval,
		client:       &http.Client{Timeout: 8 * time.Second},
		cached: WeatherMetrics{
			TemperatureC: 20.0,
			IsDay:        true,
		},
	}
}

func (w *WeatherProvider) GetWeather() WeatherMetrics {
	w.mu.RLock()
	if time.Since(w.lastFetch) < w.pollInterval && !w.lastFetch.IsZero() {
		defer w.mu.RUnlock()
		return w.cached
	}
	w.mu.RUnlock()

	w.mu.Lock()
	defer w.mu.Unlock()

	// Double check inside write lock
	if time.Since(w.lastFetch) < w.pollInterval && !w.lastFetch.IsZero() {
		return w.cached
	}

	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,cloud_cover,direct_normal_irradiance,global_tilted_irradiance,diffuse_radiation,direct_radiation,is_day&timezone=auto",
		w.lat, w.lon,
	)

	resp, err := w.client.Get(url)
	if err != nil {
		return w.cached
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return w.cached
	}

	var data openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return w.cached
	}

	w.cached = WeatherMetrics{
		TemperatureC:        data.Current.Temperature2M,
		CloudCoverPct:       data.Current.CloudCover,
		GHIIrradianceWM2:    data.Current.GlobalTiltedIrradiance,
		DNIIrradianceWM2:    data.Current.DirectNormalIrradiance,
		DirectRadiationWM2:  data.Current.DirectRadiation,
		DiffuseRadiationWM2: data.Current.DiffuseRadiation,
		IsDay:               data.Current.IsDay == 1,
	}
	w.lastFetch = time.Now()

	return w.cached
}
