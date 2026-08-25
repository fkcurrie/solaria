package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Incident represents an operational, security, or physics anomaly
type Incident struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"` // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Category    string    `json:"category"` // "BATTERY_SAFETY", "ARRAY_TOPOLOGY", "EDGE_CONNECTIVITY", "SECURITY"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation"`
	Resolved    bool      `json:"resolved"`
}

// SREStatus captures overall system operational health
type SREStatus struct {
	LastAuditTime      time.Time  `json:"last_audit_time"`
	OverallHealth      string     `json:"overall_health"` // "HEALTHY", "DEGRADED", "UNHEALTHY"
	BridgeActive       bool       `json:"bridge_active"`
	CloudServerActive  bool       `json:"cloud_server_active"`
	TotalIncidents     int        `json:"total_incidents"`
	ActiveIncidents    int        `json:"active_incidents"`
	RecentIncidents    []Incident `json:"recent_incidents"`
	LiFePO4SafetyPass  bool       `json:"lifepo4_safety_pass"`
	StringTopologyPass bool       `json:"string_topology_pass"`
	SpoolHealthPass    bool       `json:"spool_health_pass"`
	SecurityAuditPass  bool       `json:"security_audit_pass"`
}

type SREAgent struct {
	mu           sync.RWMutex
	status       SREStatus
	incidents    []Incident
	incidentFile string
	bridgeURL    string
	cloudURL     string
	apiToken     string
}

func NewSREAgent(bridgeURL, cloudURL, incidentFile, token string) *SREAgent {
	agent := &SREAgent{
		incidentFile: incidentFile,
		bridgeURL:    bridgeURL,
		cloudURL:     cloudURL,
		apiToken:     token,
		status: SREStatus{
			OverallHealth: "HEALTHY",
		},
		incidents: make([]Incident, 0),
	}
	agent.loadIncidents()
	return agent
}

func (a *SREAgent) loadIncidents() {
	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := os.ReadFile(a.incidentFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &a.incidents)
}

func (a *SREAgent) saveIncidents() {
	_ = os.MkdirAll(filepath.Dir(a.incidentFile), 0750)
	data, err := json.MarshalIndent(a.incidents, "", "  ")
	if err == nil {
		tmp := a.incidentFile + ".tmp"
		if wErr := os.WriteFile(tmp, data, 0600); wErr == nil {
			_ = os.Rename(tmp, a.incidentFile)
		}
	}
}

func (a *SREAgent) ResolveCategory(category, title string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	modified := false
	for i := range a.incidents {
		if !a.incidents[i].Resolved && a.incidents[i].Category == category {
			if title == "" || a.incidents[i].Title == title {
				a.incidents[i].Resolved = true
				modified = true
			}
		}
	}
	if modified {
		a.saveIncidents()
	}
}

func (a *SREAgent) RecordIncident(inc Incident) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Avoid duplicate active incidents with the same Category and Title within 10 minutes
	for _, existing := range a.incidents {
		if !existing.Resolved && existing.Category == inc.Category && existing.Title == inc.Title {
			if time.Since(existing.Timestamp) < 10*time.Minute {
				return
			}
		}
	}

	inc.ID = fmt.Sprintf("INC-%d", time.Now().UnixNano()/1e6)
	inc.Timestamp = time.Now()
	a.incidents = append([]Incident{inc}, a.incidents...)
	if len(a.incidents) > 100 {
		a.incidents = a.incidents[:100]
	}
	a.saveIncidents()
}

func (a *SREAgent) RunAudit(ctx context.Context) SREStatus {
	var (
		bridgeActive      = false
		cloudActive       = false
		lifepo4Pass       = true
		stringPass        = true
		spoolPass         = true
		securityPass      = true
		client            = &http.Client{Timeout: 3 * time.Second}
	)

	// 1. Audit Bridge Daemon
	bridgeReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.bridgeURL+"/api/v1/bridge-status", nil)
	if a.apiToken != "" {
		bridgeReq.Header.Set("Authorization", "Bearer "+a.apiToken)
	}
	resp, err := client.Do(bridgeReq)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil && resp.StatusCode == http.StatusOK {
		bridgeActive = true
		var bridgeStatus struct {
			SpoolCount         int    `json:"spool_count"`
			LastBlePacketTime  string `json:"last_ble_packet_time"`
			TotalBleFrames     int64  `json:"total_ble_frames"`
			TotalUploads       int64  `json:"total_successful_uploads"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&bridgeStatus); err == nil {
			if bridgeStatus.SpoolCount > 50 {
				spoolPass = false
				a.RecordIncident(Incident{
					Severity:    "HIGH",
					Category:    "EDGE_CONNECTIVITY",
					Title:       "Offline Spool Queue Elevated",
					Description: fmt.Sprintf("Disk spooler has %d unsynced telemetry records queued.", bridgeStatus.SpoolCount),
					Remediation: "Verify Cloud Run ingestion endpoint and internet connectivity on cottage edge gateway.",
				})
			}
		}
	} else {
		a.RecordIncident(Incident{
			Severity:    "MEDIUM",
			Category:    "EDGE_CONNECTIVITY",
			Title:       "Local Edge Bridge Unreachable",
			Description: fmt.Sprintf("Could not reach bridge at %s (Status: %v)", a.bridgeURL, err),
			Remediation: "Ensure 'cmd/bridge' daemon is running on port 8080.",
		})
	}

	// 2. Audit Cloud Server & Live Telemetry Safety Invariants
	cloudReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.cloudURL+"/api/v1/live", nil)
	cResp, cErr := client.Do(cloudReq)
	if cResp != nil {
		defer cResp.Body.Close()
	}
	if cErr == nil && cResp.StatusCode == http.StatusOK {
		cloudActive = true
		var liveRecord struct {
			Telemetry struct {
				PVPowerW        int     `json:"pv_power_w"`
				PVVoltageV      float64 `json:"pv_voltage_v"`
				PVCurrentA      float64 `json:"pv_current_a"`
				BatteryVoltageV float64 `json:"battery_voltage_v"`
				BatteryCurrentA float64 `json:"battery_current_a"`
				BatteryTempC    int     `json:"battery_temp_c"`
				ControllerTempC int     `json:"controller_temp_c"`
			} `json:"telemetry"`
			Weather struct {
				DirectRadiationWM2 float64 `json:"direct_radiation_w_m2"`
			} `json:"weather"`
		}
		if err := json.NewDecoder(cResp.Body).Decode(&liveRecord); err == nil {
			// Invariant 1: LiFePO4 Sub-Zero Charge Inhibit
			if liveRecord.Telemetry.BatteryTempC <= 0 && liveRecord.Telemetry.BatteryCurrentA > 0.1 {
				lifepo4Pass = false
				a.RecordIncident(Incident{
					Severity:    "CRITICAL",
					Category:    "BATTERY_SAFETY",
					Title:       "Sub-Zero LiFePO4 Charging Violation Detected",
					Description: fmt.Sprintf("Battery temperature is %d°C (<=0°C) with active charging current %.2fA. Risk of lithium plating.", liveRecord.Telemetry.BatteryTempC, liveRecord.Telemetry.BatteryCurrentA),
					Remediation: "Renogy Rover low-temperature charge protection must inhibit charging below 0°C immediately.",
				})
			}

			// Invariant 2: 2S2P String Imbalance / Diode Drop
			if liveRecord.Weather.DirectRadiationWM2 > 300.0 && liveRecord.Telemetry.PVVoltageV > 0 && liveRecord.Telemetry.PVVoltageV < 22.0 {
				stringPass = false
				a.RecordIncident(Incident{
					Severity:    "HIGH",
					Category:    "ARRAY_TOPOLOGY",
					Title:       "2S2P String Imbalance / Diode Drop Detected",
					Description: fmt.Sprintf("PV Voltage is %.1fV under high sun (%.0f W/m²). Expected ~34-38V for 2S series strings.", liveRecord.Telemetry.PVVoltageV, liveRecord.Weather.DirectRadiationWM2),
					Remediation: "Inspect array MC4 series interconnects and bypass diodes on roof panels.",
				})
			}
		}
	}

	// 3. Audit Security Mutation Authorization
	secReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.cloudURL+"/api/v1/hardware-config", nil)
	sResp, sErr := client.Do(secReq)
	if sResp != nil {
		defer sResp.Body.Close()
	}
	if sErr == nil {
		if sResp.StatusCode != http.StatusUnauthorized {
			securityPass = false
			a.RecordIncident(Incident{
				Severity:    "CRITICAL",
				Category:    "SECURITY",
				Title:       "Unauthenticated Hardware Mutation Endpoint Exposed",
				Description: fmt.Sprintf("POST /api/v1/hardware-config returned %d instead of 401 Unauthorized.", sResp.StatusCode),
				Remediation: "Ensure verifyAuth(r) middleware is enforced for all configuration mutations.",
			})
		}
	}

	// Auto-resolve recovered incidents
	if bridgeActive {
		a.ResolveCategory("EDGE_CONNECTIVITY", "Local Edge Bridge Unreachable")
	}
	if spoolPass {
		a.ResolveCategory("EDGE_CONNECTIVITY", "Offline Spool Queue Elevated")
	}
	if cloudActive && lifepo4Pass {
		a.ResolveCategory("BATTERY_SAFETY", "")
	}
	if cloudActive && stringPass {
		a.ResolveCategory("ARRAY_TOPOLOGY", "")
	}
	if cloudActive && securityPass {
		a.ResolveCategory("SECURITY", "")
	}

	overall := "HEALTHY"
	if !lifepo4Pass || !securityPass {
		overall = "UNHEALTHY"
	} else if !bridgeActive || !cloudActive || !stringPass || !spoolPass {
		overall = "DEGRADED"
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	activeCount := 0
	for _, inc := range a.incidents {
		if !inc.Resolved {
			activeCount++
		}
	}

	recent := a.incidents
	if len(recent) > 10 {
		recent = recent[:10]
	}

	a.status = SREStatus{
		LastAuditTime:      time.Now(),
		OverallHealth:      overall,
		BridgeActive:       bridgeActive,
		CloudServerActive:  cloudActive,
		TotalIncidents:     len(a.incidents),
		ActiveIncidents:    activeCount,
		RecentIncidents:    recent,
		LiFePO4SafetyPass:  lifepo4Pass,
		StringTopologyPass: stringPass,
		SpoolHealthPass:    spoolPass,
		SecurityAuditPass:  securityPass,
	}

	return a.status
}

