# Headless SD Card Setup for Raspberry Pi

Setting up a dedicated Raspberry Pi appliance for Solaria takes less than 5 minutes using the official Raspberry Pi Imager.

---

## 🖴 3-Step Headless Flashing

1. Download and launch **[Raspberry Pi Imager](https://www.raspberrypi.com/software/)**.
2. **Choose OS:** Select `Raspberry Pi OS (other)` $\rightarrow$ `Raspberry Pi OS Lite (64-bit)`.
3. **Configure Customization (Gear Icon / Ctrl+Shift+X):**
   - **Hostname:** `solaria-cottage`
   - **User / Password:** `solaria` / `<your-secure-password>`
   - **Enable SSH:** Check `Use password authentication` or inject your public SSH key.
   - **Configure Wireless LAN:** Enter your local cottage WiFi SSID and Pre-Shared Key (PSK).
   - **Set Locale:** `America/Toronto` (or your local time zone).
4. **Flash & Insert:** Write to your MicroSD card, insert into the Raspberry Pi, and apply 5V power (via USB-C or DC buck converter connected to the 12V LiFePO4 bus).

---

## 🔑 First Boot Connection

Once the Pi boots (typically 30–45 seconds), connect over SSH from your computer:

```bash
ssh solaria@solaria-cottage.local
```

Then run the 1-click installer:

```bash
curl -fsSL https://raw.githubusercontent.com/fkcurrie/solaria/main/setup.sh | bash
```
