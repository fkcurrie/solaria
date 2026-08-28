package location

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// SiteLocation encapsulates site coordinates, location name, and resolution source
type SiteLocation struct {
	Name      string  `json:"site_name"`
	Latitude  float64 `json:"site_latitude"`
	Longitude float64 `json:"site_longitude"`
	Source    string  `json:"source"` // "ENV_OVERRIDE", "GPS_HARDWARE", "IP_GEOLOCATION", "DEFAULT_FALLBACK"
}

// ResolveLocation determines site coordinates using the 3-tiered resolution hierarchy:
// Tier 1: Explicit Environment Variables (SITE_LATITUDE, SITE_LONGITUDE in .env)
// Tier 2: Hardware GPS Receiver (gpsd on localhost:2947)
// Tier 3: Network IP Geolocation APIs (ip-api.com / ipinfo.io)
func ResolveLocation(envLatStr, envLonStr, envName string) SiteLocation {
	// --------------------------------------------------------------------------
	// Tier 1: Check Environment Variable Overrides
	// --------------------------------------------------------------------------
	if envLatStr != "" && envLonStr != "" {
		lat, err1 := strconv.ParseFloat(strings.TrimSpace(envLatStr), 64)
		lon, err2 := strconv.ParseFloat(strings.TrimSpace(envLonStr), 64)
		if err1 == nil && err2 == nil && lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 {
			name := envName
			if name == "" {
				name = fmt.Sprintf("Configured Site (%.4f, %.4f)", lat, lon)
			}
			return SiteLocation{
				Name:      name,
				Latitude:  lat,
				Longitude: lon,
				Source:    "ENV_OVERRIDE",
			}
		}
	}

	// --------------------------------------------------------------------------
	// Tier 2: Hardware GPS Receiver (gpsd daemon at localhost:2947)
	// --------------------------------------------------------------------------
	if gpsLoc, err := queryGPSD("127.0.0.1:2947"); err == nil {
		if envName != "" {
			gpsLoc.Name = envName
		}
		return gpsLoc
	}

	// --------------------------------------------------------------------------
	// Tier 3: Network IP Geolocation API
	// --------------------------------------------------------------------------
	if ipLoc, err := queryIPGeo(); err == nil {
		if envName != "" {
			ipLoc.Name = envName
		}
		return ipLoc
	}

	// --------------------------------------------------------------------------
	// Tier 4: Default Safe Baseline Fallback
	// --------------------------------------------------------------------------
	name := envName
	if name == "" {
		name = "Default Site Location"
	}
	return SiteLocation{
		Name:      name,
		Latitude:  43.6752,
		Longitude: -79.3472,
		Source:    "DEFAULT_FALLBACK",
	}
}

// queryGPSD connects to local gpsd daemon and reads TPV fix
func queryGPSD(addr string) (SiteLocation, error) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return SiteLocation{}, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	// Enable WATCH command to stream JSON reports
	_, err = fmt.Fprintf(conn, "?WATCH={\"enable\":true,\"json\":true};\n")
	if err != nil {
		return SiteLocation{}, err
	}

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var report struct {
			Class string  `json:"class"`
			Mode  int     `json:"mode"`
			Lat   float64 `json:"lat"`
			Lon   float64 `json:"lon"`
		}
		if err := json.Unmarshal(line, &report); err == nil {
			if report.Class == "TPV" && report.Mode >= 2 && report.Lat != 0 && report.Lon != 0 {
				return SiteLocation{
					Name:      fmt.Sprintf("GPS Fix (%.4f, %.4f)", report.Lat, report.Lon),
					Latitude:  report.Lat,
					Longitude: report.Lon,
					Source:    "GPS_HARDWARE",
				}, nil
			}
		}
	}
	return SiteLocation{}, fmt.Errorf("no valid GPS TPV fix received from %s", addr)
}

// queryIPGeo queries public IP geolocation endpoints
func queryIPGeo() (SiteLocation, error) {
	client := &http.Client{Timeout: 4 * time.Second}

	// Primary Endpoint: ip-api.com
	resp, err := client.Get("http://ip-api.com/json/")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var res struct {
				Status      string  `json:"status"`
				City        string  `json:"city"`
				RegionName  string  `json:"regionName"`
				CountryCode string  `json:"countryCode"`
				Lat         float64 `json:"lat"`
				Lon         float64 `json:"lon"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Status == "success" {
				siteName := fmt.Sprintf("%s, %s, %s", res.City, res.RegionName, res.CountryCode)
				return SiteLocation{
					Name:      siteName,
					Latitude:  res.Lat,
					Longitude: res.Lon,
					Source:    "IP_GEOLOCATION",
				}, nil
			}
		}
	}

	// Fallback Endpoint: ipinfo.io
	resp2, err := client.Get("https://ipinfo.io/json")
	if err == nil {
		defer resp2.Body.Close()
		if resp2.StatusCode == http.StatusOK {
			var res2 struct {
				City    string `json:"city"`
				Region  string `json:"region"`
				Country string `json:"country"`
				Loc     string `json:"loc"`
			}
			if err := json.NewDecoder(resp2.Body).Decode(&res2); err == nil && res2.Loc != "" {
				parts := strings.Split(res2.Loc, ",")
				if len(parts) == 2 {
					lat, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
					lon, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
					if e1 == nil && e2 == nil {
						siteName := fmt.Sprintf("%s, %s, %s", res2.City, res2.Region, res2.Country)
						return SiteLocation{
							Name:      siteName,
							Latitude:  lat,
							Longitude: lon,
							Source:    "IP_GEOLOCATION",
						}, nil
					}
				}
			}
		}
	}

	return SiteLocation{}, fmt.Errorf("ip geolocation failed across all endpoints")
}
