# [ISSUE-007] Sun-Arc Diurnal Elevation & Peak Hour Radial Tracker

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Solar & Atmospheric Advisor)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Celestial Math
Currently, solar azimuth and elevation are printed as raw numbers: `Azimuth: 142.3°, Elevation: 49.8°`. 

For off-grid cottage owners trying to plan high-draw electrical activities (running power tools, water pumps, or charging EVs), raw coordinate numbers fail to communicate:
- *Where is the sun relative to the cottage tree line right now?*
- *When does peak solar harvest start and end today?*
- *How much harvesting daylight remains before tree shadows begin?*

---

## 💡 UX Recommendation & Solar Arc Dome

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                 NOAA SUN-ARC ELEVATION & HARVEST WINDOW DOME                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│                            ☀️ (12:45 PM Peak 52.4°)                          │
│                        ╭───────────────╮                                    │
│                    ╭───╯               ╰───╮                                │
│                ╭───╯   [PEAK HARVEST]      ╰───╮                            │
│            ╭───╯       (10:30 - 14:00)         ╰───╮                        │
│        ●───╯ (Now: 10:15 AM, 41.2° Elev)            ╰───╮                   │
│  [🌅 Sunrise 6:18 AM]                          [🌇 Sunset 8:04 PM]           │
│  ─────────────────────────────────────────────────────────────────────────  │
│  Remaining Peak Sun: 3h 45m  •  Total Day Energy Potential: 2.24 kWh        │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Visual Sky-Arc Curve:**
   - Render a celestial dome based on NOAA solar calculations for Dorset, ON ($45.186^\circ\text{N}, -78.863^\circ\text{W}$).
   - Highlight the **Golden Window (Peak Generation Zone)** where solar elevation $\ge 35^\circ$.
   - Display a glowing sun icon that advances along the arc in real-time.
2. **Dynamic Horizon Countdown:**
   - Show direct countdowns to Solar Noon, Afternoon Tree Shading, and Sunset.

---

## 📋 Acceptance Criteria
- [ ] Build vector SVG celestial dome in `#tab-advisor` or `#tab-live`.
- [ ] Calculate arc parameters from NOAA solar calculations (`cmd/cloud-server/main.go`).
- [ ] Highlight 400W array golden harvest window in amber.
- [ ] Render sunrise, sunset, and solar noon timestamps in localized EDT (`America/Toronto`).
