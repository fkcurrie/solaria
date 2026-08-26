# [ISSUE-011] High-Contrast "Sunlight Mode" Auto-Trigger & Outdoor Theme

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P2 (Medium)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Theme Engine & CSS Tokens)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Direct Glare Contrast
Dark mode dashboards look sleek indoors, but when viewed outdoors on a dock or deck under $80,000+\text{ lux}$ of direct Ontario summer sunlight, dark glass containers suffer severe glare and become unreadable without setting phone screen brightness to 100% (draining the phone battery).

---

## 💡 UX Recommendation & High-Contrast Mode

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    HIGH-CONTRAST OUTDOOR SUNLIGHT THEME                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Background: Pure Crisp White `#FFFFFF`                                     │
│  Cards: #F8FAFC with 2px Solid Dark Slate `#0F172A` Borders                 │
│  Typography: Pure Jet Black `#000000` (Contrast Ratio: 21:1 WCAG AAA)       │
│  Metrics: Deep Amber `#B45309` and Forest Green `#047857`                   │
│                                                                             │
│  Auto-Trigger Options:                                                      │
│  • Ambient Light Sensor API (`AmbientLightSensor` in supported browsers)    │
│  • Astronomical Schedule (Auto-activate between Sunrise and Sunset)         │
│  • Manual Quick Toggle in Header                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 📋 Acceptance Criteria
- [ ] Refine `.sunlight-mode` CSS class to deliver strict WCAG AAA contrast ratio ($\ge 7:1$).
- [ ] Implement smart automatic theme trigger based on real-time solar elevation ($> 10^\circ$).
- [ ] Add header toggle button with instantaneous transitions (no flash of unstyled theme).
- [ ] Persist preference in `localStorage.getItem("solaria_theme")`.
