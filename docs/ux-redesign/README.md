# 🎨 Google Principal UX Review & Design Systems Architecture

This directory contains the complete design architecture, heuristics review, and implementation issues produced by the **Google Principal UX & Design Systems Architect** persona for Project Solaria.

---

## 📌 Master Epic
- **[EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)**

---

## 📋 15 Prioritized Actionable Sub-Issues (Implementation Status)

| Issue ID | Priority | Title | Key Target Area | Status |
| :--- | :--- | :--- | :--- | :--- |
| [**ISSUE-001**](./ISSUE-001-Hero-Energy-Flow-Vector-Sankey.md) | **P0 (Critical)** | Hero Energy Flow Vector (Live Animated Power Sankey) | Live Operations Tab / Hero Header | ✅ **COMPLETED** |
| [**ISSUE-002**](./ISSUE-002-Primary-Glance-Card-Trio-Anchor-Hierarchy.md) | **P0 (Critical)** | Primary Glance Card Trio (Generation, Storage, Horizon) | Summary Anchor Dashboard | ✅ **COMPLETED** |
| [**ISSUE-003**](./ISSUE-003-Tab-Navigation-Re-Architecture-Segmented-Controls.md) | **P1 (High)** | Tab Navigation Re-Architecture (4-Category Segmented Control) | Global Shell & Navigation | ✅ **COMPLETED** |
| [**ISSUE-004**](./ISSUE-004-Semantic-Color-Harmony-Material-3-WCAG-Audit.md) | **P1 (High)** | Semantic Color Harmony & WCAG 2.1 AA/AAA Contrast Audit | Global CSS & Design Tokens | ✅ **COMPLETED** |
| [**ISSUE-005**](./ISSUE-005-Tabular-Numerics-Zero-Jitter-Live-Typography.md) | **P1 (High)** | Tabular Numerics & Zero-Jitter Streaming Typography | Global Typography & Live Cards | ✅ **COMPLETED** |
| [**ISSUE-006**](./ISSUE-006-LiFePO4-Non-Linear-SOC-Voltage-Plateau-Widget.md) | **P1 (High)** | LiFePO4 Non-Linear SOC Voltage Curve Visualizer | Battery Diagnostics & Live Pane | ✅ **COMPLETED** |
| [**ISSUE-007**](./ISSUE-007-Sun-Arc-Diurnal-Elevation-Radial-Compass-Tracker.md) | **P1 (High)** | Sun-Arc Diurnal Elevation & Peak Hour Radial Tracker | Solar Advisor & Live Tab | ✅ **COMPLETED** |
| [**ISSUE-008**](./ISSUE-008-Power-Budget-Scenario-Quick-Pick-Presets.md) | **P1 (High)** | Power Budget Scenario Quick-Picks (Eco, Workday, Winter) | Off-Grid Power Budget Advisor | ✅ **COMPLETED** |
| [**ISSUE-009**](./ISSUE-009-Contextual-ELI5-Glossary-Tooltips.md) | **P2 (Medium)** | Contextual "Explain Like I'm Five" (ELI5) Info Tooltips | Global System Metrics & Headers | ✅ **COMPLETED** |
| [**ISSUE-010**](./ISSUE-010-Tactile-Mobile-Floating-Bottom-Navigation-Bar.md) | **P1 (High)** | Tactile Mobile Floating Bottom Navigation Bar (< 768px) | Mobile Viewport / PWA Shell | ✅ **COMPLETED** |
| [**ISSUE-011**](./ISSUE-011-High-Contrast-Sunlight-Mode-Auto-Trigger.md) | **P2 (Medium)** | High-Contrast "Sunlight Mode" Auto-Trigger & Outdoor Theme | Outdoor Readability / Sensor API | ✅ **COMPLETED** |
| [**ISSUE-012**](./ISSUE-012-Human-Centered-Actionable-Outage-Cards.md) | **P0 (Critical)** | Human-Centered Actionable Outage Cards (1-2-3 Remediation) | Incident Banner & Error UI | ✅ **COMPLETED** |
| [**ISSUE-013**](./ISSUE-013-Diagnostic-Log-Stream-Filtering-and-Scroll-Lock.md) | **P2 (Medium)** | Diagnostic Log Stream Filter Chips & Auto-Scroll Lock | SRE Diagnostics Console | ✅ **COMPLETED** |
| [**ISSUE-014**](./ISSUE-014-Smart-Loading-Skeleton-States-No-NaN-Jitter.md) | **P2 (Medium)** | Smart Skeleton Loading States (Zero "NaN / --" Layout Shift) | Initial Handshake & Async Fetch | ✅ **COMPLETED** |
| [**ISSUE-015**](./ISSUE-015-Sound-and-Micro-Haptic-Feedback-for-Mutations.md) | **P3 (Low)** | Sound & Micro-Haptic Confirmation for Control Mutations | Modbus Mutation Buttons | ✅ **COMPLETED** |

---

## 🎯 Guiding Design Heuristics
1. **The 3-Second Glanceability Test:** Cottage owners must understand battery flux in under 3 seconds.
2. **Progressive Disclosure:** Separate primary metrics from tertiary engineering registers.
3. **WCAG 2.1 AA/AAA Accessibility:** Ensure sunlight readability and accessible touch targets.
4. **Fitts's Law Mobile Ergonomics:** Anchor interactive navigation in bottom thumb zone on smartphones.
