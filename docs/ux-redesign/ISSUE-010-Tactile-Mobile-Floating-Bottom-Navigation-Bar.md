# [ISSUE-010] Tactile Mobile Floating Bottom Navigation Bar (< 768px)

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P1 (High)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Mobile Shell & PWA)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Thumb-Zone Ergonomics
On modern smartphones (e.g. 6.5"+ screens), reaching top-mounted tabs with one hand while walking outdoors or standing on a cottage dock requires awkward thumb gymnastics or two hands. This violates **Fitts's Law** and mobile ergonomics.

---

## 💡 UX Recommendation & Frosted Bottom Bar

On viewports $< 768\text{px}$, anchor a frosted-glass floating navigation bar in the natural bottom thumb zone:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    MOBILE FLOATING BOTTOM NAVIGATION BAR                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  [  ⚡ Live  ]       [  📊 History  ]       [  🌲 Advisor  ]       [  🩺 Health  ] │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Frosted Glass Styling (CSS Backdrop Filter):**
   - `background: rgba(15, 23, 42, 0.85); backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);`
   - Safe-area inset padding for iOS home indicator (`padding-bottom: env(safe-area-inset-bottom)`).
2. **Haptic & Visual Feedback:**
   - Active icon lights up in gold amber (`#F59E0B`) with an upward glowing micro-bar.
   - Smooth active view switching with zero scroll jumping.

---

## 📋 Acceptance Criteria
- [ ] Show floating bottom navigation bar on screens $< 768\text{px}$; hide top navigation tabs on mobile.
- [ ] Implement `env(safe-area-inset-bottom)` to support modern bezel-less iPhones and Android devices.
- [ ] Ensure minimum touch target size of $48\text{px} \times 48\text{px}$ per WCAG 2.1 touch criteria.
- [ ] Add active indicator badge with amber illumination.
