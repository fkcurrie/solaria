# ⚡ Solar Array & Charge Controller Engineering Specifications

## Solar Array Topology: 4x100W 2S2P

The solar generation array at the Dorset, ON site (1296 Wren Lake Dr) consists of four 100W monocrystalline photovoltaic panels arranged in a **2-Series, 2-Parallel (2S2P)** configuration.

```
       [ Panel 1: 100W ] (+) --- (-) [ Panel 2: 100W ]  (Series String 1: ~36V-40V, ~5.5A)
              |                                  |
              +-----------(+)     (-)------------+
                           |       |
                         [+]     [-] ====> Combined 2S2P Input to Rover (36-40V Vmp, 11A Imp)
                           |       |
              +-----------(+)     (-)------------+
              |                                  |
       [ Panel 3: 100W ] (+) --- (-) [ Panel 4: 100W ]  (Series String 2: ~36V-40V, ~5.5A)
```

---

## Technical Specifications Table

| Parameter | Single Panel (100W) | 2S String (200W) | Array Total (2S2P, 400W) | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **Peak Power ($P_{\text{max}}$)** | $100\,\text{W}$ | $200\,\text{W}$ | **$400\,\text{W}$** | Total nominal array capacity |
| **Optimum Operating Voltage ($V_{\text{mp}}$)** | $18.0\,\text{V} - 20.4\,\text{V}$ | $36.0\,\text{V} - 40.8\,\text{V}$ | **$36.0\,\text{V} - 40.8\,\text{V}$** | Optimal MPPT tracking window |
| **Optimum Operating Current ($I_{\text{mp}}$)** | $4.9\,\text{A} - 5.5\,\text{A}$ | $4.9\,\text{A} - 5.5\,\text{A}$ | **$9.8\,\text{A} - 11.0\,\text{A}$** | Within 10AWG wiring capacity |
| **Open-Circuit Voltage ($V_{\text{oc}}$)** | $21.6\,\text{V} - 24.3\,\text{V}$ | $43.2\,\text{V} - 48.6\,\text{V}$ | **$43.2\,\text{V} - 48.6\,\text{V}$** | Safely below Rover 100V limit |
| **Short-Circuit Current ($I_{\text{sc}}$)** | $5.4\,\text{A} - 5.9\,\text{A}$ | $5.4\,\text{A} - 5.9\,\text{A}$ | **$10.8\,\text{A} - 11.8\,\text{A}$** | Fused with 15A inline MC4 fuses |

---

## Charge Controller: Renogy Rover 20A MPPT (`RNG-CTRL-RVR20-CAN`)

* **Maximum PV Input Voltage:** $100\,\text{V DC}$ (absolute maximum).
* **Maximum Battery Charging Current:** $20\,\text{A}$ to nominal 12V battery bank ($\approx 288\,\text{W}$ charging limit).
* **Over-Paneling Ratio:** $\frac{400\,\text{W}}{288\,\text{W}} \approx 138\%$.
  * **Engineering Rationale:** In the Dorset, Ontario climate, overcast and shoulder hours are common. An over-paneled 400W array delivers earlier wake-up voltages and higher harvest yields in low-light and diffuse conditions, while the controller smoothly clips power at its 20A charging ceiling during peak summer noon hours.

---

## Battery Bank Chemistry Profiles

The Renogy Rover supports 4 standard battery types and a customizable user profile:

1. **LiFePO4 (Lithium Iron Phosphate):**
   * Nominal Voltage: `12.8V` (4S)
   * Boost Charging: `14.4V`
   * Overvoltage Disconnect: `14.8V`
   * Low Voltage Warning: `12.0V`
   * Low Voltage Cutoff: `11.1V`
   * *Cold Weather Protection:* Charging must be inhibited below $0^\circ\text{C}$ ($32^\circ\text{F}$) to prevent lithium plating.
2. **Sealed (AGM):**
   * Boost Charging: `14.4V`, Float: `13.8V`
3. **Gel:**
   * Boost Charging: `14.2V`, Float: `13.8V`
4. **Flooded:**
   * Boost Charging: `14.6V`, Float: `13.8V`, Equalize: `14.8V`
