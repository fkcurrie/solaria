# Resilience & Troubleshooting

Solaria incorporates automated recovery mechanisms for remote, unattended deployments.

## Supervisor Architecture

1. **Watchdog Timer:** If no valid Modbus frame arrives for 30 seconds, the bridge flags an outage, logs downtime, and executes host BlueZ recovery (`bluetoothctl power off/on`).
2. **Web Bluetooth Auto-Reconnect:** The dashboard acquires a browser Screen WakeLock and runs an infinite reconnection loop against the BT-1 module (`BT-TH-66F984D6`).
3. **Chunk Buffer Reset:** Modbus frame assembly buffers reset automatically on connection changes to prevent packet corruption.

## Outage Metrics & Logging

The supervisor tracks uptime and records interruptions in a ring buffer exposed on `http://localhost:8080`:

* **System Availability (%):** Operational uptime divided by total session time.
* **Total Outages:** Count of communication drop events.
* **Total Downtime:** Accumulated offline seconds.
* **Audit Log:** Timestamped log of disconnects, recovery actions, and durations.

## Troubleshooting

### GATT Disconnection / Device Out of Range

* **Cause:** BLE radio interference or host adapter freeze.
* **Automated Action:** Watchdog automatically cycles host Bluetooth power after 30 seconds of silence.
* **Manual Reset:**

  ```bash
  sudo systemctl restart bluetooth
  bluetoothctl power off && sleep 1 && bluetoothctl power on
  ```

### Missing Irradiance Data

* **Cause:** Outbound HTTP failure to Open-Meteo API.
* **Check:** Verify network connectivity to `https://api.open-meteo.com`.

### Ingestion Authorization Failures

* **Cause:** Mismatched API tokens.
* **Check:** Verify `SOLARIA_API_TOKEN` matches in both `.env` and Cloud Run environment settings.
