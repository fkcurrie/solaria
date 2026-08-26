# [ISSUE-006] Chaos Testing & Fault Injection Control Plane (CLI / REST)

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: Chaos Control Plane & Fault Injection (Go)

---

## 🎯 Description
Implement an external **Chaos Testing & Fault Injection Control Plane** accessible via an interactive CLI and an HTTP REST API. This subsystem enables developers and automated SRE test runners to inject physical hardware malfunctions, environmental anomalies, and Bluetooth connection dropouts into the live emulator without modifying source code.

---

## 📐 Technical Specifications

### 1. HTTP REST Chaos Endpoints (`:8999`)
- `POST /api/v1/chaos/fault`: Injects a specific fault condition.
- `POST /api/v1/chaos/reset`: Restores the emulator to healthy nominal operating conditions.
- `GET /api/v1/chaos/status`: Returns the active fault injection matrix.

### 2. Supported Fault Scenarios

| Fault Identifier | Simulated Condition | Physics & Register Impact |
| :--- | :--- | :--- |
| `DIODE_FAILURE` | Blown bypass diode in 2S2P String 1 | String voltage drops from ~36V to ~18V ($V_{\text{mp}}$ halving), array power cuts by ~50%. |
| `TREE_SHADOW` | Tree canopy occlusion (11:00 - 13:00) | PV power abruptly drops from ~350W down to ~60W diffuse yield with erratic voltage dips. |
| `SUBZERO_FREEZE` | Ambient battery temperature drops to -5°C | Register `0x0103` low byte sets to `0xFB` (-5°C), $I_{\text{charge}}$ drops to 0.00A, subzero alarm bit sets. |
| `LVD_CUTOFF` | Excessive cottage inverter load depletes battery | Battery voltage collapses below 10.8V, load circuit disconnects (`0x0104` = 0V), controller fault bitmask asserts LVD. |
| `PROBE_DISCONNECTED` | RTS battery temperature sensor disconnected | Controller falls back to default 25°C baseline or asserts Open Circuit fault code. |
| `BLE_SILENT_DROP` | Bluetooth RF jamming or module freeze | Emulator stops sending notification packets on `0xFFF1` to test central client watchdog reconnect logic. |
| `CRC_CORRUPTION` | RS-232 UART electrical noise | Inverts the low CRC byte on 10% of packets to test central parser error recovery. |

### 3. Interactive CLI Interface
Provide a Go CLI command:
```bash
go run ./renogy-bt-emulator/cmd/chaos --inject DIODE_FAILURE --duration 10m
go run ./renogy-bt-emulator/cmd/chaos --set-temp -2
go run ./renogy-bt-emulator/cmd/chaos --set-irradiance 800
go run ./renogy-bt-emulator/cmd/chaos --reset
```

---

## 🧪 Acceptance Criteria
- [ ] Injecting `DIODE_FAILURE` immediately halves PV voltage and power in telemetry packets received by connected Web Bluetooth clients.
- [ ] Injecting `SUBZERO_FREEZE` trips the sub-zero lithium safety alarm on the Solaria dashboard in real-time.
- [ ] Calling `POST /api/v1/chaos/reset` immediately restores healthy 2S2P baseline telemetry.

---

## 🔗 References
- [Chaos Engineering: System Resiliency in IoT Systems](https://principlesofchaos.org/)
