# [ISSUE-001] BLE GATT Server Stack & D-Bus / BlueZ Peripheral Layer

**Parent Epic**: [EPIC-001](./EPIC-001-Renogy-BT1-BT2-Hardware-Emulator.md)  
**Status**: 📋 Planned  
**Component**: Bluetooth LE Transport Layer (Go)

---

## 🎯 Description
Implement the core Bluetooth Low Energy (BLE) GATT peripheral stack in Go. The emulator must register an LE advertisement packet with the local BlueZ daemon via D-Bus and expose the exact Service UUIDs and Characteristic UUIDs expected by Renogy central clients (such as Web Bluetooth in Chrome and the Renogy DC Home app).

---

## 📐 Technical Specifications

### 1. BLE Advertising Profile
- **Advertising Flags**: `0x06` (LE General Discoverable Mode, BR/EDR Not Supported).
- **Local Name**: `BT-TH-EMULATOR` (or user-configurable prefix `BT-TH-XXXXXXXX`).
- **Advertised Service UUIDs**:
  - `0000ffd0-0000-1000-8000-00805f9b34fb` (`0xFFD0` - Primary Renogy Serial Service)
  - `0000fff0-0000-1000-8000-00805f9b34fb` (`0xFFF0` - Renogy Telemetry Service)

### 2. GATT Hierarchy & Services
```
Primary Service: 0000ffd0-0000-1000-8000-00805f9b34fb (0xFFD0)
 └─ Characteristic: 0000ffd1-0000-1000-8000-00805f9b34fb (0xFFD1)
      Properties: Write, Write Without Response, Read
      Description: Receives 8-byte Modbus RTU query frames from Central.

Primary Service: 0000fff0-0000-1000-8000-00805f9b34fb (0xFFF0)
 └─ Characteristic: 0000fff1-0000-1000-8000-00805f9b34fb (0xFFF1)
      Properties: Notify, Read
      Descriptors: Client Characteristic Configuration (0x2902)
      Description: Transmits Modbus response chunks (20 bytes max) to Central.
```

### 3. Go Implementation Architecture
- Utilize D-Bus object paths (`/org/bluez/example/service0`, `/org/bluez/example/char0`) or high-level Go bindings (`github.com/muka/go-bluetooth`).
- Implement `org.bluez.GattCharacteristic1` interface:
  - `WriteValue(value []byte, options map[string]interface{})`: Passes incoming raw Modbus query to the Modbus Dispatcher ([ISSUE-002](./ISSUE-002-Renogy-Modbus-RTU-Protocol-Engine.md)).
  - `StartNotify()` / `StopNotify()`: Manages client subscription state for the telemetry stream.

---

## 🧪 Acceptance Criteria
- [ ] `bluetoothctl scan on` discovers the emulator device broadcasting name `BT-TH-EMULATOR`.
- [ ] Google Chrome Web Bluetooth API (`navigator.bluetooth.requestDevice`) can discover, connect, and negotiate MTU with the peripheral.
- [ ] Subscribing to `0xFFF1` notifications successfully enables the GATT Client Characteristic Configuration Descriptor (CCCD).
- [ ] Writing 8 bytes to `0xFFD1` triggers the internal dispatch callback without D-Bus timeouts.
- [ ] Clean shutdown handling releases D-Bus paths and stops advertising on `SIGINT`/`SIGTERM`.

---

## 🔗 References
- [Linux BlueZ GATT D-Bus API](https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc/gatt-api.txt)
- [Linux BlueZ Advertising D-Bus API](https://git.kernel.org/pub/scm/bluetooth/bluez.git/tree/doc/advertising-api.txt)
