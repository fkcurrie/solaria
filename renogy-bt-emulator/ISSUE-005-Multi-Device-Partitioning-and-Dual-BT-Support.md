# [ISSUE-005] Multi-Device Bluetooth Partitioning (Dual Controller & Battery)

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: BLE Multi-Peripheral Architecture (Go)

---

## 🎯 Description
Implement **Multi-Device Bluetooth Partitioning** to allow a single host machine to emulate multiple Renogy Bluetooth devices simultaneously. This enables simulating complex multi-node systems (e.g., Device 1: Renogy Rover 20A MPPT, Device 2: Renogy 170Ah Smart LiFePO4 Battery with BMS telemetry, or Device 3: Renogy Inverter).

---

## 📐 Technical Specifications

### 1. Partitioning Approaches

#### Approach A: Multi-Advertising Sets on a Single BLE 5.0 Adapter (`hci0`)
- Utilizing Bluetooth 5.0 LE Extended Advertising & Multiple Advertising Sets (BlueZ 5.50+).
- The Linux host registers two distinct advertising instances with unique Random Static BLE MAC addresses:
  - **Instance 1 (BT-1 Controller)**: `BT-TH-ROVER20` (UUIDs `0xFFD0`, `0xFFF0`, Device ID `0x01`).
  - **Instance 2 (BT-2 Smart Battery)**: `BT-TH-BATT170` (UUIDs `0xFFD0`, `0xFFF0`, Device ID `0xF7`).

#### Approach B: Multi-HCI Adapter Binding (`hci0` + `hci1`)
- When two physical USB BLE dongles are plugged into the host (e.g., `hci0` and `hci1`):
  - Go daemon spawns two independent D-Bus connection worker pools, binding one device emulator per physical HCI index.
  - Zero radio time-slicing contention and maximum RF throughput.

### 2. Device Profile Multiplexing
```
┌─────────────────────────────────────────────────────────────┐
│                 GO MULTI-DEVICE CONTROLLER                  │
├──────────────────────────────┬──────────────────────────────┤
│  Device Node 1: ROVER 20A    │  Device Node 2: BATT 170AH   │
│  • Modbus Address: 0x01      │  • Modbus Address: 0xF7      │
│  • BLE Name: BT-TH-ROVER20   │  • BLE Name: BT-TH-BATT170   │
│  • Telemetry: PV & Charge    │  • Telemetry: 4S Cell Volts, │
│  • Registers: 0x0100-0x0122  │    Cycle Count, BMS Status   │
└──────────────────────────────┴──────────────────────────────┘
```

### 3. Battery BMS Register Emulation (Renogy Smart Battery Protocol)
When queried at device address `0xF7` (Smart Battery):
- Registers `0x5000` - `0x5003`: Individual 4S Cell Voltages ($V_1, V_2, V_3, V_4$).
- Register `0x5004`: Battery Internal Temperature Sensors (Probe 1 & Probe 2).
- Register `0x5008`: Battery Cycle Count (e.g., 142 cycles).
- Register `0x500A`: BMS Alarm & Protection Bitmask (Over-voltage, Under-voltage, High-temp, Low-temp charge inhibit).

---

## 🧪 Acceptance Criteria
- [ ] Central device scanning discovers both `BT-TH-ROVER20` and `BT-TH-BATT170` as separate, concurrent peripherals.
- [ ] Client can connect to `BT-TH-ROVER20` to stream solar telemetry while simultaneously connecting to `BT-TH-BATT170` to stream cell-level BMS telemetry.
- [ ] Modbus commands intended for Node 1 do not cross-talk or interfere with Node 2.

---

## 🔗 References
- [Bluetooth Core Specification v5.3: LE Extended Advertising](https://www.bluetooth.com/specifications/specs/core-specification-5-3/)
- [Renogy Smart Lithium Battery RS485 / Bluetooth Protocol](https://www.renogy.com)
