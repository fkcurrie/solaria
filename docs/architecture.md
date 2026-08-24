# 🏛️ Solaria System Architecture

Solaria is an end-to-end, high-performance **Golang** telemetry, atmospheric intelligence, and BigQuery analytics platform designed for Renogy Solar Charge Controllers equipped with BT-1 (RS232) or BT-2 (RS485) Bluetooth Low Energy modules.

---

## High-Level Data Flow

```mermaid
flowchart TD
    subgraph Array["☀️ PV Array (1296 Wren Lake Dr, Dorset, ON)"]
        P1["100W Panel A1"] --- P2["100W Panel A2\n(Series String 1: ~36V-40V)"]
        P3["100W Panel B1"] --- P4["100W Panel B2\n(Series String 2: ~36V-40V)"]
        P2 -->|"Parallel Join (2S2P = 400Wp)"| Rover["Renogy Rover 20A MPPT\n(100V PV max, 20A charge)"]
        P4 -->|"Parallel Join (2S2P = 400Wp)"| Rover
        Rover -->|"12V DC"| Batt["Battery Bank (12V LiFePO4)"]
    end

    subgraph Edge["📡 Edge Layer (Linux Gateway / Raspberry Pi)"]
        Rover -->|"BT-1 Module (RS232)"| BT["BLE GATT Service 0xFFD0\n(BT-TH-66F984D6)"]
        BT -->|"Modbus RTU over BLE\n(FFD1 TX / FFF1 RX)"| Bridge["Go Bridge Daemon (:8765 / :8080)\n(cmd/bridge)"]
        Weather["Open-Meteo Weather API\n(45.186° N, -78.863° W)"] -->|"Solar Irradiance (GHI/DHI)\nCloud Cover %"| Bridge
        Bridge -->|"Disk Spooler\n(Zero Data Loss)"| Spooler[("spool.jsonl")]
        Spooler -->|"Replay on Restore"| Uploader["HTTPS Cloud Ingest Client"]
    end

    subgraph GCP["☁️ Google Cloud Platform (Project: solaria-solar)"]
        Uploader -->|"HTTPS POST /api/v1/telemetry\n(Bearer Token Auth)"| CloudRun["Go Cloud Run Microservice\n(solaria-dashboard)"]
        CloudRun -->|"Asynchronous Streaming Insert"| BigQuery[("Google BigQuery\nsolaria-solar.solaria.telemetry\n(Day-Partitioned)")]
        CloudRun -->|"Live Web UI & REST APIs"| CloudDashboard["Cloud Dashboard\n(Chart.js, 400W Gauge, Wx Overlay)"]
    end

    subgraph User["📱 User & Monitoring Clients"]
        Bridge --> LocalUI["Localhost Web Dashboard (:8080)\n(Live Telemetry, GATT Controller, Outage Monitor)"]
        CloudDashboard --> RemoteUI["Remote Mobile & Desktop Web\n(Historical Analytics, Atmospheric PR)"]
    end
```

---

## Component Breakdown

### 1. Edge Bridge (`cmd/bridge`)
* **Pure Go Server:** Runs on the local host (Linux/macOS/Raspberry Pi) serving the local dashboard on `http://localhost:8080` and a WebSocket control gateway on `ws://localhost:8765`.
* **Modbus RTU Framing & Verification:**
  * Reassembles streaming BLE chunks.
  * Validates Modbus RTU CRC16 checksums before decoding.
  * Handles standard 34-register telemetry blocks (0x0100–0x0122) and battery profile registers (0xE004–0xE012).
* **Autonomous Resilience Supervisor:**
  * Background goroutine monitors telemetry arrival every 5 seconds.
  * Detects disconnections and triggers automatic `bluetooth.service` restarts and `bluetoothctl` power re-engagements.
  * Emits `watchdog_reconnect` signals over WebSocket to initiate Web Bluetooth GATT re-pairing.
* **Outage Tracker & Logger:**
  * Tracks availability percentage, downtime duration, outage count, and session uptime.
  * Streams real-time outage notifications and session history to connected browser clients.

### 2. Standalone Edge Agent (`cmd/edge-agent`)
* Headless background daemon for headless Linux/Raspberry Pi deployments.
* Manages direct BlueZ DBus BLE connections, queries Open-Meteo weather coordinates, buffers data to local disk (`spool.jsonl`), and syncs to Google Cloud Run when online.

### 3. Cloud Server (`cmd/cloud-server`)
* **Google Cloud Run Microservice:**
  * Provides authenticated ingestion endpoint `POST /api/v1/telemetry` with Bearer token authentication.
  * Streams incoming records directly to Google BigQuery asynchronously.
  * Serves server-side rendered dashboard with dynamic timeframes (Day, Week, Month, Year).

---

## BLE & Modbus Communication Specification

* **Primary Service UUID:** `0000ffd0-0000-1000-8000-00805f9b34fb` (Write / Command Channel)
  * **TX Characteristic:** `0000ffd1-0000-1000-8000-00805f9b34fb` (Write Without Response / Write Request)
* **Secondary Service UUID:** `0000fff0-0000-1000-8000-00805f9b34fb` (Read / Notification Channel)
  * **RX Characteristic:** `0000fff1-0000-1000-8000-00805f9b34fb` (Notify / Modbus Response)
* **Standard Polling Query:**
  * Device Address: `0x01`
  * Function Code: `0x03` (Read Holding Registers)
  * Start Register: `0x0100`
  * Register Count: `34` (`0x0022`)
  * Hex Payload: `01 03 01 00 00 22 75 E6`
