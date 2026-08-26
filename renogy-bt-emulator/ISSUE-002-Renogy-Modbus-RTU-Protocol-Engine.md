# [ISSUE-002] Renogy Modbus RTU Protocol Engine & Register Map

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: Modbus RTU Register Engine (Go)

---

## 🎯 Description
Implement the complete **Renogy Modbus RTU Protocol Engine** in Go. The engine receives raw query frames written to characteristic `0xFFD1`, validates the function codes and CRC-16 checksums, queries the internal memory register map, constructs the binary Modbus response frame, calculates CRC-16, and slices the response into 20-byte chunks transmitted over characteristic `0xFFF1` with realistic UART inter-chunk timing.

---

## 📐 Technical Specifications

### 1. Supported Function Codes
- **`0x03` (Read Holding Registers)**: Queries real-time solar, battery, load, and history telemetry.
- **`0x06` (Write Single Register)**: Executes controller commands (e.g., Manual Load Switching at `0x010A`, Battery Type selection at `0xE004`).

### 2. Renogy Rover Modbus Register Map
The memory store must maintain the following register offsets:

| Register Range | Size | Description | Units / Scale |
| :--- | :--- | :--- | :--- |
| `0x000A` - `0x0011` | 8 words | Controller Model String | ASCII (e.g. `RNG-CTRL-RVR20`) |
| `0x0014` - `0x0019` | 6 words | Controller Serial Number | Hex string |
| `0x0100` | 1 word | Battery State of Charge (SOC) | `0% - 100%` |
| `0x0101` | 1 word | Battery Voltage | `0.1 V` |
| `0x0102` | 1 word | Battery Charging Current | `0.01 A` |
| `0x0103` | 1 word | Controller Temp (High Byte) / Battery Temp (Low Byte) | `1 °C` (Sign-extended) |
| `0x0104` | 1 word | Load Voltage | `0.1 V` |
| `0x0105` | 1 word | Load Current | `0.01 A` |
| `0x0106` | 1 word | Load Power | `1 W` |
| `0x0107` | 1 word | Solar PV Voltage | `0.1 V` |
| `0x0108` | 1 word | Solar PV Current | `0.01 A` |
| `0x0109` | 1 word | Solar PV Power | `1 W` |
| `0x010B` - `0x010C` | 2 words | Daily Min / Max Battery Voltage | `0.1 V` |
| `0x010D` | 1 word | Daily Max Charging Current | `0.01 A` |
| `0x010F` | 1 word | Daily Max Solar Power | `1 W` |
| `0x0111` - `0x0112` | 2 words | Daily Charging / Discharging Amp-Hours | `1 Ah` |
| `0x0113` - `0x0114` | 2 words | Daily Generated / Consumed Watt-Hours | `1 Wh` |
| `0x0115` | 1 word | Operating Days | `1 day` |
| `0x011C` - `0x011D` | 2 words | Cumulative Generated Energy | `1 kWh` |
| `0x0120` | 1 word | Charging State Code | `0x00` (Deact), `0x02` (MPPT), `0x03` (EQ), `0x04` (Boost), `0x05` (Float), `0x06` (Current Limit) |
| `0x0121` - `0x0122` | 2 words | Controller Fault Bitmask | 32-bit bitfield |

### 3. Response Construction & 20-Byte ATT Chunking
- Standard query: `FF 03 01 00 00 22 10 1E` (Read 34 registers / 68 bytes starting at `0x0100`).
- Generated Response: `[0xFF, 0x03, 0x44, <68 data bytes>, <CRC_LOW>, <CRC_HIGH>]` (Total 73 bytes).
- **Chunking Pipeline**:
  - Slices 73 bytes into 4 packets:
    - Packet 1: 20 bytes (`bytes[0:20]`)
    - Packet 2: 20 bytes (`bytes[20:40]`)
    - Packet 3: 20 bytes (`bytes[40:60]`)
    - Packet 4: 13 bytes (`bytes[60:73]`)
  - Transmits packets over `0xFFF1` with a **15ms - 25ms delay** between notifications to simulate the Renogy BT-1 RS-232 UART bridge baud rate (9600 baud).

---

## 🧪 Acceptance Criteria
- [ ] Modbus CRC-16 algorithm produces exact checksums adhering to standard Modbus polynomial (`0xA001`).
- [ ] Querying `0x0100` (34 words) responds with a 73-byte RTU payload.
- [ ] Chunking delivers 4 consecutive GATT notifications without dropping frames.
- [ ] Writing to register `0x010A` toggles the emulated 12V DC load circuit and updates register `0x0106`.

---

## 🔗 References
- [Modbus Application Protocol Specification v1.1b3](https://modbus.org/docs/Modbus_Application_Protocol_V1_1b3.pdf)
- [Renogy Rover Modbus Protocol Manual](https://www.renogy.com)
