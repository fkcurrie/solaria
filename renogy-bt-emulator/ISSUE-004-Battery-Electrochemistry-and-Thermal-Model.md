# [ISSUE-004] LiFePO4 Electrochemistry, OCV Curve & Thermal Safety Model

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: Battery Chemistry & Thermal Simulation (Go)

---

## 🎯 Description
Implement a high-fidelity **12V 170Ah Lithium Iron Phosphate (LiFePO4)** battery simulation model that emulates open-circuit voltage curves, internal resistance, multi-stage MPPT charging (Bulk $\to$ Absorption/Boost $\to$ Float), sub-zero charging cutoffs, and low-voltage disconnects.

---

## 📐 Technical Specifications

### 1. 12V 170Ah LiFePO4 Chemistry Characteristics
- **Nominal Capacity**: $170\text{ Ah} = 2,176\text{ Wh}$ (4S prismatic cells).
- **Voltage Zones & Open Circuit Voltage (OCV)**:
  - **Overcharge Warning**: $> 14.6\text{ V}$ ($3.65\text{V/cell}$)
  - **Boost / Absorption Threshold**: $14.4\text{ V}$ ($3.60\text{V/cell}$)
  - **Float Voltage**: $13.6\text{ V}$ ($3.40\text{V/cell}$)
  - **Upper Working Plateau (90% - 99% SOC)**: $13.4\text{ V} - 13.6\text{ V}$
  - **Mid-Discharge Plateau (20% - 90% SOC)**: $13.1\text{ V} - 13.3\text{ V}$ (Extremely flat curve)
  - **Low-Knee Knee Point (< 20% SOC)**: $12.5\text{ V} \to 12.0\text{ V}$ (Steep drop-off)
  - **Low-Voltage Disconnect (LVD)**: $10.6\text{ V} - 11.0\text{ V}$

### 2. State of Charge (SOC) Coulomb Counting Model
$$SOC(t) = SOC(t_0) + \frac{1}{C_{\text{rated}}} \int \left( I_{\text{charge}}(t) \times \eta_{\text{coulombic}} - I_{\text{discharge}}(t) \right) dt$$
- $\eta_{\text{coulombic}} = 99.0\%$ for LiFePO4.
- Internal resistance $R_{\text{int}} \approx 12\text{ m}\Omega$, causing dynamic terminal voltage sag under load and rise under high charging current:
  $$V_{\text{terminal}} = V_{\text{ocv}}(SOC) + I \cdot R_{\text{int}}$$

### 3. Sub-Zero Low-Temperature Lithium Inhibit Safety
- LiFePO4 batteries suffer permanent metallic lithium plating if charged below $0^\circ\text{C}$ ($32^\circ\text{F}$).
- When emulated Battery Temperature $\le 0^\circ\text{C}$:
  - Controller Charging State transitions to `0x00` (Deactivated).
  - Register `0x0102` ($I_{\text{charge}}$) drops to $0.00\text{A}$.
  - Bit 7 of Fault Register `0x0121` is set (`0x0080` Low Temperature Charge Prohibited).

---

## 🧪 Acceptance Criteria
- [ ] During bulk charging, battery terminal voltage smoothly ramps from 13.2V up to 14.4V.
- [ ] Reaching 14.4V triggers the Absorption stage, holding 14.4V while current tapers down to < 2A before entering Float (13.6V).
- [ ] Lowering temperature to -1°C instantly forces charging current to 0.00A and asserts the low-temperature fault bitmask.
- [ ] Under 5A load with no solar generation, battery SOC decreases according to Amp-hour integration.

---

## 🔗 References
- [Renogy 12V 170Ah Smart LiFePO4 Battery Manual (RBT170LFP12-BT)](https://www.renogy.com)
- [Journal of The Electrochemical Society: Degradation Mechanism of LiFePO4 at Sub-Zero Temperatures](https://iopscience.iop.org)
