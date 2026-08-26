# Edge BLE Bridge Daemon (`solaria-bridge`)

The **Solaria Bridge Daemon** is a lightweight, high-concurrency Go service that runs continuously on the edge computer (e.g. Raspberry Pi).

---

## 🚀 Key Responsibilities

1. **Web Bluetooth & WebSocket Gateway:** Provides a clean HTTP/WebSocket interface (`:8080`) allowing browsers to connect directly to the BT-1 module using Web Bluetooth API.
2. **Binary Frame Decoding:** Parses Modbus RTU telemetry packets and validates CRC-16 checksums.
3. **Solar Physics & Health Assessment:** Evaluates LiFePO4 cold charging rules, string symmetry, and efficiency in real time.
4. **Live ASCII Console:** Prints a real-time terminal readout of battery, array, and atmospheric conditions.
5. **Persistent Disk Spooler:** Implements zero-data-loss FIFO buffering to `spool/` when cloud endpoints are unreachable.

---

## 🖥️ Live Terminal Output Example

```text
[14:10:45.423 ☀️ RENOGY LIVE TELEMETRY | DIFFUSE_OVERCAST]
  ├─ Array (400W 2S2P): 300 W (36.5V @ 8.21A) | Util: 75.0% | Peak: 385W
  ├─ Battery:           13.5 V | SOC: 80% | Charge: 20.00A
  ├─ State:             MPPT Charging | Health: NORMAL
  ├─ Dorset Wx:         15.7°C | Clouds: 72% | Rad: 43.0 W/m² (PR: 264.5%)
  ├─ Temps:             Controller 25°C | Battery 20°C
  └─ Daily Yield:       1450 Wh | Lifetime: 412 kWh
```

---

## 📡 HTTP Endpoints (`:8080`)

| Method | Route | Description | Auth Required |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | Web Bluetooth pairing UI & oscilloscope | No |
| `GET` | `/api/v1/health` | Service uptime and packet statistics | No |
| `GET` | `/api/v1/status` | Current live telemetry frame and spool backlog | No |
| `GET` | `/api/v1/diagnostics` | Goroutines, memory allocation, and spool metrics | No |
| `POST` | `/api/v1/reload` | Dynamic token and endpoint configuration reload | Yes (`Bearer Token`) |
