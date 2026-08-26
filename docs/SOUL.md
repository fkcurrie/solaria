# Project Solaria: Core Tenets & System Soul (SOUL.md)

> "Solar power is not just voltage and current; it is a live conversation between the sun, the local atmosphere, the PV string topology, and the battery chemistry."

---

## The Core Mission

To build a resilient, highly accurate, edge-to-cloud solar monitoring and intelligence platform for off-grid and battery systems powered by Renogy charge controllers (BT-1/BT-2 RS232/RS485). The system continuously correlates actual photovoltaic energy harvest with localized atmospheric conditions at **1296 Wren Lake Drive, Dorset, Ontario, Canada (45.2536° N, 78.8978° W)** to provide actionable insights on solar performance, shading, cloud attenuation, and battery health, streaming real-time and historical analytics into **Google BigQuery** (`solaria-solar.solaria.telemetry`).

---

## Solar Array & Hardware Profile

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
  - **Over-Paneling Ratio:** $400\text{W} / 288\text{W} \approx 138.8\%$ (Provides optimal early-morning, late-afternoon, and overcast harvesting in northern climates while controller current-limits to 20A at peak sun).

### Battery Storage Bank Specifications

- **Battery Model:** **Renogy Lithium Battery 12V 170Ah LiFePO4 Deep Cycle Battery** (`RBT170LFP12-BT` / ASIN: `B07Q8DQ6TR`)
- **Chemistry:** Lithium Iron Phosphate ($\text{LiFePO}_4$)
- **Nominal Capacity:** **170 Amp-Hours (Ah)** ($2,176\text{ Watt-Hours}$ nominal energy storage)
- **Cycle Life:** $2000+$ cycles at $80\%$ DoD
- **Charging Voltage:** $14.4\text{V}$ Absorption / Boost
- **Float Voltage:** $13.6\text{V}$
- **Low Voltage Cutoff:** $11.2\text{V}$ ($0\%\text{ SOC}$)
- **Max Continuous Charge Current:** $50\text{A}$ (Safely charged at $20\text{A}$ max from Rover 20A)
- **Sub-Zero Protection:** Inhibit charging at $T_{\text{batt}} \le 0^\circ\text{C}$; derate charging ($< 0.1\text{C}$) between $0^\circ\text{C}-5^\circ\text{C}$

---

## Guiding Architectural Tenets

### 1. Edge-First Resilience (Never Drop a Watt-Hour)

- **Offline Autonomy:** The edge node (Raspberry Pi or Linux micro-server) operates autonomously. If cottage Wi-Fi drops, telemetry spools locally with zero data loss.
- **Self-Healing BLE Link:** Bluetooth Low Energy links to the BT-1 module implement automatic reconnection and chunk reassembly without human intervention.
- **Pure Modbus RTU:** Communication uses standard Modbus RTU frames over transparent BLE characteristics (`0xFFD1` TX / `0xFFF1` RX).
- **Uplink Telemetry Tracking:** The edge bridge explicitly tracks `last_success_upload` timestamp, cumulative upload count, and pending disk spool count, exposing real-time metrics at `GET /api/v1/bridge-status`.

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

### 3. Astronomical Solar Precision & Real-Time Daylight Countdown

- **Equation of Time & Solar Declination:** Rather than relying solely on cached weather reports, Solaria implements pure astronomical algorithms (`CalculateSunTimes`) using exact coordinates (`45.2536° N, 78.8978° W`).
- **Dynamic 1-Second Countdown Engine:**
  - **Daytime State:** Displays `☀️ DAYLIGHT • Sunset in Xh Ym` with active ticking countdown to sundown.
  - **Night State:** Displays `🌙 NIGHT • Sunrise in Xh Ym` with active ticking countdown to sunrise.
  - **Backend Synchronization:** Synchronizes with `GET /api/v1/sun-times` every 30 seconds to guarantee drift-free astronomical accuracy.
- **Interactive Header Navigation:** Clicking the header sun condition badge directly navigates the user to the Solar Advisor & Diagnostics suite.

