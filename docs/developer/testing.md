# End-to-End Test Suite & E2E Audit

Solaria includes unit tests, concurrency race condition detectors, and a 21-probe end-to-end integration verifier.

---

## 🧪 Running Automated Tests

### Run All Unit & Integration Tests
```bash
go test -v ./...
```

### Run with Concurrency Race Detector
```bash
go test -race ./...
```

### Run 21-Probe E2E System Audit
```bash
./bin/solaria-e2e-audit
```

The E2E audit tests 5 critical system layers:
1. **Bridge Layer:** Process health, uptime, Modbus frame rate, zero spool backlog.
2. **Cloud API Layer:** Batch ingestion, live stream freshness, ML prediction, NOAA sun times.
3. **Security Layer:** 401 Unauthorized rejection on unauthenticated mutations, Bearer token verification.
4. **Frontend PWA:** Service worker `/sw.js` cache, `manifest.json`, SVG vector assets, 10 dashboard tab DOM elements.
5. **Physics Safety:** LiFePO4 sub-zero charging inhibit invariants, 2S2P series string symmetry.
