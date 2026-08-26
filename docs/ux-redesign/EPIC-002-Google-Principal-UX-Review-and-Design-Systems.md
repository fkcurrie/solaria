# [EPIC-002] Google Principal UX Review & Design Systems Architecture

## 📌 Executive Summary
Transform the **Solaria Solar & Atmospheric Intelligence Dashboard** (`cmd/cloud-server/templates/index.html`) from an engineering-dense telemetry console into a world-class, glanceable, accessible, and human-centered monitoring platform. Applying Google's core design heuristics, Material Design 3 (M3) progressive disclosure, and WCAG 2.1 AA/AAA accessibility standards, this EPIC establishes the design system, navigation hierarchy, data visualization models, and 15 actionable sub-issues required to achieve effortless 3-second situational awareness for off-grid cottage living at **1296 Wren Lake Drive, Dorset, Ontario**.

---

## 🧭 Persona & Governance
- **Lead UX Reviewer / Persona:** **Google Principal UX & Design Systems Architect** (Persona 6 in [`docs/SOUL.md`](../SOUL.md) & [`docs/review-personas.md`](../review-personas.md))
- **Mindset:** *"Focus on the user and all else will follow."* Fast is better than slow. Reduce visual noise, eliminate cognitive scanning friction, and elevate the primary questions:
  1. *Is my battery charging or discharging right now?*
  2. *How many hours of power do I have left?*
  3. *When does peak solar harvest occur today?*

---

## 🎯 High-Level Objectives

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                 EPIC-002: 4 CORE UX TRANSFORMATION DOMAINS                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  1. 3-Second Situational Awareness (Glanceability)                          │
│     - Hero Animated Energy Flow Vector (PV -> MPPT -> Battery -> Loads)     │
│     - Primary 3-Card Summary Anchor (Harvest, Storage, Daylight Horizon)    │
│     - 4-Category Segmented Navigation Hierarchy                             │
│                                                                             │
│  2. Scientific Data Visualization & Chromatic Harmony                       │
│     - Material Design 3 (M3) 4-Role Semantic Color Palette                  │
│     - Zero-Jitter Tabular Typography (`tabular-nums`, `tnum`)                │
│     - LiFePO4 Non-Linear SOC Voltage Plateau Curve Widget                   │
│     - NOAA Sun-Arc Diurnal Elevation & Peak Window Radial Dome              │
│                                                                             │
│  3. Ergonomic Interaction & Mobile Cottage Usability                        │
│     - 1-Tap Power Budget Scenario Presets (Eco, Workday, Winter)            │
│     - Contextual "Explain Like I'm Five" (ELI5) Info Tooltips               │
│     - Thumb-Zone Frosted Floating Bottom Navigation (< 768px)               │
│     - Automatic High-Contrast "Sunlight Mode" Auto-Trigger                  │
│                                                                             │
│  4. SRE Resilience & Error Ergonomics                                       │
│     - Human-Centered Actionable Outage Cards (1-2-3 Remediation)            │
│     - Diagnostic Log Streaming Chips & Auto-Scroll Pause Lock               │
│     - Fluid Skeleton Pulse Loaders (Zero "NaN / --" Layout Shift)           │
│     - Audible / Haptic Confirmation for Hardware Parameter Writes           │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 📋 Breakdown of 15 Sub-Issues (Implementation Status)

