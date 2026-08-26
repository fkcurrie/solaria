# [ISSUE-007] E2E Validation Suite & Web Bluetooth Interoperability

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: Automated E2E Testing & Integration (Go)

---

## 🎯 Description
Build an automated **End-to-End (E2E) Test Suite** that launches the Renogy BLE Emulator in a headless Linux environment and executes automated integration probes against Central clients (such as the Solaria Bridge, Chrome headless Web Bluetooth tests, and Python Gattlib clients) to verify 100% interoperability with real Renogy software.

---

## 📐 Technical Specifications

### 1. Test Harness Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                    AUTOMATED E2E HARNESS                    │
├──────────────────────────────┬──────────────────────────────┤
│  Virtual BLE Emulator (Go)   │  Central Client / Test Agent │
│  • Registers BlueZ GATT      │  • Web Bluetooth / Edge SDK  │
│  • Emulates 400W 2S2P Solar  │  • Queries 0x0100 (34 words) │
│  • Listens on FFD1 / FFF1    │  • Validates CRC & Chunking  │
└──────────────────────────────┴──────────────────────────────┘
                               ▲
                               │ Physical / Virtual HCI Loopback
                               ▼
```

### 2. Automated Test Matrix

1. **Discovery & GATT Enumeration Probe**:
   - Scans for `BT-TH-EMULATOR` over BLE.
   - Connects and discovers Service `0xFFD0` and `0xFFF0`.
   - Confirms characteristic `0xFFD1` has `WRITE` and `0xFFF1` has `NOTIFY`.
2. **Chunk Reassembly & Framing Probe**:
   - Subscribes to `0xFFF1`.
   - Sends standard 8-byte query `FF 03 01 00 00 22 10 1E`.
   - Reassembles the 4 incoming notification chunks (20B + 20B + 20B + 13B = 73B).
   - Validates CRC-16 checksum matches the payload.
3. **Telemetry Value Sanity Probes**:
   - Asserts Battery Voltage is between $10.0\text{V} - 15.0\text{V}$.
   - Asserts SOC is between $0\% - 100\%$.
   - Asserts PV Voltage is between $0.0\text{V} - 45.0\text{V}$.
   - Asserts Controller Model matches `RNG-CTRL-RVR20`.
4. **Command Execution & Write Register Probe**:
   - Writes `FF 06 01 0A 00 01 58 1E` to turn Load ON.
   - Confirms immediate ACK response frame.
   - Queries `0x0104` to verify Load Voltage matches battery terminal voltage.

---

## 🧪 Acceptance Criteria
- [ ] Running `go test -v ./renogy-bt-emulator/...` executes all register and framing unit tests.
- [ ] Running the E2E suite against an active HCI adapter passes all 4 integration probes.
- [ ] Zero packet dropouts during continuous 1-hour sustained polling test (1 poll every 2 seconds).

---

## 🔗 References
- [Solaria End-to-End System Architecture](../../docs/architecture.md)
- [Web Bluetooth Community Group Testing Guide](https://webbluetoothcg.github.io/web-bluetooth/)