func (a *SREAgent) GetStatus() SREStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func (a *SREAgent) GetIncidents() []Incident {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.incidents
}

func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func main() {
	var (
		daemonMode = flag.Bool("daemon", false, "Run continuous autonomous SRE monitoring loop")
		auditOnce  = flag.Bool("audit", false, "Run single one-shot diagnostic audit and exit")
		port       = flag.Int("port", 8082, "SRE Agent HTTP server port")
		bridgeURL  = flag.String("bridge-url", "http://localhost:8080", "Bridge daemon URL")
		cloudURL   = flag.String("cloud-url", "http://localhost:8081", "Cloud server URL")
		interval   = flag.Duration("interval", 15*time.Second, "Monitoring interval in daemon mode")
	)
	flag.Parse()

	token := os.Getenv("SOLARIA_API_TOKEN")
	agent := NewSREAgent(*bridgeURL, *cloudURL, "logs/incidents.json", token)

	if *auditOnce {
		fmt.Println("🔍 Running Project Solaria Autonomous SRE Audit...")
		status := agent.RunAudit(context.Background())
		out, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(out))
		return
	}

	if *daemonMode {
		log.Printf("🤖 Running in continuous daemon mode (interval: %v)", *interval)
	}

	// Background continuous monitoring loop if daemon mode or server mode
	ticker := time.NewTicker(*interval)
	stopChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				st := agent.RunAudit(context.Background())
				if st.OverallHealth != "HEALTHY" {
					log.Printf("⚠️ SRE Health Alert: %s (Active Incidents: %d)", st.OverallHealth, st.ActiveIncidents)
				}
			case <-stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/api/v1/sre/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_ = json.NewEncoder(w).Encode(agent.GetStatus())
	})

	mux.HandleFunc("/api/v1/sre/incidents", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_ = json.NewEncoder(w).Encode(agent.GetIncidents())
	})

	mux.HandleFunc("/api/v1/sre/audit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		st := agent.RunAudit(r.Context())
		_ = json.NewEncoder(w).Encode(st)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	go func() {
		log.Printf("🤖 Solaria SRE Agent REST API listening on http://localhost:%d", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("SRE Agent server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Solaria Autonomous SRE Agent...")
	close(stopChan)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
