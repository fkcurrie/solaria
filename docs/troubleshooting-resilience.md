# 🛡️ Troubleshooting & Remote Resilience System

Solaria is built for remote, unattended cottage installations where physical access to the charge controller or gateway is limited.

---

## 🔄 Dual-Layer Resilience Architecture

Solaria implements a multi-layer supervisor to guarantee continuous data streaming and prevent telemetry freeze:

```
+--------------------------------------------------------------------------------+
|                             SUPERVISOR ENGINE                                  |
|                                                                                |
|  [Layer 1: Go Outage & Watchdog Engine]                                        |
|  - Tracks elapsed seconds since last valid Modbus RTU packet                   |
|  - If quiet > 30s: Emits outage_start event, flags outage, and ticks downtime   |
|  - Triggers Linux BlueZ auto-heal: bluetoothctl power off/on, service reload    |
|  - Sends watchdog_reconnect signal over ws://localhost:8765                    |
|                                                                                |
|  [Layer 2: Browser Web Bluetooth Watchdog & WakeLock]                          |
|  - Acquires Screen WakeLock to prevent OS sleeping                             |
|  - Auto-reconnects GATT session to Renogy BT-1 (BT-TH-66F984D6)                |
|  - Auto-subscribes to FFF1 notify characteristic with infinite retry loop      |
|  - Resets Modbus chunk reassembly buffer on any GATT state change              |
+--------------------------------------------------------------------------------+
```

---

## 📊 Resilience & Outages Supervisor UI

The local web dashboard (`http://localhost:8080`) provides a dedicated Outage Supervisor Panel:

* **System Availability Gauge:** Real-time uptime score (e.g. `99.8%`).
* **Total Outages Counter:** Total count of communication interruptions.
* **Total Downtime Counter:** Cumulative duration of downtime.
* **Session Uptime:** Continuous uninterrupted operational time.
* **Live Outage Audit Log Table:** Historical ring-buffer capturing:
  * `#` Outage Number
  * `Status` (`Active (Healing...)` or `Resolved`)
  * `Interrupted At` / `Restored At` timestamps
  * `Duration` formatted in minutes/seconds
  * `Root Cause & Self-Healing Action` (e.g., `Linux BLE stack power-cycle & GATT reconnection`)

---

## 🛠️ Common Troubleshooting Scenarios

### 1. Bluetooth Device Is No Longer In Range / GATT Disconnect
* **Symptom:** Browser shows `GATT attempt failed: Bluetooth Device is no longer in range`.
* **Automatic Resolution:** The Go watchdog detects no frames for > 30s and automatically initiates BlueZ host power recycling. The browser client retries GATT acquisition every 3 seconds until re-paired.
* **Manual Trigger:** Run `bluetoothctl power off && sleep 1 && bluetoothctl power on` or click **"Disconnect"** and then **"Scan for Renogy BT-1"** on `http://localhost:8080`.

### 2. Missing Weather Irradiance / Overcast Data
* **Symptom:** Irradiance shows `0.0 W/m²` during daytime or cloud cover shows `--`.
* **Check:** Ensure outbound HTTPS connectivity to Open-Meteo (`https://api.open-meteo.com/v1/forecast`).

### 3. BigQuery Streaming Ingestion Errors
* **Symptom:** Dashboard logs show `401 Unauthorized` or `BigQuery insert failed`.
* **Check:** Ensure `SOLARIA_API_TOKEN` matches the secret configured in Cloud Run and that `GOOGLE_APPLICATION_CREDENTIALS` (or Cloud Run Default Service Account) has `roles/bigquery.dataEditor` permissions.