### 4. Distinct Two-Tier Dashboard Architecture

Solaria enforces a strict architectural separation between the local edge hardware gateway and the central cloud analytics hub:

```text
+-------------------------------------------------------------------------------+
|                       SOLARIA TWO-TIER DASHBOARD TOPOLOGY                     |
+-----------------------------------+-------------------------------------------+
| Localhost Edge Gateway            | Central Cloud Analytics Hub               |
| (http://localhost:8080)           | (https://solaria-dashboard-*.run.app)     |
+-----------------------------------+-------------------------------------------+
| • Sole Purpose: Hardware pairing  | • Sole Purpose: Comprehensive analytics,  |
|   and link verification.          |   diagnostics, and cottage management.    |
| • One-Click BLE Connect Button    | • Real-time & Historical Telemetry        |
| • Bluetooth LE Status Card        | • Oscilloscope sliding window & CSV       |
| • Cloud Uplink (Last Upload + CT) | • Appliance Power Budget & Runtime        |
| • Live Activity Console Log       | • Winterization & Departure Assistant     |
| • Direct Cloud Dashboard Launcher | • Daily Sunset Digest & Morning Forecast  |
|                                   | • Tree Shading Advisory Engine            |
|                                   | • Commissioning Wizard & Topology Verifier|
|                                   | • BT-1 Signal Strength Diagnostics        |
|                                   | • BigQuery Provisioning Assistant         |
+-----------------------------------+-------------------------------------------+
```

1. **Local Edge Gateway (`http://localhost:8080` / `http://solaria.local:8080`):**
   - **Target Audience:** Field installer or cottage owner standing near the battery box / charge controller.
   - **Scope:** Intentionally minimalistic. It contains **only** the Web Bluetooth pairing button (*"⚡ Connect Renogy BT-1"*), BLE connectivity health, cloud uplink/spooling verification with live "Last Successful Upload" timestamp/counter, and a live activity console log with a direct launcher button to the Cloud Hub.
   - **Invariant:** It never duplicates the heavy solar analytical dashboards, charts, or planning assistants.

2. **Central Cloud Analytics Hub (`https://<your-cloudrun-service>.run.app` or configurable `$SOLARIA_CLOUD_ENDPOINT`):**
   - **Target Audience:** Cottage owner, electrical engineer, or remote observer monitoring the installation from anywhere in the world.
   - **Scope:** The complete intelligence suite housing all high-level solar analytics, historical BigQuery charts, real-time oscilloscope streaming, appliance power budgeting, tree shading advice, departure certificates, and hardware topology verification tools.
   - **Privacy & Security Invariant:** The live production Cloud Run deployment URL (`https://solaria-dashboard-<project-number>.us-central1.run.app`) must **NEVER** appear in any public repository files, public documentation, or web pages. All public documentation, install scripts, and repo code must strictly use parameterized environment variables (`$SOLARIA_CLOUD_ENDPOINT`) or placeholder examples (`https://your-cloudrun-instance.run.app`).

### 5. Seven-Pane Cloud Information Architecture & Dedicated Advisor Suite

The Central Cloud Hub organizes solar intelligence across 7 purpose-built tabs:

