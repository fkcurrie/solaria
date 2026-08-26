# [ISSUE-002] Primary Glance Card Trio (Generation, Storage, Horizon)

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P0 (Critical)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Summary Anchor Section)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Visual Competition
Currently, the live monitor renders 8+ small, equal-sized rectangular metric cards competing for visual hierarchy (`PV Power`, `PV Voltage`, `PV Current`, `Battery Voltage`, `Battery Current`, `Controller Temp`, `Battery Temp`, `Load Power`, etc.). 

Because every card has the same font size and visual weight, the user's eye wanders across the grid with no clear anchor point. This violates **Gestalt principles of visual dominance** and slows down reading comprehension.

---

## 💡 UX Recommendation & Layout Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PRIMARY 3-CARD ANCHOR HIERARCHY                          │
├──────────────────────────────────────┬──────────────────────────────────────┤
│  ☀️ SOLAR HARVEST (Hero Card 1)      │  🔋 BATTERY BANK (Hero Card 2)       │
│  ┌─────────────────────────────────┐ │  ┌─────────────────────────────────┐ │
│  │ 342 W                           │ │  │ 88%                             │ │
│  │ Array Util: 85.5% (400W 2S2P)   │ │  │ 13.5V  •  +20.0A Boost Charging │ │
│  │ Peak Today: 385W @ 12:45 PM     │ │  │ Est. Time to Full: 42 mins      │ │
│  └─────────────────────────────────┘ │  └─────────────────────────────────┘ │
├──────────────────────────────────────┴──────────────────────────────────────┤
│  ⏳ DAYLIGHT & SOLAR HORIZON (Hero Card 3 - Full Width Banner)             │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │ ☀️ Solar Noon in 1h 14m  •  Sunset in 4h 32m (8:04 PM)  •  Elev: 49.2°  │ │
│  │ Direct Radiation: 620 W/m²  •  Optimum Harvest Window: Active Now     │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Establish 3 Primary Anchor Cards:**
   - **Card 1 (Solar Harvest):** Massive 36px font for PV Watts, dynamic progress bar comparing against rated 400W STC, and daily peak indicator.
   - **Card 2 (Battery Bank):** Massive 36px font for LiFePO4 SOC %, voltage badge with chemical state classification (`BOOST`, `FLOAT`, `ABSORPTION`, `RESTING`), and time-to-full or time-to-empty.
   - **Card 3 (Daylight Horizon):** Full-width atmospheric status bar displaying exact countdown to solar noon and sunset calculated using NOAA celestial geometry for Dorset, ON.
2. **Secondary Metrics Drawer:**
   - Move engineering details (string voltages, individual temperatures, Modbus raw packet counts) below the anchor cards into a clean collapsable grid.

---

## 📋 Acceptance Criteria
- [ ] Refactor grid layout in `#tab-live` to place the Glance Card Trio above all secondary parameters.
- [ ] Display primary metrics with large high-contrast typography (`text-3xl font-extrabold font-numeric`).
- [ ] Integrate real-time progress bar for array utilization ($\frac{P_{\text{pv}}}{400\text{W}} \times 100\%$).
- [ ] Display intelligent dynamic time estimation (`Est. Full in Xm` when charging; `Est. Runtime Yh` when discharging).
- [ ] Fully responsive: collapses from 2 columns on desktop ($\ge 1024\text{px}$) to 1 column on mobile screens ($< 768\text{px}$).
