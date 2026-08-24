# ☀️ Solaria

> **High-Performance Renogy Solar & Atmospheric Intelligence Platform (Pure Go)**

Solaria connects Renogy Solar MPPT Charge Controllers (Rover, Wanderer, Adventurer) via BT-1/BT-2 Bluetooth Low Energy modules to live atmospheric radiometry and Google BigQuery analytics.

---

## ⚡ Quick Start (One-Liner Install)

Run the universal installer on Linux, macOS, or Raspberry Pi:

```bash
curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/setup.sh | bash
```

Or clone and start locally:

```bash
git clone https://github.com/fkcurrie/solaria.git
cd solaria
./setup.sh --start-bridge
```

Open **[http://localhost:8080](http://localhost:8080)** in Chrome to connect to your Renogy BT-1 module and view live telemetry.

---

## 🤖 AI Agent & Automation Quickstart

Point **Antigravity**, **Gemini CLI**, **Claude Code**, or **Cursor** directly at this repository. The setup script provides native machine-readable discovery and execution modes:

```bash
# Discover system capabilities, Go version, and config defaults
./setup.sh --agent-mode

# Execute non-interactive setup with environment variables
./setup.sh --non-interactive

# Automatically install Go if missing and start bridge
./setup.sh --install-deps --start-bridge
```

---

## 🌟 Key Capabilities

* **🏎️ High-Speed Go Engine:** Real-time Modbus RTU chunk reassembly and CRC16 verification running on a sub-millisecond Go runtime.
* **🌤️ Atmospheric Solar Intelligence:** Enriches every 10s telemetry frame with Open-Meteo radiometry ($\text{GHI}, \text{DHI}, \text{DNI}$, Cloud Cover %, Solar Elevation) and calculates real-time **Atmospheric Performance Ratio (PR %)**.
* **🛡️ Remote Resilience & Outages Supervisor:** Self-healing watchdog with Linux Bluetooth auto-recovery, Web Bluetooth WakeLock, and continuous outage/availability audit logging.
* **🗄️ Google Cloud BigQuery Streaming:** Zero-data-loss buffered streaming into partitioned BigQuery tables (`solaria-solar.solaria.telemetry`).
* **📊 Modern Responsive UI:** Clean live telemetry dashboard with 400W array utilization gauges, battery SOC tracking, and historical trend analysis.

---

## 📚 Detailed Engineering Documentation

Deep technical guides, hardware schematics, and schema references are organized in the [`docs/`](docs/) directory:

| Document | Description |
| :--- | :--- |
| **[🏛️ System Architecture](docs/architecture.md)** | End-to-end data pipeline, Web Bluetooth GATT characteristics, Modbus registers, and microservices. |
| **[⚡ Solar Array Specifications](docs/solar-specifications.md)** | 400W 2S2P panel wiring topology, Rover 20A MPPT window, over-paneling ratio, and battery chemistry profiles. |
| **[🌤️ Atmospheric Physics](docs/atmospheric-physics.md)** | Solar irradiance math, Performance Ratio (PR) formulation, and Sun Condition classification rules. |
| **[🗄️ BigQuery Schema](docs/bigquery-schema.md)** | Complete 34-register schema definition and analytical SQL query examples. |
| **[🚀 Deployment & Services](docs/deployment.md)** | Google Cloud Run microservice deployment and headless systemd service configuration for Raspberry Pi. |
| **[🛡️ Resilience & Outages](docs/troubleshooting-resilience.md)** | Outage tracking engine, autonomous BlueZ supervisor, and remote troubleshooting guide. |

---

## 🌐 Live Cloud Dashboard

The production cloud service is hosted on Google Cloud Run:
👉 **[https://solaria-dashboard-952659886764.us-central1.run.app](https://solaria-dashboard-952659886764.us-central1.run.app)**

---

## 📜 License

MIT License © Frank Currie
