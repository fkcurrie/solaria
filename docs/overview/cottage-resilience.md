# Cottage Resilience & Off-Grid FAQ

Off-grid cottages in remote northern environments (such as Ontario's cottage country) face unique challenges: freezing sub-zero temperatures, multi-day winter storms, frequent cellular/LTE drops, and power blackouts.

---

## ❄️ Canadian Winter & Sub-Zero Design

> [!CAUTION]
> **Sub-Zero Lithium Plating Hazard:**
> Charging lithium iron phosphate (LiFePO4) cells below $0^\circ\text{C}$ ($32^\circ\text{F}$) causes metallic lithium to deposit onto the graphite anode instead of intercalating. This permanently destroys battery capacity and creates an internal short-circuit fire risk.

Solaria incorporates software safety invariants that detect sub-zero battery conditions in real time:
- The `solaria-bridge` and `solaria-sre-agent` continuously monitor internal battery temperature via Modbus registers.
- When $T_{\text{battery}} \le 0^\circ\text{C}$, the system flags `subzero_inhibit = true` and triggers high-visibility alerts on the dashboard.

---

## 🌲 Diurnal Tree Shading Analysis

Remote installations are often surrounded by tall trees and surrounding forest canopy:
- **Morning Tree Shading (East):** Dense eastern tree canopy can block direct morning sun between 08:00 and 10:30, keeping array voltage low until the sun climbs above $25^\circ$ elevation.
- **Solar Noon Peak (South):** Direct uninhibited irradiance between 11:30 and 14:00 where harvest reaches 340W–385W (85–96% of rated STC).
- **Afternoon Tree Shading (West):** Late-afternoon tree and ridge shadows create characteristic "notches" in the diurnal power curve.

Solaria's built-in **Tree Shading Diagnostic Analyzer** calculates exactly how much daily energy (kWh) is lost to specific tree profiles and provides guidance on canopy trimming.

---

## 📡 Offline Zero-Data-Loss Architecture

When winter storms knock out local cellular towers or starlink satellite terminals:
1. The edge bridge writes all incoming 1-second telemetry samples to an atomic FIFO spool file on disk.
2. Even if power to the Raspberry Pi drops suddenly, the volatile write journal (`vm.dirty_writeback_centisecs`) ensures data consistency.
3. Once the network reconnects, Solaria automatically drains the spool in batches without overloading cloud endpoints.
