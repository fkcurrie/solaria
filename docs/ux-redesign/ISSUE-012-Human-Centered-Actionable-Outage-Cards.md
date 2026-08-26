# [ISSUE-012] Human-Centered Actionable Outage Cards (1-2-3 Remediation)

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P0 (Critical)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Error UI & Incident Notification)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Error Panic
When edge Bluetooth disconnects, WiFi glitches, or the bridge process stops, the dashboard previously rendered cryptic red banners:
`"Error: Failed to fetch /api/v1/live: WebSocket closed 1006 abnormal closure."`

This triggers user anxiety without providing actionable guidance on what physical steps to take at the cottage.

---

## 💡 UX Recommendation & Actionable Outage Card

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    HUMAN-CENTERED OUTAGE REMEDIATION CARD                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  🔴 HARDWARE DISCONNECTED                                                   │
│  Bluetooth communication with Renogy BT-1 module paused 42 seconds ago.     │
│                                                                             │
│  🛠️ QUICK 3-STEP FIX:                                                       │
│  1. Check if the green LED on the Renogy BT-1 module is blinking.           │
│  2. Ensure your bridge gateway / phone is within 15ft of the battery bank.  │
│  3. Tap the button below to force an autonomous hardware reconnect.         │
│                                                                             │
│  [ 🔄 Attempt Autonomous Reconnect ]    [ 🩺 Open SRE Diagnostic Log ]       │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Clear Plain-Language Incident Header:**
   - Clearly states what failed, how long ago, and system safety status (e.g. *"Battery charge state is safe. Offline data is spooled to disk."*).
2. **Numbered Step-by-Step Remediation:**
   - 3 actionable physical and software checks.
3. **One-Tap Autonomous Recovery Actuator:**
   - Invokes `/api/v1/sre/auto-heals` or bridge reconnect endpoint directly from the banner.

---

## 📋 Acceptance Criteria
- [ ] Replace cryptic red WebSocket error banners with the structured Actionable Outage Card.
- [ ] Display elapsed disconnect time counter (`disconnected for Xs / Xm`).
- [ ] Provide `[Attempt Autonomous Reconnect]` action button with inline loading spinner.
- [ ] Include reassuring zero-data-loss indicator: *"🛡️ Disk spooler active: 0 records lost"*.
