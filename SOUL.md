# ☀️ Project Solaria: Core Tenets & System Soul (SOUL.md)

> *"Solar power is not just voltage and current; it is a live conversation between the sun, the local atmosphere, the PV string topology, and the battery chemistry."*

---

## 🧭 The Core Mission

To build a resilient, highly accurate, edge-to-cloud solar monitoring and intelligence platform for off-grid and battery systems powered by Renogy charge controllers (BT-1/BT-2 RS232/RS485). The system continuously correlates actual photovoltaic energy harvest with localized atmospheric conditions at **1296 Wren Lake Drive, Dorset, Ontario, Canada (45.186° N, 78.863° W)** to provide actionable insights on solar performance, shading, cloud attenuation, and battery health, streaming real-time and historical analytics into **Google BigQuery** (`solaria-solar.solaria.telemetry`).

---

## ⚡ Solar Array & Hardware Profile

### Photovoltaic Array Specifications

- **Total Array Capacity:** **400 Watts Peak ($400\text{Wp}$)**
- **Modules:** $4 \times 100\text{W}$ Monocrystalline Panels
- **Array Topology:** **2S2P (2 Series $\times$ 2 Parallel Strings)**
  - Two parallel branches, each consisting of two 100W panels wired in series.
  - **Nominal Series String $V_{mp}$:** $\approx 36.0\text{V} - 40.8\text{V}$ (Ideal for MPPT buck-converter efficiency).
  - **Array Open-Circuit Voltage $V_{oc}$:** $\approx 43.2\text{V} - 48.6\text{V}$ (Safely below Rover 100V limit).
  - **Array Max Power Current $I_{mp}$:** $\approx 9.8\text{A} - 11.0\text{A}$.
- **Charge Controller:** **Renogy Rover 20A MPPT** (`RNG-CTRL-RVR20`)
  - **Bluetooth Interface:** Renogy BT-1 RS232 Module (`BT-TH-66F984D6`).
  - **Max Charging Current:** 20A DC.
  - **Max PV Input Voltage:** 100V DC.
  - **Over-Paneling Ratio:** $400\text{W} / 288\text{W} \approx 138\%$ (Provides optimal early-morning, late-afternoon, and overcast harvesting in northern climates while controller current-limits to 20A at peak sun).

---

## 🏛️ Guiding Architectural Tenets

### 1. Edge-First Resilience (Never Drop a Watt-Hour)

- **Offline Autonomy:** The edge node (Raspberry Pi or Linux micro-server) operates autonomously. If cottage Wi-Fi drops, telemetry spools locally with zero data loss.
- **Self-Healing BLE Link:** Bluetooth Low Energy links to the BT-1 module implement automatic reconnection and chunk reassembly without human intervention.
- **Pure Modbus RTU:** Communication uses standard Modbus RTU frames over transparent BLE characteristics (`0xFFD1` TX / `0xFFF1` RX).

### 2. Environmental & Performance Fusion

- **Solar Irradiance Fusion:** Every telemetry packet is paired with real-time solar physics data for Dorset:
  - **GHI (Global Horizontal Irradiance in $W/m^2$)**
  - **DNI (Direct Normal Irradiance in $W/m^2$)**
  - **DHI (Diffuse Horizontal Irradiance in $W/m^2$)**
  - **Cloud Cover Percentage (%)**
  - **Ambient Temperature (°C)** vs Controller & Battery Internal Temperatures
- **Performance Analytics:**
  - **Array Utilization %:** $\frac{P_{\text{pv}}}{400\text{W}} \times 100\%$
  - **Performance Ratio (PR %):** $\frac{P_{\text{pv}}}{\text{Expected Irradiance Watts}} \times 100\%$
- **Condition Classification Engine:**
  - `FULL_SUN`: PV harvest matches $\ge 70\%$ of 400W theoretical irradiance capacity.
  - `PARTIAL_SUN_OR_SHADE`: Variable harvest indicative of passing clouds or tree shadows.
  - `DIFFUSE_OVERCAST`: Low harvest dominated by diffuse light ($DHI \gg DNI$).
  - `ABSORPTION_FLOAT_CLIPPED`: Battery SOC $\ge 99\%$ causing controller to back off PV input.
  - `NIGHT`: Sun elevation $< 0^\circ$ or PV Voltage $< 5\text{V}$.

### 3. Serverless Analytical Store

- **Google Cloud BigQuery:** Telemetry is streamed directly into `solaria-solar.solaria.telemetry` with daily time-partitioning on `timestamp` and clustering on `site` and `sun_classification`.
- **Google Cloud Run:** Go dashboard microservice providing real-time ring buffer streaming and analytical visualization.

---

## 📍 Site Profile: Dorset, Ontario Installation

- **Location:** 1296 Wren Lake Drive, Dorset, Ontario, Canada
- **Coordinates:** `45.186° N, 78.863° W` (Algonquin Highlands)
- **Elevation:** ~350m above sea level
- **Climate Context:** Sub-boreal northern climate with high summer irradiance (~6.0 peak sun hours/day) and reduced winter insolation (~1.5 peak sun hours/day).
