# Troubleshooting & Log Streaming

Solaria provides integrated real-time diagnostic log buffers across all bridge, edge, and cloud daemons.

---

## 🔍 Diagnostic Probing Commands

### 1. Run Complete E2E Audit (21 Automated Checks)
```bash
./bin/solaria-e2e-audit
```

### 2. Stream Live Edge Bridge Diagnostic Logs
```bash
curl -s http://localhost:8080/api/v1/diagnostics | jq
```

### 3. Check Incident Forensic Log
```bash
cat logs/incidents.json | jq
```

---

## ⚠️ Common Issues & Immediate Fixes

### BT-1 Connection Refused / BLE Device Not Found
- **Cause:** Bluetooth radio is soft-blocked or adapter index changed.
- **Remediation:**
  ```bash
  sudo rfkill unblock bluetooth
  sudo hciconfig hci0 reset
  sudo systemctl restart bluetooth
  ```

### Telemetry Latency / Stale Cloud Data
- **Cause:** WiFi or LTE connection dropped; bridge is actively spooling.
- **Remediation:**
  Check spool backlog size:
  ```bash
  curl -s http://localhost:8080/api/v1/status | jq '.spool_backlog'
  ```
  If backlog $> 0$, verify internet uplink; the bridge will drain automatically once connected.