| Issue ID | Title | Priority | Target Scope | Status |
| :--- | :--- | :--- | :--- | :--- |
| [**ISSUE-001**](./ISSUE-001-Hero-Energy-Flow-Vector-Sankey.md) | **Hero Energy Flow Vector (Live Animated Power Sankey)** | P0 (Critical) | Live Operations Tab / Hero Header | ✅ **COMPLETED** |
| [**ISSUE-002**](./ISSUE-002-Primary-Glance-Card-Trio-Anchor-Hierarchy.md) | **Primary Glance Card Trio (Generation, Storage, Horizon)** | P0 (Critical) | Summary Anchor Dashboard | ✅ **COMPLETED** |
| [**ISSUE-003**](./ISSUE-003-Tab-Navigation-Re-Architecture-Segmented-Controls.md) | **Tab Navigation Re-Architecture (4-Category Segmented Control)** | P1 (High) | Global Shell & Navigation | ✅ **COMPLETED** |
| [**ISSUE-004**](./ISSUE-004-Semantic-Color-Harmony-Material-3-WCAG-Audit.md) | **Semantic Color Harmony & WCAG 2.1 AA/AAA Contrast Audit** | P1 (High) | Global CSS & Design Tokens | ✅ **COMPLETED** |
| [**ISSUE-005**](./ISSUE-005-Tabular-Numerics-Zero-Jitter-Live-Typography.md) | **Tabular Numerics & Zero-Jitter Streaming Typography** | P1 (High) | Global Typography & Live Cards | ✅ **COMPLETED** |
| [**ISSUE-006**](./ISSUE-006-LiFePO4-Non-Linear-SOC-Voltage-Plateau-Widget.md) | **LiFePO4 Non-Linear SOC Voltage Curve Visualizer** | P1 (High) | Battery Diagnostics & Live Pane | ✅ **COMPLETED** |
| [**ISSUE-007**](./ISSUE-007-Sun-Arc-Diurnal-Elevation-Radial-Compass-Tracker.md) | **Sun-Arc Diurnal Elevation & Peak Hour Radial Tracker** | P1 (High) | Solar Advisor & Live Tab | ✅ **COMPLETED** |
| [**ISSUE-008**](./ISSUE-008-Power-Budget-Scenario-Quick-Pick-Presets.md) | **Power Budget Scenario Quick-Picks (Eco, Workday, Winter)** | P1 (High) | Off-Grid Power Budget Advisor | ✅ **COMPLETED** |
| [**ISSUE-009**](./ISSUE-009-Contextual-ELI5-Glossary-Tooltips.md) | **Contextual "Explain Like I'm Five" (ELI5) Info Tooltips** | P2 (Medium) | Global System Metrics & Headers | ✅ **COMPLETED** |
| [**ISSUE-010**](./ISSUE-010-Tactile-Mobile-Floating-Bottom-Navigation-Bar.md) | **Tactile Mobile Floating Bottom Navigation Bar (< 768px)** | P1 (High) | Mobile Viewport / PWA Shell | ✅ **COMPLETED** |
| [**ISSUE-011**](./ISSUE-011-High-Contrast-Sunlight-Mode-Auto-Trigger.md) | **High-Contrast "Sunlight Mode" Auto-Trigger & Outdoor Theme** | P2 (Medium) | Outdoor Readability / Sensor API | ✅ **COMPLETED** |
| [**ISSUE-012**](./ISSUE-012-Human-Centered-Actionable-Outage-Cards.md) | **Human-Centered Actionable Outage Cards (1-2-3 Remediation)** | P0 (Critical) | Incident Banner & Error UI | ✅ **COMPLETED** |
| [**ISSUE-013**](./ISSUE-013-Diagnostic-Log-Stream-Filtering-and-Scroll-Lock.md) | **Diagnostic Log Stream Filter Chips & Auto-Scroll Lock** | P2 (Medium) | SRE Diagnostics Console | ✅ **COMPLETED** |
| [**ISSUE-014**](./ISSUE-014-Smart-Loading-Skeleton-States-No-NaN-Jitter.md) | **Smart Skeleton Loading States (Zero "NaN / --" Layout Shift)** | P2 (Medium) | Initial Handshake & Async Fetch | ✅ **COMPLETED** |
| [**ISSUE-015**](./ISSUE-015-Sound-and-Micro-Haptic-Feedback-for-Mutations.md) | **Sound & Micro-Haptic Confirmation for Control Mutations** | P3 (Low) | Modbus Mutation Buttons | ✅ **COMPLETED** |

---

## 📐 Design Tokens & Architecture Specifications

```css
/* Core Design Tokens (Material Design 3 Solar Spectrum) */
:root {
  --color-solar-amber: #F59E0B;      /* PV Generation & Solar Influx */
  --color-solar-amber-dim: #D97706;
  --color-solar-amber-bg: rgba(245, 158, 11, 0.12);
  
  --color-battery-emerald: #10B981;    /* Battery Normal & Charging */
  --color-battery-warning: #F59E0B;    /* Battery SOC < 40% */
  --color-battery-critical: #EF4444;   /* Battery Voltage < 11.8V */
  
  --color-sky-azure: #38BDF8;          /* Irradiance, NOAA Solstice */
  --color-load-indigo: #818CF8;        /* Cottage AC/DC Consumption */
  
  --font-numeric: "Inter", "Roboto Mono", monospace;
  --font-feature-numeric: "tnum" 1, "zero" 1;
}
```

---

## 🏁 Verification & Acceptance Criteria
1. **Lighthouse Performance & Accessibility:** $\ge 98$ on Mobile and Desktop.
2. **WCAG 2.1 AA Compliance:** Minimum 4.5:1 text contrast ratio in standard mode; 7:1 in Sunlight Mode.
3. **Cumulative Layout Shift (CLS):** $0.00$ across dynamic telemetry streaming and tab transitions.
4. **First Contentful Paint (FCP):** $< 800\text{ms}$ on LTE / 3G cottage connections.
