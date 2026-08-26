# 📡 Renogy BT-1 / BT-2 Hardware Bluetooth LE Emulator (Go)

> **Autonomous Virtual Hardware Emulator for Renogy Solar Charge Controllers (Rover/Wanderer) and Smart Lithium Iron Phosphate (LiFePO4) Batteries**

This standalone project provides a physical/virtual Bluetooth Low Energy (BLE) peripheral written in **Go**. It emulates genuine Renogy BT-1 and BT-2 RS-232/RS-485 Bluetooth adapters, exposing authentic GATT services, Modbus RTU telemetry framing, solar diurnal physics, battery electrochemistry curves, and dynamic multi-device partitioning.

📖 Read the [**Core Tenets & System Soul (SOUL.md)**](./SOUL.md) for this project's guiding philosophy, electrochemistry rules, and physics invariants.

---

## 📑 Issue Tracker & Implementation Roadmap

| Issue | Title | Scope | Status |
| :--- | :--- | :--- | :--- |
| [**#54 EPIC**](https://github.com/fkcurrie/solaria/issues/54) | [**Master Epic: Standalone Renogy BT-1/BT-2 BLE Emulator in Go**](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md) | Full Architecture & Coordination | 📋 Planned |
| [**#55 ISSUE-001**](https://github.com/fkcurrie/solaria/issues/55) | [**BLE GATT Server Stack & D-Bus / BlueZ Peripheral Layer**](./ISSUE-001-Bluetooth-LE-GATT-Server-Stack.md) | Linux BlueZ GATT Server & Advertising | 📋 Planned |
| [**#56 ISSUE-002**](https://github.com/fkcurrie/solaria/issues/56) | [**Renogy Modbus RTU Protocol Engine & Register Map**](./ISSUE-002-Renogy-Modbus-RTU-Protocol-Engine.md) | Registers `0x0100`-`0x0122`, CRC-16 & 20B Chunking | 📋 Planned |
| [**#57 ISSUE-003**](https://github.com/fkcurrie/solaria/issues/57) | [**Solar Diurnal Physics, Irradiance & MPPT Tracking Engine**](./ISSUE-003-Solar-Physics-and-Diurnal-Cycle-Engine.md) | Solar elevation, 400W 2S2P curve & temperature derating | 📋 Planned |
| [**#58 ISSUE-004**](https://github.com/fkcurrie/solaria/issues/58) | [**LiFePO4 Electrochemistry, OCV Curve & Thermal Safety Model**](./ISSUE-004-Battery-Electrochemistry-and-Thermal-Model.md) | 12V 170Ah curves, sub-zero lockout & LVD cutoffs | 📋 Planned |
| [**#59 ISSUE-005**](https://github.com/fkcurrie/solaria/issues/59) | [**Multi-Device Bluetooth Partitioning (Dual Controller/Battery)**](./ISSUE-005-Multi-Device-Partitioning-and-Dual-BT-Support.md) | Multi-advertising sets & dual-device emulation | 📋 Planned |
| [**#60 ISSUE-006**](https://github.com/fkcurrie/solaria/issues/60) | [**Chaos Testing & Fault Injection Control Plane (CLI / REST)**](./ISSUE-006-Chaos-and-Anomaly-Injection-CLI.md) | Diode failure, shadow occlusion & hardware dropouts | 📋 Planned |
| [**#61 ISSUE-007**](https://github.com/fkcurrie/solaria/issues/61) | [**E2E Validation Suite & Web Bluetooth Interoperability**](./ISSUE-007-E2E-Validation-and-Web-Bluetooth-Interoperability.md) | Automated test suite against Solaria Web Bridge & Renogy App | 📋 Planned |

---

## 📚 Renogy Protocol & Hardware References

- **Renogy Rover Modbus Protocol Specification**: RS-232 / RS-485 RJ12 Modbus RTU specification covering register banks `0x000A` through `0x0122`.
- **Renogy BT-1 Bluetooth Module Manual**: Specifications for 2.4 GHz BLE GATT communication over Service `0xFFD0` (Rx/Tx) and `0xFFF0` (Notifications).
- **Renogy BT-2 Bluetooth Module Manual**: Multi-device RS-485 communication protocol for smart lithium batteries and parallel Rover MPPT controllers.
- **Renogy 12V 170Ah Smart LiFePO4 Battery Manual (`RBT170LFP12-BT`)**: BMS registers, cell voltage balance, low-temperature charge cutoffs (0°C / 32°F).
