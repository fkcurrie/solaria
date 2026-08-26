# [ISSUE-005] Tabular Numerics & Zero-Jitter Live Streaming Typography

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (CSS & Typography System)
- **Status:** In Progress (CSS Base Merged, Component Expansion Needed)

---

## 🧐 Problem Statement & Optical Jitter
During live WebSocket streaming (1-second telemetry ticks), decimal numbers update frequently (e.g., `13.41V` $\to$ `13.39V`, `341.2W` $\to$ `348.9W`). 

In standard proportional fonts (like standard Sans-Serif or Inter without OpenType flags), the digit `1` is significantly narrower than the digit `8` or `0`. As numbers update, card widths, text baselines, and alignment jitter back and forth. This micro-vibration causes visual fatigue and gives the interface an unpolished, unstable feel.

---

## 💡 UX Recommendation & Technical Solution

1. **OpenType Tabular Figures (`tnum`):**
   - Apply OpenType font feature `font-variant-numeric: tabular-nums;` and `font-feature-settings: "tnum" 1, "zero" 1;` across all telemetry value spans, tables, chart tooltips, and badges.
2. **Fixed Min-Widths on Value Containers:**
   - Add `.numeric-metric { display: inline-block; min-width: 3.5ch; text-align: right; }` to guarantee zero layout shift (CLS: $0.00$) when metrics transition across significant digits.

```css
.tabular-nums, .stat-value, .metric-val, .badge-val {
  font-variant-numeric: tabular-nums;
  -webkit-font-feature-settings: "tnum" 1, "zero" 1;
  font-feature-settings: "tnum" 1, "zero" 1;
}
```

---

## 📋 Acceptance Criteria
- [ ] Verify `font-variant-numeric: tabular-nums` is active on all dynamic text fields (`#val-pv-w`, `#val-batt-v`, `#val-batt-soc`, `#val-load-w`, etc.).
- [ ] Measure Cumulative Layout Shift (CLS) during 60 seconds of live telemetry stream; verify CLS = $0.000$.
- [ ] Verify clean right-alignment across tabular telemetry rows in summary and diagnostic panes.
