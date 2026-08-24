# Atmospheric Intelligence & Performance Ratio

Solaria pairs real-time Modbus telemetry with atmospheric data retrieved from Open-Meteo for site coordinates `45.186°N, -78.863°W`.

## Radiometry Metrics

* **Global Horizontal Irradiance (GHI, $\text{W/m}^2$):** Total solar radiation received on a horizontal surface.
* **Direct Normal Irradiance (DNI, $\text{W/m}^2$):** Radiation coming directly from the sun perpendicular to rays.
* **Diffuse Horizontal Irradiance (DHI, $\text{W/m}^2$):** Radiation scattered by clouds and atmospheric particles.
* **Cloud Cover ($N, \%$):** Percentage of total sky covered by cloud layers.
* **Solar Elevation ($\alpha, ^\circ$):** Angular height of the sun above the geometric horizon.

## Calculated Performance Metrics

### 1. Array Utilization (%)

Compares instantaneous output against the nominal 400 W array peak capacity:

$$\text{Utilization (\%)} = \left( \frac{P_{\text{pv}}}{400\,\text{W}} \right) \times 100$$

### 2. Atmospheric Performance Ratio (PR %)

Compares actual generation against theoretical power expected from current ambient irradiance:

$$P_{\text{expected}} = \left( \frac{\text{GHI}}{1000\,\text{W/m}^2} \right) \times 400\,\text{W}$$

$$\text{PR (\%)} = \begin{cases}
\left( \frac{P_{\text{pv}}}{P_{\text{expected}}} \right) \times 100 & \text{for } \text{GHI} \ge 50\,\text{W/m}^2 \\
0 & \text{for } \text{GHI} < 50\,\text{W/m}^2
\end{cases}$$

## Sun Condition Classification

Every telemetry record is categorized using deterministic threshold rules:

| Condition | Criteria | Description |
| :--- | :--- | :--- |
| `FULL_SUN` | $P_{\text{pv}} \ge 0.65 \cdot P_{\text{rated}}$ AND $\text{GHI} > 300\,\text{W/m}^2$ AND Clouds $< 25\%$ | Clear sky, direct insolation. |
| `PARTIAL_SUN_OR_SHADE` | $\text{PR} < 60\%$ OR Cloud Cover $25\% - 80\%$ | Intermittent clouds or physical obstruction. |
| `DIFFUSE_OVERCAST` | $\text{GHI} < 200\,\text{W/m}^2$ AND $\text{DHI} \approx \text{GHI}$ AND Clouds $> 80\%$ | Heavy overcast conditions. |
| `ABSORPTION_FLOAT_CLIPPED` | $\text{SOC} \ge 99\%$ AND Charging State $\in \{\text{Boost}, \text{Float}\}$ | Throttled output due to full battery bank. |
| `NIGHT` | Solar Elevation $< 0^\circ$ OR $V_{\text{pv}} < 5.0\,\text{V}$ | Dormant array (nighttime). |
