# [ISSUE-003] Tab Navigation Re-Architecture (4-Category Segmented Controls)

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Global Navigation Shell)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Navigation Overload
The dashboard currently exposes 9 horizontal tab buttons in a single un-nested row:
`[Live]` `[Day]` `[Week]` `[Month]` `[Year]` `[Advisor]` `[Forecast]` `[Specs]` `[Diagnostics]`

On tablets, laptops, and mobile viewports, this causes horizontal scrolling, visual clutter, and decision fatigue. The flat navigation does not distinguish between **real-time operations**, **historical trends**, **seasonal planning**, and **engineering diagnostics**.

---

## 💡 UX Recommendation & Information Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                 RE-ARCHITECTED 4-CATEGORY SEGMENTED NAVIGATION              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  [⚡ Live Monitor]   [📊 History]   [🌲 Solar Advisor]   [🩺 Diagnostics]   │
│                                                                             │
│  Sub-View Switches (Contextual Segmented Pill Bar):                         │
│  • Under [📊 History]:         [ 1D ]  [ 7D ]  [ 30D ]  [ 1Y ]  [ All ]     │
│  • Under [🌲 Solar Advisor]:   [ Runtime Budget ] [ Shading ] [ Winterize ] │
│  • Under [🩺 Diagnostics]:     [ LiFePO4 Chemistry ] [ Specs ] [ SRE Logs ] │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **4 Primary Top-Level Destinations:**
   - **⚡ Live Monitor:** Real-time energy flow vector, glance card trio, oscilloscope, and live alarms.
   - **📊 Historical Analytics:** Unified diurnal charts with time-range segmented pills (`1D`, `7D`, `30D`, `1Y`).
   - **🌲 Solar & Cottage Advisor:** Off-grid runtime advisor, seasonal peak forecasting, and winterization checklists.
   - **🩺 System Health & Diagnostics:** Battery cell physics, commissioning wizard, and SRE audit logs.
2. **Contextual Sub-Navigation:**
   - Selecting a primary tab exposes smooth secondary pills without reloading or scrolling.
   - URL hash updates automatically (e.g., `#history/7d`, `#advisor/budget`) to support bookmarking and browser back-button navigation.

---

## 📋 Acceptance Criteria
- [ ] Collapse 9 top-level tabs into 4 primary segmented tabs.
- [ ] Implement smooth animated transition when switching active views.
- [ ] Add URL deep-linking support (`window.location.hash`).
- [ ] Keyboard accessible: Full `Tab`, `ArrowLeft`, `ArrowRight`, `Enter` navigation with ARIA attributes (`role="tablist"`, `role="tab"`, `aria-selected="true"`).
