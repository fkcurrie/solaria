# Multi-Distro Compatibility

Solaria is rigorously tested against multiple Linux distributions using automated container CI matrix tests.

---

## 🐧 Supported Operating Systems

| Operating System | Architecture | Package Manager | Bluetooth Stack | Status |
| :--- | :--- | :--- | :--- | :--- |
| **Raspberry Pi OS (Bookworm Lite)** | `arm64`, `armv7` | `apt` | BlueZ 5.66+ | 🟢 **Tier 1 (Recommended)** |
| **Debian 12 Bookworm** | `x86_64`, `arm64` | `apt` | BlueZ 5.66+ | 🟢 **Tier 1 (Fully Supported)** |
| **Ubuntu Server 24.04 LTS** | `x86_64`, `arm64` | `apt` | BlueZ 5.72+ | 🟢 **Tier 1 (Fully Supported)** |
| **Alpine Linux (Container)** | `x86_64`, `arm64` | `apk` | Minimal | 🟡 **Tier 2 (Cloud Run Container)** |
| **macOS (Darwin)** | `arm64`, `x86_64` | `brew` | Native CoreBluetooth | 🟡 **Tier 2 (Local Dev / UI)** |

---

## 🧪 Automated CI Matrix Verification

Every pull request and commit to `main` executes continuous regression tests across:
1. `debian:bookworm-slim` (x86_64)
2. `ubuntu:24.04` (x86_64)
3. `debian:bookworm-slim` (ARM64 emulation via QEMU)

The CI matrix explicitly asserts that:
- Core dependencies (`bluez`, `dbus`, `curl`, `jq`) resolve cleanly without package breakage.
- `/etc/systemd/journald.conf.d/solaria-volatile.conf` is correctly provisioned with `Storage=volatile`.
- `/etc/sysctl.d/99-solaria-resilience.conf` contains correct writeback parameters (`vm.dirty_writeback_centisecs = 1500`).
