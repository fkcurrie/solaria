# 3-Tier Hardware Bluetooth Radio Watchdog

Embedded Linux Bluetooth chipsets (e.g., Broadcom BCM43438 on Raspberry Pi) occasionally experience hardware lockups or GATT handle exhaustion when communicating continuously with BLE peripheral modules.

---

## 🛠️ Progressive 3-Tier Reset Algorithm

When the SRE agent detects that no valid BLE frames have been decoded for $> 60\text{ seconds}$, it executes a **progressive 3-tier recovery sequence**:

```mermaid
graph TD
    STALL["BLE Telemetry Stalled (> 60s)"] --> T1["Tier 1: Soft Radio Unblock<br/>(rfkill unblock bluetooth)"]
    T1 -. "If still stalled (+15s)" .-> T2["Tier 2: HCI Controller Reset<br/>(hciconfig hci0 reset / systemctl restart bluetooth)"]
    T2 -. "If still stalled (+30s)" .-> T3["Tier 3: Process Re-Spawn & Clean Spool<br/>(kill -9 solaria-bridge && ./bin/solaria-bridge)"]
    T3 --> RECOVERED["Telemetry Resumed (Lag < 5s)"]
```

---

## 🔒 BlueZ Stack Configuration

Solaria configures optimal Bluetooth Low Energy timeouts in `/etc/bluetooth/main.conf`:
- `FastConnectable = true`
- `LEAutoConnect = true`
- `JustWorksRepairing = always`