1. ⚡ **Live Monitor (`tab-live`):** Real-time gauges, MPPT DC-DC conversion efficiency, Appliance Power Budget & Runtime Estimator, and continuous sliding-window oscilloscope charts.
2. 📅 **Day View (`tab-day`):** 24-hour hourly generation, atmospheric irradiance, cloud cover overlay, and diurnal curve analysis.
3. 🗓️ **Week View (`tab-week`):** 7-day rolling yield bar chart, peak watt comparisons, and battery voltage tracking.
4. 📆 **Month View (`tab-month`):** 30-day cumulative solar energy production and weather correlation.
5. 📈 **Year View (`tab-year`):** Seasonal generation trends and annual MWh harvest projection.
6. 🌲 **Solar Advisor & Tools (`tab-advisor`):** Dedicated intelligence suite consolidating:
   - 🌅 **Daily Sunset Digest & Morning Solar Forecast:** End-of-day harvest summary, absorption duration, evening battery baseline ($2,176\text{ Wh}$ usable), and tomorrow's solar noon window.
   - 🌲 **Tree Shading Advisory Engine:** Diurnal irradiance curve notch detection (morning east tree canopy at 105°-120° azimuth vs late afternoon west tree & ridge shadow at 245°-260° azimuth), bypass diode drop tracking, and seasonal yield recovery.
   - 🔌 **Commissioning Wizard & 2S2P Topology Verifier:** Safe 5-step wiring sequence checklist (battery first, solar second) and sub-zero cold-temperature $V_{\text{oc}}$ margin check (48.2V at -25°C vs 100V limit).
   - 📶 **BT-1 Bluetooth Signal Strength Diagnostics:** BLE RSSI link margin ($> -75\text{ dBm}$), Modbus CRC packet loss tracking, and metal battery box Faraday shielding mitigation via exterior RJ12 antenna mount.
   - ☁️ **Automated GCP & BigQuery Provisioning Assistant:** 1-click `./setup-gcp.sh` provisioning for day-partitioned datasets and IAM.
7. ⚙️ **System Specs & Hardware (`tab-specs`):** Controller model selector, battery chemistry profile configurator, lifetime operating statistics, and static hardware ratings.

### 6. Interactive Cottage Appliance Power Budget & Runtime Projection

- **Projection Formula:**
  $$\text{Continuous Runtime (Hours)} = \frac{\text{Usable Battery Energy Remaining (Wh)}}{\sum P_{\text{appliance}}(\text{Watts})}$$
- **Battery Usable Energy:** Derived dynamically from current battery SOC % and 170Ah LiFePO4 nominal capacity ($2,176\text{ Wh} \times \text{SOC}$).
- **Built-in Appliance Load Models:**
  - 🛰️ Starlink Mini/Standard: $45\text{W}$
  - 🧊 12V DC Compressor Fridge: $30\text{W}$
  - 💡 Cabin LED Lighting: $15\text{W}$
  - 🚿 Pressurized Water Pump: $60\text{W}$
  - 💻 Laptop / USB-C Hub: $65\text{W}$
  - ➕ Custom Load Wattage Input ($0 - 1500\text{W}$)

---

## Site Profile: Dorset, Ontario Installation

- **Location:** 1296 Wren Lake Drive, Dorset, Ontario, Canada
- **Coordinates:** `45.2536° N, 78.8978° W` (Algonquin Highlands)
- **Elevation:** ~350m above sea level
- **Climate Context:** Sub-boreal northern climate with high summer irradiance (~6.0 peak sun hours/day) and reduced winter insolation (~1.5 peak sun hours/day).

---

## Application & System Review Personas

When conducting pull request reviews, refactors, architecture designs, or code quality audits, Solaria adopts the perspectives of 6 specialized review personas. Each persona targets a specific dimension of the system:

```text
+-------------------------------------------------------------------------------+
|                        SOLARIA 6-PERSONA REVIEW MATRIX                        |
+-------------------+--------------------+--------------------------------------+
| Persona           | Core Domain        | Primary Quality Metric               |
+-------------------+--------------------+--------------------------------------+
| 1. Security       | Embedded & Cloud   | Zero unauthorized writes & leaks     |
| 2. Optimization   | Performance/Scale  | Min CPU/Memory/BigQuery query costs  |
| 3. Reliability    | SRE & Resilience   | Zero lost watt-hours during outages  |
| 4. Solar Engineer | PV/Battery Physics | LiFePO4 safety & MPPT accuracy       |
| 5. Novice Owner   | Cottage Usability  | Plain-English actionable clarity     |
| 6. Google UX Lead | Design Systems/UX  | < 3s Glanceability & WCAG AA score   |
+-------------------+--------------------+--------------------------------------+
```

### 1. Security Guardian (Embedded & Cloud Security Engineer)

