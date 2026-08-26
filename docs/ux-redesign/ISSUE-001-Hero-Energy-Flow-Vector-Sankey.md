# [ISSUE-001] Hero Energy Flow Vector (Live Animated Power Sankey)

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P0 (Critical)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Hero Section / Live Monitor Tab)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & User Friction
Currently, solar generation watts, battery current/voltage, and cottage load metrics are rendered in separate, disconnected rectangular cards. When walking past the dashboard or checking on a phone, a user must mentally compute:
$$\text{Net Battery Flux} = P_{\text{array}} \times \eta_{\text{mppt}} - P_{\text{load}}$$
This creates high cognitive friction and fails the **3-Second Situational Awareness Test**. Users cannot tell at a glance whether solar energy is actively filling the battery, bypassing to loads, or if the battery is being drained to sustain cottage appliances.

---

## 💡 UX Recommendation & Visual Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    LIVE HERO ENERGY FLOW VECTOR                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   [☀️ Solar Array] ──── 320W ────► [⚡ MPPT 98%] ─── +21.4A ───► [🔋 LiFePO4] │
│      (36.2V @ 8.8A)                     │ (298W Net)                (84% SOC)│
│                                         │                                    │
│                                         └──── 22W ────► [🏡 Cottage Loads]   │
│                                                            (LEDs + Router)   │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Dynamic SVG / CSS Energy Flow Diagram:**
   - Render animated pulsating dash arrays (`stroke-dashoffset`) along SVG connecting curves.
   - Flow speed and line thickness proportional to real-time power (W).
   - Particle direction indicates charging vs. discharging:
     - **Charging:** Influx particles stream from Solar $\to$ MPPT $\to$ Battery.
     - **Discharging (Night / Heavy Load):** Particles reverse from Battery $\to$ Loads.
2. **Interactive Node Cards:**
   - Clicking each node (Solar, MPPT, Battery, Loads) opens an inline drawer with deep electrical diagnostics.

---

## 📋 Acceptance Criteria
- [ ] Implement an SVG/Canvas vector diagram at the top of the `#tab-live` container.
- [ ] Dynamically update flow paths in real-time based on Modbus RTU telemetry:
  - `pv_power_w` > 0: Solar line active with amber particles (`#F59E0B`).
  - `battery_current_a` > 0: Charge line active with emerald particles (`#10B981`).
  - `battery_current_a` < 0: Discharge line active with amber/red particles.
  - `load_power_w` > 0: Load line active with indigo particles (`#818CF8`).
- [ ] CSS animation uses GPU acceleration (`transform: translateZ(0)` / `will-change: stroke-dashoffset`) for smooth 60fps rendering without CPU heating.
- [ ] Fallback static vector rendered gracefully on low-power devices with `prefers-reduced-motion: reduce`.

---

## 🛠️ Implementation Plan
1. Add `<div class="energy-flow-container">` inside `templates/index.html`.
2. Construct responsive SVG viewBox (`0 0 800 240`) with 4 node groups (`#node-solar`, `#node-mppt`, `#node-battery`, `#node-loads`).
3. Bind JS update loop in `updateLiveDashboard(data)` to calculate particle velocity and flow direction.
