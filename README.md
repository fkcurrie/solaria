# Solaria: Solar Energy Intelligence & Cloud Analytics

<p align="center">
  <img src="cmd/cloud-server/static/assets/solaria-logo.svg" alt="Solaria Solar Energy Intelligence & Analytics" width="180" />
</p>

<p align="center">
  <strong>Autonomous Off-Grid Solar & LiFePO4 Energy Intelligence Platform for Renogy Rover MPPT & Google Cloud Run.</strong>
</p>

<p align="center">
  <a href="https://fkcurrie.github.io/solaria/"><img src="https://img.shields.io/badge/docs-Material_MkDocs-amber.svg?logo=materialformkdocs" alt="Documentation Site"></a>
  <a href="https://github.com/fkcurrie/solaria/actions/workflows/ci.yml"><img src="https://github.com/fkcurrie/solaria/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
  <a href="https://github.com/fkcurrie/solaria/actions/workflows/bootstrap-ci.yml"><img src="https://github.com/fkcurrie/solaria/actions/workflows/bootstrap-ci.yml/badge.svg" alt="Multi-Distro Matrix"></a>
  <a href="https://github.com/fkcurrie/solaria/actions/workflows/docs.yml"><img src="https://github.com/fkcurrie/solaria/actions/workflows/docs.yml/badge.svg" alt="Docs CI"></a>
  <img src="https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT">
  <img src="https://img.shields.io/badge/Platform-Raspberry_Pi_%7C_Linux_%7C_Cloud_Run-green" alt="Platform">
</p>

---

## 📖 Complete Documentation & Guides

Comprehensive architecture diagrams, wiring schematics, LiFePO4 safety guidelines, and API references are available on our official documentation site:

👉 **[https://fkcurrie.github.io/solaria/](https://fkcurrie.github.io/solaria/)**

---

## ⚡ Overview

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
* **Atmospheric Correlation & Shading Analyzer:** Enriches telemetry with solar radiometry (GHI, DHI, DNI, cloud cover) from Open-Meteo, computes real-time Performance Ratio (PR), and runs the built-in Tree Shading Diagnostic Analyzer to quantify harvest energy lost to canopy occlusion.
* **Celestial Sun Trajectory Engine:** Real-time SVG celestial dome mapping solar elevation, azimuth, and panel normal angles ($45.186^\circ\text{N}, -78.863^\circ\text{W}$).
* **Autonomous SRE Supervisor (`cmd/sre-agent`):** Continuous health auditing, auto-healing restarts, and 3-tier Bluetooth radio recovery.
* **Connection Resilience:** Background watchdog with automatic BlueZ power cycling, Web Bluetooth WakeLock, and continuous outage logging.
* **Storage Options:** Configurable local CSV/spool storage and buffered streaming inserts into date-partitioned BigQuery tables.
* **Renogy BLE Hardware Emulator (`renogy-bt-emulator`):** Standalone pure Go BLE GATT server with 23,800+ real-world telemetry replay fixtures for offline testing.

---

## Documentation & Roadmap

* [System Architecture](docs/architecture.md) — BLE GATT services, data pipeline, and Modbus frame format.
* [Solar Hardware Specifications](docs/solar-specifications.md) — 400W 2S2P electrical wiring, Rover 20A MPPT limits, and battery profiles.
* [Atmospheric Formulas & Tree Shading](docs/physics/tree-shading.md) — Irradiance math, shading occlusion formulas, and canopy analysis.
* [BigQuery Telemetry Schema](docs/bigquery-schema.md) — 34-column schema reference and analytical SQL queries.
* [Deployment Guide](docs/deployment.md) — Local edge setup, storage configuration, and Cloud Run deployment.
* [Resilience & Troubleshooting](docs/troubleshooting-resilience.md) — Watchdog mechanics, outage tracking, and recovery procedures.
* [Core Soul & Multi-Persona Governance](docs/SOUL.md) — Architectural invariants and multi-persona review system.
* [Google Principal UX Review & 15 Issues](docs/ux-redesign/README.md) — [EPIC-002](docs/ux-redesign/EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md) and 15 design issues.
* [Renogy BT-1 / BT-2 BLE Emulator](renogy-bt-emulator/README.md) — Standalone pure Go physical BLE hardware emulator ([EPIC-001](renogy-bt-emulator/EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)) and [Telemetry Fixtures](renogy-bt-emulator/fixtures/telemetry/README.md).


---

## License

MIT (c) Frank Currie
