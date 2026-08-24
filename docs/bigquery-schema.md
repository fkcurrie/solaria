# 🗄️ BigQuery Schema & Querying Reference

Solaria streams telemetry records directly into Google BigQuery partitioned by timestamp date:

* **Project:** `solaria-solar`
* **Dataset:** `solaria`
* **Table:** `telemetry` (`solaria-solar.solaria.telemetry`)
* **Partitioning:** Day-partitioned on `timestamp` column.

---

## Complete Table Schema

| Field Name | Type | Mode | Description |
| :--- | :--- | :--- | :--- |
| `timestamp` | `TIMESTAMP` | REQUIRED | UTC telemetry ingestion timestamp |
| `site` | `STRING` | NULLABLE | Geographic site identifier (e.g., `1296 Wren Lake Drive`) |
| `latitude` | `FLOAT64` | NULLABLE | Site latitude coordinate (`45.186`) |
| `longitude` | `FLOAT64` | NULLABLE | Site longitude coordinate (`-78.863`) |
| `array_capacity_w` | `FLOAT64` | NULLABLE | Total array peak nameplate capacity (`400.0`) |
| `array_topology` | `STRING` | NULLABLE | Wiring topology (`2S2P`) |
| `array_utilization_pct` | `FLOAT64` | NULLABLE | Instantaneous generation utilization ($P_{\text{pv}} / 400\text{W}$) |
| `performance_ratio_pct` | `FLOAT64` | NULLABLE | Harvesting ratio relative to atmospheric irradiance |
| `pv_power_w` | `FLOAT64` | NULLABLE | Solar array output power ($W$) |
| `pv_voltage_v` | `FLOAT64` | NULLABLE | Solar array terminal voltage ($V$) |
| `pv_current_a` | `FLOAT64` | NULLABLE | Solar array output current ($A$) |
| `battery_soc_pct` | `INT64` | NULLABLE | Battery State of Charge ($0\% - 100\%$) |
| `battery_voltage_v` | `FLOAT64` | NULLABLE | Battery terminal voltage ($V$) |
| `battery_current_a` | `FLOAT64` | NULLABLE | Net charging/discharging current ($A$) |
| `controller_temp_c` | `FLOAT64` | NULLABLE | Internal Rover controller temperature ($^\circ\text{C}$) |
| `battery_temp_c` | `FLOAT64` | NULLABLE | Battery temperature sensor reading ($^\circ\text{C}$) |
| `charging_state` | `STRING` | NULLABLE | Modbus state (`Deactivated`, `Activated`, `MPPT Charging`, `Equalizing`, `Boost`, `Float`, `Current Limiting`) |
| `load_status` | `STRING` | NULLABLE | DC load status (`ON` or `OFF`) |
| `fault_flags` | `INT64` | NULLABLE | Bitmask of controller fault flags |
| `daily_min_battery_voltage_v` | `FLOAT64` | NULLABLE | Daily minimum battery voltage recorded |
| `daily_max_battery_voltage_v` | `FLOAT64` | NULLABLE | Daily maximum battery voltage recorded |
| `daily_max_pv_w` | `FLOAT64` | NULLABLE | Daily peak solar power recorded ($W$) |
| `daily_generated_wh` | `FLOAT64` | NULLABLE | Daily cumulative energy generated ($\text{Wh}$) |
| `operating_days` | `INT64` | NULLABLE | Total operating days counter |
| `total_battery_fullcharge_count`| `INT64` | NULLABLE | Lifetime full charge cycles counter |
| `total_charging_ah` | `INT64` | NULLABLE | Lifetime charging Amp-hours |
| `total_generated_kwh` | `INT64` | NULLABLE | Lifetime cumulative energy ($\text{kWh}$) |
| `weather_temp_c` | `FLOAT64` | NULLABLE | Ambient outdoor temperature from Open-Meteo ($^\circ\text{C}$) |
| `weather_cloud_cover_pct` | `FLOAT64` | NULLABLE | Cloud cover percentage ($0\% - 100\%$) |
| `weather_direct_rad_w_m2` | `FLOAT64` | NULLABLE | Direct normal irradiance ($\text{W/m}^2$) |
| `weather_diffuse_rad_w_m2` | `FLOAT64` | NULLABLE | Diffuse horizontal irradiance ($\text{W/m}^2$) |
| `sun_classification` | `STRING` | NULLABLE | Classified state (`FULL_SUN`, `PARTIAL_SUN_OR_SHADE`, `DIFFUSE_OVERCAST`, `ABSORPTION_FLOAT_CLIPPED`, `NIGHT`) |

---

## Analytical SQL Examples

### 1. Today's Hourly Generation & Atmospheric Irradiance Correlation
```sql
SELECT
  TIMESTAMP_TRUNC(timestamp, HOUR) AS hour_window,
  ROUND(AVG(pv_power_w), 1) AS avg_solar_w,
  ROUND(MAX(pv_power_w), 1) AS peak_solar_w,
  ROUND(AVG(weather_direct_rad_w_m2 + weather_diffuse_rad_w_m2), 1) AS avg_irradiance_ghi,
  ROUND(AVG(array_utilization_pct), 1) AS avg_array_utilization_pct,
  ROUND(AVG(battery_soc_pct), 1) AS avg_battery_soc_pct
FROM `solaria-solar.solaria.telemetry`
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
GROUP BY hour_window
ORDER BY hour_window ASC;
```

### 2. Daily Energy Yield vs Total Insolation (Past 30 Days)
```sql
SELECT
  DATE(timestamp) AS date,
  MAX(daily_generated_wh) AS total_yield_wh,
  MAX(daily_max_pv_w) AS peak_power_w,
  MIN(daily_min_battery_voltage_v) AS min_batt_v,
  MAX(daily_max_battery_voltage_v) AS max_batt_v,
  ROUND(AVG(weather_cloud_cover_pct), 1) AS avg_cloud_cover_pct
FROM `solaria-solar.solaria.telemetry`
WHERE _PARTITIONDATE >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY date
ORDER BY date DESC;
```
