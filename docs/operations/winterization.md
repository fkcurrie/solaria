# Winterization & Departure Checklist

Before vacating an off-grid cottage for the winter freeze, follow the **Solaria Winterization Runbook** to protect LiFePO4 cells from sub-zero degradation and ensure safe solar trickle maintenance.

---

## ❄️ 5-Point Departure Checklist

```text
[ ] 1. Battery State of Charge: Ensure battery SOC is between 50% and 70% (~13.1V - 13.2V).
[ ] 2. Disconnect Inverters & High Loads: Turn off all 120V AC inverters and parasitic DC appliances.
[ ] 3. Enable Charge Controller Sub-Zero Inhibit: Verify low-temperature charging cutoff is active on the Rover MPPT or BMS.
[ ] 4. Check Mechanical Array Clearances: Ensure 2S2P panels are tilted (> 45°) to shed heavy snow buildup.
[ ] 5. Confirm Edge Spool Health: Run ./bin/solaria-e2e-audit to confirm zero uncommitted spool backlogs.
```

---

## 🌡️ LiFePO4 Winter Storage Guidelines

- **Optimal Storage Temperature:** $-10^\circ\text{C}$ to $25^\circ\text{C}$ (Discharging is safe down to $-20^\circ\text{C}$, but **NEVER** charge below $0^\circ\text{C}$).
- **Self-Discharge Rate:** LiFePO4 self-discharges at only $< 2\%$ per month; a 60% charged bank easily survives 6 months of winter without recharging.
