# 📁 Real Hardware Telemetry & Trace Replay Fixtures

> **Authentic solar generation, battery electrochemistry, and anomaly traces captured from Project Solaria's off-grid Renogy Rover 20A MPPT + 12V 170Ah LiFePO4 battery system.**

This directory contains high-resolution (10-second sampling interval) time-series telemetry recorded directly from physical Renogy hardware via the BT-1 BLE adapter. These datasets serve as ground-truth test fixtures and trace-replay data for the **Renogy BT-1 / BT-2 BLE Hardware Emulator in Go**.

---

## 📊 Dataset Inventory

| Trace ID | Filename | Records | Time Span | Key Characteristics & Phenoma |
| :--- | :--- | :---: | :--- | :--- |
| `trace-2026-08-23` | [`solar_telemetry_2026-08-23.csv`](./solar_telemetry_2026-08-23.csv) | 1,463 | Aug 23 20:00 - 23:59 UTC | Late afternoon solar taper, dusk transition, inverter standby load (15W). |
| `trace-2026-08-24` | [`solar_telemetry_2026-08-24.csv`](./solar_telemetry_2026-08-24.csv) | 13,262 | Full 24 Hours | Complete diurnal solar trajectory, sunrise wake-up, Bulk MPPT charging, Boost/Absorption plateau at 14.2V–14.4V, Float transition (13.6V), 100% SoC maintenance. |
| `trace-2026-08-25` | [`solar_telemetry_2026-08-25.csv`](./solar_telemetry_2026-08-25.csv) | 6,623 | Aug 25 00:00 - 20:53 UTC | Peak irradiance, distinct midday tree shading occlusion dips, controller heat sink thermal derating. |
| `trace-2026-08-26` | [`solar_telemetry_2026-08-26.csv`](./solar_telemetry_2026-08-26.csv) | 2,477 | Aug 26 00:00 - 18:30 UTC | Real-world transient string imbalance / bypass diode drop event at 17:36 UTC ($V_{\text{pv}} = 21.9\text{V}$ under $676\text{ W/m}^2$). |

---

## 📐 CSV Column Schema & Modbus Register Mapping

Every CSV row directly maps to Renogy Modbus RTU holding registers (`0x0100` through `0x0122`):

| CSV Column | Unit | Renogy Modbus Register | Scaling / Decoding |
| :--- | :---: | :---: | :--- |
| `timestamp` | ISO 8601 | Edge system clock | Nanosecond UTC timestamp |
| `pv_power_w` | W | `0x0109` | 16-bit unsigned integer (Watts) |
| `pv_voltage_v` | V | `0x0107` | 16-bit integer divided by 10 (0.1V resolution) |
| `pv_current_a` | A | `0x0108` | 16-bit integer divided by 100 (0.01A resolution) |
| `battery_soc_pct` | % | `0x0100` | 16-bit integer (0–100%) |
| `battery_voltage_v` | V | `0x0101` | 16-bit integer divided by 10 (0.1V resolution) |
| `battery_current_a` | A | `0x0102` | 16-bit integer divided by 100 (0.01A resolution) |
| `charging_state` | Text | `0x0120` (bits 0-3) | `Deactivated`, `MPPT_Bulk`, `Boost`, `Float`, `Equalizing` |
| `controller_temp_c` | °C | `0x0103` (High byte) | 8-bit signed integer (°C) |
| `battery_temp_c` | °C | `0x0103` (Low byte) | 8-bit signed integer (°C) |
| `load_power_w` | W | `0x010E` | 16-bit integer (Watts) |
| `daily_generated_wh`| Wh | `0x0113` | 16-bit integer (Watt-hours) |
| `total_generated_kwh`| kWh | `0x0114`-`0x0115` | 32-bit unsigned integer (kWh) |

---

## 🧠 Learned Solar Array & Shading Profile

- [`solar_model_learned.json`](./solar_model_learned.json):
  Contains calibrated electrical and astronomical model parameters learned from the physical installation:
  - **Peak Array Capacity**: 400W nominal ($V_{\text{mp}} \approx 36\text{V}$, $I_{\text{mp}} \approx 11.1\text{A}$)
  - **Panel Temperature Coefficient**: $-0.4\% / ^\circ\text{C}$ relative to $25^\circ\text{C}$ STC
  - **Horizon Elevation & Shading Loss Profile**: Calibrated loss coefficients across solar azimuths ($90^\circ \to 270^\circ$).

---

## 🔁 Using with the Emulator

When running the emulator in trace-replay mode:
```bash
./bin/renogy-bt-emulator -mode=replay -trace=fixtures/telemetry/solar_telemetry_2026-08-24.csv -speed=10x
```
