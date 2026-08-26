# 1-Click Bootstrap Quickstart

The quickest way to install and configure Solaria on any Linux system (Raspberry Pi, Debian, Ubuntu) is using the automated bootstrap script.

---

## ⚡ Automated 1-Click Installation

Run the following command directly in your edge machine's terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/setup.sh | bash
```

### What the Installer Does Automatically
1. **Detects Target OS:** Checks if running on Raspberry Pi OS, Debian 12 Bookworm, or Ubuntu 24.04 LTS.
2. **Installs System Dependencies:** Automatically installs `bluez`, `dbus`, `bluetooth`, `curl`, and `jq` via `apt-get`.
3. **Applies Flash Longevity Tuning:** Configures `/etc/systemd/journald.conf.d/solaria-volatile.conf` to use volatile RAM journal storage (`Storage=volatile`, `RuntimeMaxUse=32M`) to protect the MicroSD card from wear.
4. **Applies Kernel Resilience:** Sets `vm.dirty_writeback_centisecs = 1500` and `kernel.panic = 10` for resilient crash recovery.
5. **Compiles & Registers Systemd Services:** Builds `solaria-bridge` and `solaria-sre-agent` and creates auto-restarting systemd units.

---

## 🛠️ Manual Build from Source

If you prefer building from source:

```bash
# 1. Clone the repository
git clone https://github.com/fkcurrie/solaria.git
cd solaria

# 2. Build the binaries
go build -o bin/solaria-bridge ./cmd/bridge
go build -o bin/solaria-cloud-server ./cmd/cloud-server
go build -o bin/solaria-sre-agent ./cmd/sre-agent

# 3. Start the local bridge
./bin/solaria-bridge
```

Open `http://localhost:8080` in Google Chrome to connect to your Renogy BT-1 module using Web Bluetooth.