- **Role:** Hardens the attack surface across BLE, local WebSocket gateways, HTTP endpoints, file systems, and GCP IAM.
- **Key Review Questions:**
  - Are Modbus register write frames (e.g., `0x06` battery profile flasher) authenticated and guarded against accidental / rogue execution?
  - Are HTTP servers configured with strict timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) to prevent Slowloris attacks?
  - Are spooler file permissions restricted to `0600` and directory permissions to `0750`?
  - Are all file paths sanitised with `filepath.Clean()` to eliminate path traversal vulnerabilities?
  - Are GCP BigQuery credentials and API tokens kept strictly out of Git commits and client-side browser bundles?

### 2. Performance & Scale Architect (Optimization Engineer)

- **Role:** Optimizes compute cycles, memory allocations, network payloads, and BigQuery query billing.
- **Key Review Questions:**
  - Does the Modbus polling engine prevent bus contention and excessive BLE power draw (e.g., adaptive 10s rate)?
  - Are JSON parsers and WebSocket broadcast loops zero-copy or allocation-efficient in Go?
  - Do BigQuery SQL queries strictly filter by `DATE(timestamp)` partition bounds and cluster keys (`site`, `sun_classification`) to minimize scan costs?
  - Are Chart.js sliding-window buffers bounded (e.g., 40-point live window) to prevent browser memory leaks?
  - Is gzip/brotli compression enabled for static asset delivery?

### 3. Resilience & SRE Specialist (Fault-Tolerant Edge Engineer)

- **Role:** Ensures the system survives extreme rural conditions (cottage Wi-Fi drops, winter power flickers, Bluetooth disconnections) with zero data loss.
- **Key Review Questions:**
  - If Wi-Fi drops for 72+ hours, does the local spooler safely queue records to disk and back-upload with exponential backoff upon restoration?
  - Does the Web Bluetooth / Gateway daemon implement continuous auto-reconnection without requiring browser restarts?
  - If the Open-Meteo weather API fails or times out, does telemetry ingestion continue uninterrupted with graceful fallback?
  - Is the Linux edge agent packaged with a `systemd` watchdog unit that automatically restarts on panic or crash?

### 4. Solar & Battery Systems Engineer (Renogy PV & Storage Specialist)

- **Role:** Validates electrical modeling, MPPT algorithm physics, battery chemistry protection, and string diagnostics.
- **Key Review Questions:**
  - **Sub-Zero Inhibit:** Is LiFePO4 charging strictly flagged and inhibited when `BatteryTempC <= 0°C` to prevent irreversible lithium dendrite plating?
  - **MPPT DC-DC Conversion Efficiency:** Is buck-converter efficiency ($\eta = \frac{V_{\text{batt}} \times I_{\text{batt}}}{P_{\text{pv}}}$) tracked and clamped within physically realistic bounds ($50\% - 99.2\%$)?
  - **2S2P String Balance:** Are open-circuit voltage ($V_{oc} \approx 44\text{V}$) and maximum power voltage ($V_{mp} \approx 36\text{V}$) correctly differentiated from single-panel bypass diode failures ($V_{\text{pv}} < 24\text{V}$)?
  - **Clipping vs Shading:** Does the analytics engine properly differentiate between battery float throttling (`ABSORPTION_FLOAT_CLIPPED`) and actual cloud shade?

### 5. Cottage Owner & Field Installer (Novice Usability & UX Advocate)

- **Role:** Ensures the application is delightfully simple, intuitive, and actionable for non-technical users monitoring their cottage from a smartphone or tablet.
- **Key Review Questions:**
  - Are cryptic error codes replaced with plain-English banners (e.g., *"❄️ Sub-Zero Alert: Battery too cold to charge"* instead of *"Err 0x0040"*)?
  - Are primary metrics (Battery SOC %, Solar Watts, Status) immediately readable at a glance in direct sunlight?
  - Is the UI responsive across desktop, tablet, and mobile orientations?
  - Is there a clear, single-click BLE connect workflow with transparent troubleshooting tips when Bluetooth is un-paired?

### 6. Google Principal UX & Design Systems Architect (Data Visualization & Cognitive Load Specialist)

