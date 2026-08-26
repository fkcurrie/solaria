# Autonomous SRE Supervisor (`solaria-sre-agent`)

The **Solaria SRE Supervisor** is an autonomous self-healing agent running on port `:8082`. It continuously executes end-to-end health audits, detects stalled processes, and self-heals system degradation.

---

## 🤖 3-Minute Autonomous Health Cycle

```mermaid
sequenceDiagram
    autonumber
    participant SRE as solaria-sre-agent (:8082)
    participant Bridge as solaria-bridge (:8080)
    participant Cloud as Cloud Run (:8081)
    participant Radio as Linux Bluetooth Stack

    loop Every 2 Minutes
        SRE->>Bridge: GET /api/v1/status
        alt Bridge Silent (> 60s)
            SRE->>Radio: Step 1: Execute 3-Tier Radio Reset
            SRE->>Bridge: Step 2: Restart solaria-bridge process
            SRE->>SRE: Log Auto-Heal Incident (logs/incidents.json)
        else Bridge Healthy
            SRE->>Cloud: GET /api/v1/live
            SRE->>SRE: Assert SubZero Inhibit & Diode Symmetry
        end
    end
```

---

## 📋 Forensic Incident Tracking

All incidents are persisted in `logs/incidents.json` with root cause diagnostics and resolution timestamps:

```json
{
  "id": "INC-1787748923514",
  "timestamp": "2026-08-26T12:55:23Z",
  "severity": "HIGH",
  "category": "AUTO_HEAL",
  "title": "Autonomous Bridge Daemon Self-Heal Triggered",
  "description": "Bridge process failed health check; spawned ./bin/solaria-bridge (PID: 781155)",
  "resolved": true
}
```
