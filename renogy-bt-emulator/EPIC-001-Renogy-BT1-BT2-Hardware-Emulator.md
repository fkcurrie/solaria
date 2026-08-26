# [EPIC-001] Standalone Renogy BT-1 / BT-2 Bluetooth LE Hardware Emulator (Go)

## 📌 Executive Summary
Build a standalone, high-fidelity **Bluetooth Low Energy (BLE) Hardware Emulator** in **Go** that mimics genuine Renogy BT-1 and BT-2 communication modules connected to Renogy Rover/Wanderer charge controllers and smart LiFePO4 batteries. The emulator operates over the real physical 2.4 GHz Bluetooth protocol (via Linux BlueZ / D-Bus or tinygo-bluetooth), allowing any central device (Solaria Web Bluetooth UI, Google Chrome, Python edge scripts, or the official Renogy DC Home mobile app) to discover, pair, query, and program the virtual solar system as if physical hardware were connected.

---

## 🎯 Objectives & User Stories
1. **As a Solar IoT Engineer**, I want an over-the-air BLE peripheral that advertises genuine Renogy GATT services (`0xFFD0`, `0xFFF0`) so I can test Web Bluetooth pairing, chunk reassembly, and Modbus polling without running down to the off-grid battery bank.
2. **As an SRE / QA Tester**, I want the emulator to simulate realistic diurnal solar cycles (sunrise, peak solar noon, clouds, sunset) and realistic LiFePO4 battery charge/discharge physics (Bulk, Absorption, Float, Low-Knee) based on solar angles and electrical models.
3. **As a System Architect**, I want to partition a single Bluetooth adapter to emulate multiple Renogy devices simultaneously (e.g., a 20A Rover MPPT on Device 1 and a 170Ah Smart Lithium Battery on Device 2).
4. **As a Safety & Chaos Engineer**, I want an external control plane (CLI/REST) to inject faults on demand (such as sub-zero 0°C lithium lockout, bypass diode failures, shadow occlusion, and dropped BLE packets) to validate monitoring alarms and fail-safes.

---

## 🏗️ High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    RENOGY BT-1 / BT-2 GO EMULATOR                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  [CLI / REST Chaos API] ───► [Fault & Anomaly Injector]                    │
│                                    │ (diode fault, subzero lockout, drop)   │
│                                    ▼                                        │
│  [Solar Physics Engine] ────► [State & Modbus Store] ◄──── [LiFePO4 Model]  │
│  (NOAA Sun Math, Incline,      (Registers 0x0100-0x0122,   (12V 170Ah OCV,  │
│   400W 2S2P String Curves)      0x000A-0x001A specs)        Thermal state)  │
│                                    │                                        │
│                                    ▼                                        │
│                     [Modbus RTU Dispatcher Engine]                          │
│                     - Parses 0x03 Read & 0x06 Write                         │
│                     - Verifies Modbus CRC-16                                │
│                     - 20-Byte ATT Notification Slicer                       │
│                                    │                                        │
│                                    ▼                                        │
│                     [Linux BlueZ GATT Server Stack]                         │
│                     - Multi-Advertising Sets (BT-1 + BT-2)                  │
│                     - Service 0xFFD0 / Characteristic 0xFFD1 (Rx/Tx)        │
│                     - Service 0xFFF0 / Characteristic 0xFFF1 (Notify)       │
│                                    │                                        │
└────────────────────────────────────┼────────────────────────────────────────┘
                                     │ 2.4 GHz BLE Physical Broadcast
                                     ▼
         ┌───────────────────────────────────────────────────────┐
         │                  Central Clients                      │
         │  • Solaria Web Bluetooth Dashboard (Chrome)           │
         │  • Solaria Edge Daemon (cmd/edge-agent)               │
         │  • Official Renogy DC Home App (iOS / Android)        │
         └───────────────────────────────────────────────────────┘
