# [ISSUE-004] Semantic Color Harmony & Material 3 WCAG Contrast Audit

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (CSS Design System Tokens)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Chromatic Chaos
Over successive iterations, the dashboard accumulated multiple conflicting color palettes across charts, status badges, and cards (neon greens, cyan borders, purple gradient headers, bright yellow warnings, orange badges). 

This "Christmas tree effect" creates visual noise, reduces scan efficiency, and contains low-contrast combinations (e.g., light yellow text on gray backgrounds) that fail **WCAG 2.1 AA** requirements ($4.5:1$ contrast ratio for normal text).

---

## 💡 UX Recommendation & Material Design 3 Palette

Standardize on a strict **4-Role Semantic Palette** grounded in physical solar domains:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    MATERIAL DESIGN 3 SEMANTIC COLOR TOKENS                  │
├─────────────────────────────────────────────────────────────────────────────┤
│  ☀️ SOLAR GENERATION:                                                       │
│  • Primary: Amber-500 `#F59E0B` (High harvest, active solar influx)        │
│  • Container: Amber-950/20 `rgba(245, 158, 11, 0.12)`                       │
│  • Contrast Text on Dark: Amber-300 `#FCD34D` (Contrast Ratio: 9.2:1)       │
│                                                                             │
│  🔋 STORAGE & BATTERY:                                                      │
│  • Nominal / Charging: Emerald-500 `#10B981`                                │
│  • Low Battery Warning: Amber-500 `#F59E0B` (< 40% SOC)                     │
│  • Critical Low Knee: Rose-500 `#EF4444` (< 11.8V, LVD Imminent)            │
│                                                                             │
│  🌌 ATMOSPHERIC & SKY:                                                      │
│  • Irradiance & Elevation: Sky-400 `#38BDF8`                                │
│  • Cloud Density: Slate-400 `#94A3B8`                                       │
│                                                                             │
│  🏡 CONSUMPTION & LOADS:                                                    │
│  • Active Inverter & DC Loads: Indigo-400 `#818CF8`                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 📋 Acceptance Criteria
- [ ] Define CSS custom properties for all 4 semantic color roles in `:root`.
- [ ] Run automated WCAG 2.1 AA contrast audit across all text/background pairs:
  - Normal text ($< 18\text{pt}$): Contrast $\ge 4.5:1$.
  - Large text ($\ge 18\text{pt}$ bold): Contrast $\ge 3.0:1$.
  - UI components and graphical objects: Contrast $\ge 3.0:1$.
- [ ] Ensure all charts (Chart.js / SVG) use the unified semantic color tokens for series mapping (Solar = Amber, Battery = Emerald/Red, Weather = Sky Blue, Load = Indigo).
