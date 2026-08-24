# BigQuery Telemetry Schema

Solaria writes telemetry rows into Google BigQuery partitioned daily on `timestamp`.

* **Dataset:** `solaria`
* **Table:** `telemetry` (`solaria-solar.solaria.telemetry`)
* **Partitioning:** Day-partitioned on `timestamp`

## Schema Reference

| Column | Type | Mode | Unit / Format | Description |
| :--- | :--- | :--- | :--- | :--- |
| `timestamp` | `TIMESTAMP` | REQUIRED | UTC | Telemetry record timestamp |
| `site` | `STRING` | NULLABLE | — | Site name or label |
| `latitude` | `FLOAT64` | NULLABLE | Degrees | GPS latitude (`45.186`) |
| `longitude` | `FLOAT64` | NULLABLE | Degrees | GPS longitude (`-78.863`) |
| `array_capacity_w` | `FLOAT64` | NULLABLE | W | Array nameplate peak rating (`400.0`) |
| `array_topology` | `STRING` | NULLABLE | — | Wiring topology (`2S2P`) |
| `array_utilization_pct` | `FLOAT64` | NULLABLE | % | Instantaneous generation utilization |
| `performance_ratio_pct` | `FLOAT64` | NULLABLE | % | Harvest efficiency relative to GHI |
| `pv_power_w` | `FLOAT64` | NULLABLE | W | Array generation power |
| `pv_voltage_v` | `FLOAT64` | NULLABLE | V | Array operating voltage |
| `pv_current_a` | `FLOAT64` | NULLABLE | A | Array output current |
| `battery_soc_pct` | `INT64` | NULLABLE | % | Battery state of charge (0–100) |
| `battery_voltage_v` | `FLOAT64` | NULLABLE | V | Battery terminal voltage |
| `battery_current_a` | `FLOAT64` | NULLABLE | A | Net charge/discharge current |
| `controller_temp_c` | `FLOAT64` | NULLABLE | °C | Internal controller temperature |
| `battery_temp_c` | `FLOAT64` | NULLABLE | °C | Battery temperature sensor |
| `charging_state` | `STRING` | NULLABLE | — | Modbus state (e.g. `MPPT Charging`, `Float`) |
| `load_status` | `STRING` | NULLABLE | — | DC auxiliary load state (`ON` / `OFF`) |
| `fault_flags` | `INT64` | NULLABLE | Bitfield | Modbus fault bitmask |
| `daily_min_battery_voltage_v` | `FLOAT64` | NULLABLE | V | Daily recorded minimum battery voltage |
| `daily_max_battery_voltage_v` | `FLOAT64` | NULLABLE | V | Daily recorded maximum battery voltage |
| `daily_max_pv_w` | `FLOAT64` | NULLABLE | W | Daily recorded peak solar power |
| `daily_generated_wh` | `FLOAT64` | NULLABLE | Wh | Daily cumulative energy yield |
| `operating_days` | `INT64` | NULLABLE | Days | Total controller operating days |
| `total_battery_fullcharge_count` | `INT64` | NULLABLE | Count | Total battery full charge cycles |
| `total_charging_ah` | `INT64` | NULLABLE | Ah | Total accumulated charging amp-hours |
| `total_generated_kwh` | `INT64` | NULLABLE | kWh | Total lifetime generated energy |
| `weather_temp_c` | `FLOAT64` | NULLABLE | °C | Ambient outdoor temperature |
| `weather_cloud_cover_pct` | `FLOAT64` | NULLABLE | % | Atmospheric cloud cover fraction |
| `weather_direct_rad_w_m2` | `FLOAT64` | NULLABLE | W/m² | Direct normal irradiance (DNI) |
| `weather_diffuse_rad_w_m2` | `FLOAT64` | NULLABLE | W/m² | Diffuse horizontal irradiance (DHI) |
| `sun_classification` | `STRING` | NULLABLE | — | State (`FULL_SUN`, `DIFFUSE_OVERCAST`, etc.) |

## Example Queries

### Hourly Energy Profile (Last 24 Hours)

```sql
SELECT
  TIMESTAMP_TRUNC(timestamp, HOUR) AS hour_window,
  ROUND(AVG(pv_power_w), 1) AS avg_solar_w,
  ROUND(MAX(pv_power_w), 1) AS peak_solar_w,
  ROUND(AVG(weather_direct_rad_w_m2 + weather_diffuse_rad_w_m2), 1) AS avg_ghi,
  ROUND(AVG(battery_soc_pct), 1) AS avg_battery_soc_pct
FROM `solaria-solar.solaria.telemetry`
WHERE timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
GROUP BY hour_window
ORDER BY hour_window ASC;
```

### Daily Summary (Last 30 Days)

```sql
SELECT
  DATE(timestamp) AS date,
  MAX(daily_generated_wh) AS daily_yield_wh,
  MAX(daily_max_pv_w) AS peak_power_w,
  MIN(daily_min_battery_voltage_v) AS min_battery_v,
  MAX(daily_max_battery_voltage_v) AS max_battery_v,
  ROUND(AVG(weather_cloud_cover_pct), 1) AS avg_cloud_cover_pct
FROM `solaria-solar.solaria.telemetry`
WHERE _PARTITIONDATE >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY date
ORDER BY date DESC;
```
