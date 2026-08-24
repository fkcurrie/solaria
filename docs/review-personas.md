# Solaria Application & System Review Personas (`docs/review-personas.md`)

> **Multi-Perspective Quality & Engineering Review Matrix for Project Solaria**

When conducting pull request reviews, refactors, architecture designs, or code quality audits, Solaria leverages 5 specialized review personas. Each persona brings domain-specific expertise, targeted checklists, and evaluation criteria to ensure security, performance, resilience, photovoltaic accuracy, and novice usability.

---

## 5-Persona Review Matrix Summary

```text
+-----------------------------------------------------------------------------------------+
|                              SOLARIA REVIEW PERSONA MATRIX                              |
+----+-------------------+-----------------------+----------------------------------------+
| #  | Persona           | Core Domain           | Primary Quality Metric                 |
+----+-------------------+-----------------------+----------------------------------------+
| 1  | Security Guardian | Embedded & Cloud Sec  | Zero unauthorized writes, leaks & vuln |
| 2  | Perf Architect    | Performance & Scale   | Min CPU/Memory/BigQuery query costs    |
| 3  | SRE Specialist    | Edge Resilience & SRE | Zero lost watt-hours during outages    |
| 4  | Solar Engineer    | PV & Battery Physics  | LiFePO4 safety & MPPT tracking acc     |
| 5  | Cottage Owner     | Field UX & Usability  | Plain-English actionable clarity       |
+----+-------------------+-----------------------+----------------------------------------+
```

---

## Persona 1: Security Guardian (Embedded & Cloud Security Engineer)

### Domain & Mindset

Guards the attack surface across Web Bluetooth (BLE), local WebSocket IPC, HTTP endpoints, edge file storage, and GCP IAM. Assumes hostile or noisy local network environments and untrusted inputs.

### Review Checklist

- **Modbus Write Protection:** Are Modbus register write frames (e.g., `0x06` battery profile flasher) authenticated, rate-limited, and protected against rogue execution?
- **HTTP Server Hardening:** Are all `http.Server` instances configured with explicit timeouts (`ReadHeaderTimeout: 10s`, `ReadTimeout: 30s`, `WriteTimeout: 30s`, `IdleTimeout: 60s`)?
- **Least Privilege Permissions:** Are spooler files restricted to `0600` and directory creations to `0750`?
- **Path Sanitization:** Are all file operations cleaned with `filepath.Clean()` to eliminate path traversal vulnerabilities?
- **Secret & Credential Isolation:** Are BigQuery service account keys, GCP tokens, and Bluetooth PINs excluded from Git commits, logs, and browser bundles?

---

## Persona 2: Performance & Scale Architect (Optimization Engineer)

### Domain & Mindset

Minimizes edge CPU consumption, memory allocations, network payloads, and BigQuery query billing costs.

### Review Checklist

- **Adaptive Modbus Polling:** Does the polling loop balance responsiveness with low power consumption (e.g., adaptive 10s intervals vs fast bursts)?
- **Allocation Efficiency:** Are Go JSON serializers, slice buffers, and WebSocket broadcasts allocation-efficient?
- **BigQuery Query Pruning:** Do all dashboard analytics queries enforce `DATE(timestamp)` partition bounds and cluster keys (`site`, `sun_classification`) to minimize scanned bytes?
- **Sliding Window Buffer Caps:** Are browser Chart.js datasets capped with sliding-window FIFO arrays (e.g., 40 to 1440 points) to prevent browser tab memory exhaustion?
- **Asset Compression:** Are static web assets properly formatted and compressed for low-bandwidth cellular links?

---

## Persona 3: Resilience & SRE Specialist (Fault-Tolerant Edge Engineer)

### Domain & Mindset

Ensures the edge collector survives remote cottage conditions (Wi-Fi outages, winter power flickers, Bluetooth disconnections, cloud API downtime) with zero data loss.

### Review Checklist

- **Offline Spooling:** If cottage Wi-Fi drops for 72+ hours, does the disk spooler safely persist records and flush them with exponential backoff upon reconnect?
- **Autonomous Reconnect:** Does the BLE GATT client automatically retry connections indefinitely without requiring manual browser or server reloads?
- **Graceful API Degradation:** If the Open-Meteo weather API is unreachable, does telemetry ingestion proceed normally with fallback default metrics?
- **Process Supervision:** Is the edge agent packaged with `systemd` restart policies (`Restart=always`, `RestartSec=5s`) for crash recovery?

---

## Persona 4: Solar & Battery Systems Engineer (Renogy PV & Storage Specialist)

### Domain & Mindset

Validates electrical physics, 2S2P solar string impedance, MPPT buck conversion, and LiFePO4 chemical safety constraints.

### Review Checklist

- **Sub-Zero Thermal Protection:** Is LiFePO4 charging strictly flagged and inhibited when `BatteryTempC <= 0°C` to prevent irreversible metallic lithium plating?
- **MPPT DC-DC Efficiency:** Is buck conversion efficiency ($\eta = \frac{V_{\text{batt}} \times I_{\text{batt}}}{P_{\text{pv}}}$) tracked and clamped within physically realistic bounds ($50\% - 99.2\%$)?
- **2S2P String Balance:** Are string open-circuit voltage ($V_{oc} \approx 44\text{V}$) and maximum power voltage ($V_{mp} \approx 36\text{V}$) distinguished from single-panel bypass diode failures ($V_{\text{pv}} < 24\text{V}$)?
- **Clipping vs Shading:** Does the telemetry classification engine distinguish between controller float throttling (`ABSORPTION_FLOAT_CLIPPED`) and actual cloud shading?

---

## Persona 5: Cottage Owner & Field Installer (Novice Usability & UX Advocate)

### Domain & Mindset

Ensures that non-technical users monitoring their cottage from a smartphone or tablet have immediate, unambiguous situational awareness without needing engineering expertise.

### Review Checklist

- **Plain-English Notifications:** Are technical fault codes accompanied by actionable explanations (e.g., *"❄️ Sub-Zero Alert: Battery too cold to charge"* instead of *"Err 0x0040"*)?
- **Glanceable Status:** Are essential metrics (Battery SOC %, Solar Generation Watts, System Health) visible at a glance with distinct color coding?
- **Mobile & Dock Friendly:** Is the layout fully responsive on mobile screens with touch-friendly buttons and high-contrast text in direct sunlight?
- **One-Click Pairing:** Is the Bluetooth connection workflow simple, with clear visual prompts and helpful troubleshooting guidance?
