# [ISSUE-009] Contextual "Explain Like I'm Five" (ELI5) Info Tooltips

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P2 (Medium)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Global Tooltip System)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Domain Jargon Friction
Solar energy systems involve dense engineering terminology (*"Performance Ratio (PR)"*, *"MPPT DC-DC Buck Efficiency"*, *"Voc Cold Headroom"*, *"2S2P String Balance"*, *"LiFePO4 OCV Plateau"*).

When non-technical family members or guests look at the dashboard, these terms feel intimidating and incomprehensible.

---

## 💡 UX Recommendation & Material Tooltip Popovers

Implement lightweight, accessible Material Design `(?)` info buttons that trigger plain-language popovers on hover (desktop) or tap (mobile):

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CONTEXTUAL ELI5 EXPLANATION POPOVER                      │
├─────────────────────────────────────────────────────────────────────────────┤
│  Performance Ratio (PR): 94.2%  [ ? ]                                       │
│                                 ┌────────────────────────────────────────┐  │
│                                 │ 💡 What does Performance Ratio mean?   │  │
│                                 │ It compares how much energy your roof  │  │
│                                 │ panels are generating compared to the  │  │
│                                 │ maximum theoretical sunlight shining   │  │
│                                 │ on Wren Lake right now.                │  │
│                                 │ > 85% is exceptional.                  │  │
│                                 └────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 📋 Acceptance Criteria
- [ ] Add accessible `<button class="info-tooltip" aria-label="Learn more about...">` components next to technical headers.
- [ ] Popovers format cleanly on touchscreens with click-away backdrop dismiss.
- [ ] High-contrast styling: Dark slate container (`#1E293B`) with white text and gold highlight keywords.
- [ ] 10 core terms defined in plain English: PR %, Voc Headroom, MPPT Buck, SOC %, Low Voltage Disconnect, Absorption, Float, Diode Drop, LiFePO4 Inhibit, String Imbalance.
