# [ISSUE-014] Smart Skeleton Loading States (Zero "NaN / --" Layout Shift)

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P2 (Medium)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (CSS Animation & State Management)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Initial Render Jitter
When the dashboard first opens in a browser (or wakes up from mobile background), card containers display raw unformatted placeholder text like `-- W`, `-- V`, `NaN%`, or blank spaces before WebSocket handshake finishes. 

When data arrives, the text expands, causing visible Cumulative Layout Shift (CLS) and looking like a broken application for 500ms-1500ms.

---

## 💡 UX Recommendation & Shimmer Skeleton Loaders

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    PULSING SKELETON SHIMMER LOADERS                         │
├─────────────────────────────────────────────────────────────────────────────┤
│  SOLAR HARVEST                    BATTERY BANK                              │
│  ┌───────────────────────────┐    ┌───────────────────────────┐             │
│  │ ████████████  (Shimmer)   │    │ ████████████  (Shimmer)   │             │
│  │ ████████                  │    │ ████████                  │             │
│  └───────────────────────────┘    └───────────────────────────┘             │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **CSS Shimmer Pulse Animation:**
   - Subtle wave gradient sweeping across placeholder bars:
     ```css
     @keyframes shimmer {
       0% { background-position: -200% 0; }
       100% { background-position: 200% 0; }
     }
     .skeleton {
       background: linear-gradient(90deg, #1e293b 25%, #334155 50%, #1e293b 75%);
       background-size: 200% 100%;
       animation: shimmer 1.5s infinite;
       border-radius: 6px;
     }
     ```
2. **Instant Graceful Transition:**
   - Seamlessly fade from skeleton to live numbers when the first valid telemetry frame arrives.

---

## 📋 Acceptance Criteria
- [ ] Replace all `--` and empty text placeholders with `.skeleton` bars during initial loading.
- [ ] Ensure initial DOM layout dimensions match populated metric cards to guarantee $0.00$ CLS.
- [ ] Remove skeletons smoothly with CSS opacity transitions upon first WebSocket packet.
