# [ISSUE-006] LiFePO4 Non-Linear SOC Voltage Curve Visualizer

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Battery Monitor & Diagnostics)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Lithium Chemistry Mental Model
Lithium Iron Phosphate (LiFePO4) has an exceptionally flat discharge curve:
- $13.6\text{V} - 14.4\text{V}$: Upper Boost Knee (90% to 100% SOC)
- $13.1\text{V} - 13.4\text{V}$: Extremely Flat Working Plateau (20% to 80% SOC)
- $< 12.8\text{V}$: Rapidly Descending Lower Knee (< 15% SOC)
- $< 11.5\text{V}$: Critical Low Voltage Disconnect

When users see `13.2V` on a linear dial or basic meter, they assume the battery is almost empty (thinking of traditional Lead-Acid 12.0V-12.7V curves) or fail to realize when voltage begins plunging off the lower cliff.

---

## 💡 UX Recommendation & Curve Widget

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    LiFePO4 VOLTAGE & CHEMICAL PROFILE WIDGET                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  Voltage: 13.28V | SOC: 72% | State: MPPT Bulk Charge | Temp: 22°C          │
│                                                                             │
│  14.4V ┐                                            ╭──────── (Boost Knee)  │
│  13.4V │                      ● (Current: 13.28V)───╯                       │
│  13.2V │      ────────────────────────────────── (Flat Working Plateau)     │
│  12.5V │   ──╯ (Low Knee Warning)                                           │
│  10.6V ┴─── (Critical Cutoff LVD)                                           │
│        0%        20%        40%        60%        80%       100% (SOC)      │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Interactive SVG OCV Curve:**
   - Render the non-linear LiFePO4 Open Circuit Voltage (OCV) profile.
   - Plot a glowing current-state puck at the real-time voltage/SOC coordinate.
   - Visually shade the 3 distinct chemical zones:
     - 🟢 **Safe Plateau:** 13.0V - 13.4V
     - 🟡 **Boost / Absorption Zone:** 13.5V - 14.4V
     - 🔴 **Low Voltage Danger Zone:** < 12.8V

---

## 📋 Acceptance Criteria
- [ ] Build responsive SVG chemical curve widget inside the Battery section of `#tab-live` and `#tab-diagnostics`.
- [ ] Smoothly animate marker position on incoming Modbus telemetry.
- [ ] Add tooltips explaining why LiFePO4 voltage stays flat between 30% and 80% SOC.
- [ ] Trigger warning state styling when voltage enters lower knee ($< 12.8\text{V}$).
