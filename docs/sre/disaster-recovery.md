# Disaster Recovery & Zero-Data-Loss Spooling

Remote cottage environments suffer frequent power losses from blown breakers, winter ice storms, or battery low-voltage disconnects.

---

## ⚡ Disaster Recovery & Fault Tolerance Matrix

| Failure Mode | Impact | Solaria Autonomous Mitigation |
| :--- | :--- | :--- |
| **Abrupt Power Loss** | Pi loses 5V power immediately | Dirty writeback kernel tuning (`vm.dirty_writeback_centisecs = 1500`) ensures journal consistency. On reboot, systemd automatically relaunches services. |
| **WiFi / LTE Outage** | Internet down for days | `DiskSpooler` atomically buffers all 1-second frames to `spool/queue.jsonl`. Backlog drains automatically once online. |
| **Spool Line Corruption** | Truncated record in spool file | Spool parser quarantines corrupted lines to `spool/quarantine.log` and continues processing remaining valid records. |
| **Cloud Run Ingestion 503** | Cloud server maintenance | Exponential backoff with random jitter prevents retry storms. |
| **Frozen BLE Radio** | BT-1 stops advertising | SRE supervisor invokes 3-tier radio reset (`rfkill`, `hciconfig hci0 reset`). |

---

## 🗄️ Spool Drainer Mechanics

The spool drainer operates as a background goroutine within `solaria-bridge`:

```go
func (s *DiskSpooler) DrainBatch(batchSize int, uploader Uploader) (int, error) {
    // 1. Reads up to batchSize records from queue.jsonl
    // 2. Posts payload to POST /api/v1/ingest with Bearer Token
    // 3. Atomically commits read offset or truncates processed lines
}
```
