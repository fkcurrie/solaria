# Feature Request: Fullscreen Kiosk & Ambient Screensaver Mode for Touchscreen Displays

**Issue Title:** `[FEATURE] Fullscreen Kiosk & Ambient Screensaver Mode for Touchscreen Nodes (Surface Go, Pi Displays, Wall Monitors)`

---

## 🚀 Overview

Solaria's Web Dashboard is often deployed on dedicated touchscreen devices (such as a Microsoft Surface Go, Raspberry Pi Touchscreen, or wall-mounted iPad) operating in off-grid solar cabins, vans, and battery sheds.

When left unattended, standard web dashboards suffer from small fonts that are hard to read from across the room, and displays either dim/sleep or risk static OLED/IPS image burn-in.

This feature requests a dedicated **Fullscreen Kiosk & Ambient Screensaver Mode** that transforms the dashboard into a high-visibility, glanceable solar monitor.

---

## 📐 Key Capabilities & Requirements

```text
[Solaria Web Dashboard]
        │
        ├──► Trigger 1: Auto-engage on idle inactivity timeout (e.g. 2 minutes)
        ├──► Trigger 2: Manual "Enter Kiosk Mode" button or 'F' / 'S' key shortcut
        │
        ├──► Display Management: HTML5 Screen Wake Lock API (keeps display on)
        │
        ├──► Glanceable UI Layout:
        │      ├─ Giant Solar Generation (Watts / kW & Peak %)
        │      ├─ LiFePO4 Battery SOC (SOC %, Voltage, Net Amps)
        │      ├─ Atmospheric Status (Temp, Cloud Cover, Sun Elevation)
        │      └─ Subtle Pixel Shift (OLED/IPS screen burn-in prevention)
        │
        └──► Tap-to-Wake Dismissal: Single touchscreen tap or keypress restores full dashboard
```

---

## 🛠️ Implementation Plan

### 1. Kiosk Engine & HTML5 Fullscreen API (`cmd/bridge/static/index.html`)
- Integrate HTML5 Fullscreen API (`document.documentElement.requestFullscreen()`).
- Add an idle watchdog timer (`mousemove`, `touchstart`, `keydown`) that auto-triggers Kiosk Mode after 120 seconds of inactivity.

### 2. Glanceable Screensaver UI Layer
- High-contrast, large-font typography optimized for viewing from 5–10 feet away.
- Burn-in protection: Shift UI elements by a few pixels every 5 minutes to protect OLED/IPS panels on devices like the Surface Go.

### 3. Screen WakeLock API (`navigator.wakeLock`)
- Request a screen wake lock when entering Kiosk Mode to prevent the operating system from dimming or turning off the screen.

### 4. Tap-to-Wake Navigation
- Any touchscreen tap, mouse click, or keypress seamlessly exits Kiosk Mode and restores interactive controls.

---

## ✅ Acceptance Criteria

- [ ] Clicking the "Kiosk Mode" header button or pressing `F` / `S` toggles fullscreen screensaver mode.
- [ ] Inactivity timer automatically engages screensaver after 2 minutes.
- [ ] Screen Wake Lock keeps the display active without OS dimming.
- [ ] Giant metrics (Solar Watts, Battery SOC%, Temp, Solar Elevation) are visible across the room.
- [ ] Tapping anywhere exits screensaver mode instantly.
