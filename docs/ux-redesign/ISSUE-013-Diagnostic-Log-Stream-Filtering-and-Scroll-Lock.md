# [ISSUE-013] Diagnostic Log Stream Filter Chips & Auto-Scroll Lock

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P2 (Medium)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (SRE Diagnostics Console)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & SRE Inspection Friction
The Diagnostics tab streams bridge and cloud server logs into a monospace terminal window. However:
1. When troubleshooting an intermittent Modbus CRC error or BLE disconnect, continuous log streaming forces the container to auto-scroll to the bottom, yanking text out from under the user's cursor.
2. Routine `[INFO]` logs drown out critical `[ERROR]` and `[WARN]` events.

---

## 💡 UX Recommendation & Log Stream Console Controls

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    DIAGNOSTIC LOG STREAM FILTERING & CONTROLS               │
├─────────────────────────────────────────────────────────────────────────────┤
│  Filter:  [ ALL (142) ]  [ 🔴 ERROR (2) ]  [ 🟡 WARN (5) ]  [ 🔵 INFO (135) ] │
│  Search:  [ 🔍 filter text...              ]   [ ⏸️ PAUSE SCROLL ] [ 🧹 CLEAR ] │
│                                                                             │
│  15:45:12.012 [INFO]  Renogy Modbus frame 0x0100 parsed successfully.       │
│  15:45:14.281 [WARN]  BLE RSSI dropped to -88 dBm (Weak signal warning).    │
│  15:45:15.004 [ERROR] Cloud Run POST returned 401. Failover to Local Ingest. │
│  15:45:15.018 [INFO]  Local ingest sync succeeded (200 OK).                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

1. **Interactive Severity Filter Chips:**
   - Filter stream dynamically by log severity with active event count badges.
2. **Sticky Auto-Scroll Pause Button:**
   - Automatically pauses scrolling when the user manually scrolls up; resumes when scrolled to bottom.
3. **Live Substring Search Bar:**
   - Instant client-side text filtering as you type.

---

## 📋 Acceptance Criteria
- [ ] Add filter chips (`[ALL]`, `[ERROR]`, `[WARN]`, `[INFO]`) to `#tab-diagnostics`.
- [ ] Implement user scroll detection: auto-pause streaming scroll when viewing history.
- [ ] Add client-side regex / text search input.
- [ ] Add `[Copy Logs to Clipboard]` button with visual "Copied!" checkmark animation.
