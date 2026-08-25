package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSREAgent_RunAudit_Healthy(t *testing.T) {
	tempDir := t.TempDir()
	incidentFile := tempDir + "/test_incidents.json"

	// Mock Bridge
	bridgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"spool_count":              0,
			"total_successful_uploads": 100,
			"total_ble_frames":         100,
		})
	}))
	defer bridgeServer.Close()

	// Mock Cloud Server
	cloudServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/live" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"telemetry": map[string]interface{}{
					"pv_power_w":        350,
					"pv_voltage_v":      36.2,
					"pv_current_a":      9.8,
					"battery_voltage_v": 13.6,
					"battery_current_a": 15.0,
					"battery_temp_c":    22,
					"controller_temp_c": 28,
				},
				"weather": map[string]interface{}{
					"direct_radiation_w_m2": 450.0,
				},
			})
			return
		}
		if r.URL.Path == "/api/v1/hardware-config" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer cloudServer.Close()

	agent := NewSREAgent(bridgeServer.URL, cloudServer.URL, incidentFile, "test_token")
	status := agent.RunAudit(context.Background())

	if status.OverallHealth != "HEALTHY" {
		t.Errorf("Expected OverallHealth HEALTHY, got %s", status.OverallHealth)
	}
	if !status.BridgeActive {
		t.Errorf("Expected BridgeActive true")
	}
	if !status.CloudServerActive {
		t.Errorf("Expected CloudServerActive true")
	}
	if !status.LiFePO4SafetyPass {
		t.Errorf("Expected LiFePO4SafetyPass true")
	}
	if !status.StringTopologyPass {
		t.Errorf("Expected StringTopologyPass true")
	}
	if !status.SecurityAuditPass {
		t.Errorf("Expected SecurityAuditPass true")
	}
}

func TestSREAgent_RunAudit_SubZeroViolation(t *testing.T) {
	tempDir := t.TempDir()
	incidentFile := tempDir + "/test_incidents.json"

	bridgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"spool_count": 0,
		})
	}))
	defer bridgeServer.Close()

	// Mock Sub-zero violation (Temp -5C with active charging current)
	cloudServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/live" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"telemetry": map[string]interface{}{
					"pv_power_w":        150,
					"pv_voltage_v":      36.0,
					"battery_voltage_v": 13.2,
					"battery_current_a": 8.5,
					"battery_temp_c":    -5,
				},
				"weather": map[string]interface{}{
					"direct_radiation_w_m2": 200.0,
				},
			})
			return
		}
		if r.URL.Path == "/api/v1/hardware-config" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}))
	defer cloudServer.Close()

	agent := NewSREAgent(bridgeServer.URL, cloudServer.URL, incidentFile, "test_token")
	status := agent.RunAudit(context.Background())

	if status.OverallHealth != "UNHEALTHY" {
		t.Errorf("Expected OverallHealth UNHEALTHY on sub-zero charging, got %s", status.OverallHealth)
	}
	if status.LiFePO4SafetyPass {
		t.Errorf("Expected LiFePO4SafetyPass false on sub-zero charging")
	}
	if len(agent.GetIncidents()) == 0 {
		t.Errorf("Expected incident recorded for sub-zero charging")
	}
}

func TestSREAgent_RunAudit_DiodeFault(t *testing.T) {
	tempDir := t.TempDir()
	incidentFile := tempDir + "/test_incidents.json"

	bridgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"spool_count": 0})
	}))
	defer bridgeServer.Close()

	// Mock Diode Fault (PV Voltage 18V under 500 W/m2 sun)
	cloudServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/live" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"telemetry": map[string]interface{}{
					"pv_power_w":        120,
					"pv_voltage_v":      18.2,
					"battery_voltage_v": 13.3,
					"battery_current_a": 8.0,
					"battery_temp_c":    20,
				},
				"weather": map[string]interface{}{
					"direct_radiation_w_m2": 500.0,
				},
			})
			return
		}
		if r.URL.Path == "/api/v1/hardware-config" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}))
	defer cloudServer.Close()

	agent := NewSREAgent(bridgeServer.URL, cloudServer.URL, incidentFile, "test_token")
	status := agent.RunAudit(context.Background())

	if status.StringTopologyPass {
		t.Errorf("Expected StringTopologyPass false on 18V diode drop")
	}
	if status.OverallHealth != "DEGRADED" {
		t.Errorf("Expected OverallHealth DEGRADED, got %s", status.OverallHealth)
	}
}
