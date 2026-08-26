# [ISSUE-015] Sound & Micro-Haptic Confirmation for Control Mutations

- **Epic:** [EPIC-002: Google Principal UX Review & Design Systems Architecture](./EPIC-002-Google-Principal-UX-Review-and-Design-Systems.md)
- **Priority:** P3 (Low)
- **Assignee Persona:** Google Principal UX & Design Systems Architect
- **Component:** `cmd/cloud-server/templates/index.html` (Interaction Feedback & Audio API)
- **Status:** ✅ COMPLETED & VERIFIED

---

## 🧐 Problem Statement & Mutation Ambiguity
When an operator toggles an off-grid load switch, updates battery charge parameters via Modbus 0x06, or triggers a manual SRE audit, there is no immediate sensory feedback confirming that the hardware received and executed the command. Because radio transmission over BLE takes 100ms-400ms, users may double-click buttons out of uncertainty.

---

## 💡 UX Recommendation & Multi-Sensory Feedback

1. **Web Audio API Subtle Micro-Chimes:**
   - Synthesize lightweight non-intrusive sine wave audio blips without external MP3 asset downloads:
     - **Success Mutation:** Two ascending soft sine tones (520Hz $\to$ 780Hz, 80ms).
     - **Error / Rejection:** Low frequency buzz (180Hz, 120ms).
   - Provide an in-header toggle to mute sounds anytime.
2. **Navigator Vibration API (`navigator.vibrate`):**
   - On supported mobile devices, trigger short 15ms micro-haptic pulse on successful button press.

```javascript
function playMicroChime(success) {
  if (isMuted || !window.AudioContext) return;
  const ctx = new (window.AudioContext || window.webkitAudioContext)();
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.connect(gain);
  gain.connect(ctx.destination);
  if (success) {
    osc.frequency.setValueAtTime(587.33, ctx.currentTime); // D5
    osc.frequency.exponentialRampToValueAtTime(880, ctx.currentTime + 0.08); // A5
    gain.gain.setValueAtTime(0.08, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.12);
    osc.start(); osc.stop(ctx.currentTime + 0.12);
    if (navigator.vibrate) navigator.vibrate(15);
  }
}
```

---

## 📋 Acceptance Criteria
- [ ] Implement synthetic Web Audio feedback on hardware control actions and preset selections.
- [ ] Add mobile micro-haptics (`navigator.vibrate(15)`).
- [ ] Add explicit audio mute toggle in dashboard header (`🔊 / 🔇`) with persistence in `localStorage`.
