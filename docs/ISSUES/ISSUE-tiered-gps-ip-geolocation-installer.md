# Feature Request: Multi-Tiered Location Resolver in Installer (Hardware GPS -> IP Geolocation -> Manual)

**Issue Title:** `[FEATURE] Implement Multi-Tiered Location Resolver (Hardware GPS / gpsd -> IP Geolocation -> Manual Override) in setup.sh and install.sh`

---

## 🚀 Overview

Solaria's atmospheric radiometry (Open-Meteo correlation, solar elevation, GHI, DHI, DNI, and Performance Ratio) relies on accurate site latitude and longitude coordinates (`SITE_LATITUDE`, `SITE_LONGITUDE`). 

Currently, site coordinates are either supplied via environment variables or fall back to static default values. To support both mobile/off-grid deployments (vans, RVs, marine vessels) and automated fixed installations, the installation scripts (`setup.sh` and `install.sh`) and runtime edge agent should support a **Multi-Tiered Location Resolver**.

---

## 📐 Proposed Architecture

```text
[Installer / Runtime Location Resolver]
             │
             ├──► Tier 1: Hardware GPS Receiver (gpsd / GeoClue / NMEA Serial)
             │      └─ High Precision (~1-5 meters). Ideal for mobile/off-grid rigs.
             │
             ├──► Tier 2: Network IP Geolocation API (http://ip-api.com/json / ipinfo)
             │      └─ City-level accuracy (~1-10 km). Automatic fallback when online without GPS fix.
             │
             └──► Tier 3: Explicit Environment Variables (SITE_LATITUDE, SITE_LONGITUDE in .env)
                    └─ Air-gapped / fixed user configuration override.
```

---

## 🛠️ Implementation Strategy

1. **`install.sh` / `setup.sh` Enhancement:**
   - Query `gpsd` (`localhost:2947`) or `geoclue` via D-Bus if present.
   - If no GPS fix is returned, perform HTTP fallback to `http://ip-api.com/json` (or `https://ipinfo.io/json`).
   - Populate `.env` with auto-detected `SITE_NAME`, `SITE_LATITUDE`, and `SITE_LONGITUDE`.

2. **Go Core Package Helper (`pkg/location`):**
   - Add a Go helper package to resolve runtime location updates dynamically if installed on a mobile system with a USB/UART GPS dongle.

---

## ✅ Acceptance Criteria

- [ ] Running `./install.sh -y` automatically detects host coordinates via GPS or IP geolocation without requiring hardcoded defaults.
- [ ] Fallback chain (`gpsd` -> `ip-api.com` -> `ipinfo.io` -> `.env`) works seamlessly across offline and online environments.
- [ ] Unit tests cover location resolution logic and fallback scenarios.
