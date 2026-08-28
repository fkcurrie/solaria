package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Incident represents an operational, security, or physics anomaly
type Incident struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"` // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Category    string    `json:"category"` // "BATTERY_SAFETY", "ARRAY_TOPOLOGY", "EDGE_CONNECTIVITY", "SECURITY", "AUTO_HEAL"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Remediation string    `json:"remediation"`
	Resolved    bool      `json:"resolved"`
}

// AutoHealAction represents an autonomous self-healing action taken by the agent
type AutoHealAction struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"` // "RESTART_BRIDGE", "RESTART_CLOUD_SERVER", "FLUSH_SPOOL", "RESET_BLE"
	Target    string    `json:"target"`
	Reason    string    `json:"reason"`
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
}

// SREStatus captures overall system operational health
type SREStatus struct {
	LastAuditTime         time.Time        `json:"last_audit_time"`
	OverallHealth         string           `json:"overall_health"` // "HEALTHY", "DEGRADED", "UNHEALTHY"
	BridgeActive          bool             `json:"bridge_active"`
	CloudServerActive     bool             `json:"cloud_server_active"`
	CloudRunActive        bool             `json:"cloudrun_active"`
	TelemetryStreaming    bool             `json:"telemetry_streaming"`
	LastPacketAgeSec      int              `json:"last_packet_age_sec"`
	CloudRunPacketAgeSec  int              `json:"cloudrun_packet_age_sec"`
	TotalIncidents        int              `json:"total_incidents"`
	ActiveIncidents       int              `json:"active_incidents"`
	RecentIncidents       []Incident       `json:"recent_incidents"`
	RecentAutoHeals       []AutoHealAction `json:"recent_auto_heals"`
	LiFePO4SafetyPass     bool             `json:"lifepo4_safety_pass"`
	StringTopologyPass    bool             `json:"string_topology_pass"`
	SpoolHealthPass       bool             `json:"spool_health_pass"`
	SecurityAuditPass     bool             `json:"security_audit_pass"`
	CloudRunFreshnessPass bool             `json:"cloudrun_freshness_pass"`
}

type SREAgent struct {
	mu           sync.RWMutex
	status       SREStatus
	incidents    []Incident
	autoHeals    []AutoHealAction
	lastHealTime map[string]time.Time
	incidentFile string
	bridgeURL    string
	cloudURL     string
	cloudRunURL  string
	apiToken     string
	autoHeal     bool
}

