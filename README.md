# Solaria

Solaria streams telemetry from Renogy solar charge controllers (Rover, Wanderer, Adventurer) via Bluetooth Low Energy (BT-1 / BT-2) to local disk logs and optional Google BigQuery analytics, correlating solar harvest against local atmospheric weather data in real time.

```text
[Renogy Controller] ──(BT-1 BLE)──> [Go Bridge / Edge Agent] ──(Local Logs: CSV/JSON)
                                            │
                                            ├──(Open-Meteo Weather)
                                            │
                                            └──(Optional Cloud Run HTTPS)──> [BigQuery]
```

## Quickstart

### Automated Install

Run the installer on Linux, macOS, or Raspberry Pi:

```bash
curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/setup.sh | bash
```

### Manual Setup

```bash
git clone https://github.com/fkcurrie/solaria.git
cd solaria
go run ./cmd/bridge
```

Open `http://localhost:8080` in Chrome to pair with the Renogy BT-1 module.

---

## AI Agent Integration

This repository supports automated discovery for agentic coding tools (Gemini CLI, Antigravity, Claude Code, Cursor):

```bash
# Print machine-readable environment & config schema
./setup.sh --agent-mode

# Run non-interactive configuration
./setup.sh --non-interactive

# Install dependencies and start bridge daemon
./setup.sh --install-deps --start-bridge

# Configure for local-only storage (no cloud transmission)
./setup.sh --storage-mode local --non-interactive
```

---

## Storage Modes

Solaria supports three storage configurations controlled via `STORAGE_MODE` in `.env`:

* **`local`:** Logs telemetry directly to the Raspberry Pi or Linux host filesystem (`logs/solar_telemetry_YYYY-MM-DD.csv` and `spool.jsonl`). Cloud ingestion is disabled.
* **`bigquery`:** Streams telemetry to Google Cloud Run for insertion into BigQuery (`solaria-solar.solaria.telemetry`).
* **`both` (Default):** Writes local CSV logs and streams to Google BigQuery.

---

## System Overview

* **Modbus RTU Engine (`cmd/bridge`):** Go-based BLE packet reassembly, CRC16 validation, and WebSocket gateway.
* **Atmospheric Correlation:** Enriches telemetry with solar radiometry (GHI, DHI, DNI, cloud cover) from Open-Meteo to compute real-time Performance Ratio (PR).
* **Connection Resilience:** Background watchdog with automatic BlueZ power cycling, Web Bluetooth WakeLock, and continuous outage logging.
* **Storage Options:** Configurable local CSV/spool storage and buffered streaming inserts into date-partitioned BigQuery tables.

---

## Documentation

* [System Architecture](docs/architecture.md) — BLE GATT services, data pipeline, and Modbus frame format.
* [Solar Hardware Specifications](docs/solar-specifications.md) — 400W 2S2P electrical wiring, Rover 20A MPPT limits, and battery profiles.
* [Atmospheric Formulas & PR](docs/atmospheric-physics.md) — Irradiance math, Performance Ratio formulas, and classification states.
* [BigQuery Telemetry Schema](docs/bigquery-schema.md) — 34-column schema reference and analytical SQL queries.
* [Deployment Guide](docs/deployment.md) — Local edge setup, storage configuration, and Cloud Run deployment.
* [Resilience & Troubleshooting](docs/troubleshooting-resilience.md) — Watchdog mechanics, outage tracking, and recovery procedures.

---

## License

MIT (c) Frank Currie
