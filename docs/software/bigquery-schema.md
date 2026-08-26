# BigQuery Data Lake Schema

Solaria streams telemetry into **Google BigQuery** table `solaria-solar.solaria.telemetry`, partitioned daily by `timestamp` and clustered by `site_id` and `charging_state`.

---

## 🗄️ Table Schema Definition

```sql
CREATE TABLE `solaria-solar.solaria.telemetry` (
  timestamp TIMESTAMP NOT NULL OPTIONS(description="UTC timestamp of sample"),
  site_id STRING NOT NULL OPTIONS(description="Unique installation identifier"),
  pv_voltage_v FLOAT64 OPTIONS(description="Array DC voltage (V)"),
  pv_current_a FLOAT64 OPTIONS(description="Array DC current (A)"),
  pv_power_w FLOAT64 OPTIONS(description="Instantaneous solar power (W)"),
  batt_voltage_v FLOAT64 OPTIONS(description="Battery terminal voltage (V)"),
  batt_current_a FLOAT64 OPTIONS(description="Charge/discharge current (A)"),
  batt_soc_pct INT64 OPTIONS(description="State of Charge (0-100%)"),
  batt_temp_c FLOAT64 OPTIONS(description="Battery internal temperature (°C)"),
  controller_temp_c FLOAT64 OPTIONS(description="MPPT heatsink temperature (°C)"),
  charging_state STRING OPTIONS(description="Bulk, Absorption, Float, or Standby"),
  daily_yield_wh FLOAT64 OPTIONS(description="Cumulative daily harvest (Wh)"),
  subzero_inhibit BOOL OPTIONS(description="Sub-zero lithium charging inhibit flag"),
  open_meteo_temp_c FLOAT64 OPTIONS(description="External ambient temperature (°C)"),
  open_meteo_cloud_pct INT64 OPTIONS(description="Cloud cover percentage (0-100%)"),
  open_meteo_shortwave_rad FLOAT64 OPTIONS(description="Global Horizontal Irradiance (W/m²)")
)
PARTITION BY DATE(timestamp)
CLUSTER BY site_id, charging_state;
```

---

## 📊 Useful Analytical SQL Queries

### Daily Energy Yield & Peak Power
```sql
SELECT
  DATE(timestamp) as date,
  MAX(daily_yield_wh) / 1000.0 as total_kwh,
  MAX(pv_power_w) as peak_watts,
  AVG(batt_voltage_v) as avg_battery_voltage,
  MIN(batt_temp_c) as min_battery_temp_c
FROM `solaria-solar.solaria.telemetry`
WHERE site_id = '1296_wren_lake'
GROUP BY date
ORDER BY date DESC
LIMIT 30;
```
