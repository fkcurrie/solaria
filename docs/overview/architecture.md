# System Architecture

Solaria is built upon a layered **edge-to-cloud telemetry and analytical architecture**. It guarantees continuous local operation and offline data persistence during network drops, with streaming synchronization to Google Cloud when connectivity is available.

---

## 🏛️ Layered Topology

```mermaid
graph TD
    subgraph "Layer 1: Physical Solar & Battery"
        PV["400W Solar Array<br/>(2S2P 36V @ 11A)"]
        BATT["12V 170Ah LiFePO4<br/>(2,176 Wh Capacity)"]
        LOAD["12V Cottage DC Loads<br/>(Starlink, Fridge, Lights)"]
        MPPT["Renogy Rover 20A MPPT<br/>Charge Controller"]
        
        PV --> MPPT
        BATT <--> MPPT
        MPPT --> LOAD
    end

    subgraph "Layer 2: Edge Hardware & Bluetooth Bridge"
        BT1["Renogy BT-1 BLE Module<br/>(Service: 0xFFD0)"]
        RPI["Raspberry Pi / Linux Edge<br/>(Appliance OS)"]
        BRIDGE["solaria-bridge (:8080)<br/>Web Bluetooth Gateway"]
        SPOOL["Persistent Disk Spooler<br/>(spool/queue.jsonl)"]
        SRE["solaria-sre-agent (:8082)<br/>Autonomous SRE Supervisor"]

        MPPT -- "RS232 / RJ12" --> BT1
        BT1 -. "BLE 2.4GHz" .-> BRIDGE
        BRIDGE --> SPOOL
        SRE -. "Radio Watchdog" .-> BRIDGE
    end

    subgraph "Layer 3: Google Cloud Analytics"
        CR["Google Cloud Run<br/>(Ingestion API & Dashboard)"]
        BQ[("Google BigQuery<br/>Partitioned Telemetry")]
        WX["Open-Meteo API<br/>(Atmospheric Forecast)"]

        BRIDGE -- "HTTPS / API Token" --> CR
        SPOOL -- "Auto-Drain on Reconnect" --> CR
        CR --> BQ
        CR <--> WX
    end

    subgraph "Layer 4: User Interface & PWA"
        DASH["Responsive Web PWA<br/>(10 Analytical Panes)"]
        CR --> DASH
    end
```

---

## 🔄 End-to-End Data Flow

1. **Modbus RTU Polling:** The charge controller continuously monitors panel voltage, array current, battery voltage, temperature, and internal charging states.
2. **Bluetooth Low Energy Framing:** The Renogy BT-1 module packages Modbus RTU frames into BLE characteristics under UUID `0000ffd1-0000-1000-8000-00805f9b34fb`.
3. **Edge Bridge Ingestion:** The `solaria-bridge` daemon parses binary frames, computes solar physics invariants (cold derating, string symmetry, DC-DC efficiency), and displays a live ASCII telemetry console.
4. **Resilient Spooling:** If network connectivity to Cloud Run is lost, frames are appended atomically to `spool/queue.jsonl`.
5. **BigQuery Synchronization:** As soon as LTE/WiFi connectivity returns, the spool drainer flushes buffered records in rate-limited batches into Google BigQuery.
6. **Live Analytics & PWA Dashboard:** The single-page dashboard renders 1-second real-time metrics, historical 24-hour diurnal curves, battery health indicators, and power budgeting tools.
