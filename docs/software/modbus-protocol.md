# Renogy BLE Modbus RTU Protocol

The **Renogy BT-1** communication module wraps standard Modbus RTU binary packets inside Bluetooth Low Energy (BLE) GATT characteristics.

---

## 📡 BLE GATT Service & Characteristics

- **Primary Service UUID:** `0000ffd0-0000-1000-8000-00805f9b34fb`
- **Read/Notify Characteristic:** `0000ffd1-0000-1000-8000-00805f9b34fb` (Properties: `Notify`, `Write Without Response`)

---

## 📑 Modbus RTU Register Mapping (Renogy Rover)

Solaria issues standard Modbus Function Code `0x03` (Read Holding Registers) to request telemetry blocks from controller address `0x01`:

```text
Request Frame: [0x01, 0x03, 0x01, 0x00, 0x00, 0x22, 0xC4, 0x2F]
```

| Register (Hex) | Offset | Data Description | Units / Scale |
| :--- | :--- | :--- | :--- |
| `0x0100` | 0 | Battery State of Charge (SOC) | `%` |
| `0x0101` | 2 | Battery Voltage | $0.1\text{ V}$ |
| `0x0102` | 4 | Charging Current to Battery | $0.01\text{ A}$ |
| `0x0103` | 6 | Controller Temp & Battery Temp | Byte High / Byte Low ($^\circ\text{C} - 128$) |
| `0x0107` | 14 | Solar Panel Voltage ($V_{\text{pv}}$) | $0.1\text{ V}$ |
| `0x0108` | 16 | Solar Panel Current ($I_{\text{pv}}$) | $0.01\text{ A}$ |
| `0x0109` | 18 | Charging Power ($P_{\text{pv}}$) | $1\text{ W}$ |
| `0x0113` | 38 | Daily Solar Generation (Yield) | $1\text{ Wh}$ |
| `0x0120` | 64 | Controller Charging State Code | Enum (`0`=Deactivated, `2`=MPPT, `4`=Boost, `5`=Float) |

---

## 🛡️ Modbus CRC-16 Calculation

Every frame is validated using standard Modbus CRC-16 (polynomial `0xA001`, initial value `0xFFFF`). Corrupted frames received over noisy wireless channels are instantly discarded before parsing.
