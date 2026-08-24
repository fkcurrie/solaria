# 🌤️ Atmospheric Intelligence & Performance Ratio Engine

Solaria enriches every 10-second Modbus telemetry packet with real-time atmospheric data fetched from Open-Meteo for the site coordinates (`45.186°N, -78.863°W`).

---

## Solar Radiometry Parameters

| Metric | Symbol | Unit | Meaning |
| :--- | :--- | :--- | :--- |
| **Global Horizontal Irradiance** | $\text{GHI}$ | $\text{W/m}^2$ | Total solar radiation incident on a horizontal surface (Direct + Diffuse) |
| **Direct Normal Irradiance** | $\text{DNI}$ | $\text{W/m}^2$ | Beam radiation coming directly from the sun perpendicular to rays |
| **Diffuse Horizontal Irradiance** | $\text{DHI}$ | $\text{W/m}^2$ | Solar radiation scattered by clouds, aerosols, and atmosphere |
| **Cloud Cover Fraction** | $N$ | $\%$ | Fractional cloud coverage of the sky dome ($0\% - 100\%$) |
| **Solar Elevation Angle** | $\alpha$ | Degrees | Angle of the sun above the geometric horizon |

---

## Mathematical Formulations

### 1. Array Capacity Utilization
Measures instantaneous power generated relative to the theoretical $400\text{W}$ nameplate peak:

$$\text{Capacity Utilization (\%)} = \left( \frac{P_{\text{pv}}}{400\,\text{W}} \right) \times 100\%$$

### 2. Atmospheric Performance Ratio (PR)
Measures harvesting efficiency compared to the theoretical irradiance currently available from the atmosphere:

$$P_{\text{expected}} = \left( \frac{\text{GHI}}{1000\,\text{W/m}^2} \right) \times 400\,\text{W}$$

$$\text{Performance Ratio (\%)} = \begin{cases} 
\left( \frac{P_{\text{pv}}}{P_{\text{expected}}} \right) \times 100\% & \text{if } \text{GHI} \ge 50\,\text{W/m}^2 \\
0\% & \text{if } \text{GHI} < 50\,\text{W/m}^2 
\end{cases}$$

---

## Sun Condition Classification Matrix

Every telemetry frame is dynamically categorized by the atmospheric inference engine:

| Condition Code | Visual Badge | Diagnostic Rules | Interpretation |
| :--- | :--- | :--- | :--- |
| `FULL_SUN` | ☀️ Full Sun | $P_{\text{pv}} \ge 0.65 \times P_{\text{rated}}$ AND $\text{GHI} > 300\,\text{W/m}^2$ AND Clouds $< 25\%$ | Optimal unobstructed direct sunlight |
| `PARTIAL_SUN_OR_SHADE` | ⛅ Partial Sun / Shading | $\text{PR} < 60\%$ OR Cloud Cover $25\% - 80\%$ | Intermittent clouds, cloud-edge lensing, tree obstruction |
| `DIFFUSE_OVERCAST` | ☁️ Overcast | $\text{GHI} < 200\,\text{W/m}^2$ AND $\text{DHI} \approx \text{GHI}$ AND Clouds $> 80\%$ | Dense overcast sky producing pure diffuse scattering |
| `ABSORPTION_FLOAT_CLIPPED` | 🔋 Float / Clipped | $\text{SOC} \ge 99\%$ AND State $\in \{\text{Boost}, \text{Float}\}$ | Generation throttled by controller because battery bank is fully charged |
| `NIGHT` | 🌙 Night | Solar Elevation $< 0^\circ$ OR $V_{\text{pv}} < 5.0\,\text{V}$ | Dormant array during night hours |
