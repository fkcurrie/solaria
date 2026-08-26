# Tree Shading Diagnostics & Diurnal Notch Analysis

Tree canopy shading creates distinctive, reproducible signatures in off-grid solar curves. Solaria includes an automated **Diurnal Shading Analyzer** that differentiates between atmospheric clouds and fixed tree obstructions.

---

## 🌲 Morning Pine vs. Afternoon Hemlock Profiles

```text
Power (W)
  ▲
400│                           ╭───────────╮ [Unshaded Solar Noon Peak]
300│                          ╭╯           ╰╮
200│                         ╭╯             ╰╮
100│     ╭───[Pine Notch]───╯                ╰───[Hemlock Notch]───╮
  0└───┴───────────┴───────────┴───────────┴───────────┴───────────┴──► Time
     07:00       09:00       11:00       13:00       15:00       17:00
```

1. **Morning White Pines ($08:00 - 10:30$):**
   - High trees to the east block direct low-angle solar rays ($< 25^\circ$ elevation).
   - Harvest remains capped at diffuse ambient levels (~30W–60W) before jumping sharply to >250W once the sun clears the canopy.
2. **Solar Noon Optimal Window ($11:30 - 14:00$):**
   - Direct line-of-sight southern sky access.
   - Array reaches full 340W–385W peak output.
3. **Afternoon Western Hemlocks ($15:30 - 18:00$):**
   - Steep, sudden drop in power as western hemlock branches cast shadows across series strings.

---

## 🪓 Automated Canopy Pruning Recovery Model

The Solaria analytics server computes the estimated energy gain ($\Delta\text{kWh}$) obtainable by pruning specific tree sectors:

```json
{
  "shading_analysis": {
    "morning_pine_loss_kwh": 0.42,
    "afternoon_hemlock_loss_kwh": 0.38,
    "unshaded_potential_kwh": 2.85,
    "actual_harvest_kwh": 2.05,
    "tree_canopy_efficiency": 71.9
  }
}
```
