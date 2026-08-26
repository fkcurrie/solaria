# Cottage Appliance Power Budget & Runtime Calculator

Off-grid battery management requires balancing load consumption against daily solar replenishment.

---

## ⚡ Typical Cottage Appliance Consumption

| Appliance | Power Draw | Duty Cycle | Daily Energy (Wh/day) |
| :--- | :--- | :--- | :--- |
| **Starlink Standard Actuated** | $45\text{ W}$ | 24 hrs continuous | $1,080\text{ Wh}$ |
| **12V DC Compressor Fridge** | $35\text{ W}$ | 35% duty cycle (~8.4 hrs) | $294\text{ Wh}$ |
| **LED Lighting (5 fixtures)** | $20\text{ W}$ | 5 hrs evening | $100\text{ Wh}$ |
| **Water Pressure Pump (12V)** | $65\text{ W}$ | 0.5 hrs intermittent | $32.5\text{ Wh}$ |
| **Laptop & Phone Charging** | $60\text{ W}$ | 3 hrs | $180\text{ Wh}$ |
| **Total Base Cottage Load** | — | — | **~1,686 Wh/day** |

---

## 🧮 Continuous Runtime Equation

The Solaria Power Budget engine calculates the remaining runtime in hours based on current battery state of charge (SOC) and net load:

$$\text{Runtime (hours)} = \frac{(\text{SOC} - \text{Reserve \%}) \times \text{Nominal Capacity (Wh)}}{P_{\text{load}} - P_{\text{solar}}}$$

If $P_{\text{solar}} > P_{\text{load}}$, the system reports **Infinite / Net Positive Harvest** ($\infty$).