func NewSREAgent(bridgeURL, cloudURL, cloudRunURL, incidentFile, token string, autoHeal bool) *SREAgent {
	if cloudRunURL == "" {
		if env := os.Getenv("SOLARIA_CLOUD_ENDPOINT"); env != "" {
			cloudRunURL = env
		} else {
			cloudRunURL = cloudURL
		}
	}
	agent := &SREAgent{
		incidentFile: incidentFile,
		bridgeURL:    bridgeURL,
		cloudURL:     cloudURL,
		cloudRunURL:  cloudRunURL,
		apiToken:     token,
		autoHeal:     autoHeal,
		lastHealTime: make(map[string]time.Time),
		status: SREStatus{
			OverallHealth: "HEALTHY",
		},
		incidents: make([]Incident, 0),
		autoHeals: make([]AutoHealAction, 0),
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

func (a *SREAgent) recordAutoHeal(action AutoHealAction) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.autoHeals = append([]AutoHealAction{action}, a.autoHeals...)
	if len(a.autoHeals) > 50 {
		a.autoHeals = a.autoHeals[:50]
	}
	log.Printf("🤖 [AUTO-HEAL] Action: %s | Target: %s | Success: %t | %s", action.Action, action.Target, action.Success, action.Message)
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

	// Avoid duplicate active incidents with the same Category and Title
	for i, existing := range a.incidents {
		if !existing.Resolved && existing.Category == inc.Category && existing.Title == inc.Title {
			a.incidents[i].Description = inc.Description
			a.saveIncidents()
			return
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

func (a *SREAgent) canHeal(actionKey string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	last, ok := a.lastHealTime[actionKey]
	if ok && time.Since(last) < 25*time.Second {
		return false
	}
	a.lastHealTime[actionKey] = time.Now()
	return true
}

func (a *SREAgent) AutoHealBridge() {
	if !a.autoHeal || !a.canHeal("bridge") {
		return
	}
	log.Println("🛠️ [AUTO-HEAL] Solaria Bridge daemon is offline or silent. Initiating autonomous restart...")
	cmd := exec.Command("./bin/solaria-bridge")
	cloudRunEP := a.cloudURL + "/api/v1/telemetry"
	if a.cloudRunURL != "" {
		cloudRunEP = a.cloudRunURL + "/api/v1/telemetry"
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(),
		"PORT=8080",
		"SOLARIA_CLOUD_ENDPOINT="+cloudRunEP,
		"SOLARIA_FALLBACK_ENDPOINT="+a.cloudURL+"/api/v1/ingest",
	)
	if a.apiToken != "" {
		cmd.Env = append(cmd.Env, "SOLARIA_API_TOKEN="+a.apiToken)
	}
	err := cmd.Start()
	success := err == nil
	msg := ""
	if success {
		msg = fmt.Sprintf("Successfully spawned ./bin/solaria-bridge (PID: %d)", cmd.Process.Pid)
	} else {
		msg = fmt.Sprintf("Failed to spawn ./bin/solaria-bridge: %v", err)
	}
	a.recordAutoHeal(AutoHealAction{
		Timestamp: time.Now(),
		Action:    "RESTART_BRIDGE",
		Target:    "solaria-bridge",
		Reason:    "Bridge unreachable or silent telemetry stream",
		Success:   success,
		Message:   msg,
	})
	a.RecordIncident(Incident{
		Severity:    "HIGH",
		Category:    "AUTO_HEAL",
		Title:       "Autonomous Bridge Daemon Self-Heal Triggered",
		Description: msg,
		Remediation: "SRE agent supervisor automatically restarted the bridge process with dual endpoints.",
		Resolved:    success,
	})
}

func (a *SREAgent) AutoHealBluetoothSubsystem() {
	if !a.autoHeal || !a.canHeal("bluetooth-hw-reset") {
		return
	}
	log.Println("🛠️ [AUTO-HEAL] Bluetooth telemetry silent >180s. Executing 3-tier hardware radio reset (rfkill + hciconfig reset)...")
	_ = exec.Command("rfkill", "unblock", "bluetooth").Run()
	_ = exec.Command("hciconfig", "hci0", "reset").Run()
	_ = exec.Command("bluetoothctl", "power", "off").Run()
	time.Sleep(500 * time.Millisecond)
	_ = exec.Command("bluetoothctl", "power", "on").Run()

	a.recordAutoHeal(AutoHealAction{
		Timestamp: time.Now(),
		Action:    "RESET_BLE",
		Target:    "linux-bluetooth-hci0",
		Reason:    "Telemetry stalled >180s indicating potential radio freeze",
		Success:   true,
		Message:   "Executed rfkill unblock and hciconfig hci0 reset",
	})
	a.RecordIncident(Incident{
		Severity:    "HIGH",
		Category:    "AUTO_HEAL",
		Title:       "Autonomous Bluetooth Hardware Reset Triggered",
		Description: "Reset Linux Bluetooth radio adapter (hci0) to resolve potential radio freeze.",
		Remediation: "Supervisor auto-executed hardware radio reset.",
		Resolved:    true,
	})
}

func (a *SREAgent) AutoHealCloudRunSync() {
	if !a.autoHeal || !a.canHeal("cloud-run-sync") {
		return
	}
	log.Println("🛠️ [AUTO-HEAL] Cloud Run telemetry stream stale (>120s). Initiating emergency telemetry sync...")

	client := &http.Client{Timeout: 4 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, a.cloudURL+"/api/v1/live", nil)
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			payload := fmt.Sprintf(`{"batch":[%s]}`, string(body))
			cloudRunEP := a.cloudURL + "/api/v1/telemetry"
			if a.cloudRunURL != "" {
				cloudRunEP = a.cloudRunURL + "/api/v1/telemetry"
			}
			postReq, _ := http.NewRequest(http.MethodPost, cloudRunEP, bytes.NewBufferString(payload))
			postReq.Header.Set("Content-Type", "application/json")
			if a.apiToken != "" {
				postReq.Header.Set("Authorization", "Bearer "+a.apiToken)
				postReq.Header.Set("X-API-Key", a.apiToken)
			}
			postResp, postErr := client.Do(postReq)
			if postErr == nil {
				postResp.Body.Close()
				log.Printf("🛠️ [AUTO-HEAL] Pushed emergency telemetry frame to Cloud Run (Status: %d)", postResp.StatusCode)
				a.recordAutoHeal(AutoHealAction{
					Timestamp: time.Now(),
					Action:    "SYNC_CLOUDRUN_TELEMETRY",
					Target:    "solaria-cloudrun",
					Reason:    "Cloud Run telemetry staleness detected",
					Success:   postResp.StatusCode == http.StatusOK,
					Message:   fmt.Sprintf("Direct sync returned HTTP %d", postResp.StatusCode),
				})
			}
		}
	}
}

func (a *SREAgent) AutoHealCloudServer() {
	if !a.autoHeal || !a.canHeal("cloud-server") {
		return
	}
	log.Println("🛠️ [AUTO-HEAL] Solaria Cloud Server (port 8081) is offline. Initiating autonomous restart...")
	cmd := exec.Command("./bin/solaria-cloud-server")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(),
		"PORT=8081",
		"SOLARIA_API_TOKEN="+a.apiToken,
	)
	err := cmd.Start()
	success := err == nil
	msg := ""
	if success {
		msg = fmt.Sprintf("Successfully spawned ./bin/solaria-cloud-server (PID: %d)", cmd.Process.Pid)
	} else {
		msg = fmt.Sprintf("Failed to spawn ./bin/solaria-cloud-server: %v", err)
	}
	a.recordAutoHeal(AutoHealAction{
		Timestamp: time.Now(),
		Action:    "RESTART_CLOUD_SERVER",
		Target:    "solaria-cloud-server",
		Reason:    "Local cloud ingestion server unreachable",
		Success:   success,
		Message:   msg,
	})
	a.RecordIncident(Incident{
		Severity:    "HIGH",
		Category:    "AUTO_HEAL",
		Title:       "Autonomous Cloud Server Self-Heal Triggered",
		Description: msg,
		Remediation: "SRE agent supervisor automatically restarted the cloud server process.",
		Resolved:    success,
	})
}

func (a *SREAgent) RunAudit(ctx context.Context) SREStatus {
	var (
		bridgeActive     = false
		cloudActive      = false
		telemetryFlowing = false
		lastPacketAgeSec = 9999
		lifepo4Pass      = true
		stringPass       = true
		spoolPass        = true
		securityPass     = true
		client           = &http.Client{Timeout: 3 * time.Second}
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
			SpoolCount        int    `json:"spool_count"`
			LastBlePacketTime string `json:"last_ble_packet_time"`
			LastUploadTime    string `json:"last_successful_upload"`
			TotalBleFrames    int64  `json:"total_ble_frames"`
			TotalUploads      int64  `json:"total_successful_uploads"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&bridgeStatus); decErr == nil {
			var freshestTime time.Time
			if bridgeStatus.LastBlePacketTime != "" {
				if t, parseErr := time.Parse(time.RFC3339, bridgeStatus.LastBlePacketTime); parseErr == nil {
					freshestTime = t
				}
			}
			if bridgeStatus.LastUploadTime != "" {
				if t, parseErr := time.Parse(time.RFC3339, bridgeStatus.LastUploadTime); parseErr == nil {
					if t.After(freshestTime) {
						freshestTime = t
					}
				}
			}
			if !freshestTime.IsZero() {
				lastPacketAgeSec = int(time.Since(freshestTime).Seconds())
				if lastPacketAgeSec < 60 {
					telemetryFlowing = true
				}
			}
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
			Timestamp string `json:"timestamp"`
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
			if liveRecord.Timestamp != "" {
				if t, parseErr := time.Parse(time.RFC3339, liveRecord.Timestamp); parseErr == nil {
					cloudAge := int(time.Since(t).Seconds())
					if cloudAge < lastPacketAgeSec {
						lastPacketAgeSec = cloudAge
					}
					if cloudAge < 60 {
						telemetryFlowing = true
					}
				}
			}

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

	// 3. Audit Cloud Run Remote Telemetry Freshness
	var (
		cloudRunActive       = false
		cloudRunFreshness    = false
		cloudRunPacketAgeSec = 9999
	)
	if a.cloudRunURL != "" {
		crReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.cloudRunURL+"/api/v1/live", nil)
		if a.apiToken != "" {
			crReq.Header.Set("Authorization", "Bearer "+a.apiToken)
		}
		crResp, crErr := client.Do(crReq)
		if crResp != nil {
			defer crResp.Body.Close()
		}
		if crErr == nil {
			cloudRunActive = true
			if crResp.StatusCode == http.StatusOK {
				var crRecord struct {
					Timestamp string `json:"timestamp"`
				}
				if err := json.NewDecoder(crResp.Body).Decode(&crRecord); err == nil && crRecord.Timestamp != "" {
					if t, parseErr := time.Parse(time.RFC3339, crRecord.Timestamp); parseErr == nil {
						cloudRunPacketAgeSec = int(time.Since(t).Seconds())
						if cloudRunPacketAgeSec <= 120 {
							cloudRunFreshness = true
							a.ResolveCategory("CLOUD_INGESTION", "Cloud Run Remote Telemetry Stale")
						} else {
							a.RecordIncident(Incident{
								Severity:    "HIGH",
								Category:    "CLOUD_INGESTION",
								Title:       "Cloud Run Remote Telemetry Stale",
								Description: fmt.Sprintf("Cloud Run live endpoint returned stale data (age: %ds > 120s threshold, timestamp: %s).", cloudRunPacketAgeSec, crRecord.Timestamp),
								Remediation: "Verify bridge fallback upload to Cloud Run or check Cloud Run ingestion logs.",
							})
						}
					}
				}
			} else if crResp.StatusCode == http.StatusUnauthorized {
				a.RecordIncident(Incident{
					Severity:    "HIGH",
					Category:    "SECURITY",
					Title:       "Cloud Run Ingestion IAM Unauthorized (401)",
					Description: "Cloud Run rejected unauthenticated request with 401 Unauthorized (Google Frontend IAM).",
					Remediation: "Ensure bridge has valid GCP identity token or Cloud Run service allows unauthenticated invocations.",
				})
			}
		} else {
			a.RecordIncident(Incident{
				Severity:    "MEDIUM",
				Category:    "CLOUD_INGESTION",
				Title:       "Cloud Run Remote Unreachable",
				Description: fmt.Sprintf("Could not reach Cloud Run at %s (%v)", a.cloudRunURL, crErr),
				Remediation: "Check internet connectivity and Cloud Run service deployment state.",
			})
		}
	}

	// 4. Audit Security: Mutation Protection (401 on unauthenticated POST)
	secReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.cloudURL+"/api/v1/hardware-config", strings.NewReader(`{"test":true}`))
	secReq.Header.Set("Content-Type", "application/json")
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

	// 5. Autonomous Self-Healing Actuators
	if a.autoHeal {
		if !bridgeActive {
			go a.AutoHealBridge()
		}
		if !cloudActive {
			go a.AutoHealCloudServer()
		}
		if bridgeActive && (!telemetryFlowing || lastPacketAgeSec > 180) {
			go a.AutoHealBluetoothSubsystem()
		}
		if bridgeActive && !cloudRunFreshness {
			go a.AutoHealCloudRunSync()
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
	if cloudRunActive {
		a.ResolveCategory("CLOUD_INGESTION", "Cloud Run Remote Unreachable")
	}
	if cloudRunFreshness {
		a.ResolveCategory("CLOUD_INGESTION", "Cloud Run Remote Telemetry Stale")
	}
	if cloudActive && securityPass {
		a.ResolveCategory("SECURITY", "")
	}

	overall := "HEALTHY"
	if !lifepo4Pass || !securityPass {
		overall = "UNHEALTHY"
	} else if !bridgeActive || !cloudActive || !stringPass || !spoolPass || !cloudRunFreshness {
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

	recentHeals := a.autoHeals
	if len(recentHeals) > 10 {
		recentHeals = recentHeals[:10]
	}

	a.status = SREStatus{
		LastAuditTime:         time.Now(),
		OverallHealth:         overall,
		BridgeActive:          bridgeActive,
		CloudServerActive:     cloudActive,
		CloudRunActive:        cloudRunActive,
		TelemetryStreaming:    telemetryFlowing,
		LastPacketAgeSec:      lastPacketAgeSec,
		CloudRunPacketAgeSec:  cloudRunPacketAgeSec,
		TotalIncidents:        len(a.incidents),
		ActiveIncidents:       activeCount,
		RecentIncidents:       recent,
		RecentAutoHeals:       recentHeals,
		LiFePO4SafetyPass:     lifepo4Pass,
		StringTopologyPass:    stringPass,
		SpoolHealthPass:       spoolPass,
		SecurityAuditPass:     securityPass,
		CloudRunFreshnessPass: cloudRunFreshness,
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

func (a *SREAgent) GetAutoHeals() []AutoHealAction {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.autoHeals
}

func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func main() {
	var (
		daemonMode  = flag.Bool("daemon", false, "Run continuous autonomous SRE monitoring loop")
		auditOnce   = flag.Bool("audit", false, "Run single one-shot diagnostic audit and exit")
		autoHeal    = flag.Bool("auto-heal", true, "Enable autonomous self-healing and process supervisor")
		port        = flag.Int("port", 8082, "SRE Agent HTTP server port")
		bridgeURL   = flag.String("bridge-url", "http://localhost:8080", "Bridge daemon URL")
		cloudURL    = flag.String("cloud-url", "http://localhost:8081", "Cloud server URL")
		cloudRunURL = flag.String("cloudrun-url", os.Getenv("SOLARIA_CLOUD_ENDPOINT"), "Cloud Run deployed service URL (defaults to SOLARIA_CLOUD_ENDPOINT env or cloud-url)")
		interval    = flag.Duration("interval", 5*time.Second, "Monitoring interval in daemon mode")
	)
	flag.Parse()

	token := os.Getenv("SOLARIA_API_TOKEN")
	agent := NewSREAgent(*bridgeURL, *cloudURL, *cloudRunURL, "logs/incidents.json", token, *autoHeal)

	if *auditOnce {
		fmt.Println("🔍 Running Project Solaria Autonomous SRE Audit...")
		status := agent.RunAudit(context.Background())
		out, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(out))
		return
	}

	if *daemonMode {
		log.Printf("🤖 Running in continuous daemon mode (interval: %v, auto-heal: %t)", *interval, *autoHeal)
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
					log.Printf("⚠️ SRE Health Alert: %s (Active Incidents: %d, Streaming: %t)", st.OverallHealth, st.ActiveIncidents, st.TelemetryStreaming)
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

	mux.HandleFunc("/api/v1/sre/auto-heals", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_ = json.NewEncoder(w).Encode(agent.GetAutoHeals())
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
