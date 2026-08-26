# Mobile Responsive Dashboard (PWA)

Solaria includes a zero-dependency **Progressive Web App (PWA)** styled with a modern off-grid dark theme (`#090d16` background and `#f59e0b` solar amber accents).

---

## 📱 10 Analytical Dashboard Tabs

```mermaid
graph TD
    DASH["Solaria Progressive Web App"]
    DASH --> T1["1. Overview & Live Oscilloscope"]
    DASH --> T2["2. Diurnal Solar Curve & History"]
    DASH --> T3["3. Battery & MPPT Diagnostics"]
    DASH --> T4["4. Interactive Power Budget"]
    DASH --> T5["5. ML Yield Forecasting"]
    DASH --> T6["6. Tree Shading Analyzer"]
    DASH --> T7["7. NOAA Sun Times"]
    DASH --> T8["8. Winterization Checklist"]
    DASH --> T9["9. SRE Diagnostic Logs"]
    DASH --> T10["10. System Topology & Hardware"]
```

---

## 📴 Offline PWA Support

- **Service Worker (`/sw.js`):** Automatically caches CSS, SVG icons, and HTML assets for offline execution.
- **Web App Manifest (`/manifest.json`):** Allows "Add to Home Screen" installation on iOS Safari and Android Chrome as a native fullscreen application.
- **Hardware Topology Interactive Cards:** Live indicators for panel string voltage symmetry, battery plateau stage, and controller heatsink thermal status.
