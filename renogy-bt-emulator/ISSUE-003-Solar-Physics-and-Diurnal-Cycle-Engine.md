# [ISSUE-003] Solar Diurnal Physics, Irradiance & MPPT Tracking Engine

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: Solar Physics & PV Generation Simulation (Go)

---

## 🎯 Description
Implement a realistic **Solar Physics and Diurnal Cycle Engine** that continuously calculates the physical solar irradiance and electrical characteristics of an off-grid solar array (default: 400W 2S2P Monocrystalline at 45° Incline, 135° SE Azimuth, Latitude 45.186°N, Longitude -78.863°W) and updates the Renogy MPPT controller registers dynamically.

---

## 📐 Technical Specifications

### 1. Astronomical Position & Solar Radiation Model
- Compute real-time solar elevation ($\alpha$) and azimuth ($\gamma$) using NOAA solar algorithms for the configured geographical location.
- Calculate Plane of Array (POA) Irradiance ($G_{\text{POA}}$):
  $$G_{\text{POA}} = G_{\text{beam}} \cdot \cos(\theta) + G_{\text{diffuse}} + G_{\text{ground}}$$
  where $\theta$ is the angle of incidence relative to array tilt (45°) and azimuth (135° SE).
- **Nighttime Invariant**: When $\alpha \le 0^\circ$ (or elevation below local horizon), $G_{\text{POA}} = 0 \text{ W/m}^2$, $P_{\text{pv}} = 0\text{W}$, and $I_{\text{pv}} = 0.00\text{A}$.

### 2. 400W 2S2P Electrical Model
- Modeled Array: Two series strings of two 100W panels in parallel (2S2P).
- **Nominal Ratings at STC (1000 W/m², 25°C)**:
  - $V_{\text{mp}} = 36.0\text{ V} \quad (\approx 18.0\text{V} \times 2)$
  - $I_{\text{mp}} = 11.1\text{ A} \quad (\approx 5.56\text{A} \times 2)$
  - $V_{\text{oc}} = 44.0\text{ V} \quad (\approx 22.0\text{V} \times 2)$
  - $P_{\text{max}} = 400\text{ W}$
- **Temperature Derating**:
  - Solar panel power derates by $-0.40\% / ^\circ\text{C}$ above $25^\circ\text{C}$ cell temperature.
- **MPPT Dynamic Tracking**:
  - MPPT tracking efficiency: $98.5\% - 99.2\%$.
  - Output charging current to 12V nominal battery:
    $$I_{\text{batt}} = \frac{P_{\text{pv}} \times \eta_{\text{mppt}}}{V_{\text{batt}}}$$

### 3. Diurnal State Transition Loop
- Runs every 1 second in a Go background goroutine.
- Integrates daily generated energy ($Wh = \sum P_{\text{pv}} \times \Delta t$) into register `0x0113`.
- Automatically resets daily statistics at local solar midnight.

---

## 🧪 Acceptance Criteria
- [ ] At night (22:00 - 05:00 local time), $P_{\text{pv}} == 0\text{W}$ and $I_{\text{pv}} == 0.00\text{A}$.
- [ ] At midday on a clear day, array output peaks realistically between $320\text{W} - 380\text{W}$ based on ambient temperature and panel tilt.
- [ ] Daily generated energy accumulation accurately matches trapezoidal Riemann integration over time.

---

## 🔗 References
- [NREL PVWatts System Model](https://pvwatts.nrel.gov/)
- [NOAA Solar Calculation Algorithms](https://gml.noaa.gov/grad/solcalc/solareqns.PDF)
