# [ISSUE-008] Power Budget Scenario Quick-Pick Presets

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Off-Grid Power Budget Advisor)
- **Status:** In Progress (Preset Buttons Implemented, Additional Scenarios & Storage Sync Needed)

---

## 🧐 Problem Statement & High Interaction Cost
To calculate how long the 170Ah battery will last under different cottage load combinations, users previously had to click 6 individual checkboxes (`Starlink 45W`, `12V Fridge 35W`, `LED Lights 15W`, `Inverter Idle 12W`, `Water Pump 120W`, `Laptop Charging 65W`).

This high interaction cost made quick checks frustrating during daily cottage routines.

---

## 💡 UX Recommendation & Preset Engine

Provide **1-Tap Scenario Presets** that instantly configure appliance loads and calculate battery runtime:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    ONE-TAP POWER BUDGET SCENARIO PRESETS                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  [ 🌙 Eco Night (27W) ]   [ 🛰️ Workday (142W) ]   [ ⚡ All Active (292W) ]   │
│  [ ❄️ Winter Standby (0W) ] [ 🛠️ Workshop Tools (240W) ]                    │
│                                                                             │
│  Selected Scenario: 🌙 Eco Night                                            │
│  • Total Load: 27 W (2.25 A @ 12V)                                          │
│  • Battery Reserve: 170Ah (1,840 Wh Usable @ 88% SOC)                       │
│  • Estimated Runtime: 68.1 Hours (2.8 Days)                                 │
│  • Status: 🟢 EXCELLENT OFF-GRID AUTONOMY                                   │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Preset Scenarios:**
   - **🌙 Eco Night:** Inverter Idle (12W) + LED Lighting (15W) = 27W.
   - **🛰️ Remote Workday:** Starlink (45W) + Laptop (65W) + Inverter (12W) + Fridge (35W) = 157W.
   - **⚡ All Active:** All appliances running simultaneously = 292W.
   - **❄️ Zero Standby / Winterize:** All DC loads disabled = 0W.
2. **Local Storage Persistence:**
   - Save custom user combinations to browser `localStorage` so favorite profiles are remembered.

---

## 📋 Acceptance Criteria
- [ ] Render 1-tap scenario preset pill buttons above appliance checklist in `#tab-advisor`.
- [ ] Instantly update wattage sum and calculated runtime hours upon clicking any preset pill.
- [ ] Highlight the active scenario with amber glowing ring.
- [ ] Persist custom adjustments in `localStorage.getItem("solaria_power_budget")`.
