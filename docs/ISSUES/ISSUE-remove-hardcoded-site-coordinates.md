# Refactor Request: Scrub Hardcoded Location Coordinates and Array Wattage Defaults

**Issue Title:** `[REFACTOR] Remove hardcoded site location coordinates (45.186° N, -78.863° W) and array peak wattage (400W) defaults across codebase and docs`

---

## 🚀 Overview

The codebase and documentation currently contain hardcoded site location coordinates (`45.186° N, -78.863° W`, Dorset, ON) and solar array peak capacity (`400W 2S2P`) as embedded default fallbacks in `README.md`, `setup.sh`, `install.sh`, and test strings.

Specific site coordinates and solar array hardware specifications should be dynamically auto-detected during installation (via GPS / IP Geolocation) or requested during interactive onboarding, rather than baked into repository files.

---

## 🔍 Locations Identified for Cleanup

1. **`README.md`**:
   - Celestial Trajectory Engine description (`45.186°N, -78.863°W`).
   - Setup default environment variables.
2. **`setup.sh` & `install.sh`**:
   - Default values for `SITE_NAME`, `SITE_LATITUDE`, `SITE_LONGITUDE`, and `PANEL_RATED_WATTS`.
3. **`docs/`**:
   - Architecture diagrams and solar hardware specifications mentioning hardcoded default location.

---

## 🛠️ Proposed Fix

1. Replace static coordinate defaults in `setup.sh` and `install.sh` with dynamic location resolution (calling `ip-api.com` or `gpsd`).
2. Update documentation examples to use generic placeholder parameters (e.g. `YOUR_LATITUDE`, `YOUR_LONGITUDE`).
3. Ensure default `.env` generator uses auto-detected host coordinates.

---

## ✅ Acceptance Criteria

- [ ] Zero references to hardcoded latitude `45.186` or longitude `-78.863` remain in installation scripts as hardcoded defaults.
- [ ] Installer automatically queries host location services or prompts during interactive setup.
- [ ] CI/CD tests pass with dynamic coordinate injection.