```

---

## 📋 Breakdown of Sub-Issues

| Issue ID | Title | Key Deliverables |
| :--- | :--- | :--- |
| [**#55 ISSUE-001**](https://github.com/fkcurrie/solaria/issues/55) | [**BLE GATT Server Stack & D-Bus / BlueZ Peripheral Layer**](./ISSUE-001-Bluetooth-LE-GATT-Server-Stack.md) | D-Bus BlueZ GATT peripheral implementation, LE advertisement registration, connection negotiation. |
| [**#56 ISSUE-002**](https://github.com/fkcurrie/solaria/issues/56) | [**Renogy Modbus RTU Protocol Engine & Register Map**](./ISSUE-002-Renogy-Modbus-RTU-Protocol-Engine.md) | Complete Renogy 16-bit register map (`0x000A`-`0x0122`), CRC-16 generator, 20-byte ATT chunk fragmentation engine with UART delay. |
| [**#57 ISSUE-003**](https://github.com/fkcurrie/solaria/issues/57) | [**Solar Diurnal Physics, Irradiance & MPPT Tracking Engine**](./ISSUE-003-Solar-Physics-and-Diurnal-Cycle-Engine.md) | Real-time NOAA solar position algorithm, 400W 2S2P V-I curve modeling, MPPT dynamic efficiency tracking, diurnal telemetry generation. |
| [**#58 ISSUE-004**](https://github.com/fkcurrie/solaria/issues/58) | [**LiFePO4 Electrochemistry, OCV Curve & Thermal Safety Model**](./ISSUE-004-Battery-Electrochemistry-and-Thermal-Model.md) | 12V 170Ah LiFePO4 open-circuit voltage discharge knee curve, Coulomb counting SOC, 0°C sub-zero charge lockout, over-voltage/under-voltage disconnect. |
| [**#59 ISSUE-005**](https://github.com/fkcurrie/solaria/issues/59) | [**Multi-Device Bluetooth Partitioning (Dual Controller/Battery)**](./ISSUE-005-Multi-Device-Partitioning-and-Dual-BT-Support.md) | Multi-advertising sets on single BLE 5.0 controller or dual `hci0`/`hci1` adapters for simultaneous Rover MPPT + Smart Battery emulation. |
| [**#60 ISSUE-006**](https://github.com/fkcurrie/solaria/issues/60) | [**Chaos Testing & Fault Injection Control Plane (CLI / REST)**](./ISSUE-006-Chaos-and-Anomaly-Injection-CLI.md) | Interactive CLI and REST API to trigger hardware anomalies (shaded panels, blown bypass diode, overheating, BLE connection drops). |
| [**#61 ISSUE-007**](https://github.com/fkcurrie/solaria/issues/61) | [**E2E Validation Suite & Web Bluetooth Interoperability**](./ISSUE-007-E2E-Validation-and-Web-Bluetooth-Interoperability.md) | Automated end-to-end integration test suite validating zero regressions against Web Bluetooth browser clients and edge gateways. |

---

## 🛠️ Technology Stack & Requirements
- **Language**: Go 1.22+
- **Bluetooth Stack**: `github.com/muka/go-bluetooth` or `tinygo.org/x/bluetooth` with Linux BlueZ D-Bus bindings (`org.bluez.GattManager1`, `org.bluez.LEAdvertisingManager1`).
- **Dependencies**: Native Go standard library for concurrency (`sync`, `atomic`, `context`), Modbus math, and HTTP REST interface.
- **Hardware Requirements**: Any Linux machine (PC, laptop, Raspberry Pi) equipped with a Bluetooth 4.2/5.0+ adapter (e.g. Intel AX200/AX210, CSR8510 USB dongle).

---

## 🔗 External Documentation & References
- [Renogy Rover Charge Controller Modbus Protocol v1.7](https://www.renogy.com)
- [Linux BlueZ D-Bus GATT API Documentation](https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc/gatt-api.txt)
- [Linux BlueZ D-Bus Advertising API Documentation](https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc/advertising-api.txt)
- [Web Bluetooth Community Group Specification](https://webbluetoothcg.github.io/web-bluetooth/)
