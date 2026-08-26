# Hardware Bill of Materials (BOM)

Solaria is optimized for standard off-grid solar hardware, combined with low-cost edge computing.

---

## 🛠️ Complete Hardware Components

| Component | Model / Specs | Purpose | Approximate Cost |
| :--- | :--- | :--- | :--- |
| **MPPT Charge Controller** | [Renogy Rover 20A MPPT (PG-20CC)](https://www.renogy.com) | 12V/24V Auto-sensing, up to 100V PV input, Modbus RS232 RJ12 port | ~$90 USD |
| **Bluetooth Module** | Renogy BT-1 (RS232) | BLE 4.2 communications adapter (Service `0xFFD0`) | ~$35 USD |
| **Solar Array** | 4x 100W Monocrystalline Panels (2S2P) | Total 400W STC rating, ~36V Vmp, ~11A Imp | ~$300 USD |
| **Battery Bank** | 12V 170Ah LiFePO4 (Lithium Iron Phosphate) | 2,176 Wh nominal capacity, built-in BMS | ~$450 USD |
| **Edge Computer** | Raspberry Pi 3B+ / 4B / 5 (or x86 thin client) | Runs `solaria-bridge` and `solaria-sre-agent` | ~$35 - $60 USD |
| **MicroSD Card** | 32GB/64GB SanDisk High Endurance (A2) | High-write cycle storage for OS & volatile spooling | ~$15 USD |
| **DC Breakers & Fuses** | 30A DC PV Breaker, 40A Battery Fuse | Overcurrent and lightning protection | ~$30 USD |

---

## 📐 Electrical Specifications

### 400W 2S2P Solar Array
- **Series-Parallel Layout:** Two strings in parallel, each string consisting of two 100W panels in series.
- **Max Power Voltage ($V_{\text{mp}}$):** $\approx 36.0\text{V} - 38.0\text{V}$
- **Open Circuit Voltage ($V_{\text{oc}}$):** $\approx 44.0\text{V} - 46.0\text{V}$ (Safely below Rover 100V max limit)
- **Max Power Current ($I_{\text{mp}}$):** $\approx 10.5\text{A} - 11.2\text{A}$
- **Short Circuit Current ($I_{\text{sc}}$):** $\approx 11.5\text{A} - 12.0\text{A}$

### 12V 170Ah LiFePO4 Battery Bank
- **Nominal Energy Capacity:** $12.8\text{V} \times 170\text{Ah} = 2,176\text{ Wh}$
- **Usable Energy (80% DoD):** $1,740\text{ Wh}$
- **Continuous Max Discharge Current:** $100\text{A}$ ($1,280\text{W}$)
- **Max Recommended Charge Current:** $30\text{A}$ ($0.2\text{C}$ recommended for cell longevity)