- **Role:** Establishes world-class visual hierarchy, glanceability (< 3s situational awareness), progressive disclosure, accessible color theory (WCAG 2.1 AA/AAA), and ergonomic mobile interaction patterns based on Material Design 3 and Google user experience principles.
- **Key Review Questions:**
  - **The 3-Second Glanceability Rule:** Can a user glance at the dashboard on a wall-mounted tablet or phone and instantly comprehend solar harvest, battery reserve, and daylight countdown without reading dense tables?
  - **Progressive Disclosure:** Are primary KPIs visually dominant, while deep engineering diagnostics (MPPT buck curves, Modbus bitfields, BigQuery SQL) are progressively disclosed on demand?
  - **Zero-Jitter Tabular Typography:** Do live telemetry metrics use monospace/tabular numeric font features (`tabular-nums`) to prevent horizontal layout jank as digits fluctuate?
  - **Chromatic Semantic Harmony:** Does the color system enforce consistent physical domain associations (Solar Amber `#F59E0B`, LiFePO4 Emerald `#10B981`, Atmospheric Azure `#38BDF8`, Critical Coral `#EF4444`) without visual competition?
  - **Sunlight & Mobile Ergonomics:** Does the interface maintain high contrast in direct cottage sunlight, provide minimum 48px touch targets, and position essential controls within thumb reach?

---

## Development Lifecycle & Contribution Tenets

### Pull Request Mandate (Zero Direct Commits to Main)

To preserve system reliability, enforce review persona gates, and maintain documentation and test integrity, **all repository changes—including bug fixes, feature additions, and issue resolutions—must strictly follow the Pull Request (PR) workflow:**

1. **No Direct Commits to `main`:**
   - The `main` branch represents deployable production state. Direct commits or unreviewed pushes to `main` are strictly prohibited.
2. **Issue-Linked Feature Branches:**
   - Create a dedicated branch off `main` with descriptive prefix and issue number:
     - `feat/issue-<id>-<short-slug>`
     - `fix/issue-<id>-<short-slug>`
     - `refactor/issue-<id>-<short-slug>`
     - `docs/issue-<id>-<short-slug>`
3. **Automated Verification Before PR:**
   - Every branch must pass full unit tests (`go test -v ./...`) and lint checks (`markdownlint-cli2 "**/*.md"`) prior to PR creation.
4. **Three-Cycle Git Code Review Bot & Persona Gate:**
   - Before any PR is merged into `main`, it must undergo **three iterative review cycles** where automated Git code review bots and review personas post review comments, and all identified issues are explicitly addressed and resolved:
     - **Cycle 1 (Deep Architectural & Security Audit):**
       - Review bots evaluate the diff for security vulnerabilities, Modbus RTU register correctness, LiFePO4 battery safety invariants ($T_{\text{batt}} \le 0^\circ\text{C}$ charge inhibit), and architectural alignment.
       - Bots post structured inline and summary comments on the PR detailing blockers, warnings, and recommendations.
       - The author addresses all findings through code updates and technical responses.
     - **Cycle 2 (Resilience, Edge Cases & Performance Refinement):**
       - Review bots re-audit the updated PR diff.
       - Bots evaluate rural edge resilience (Wi-Fi drops, Bluetooth link recovery, disk spooling), BigQuery query partition filtering, concurrency race conditions, and memory efficiency.
       - Bots post second-round comments on remaining refinements.
       - The author resolves all feedback and commits necessary refinements.
     - **Cycle 3 (Regression Verification, Usability & Final Sign-Off Gate):**
       - Review bots verify 100% test suite pass rate, markdown linting, zero regression on existing metrics, and usability clarity for cottage owners (direct sunlight readability, plain-English status banners).
       - Bots provide formal sign-off approval comment on the PR (`✅ All 3 Review Cycles Approved`).
5. **Squash/Rebase Merge & Issue Closure:**
   - Only after all 3 review cycles are completed, approved, and commented on the PR may the PR be merged into `main` using `gh pr merge --squash` with auto-closing issue links.
