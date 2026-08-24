#!/usr/bin/env python3
"""
Renogy BT-1 / BT-2 Solar Gateway & BLE Linux Bridge
Full Modbus RTU stream reassembly, register decoding, Dorset weather correlation,
and live Cloud Run streaming for Crostini/Linux & Raspberry Pi.
"""

import asyncio
import csv
import http.server
import json
import os
import socketserver
import struct
import subprocess
import threading
import time
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

import websockets

HTTP_PORT = 8080
WS_PORT = 8765
BASE_DIR = Path(__file__).parent
STATIC_DIR = BASE_DIR / "static"
LOG_DIR = BASE_DIR / "logs"
LOG_DIR.mkdir(exist_ok=True)

# Dorset, Ontario, Canada Coordinates
SITE_LAT = 45.186
SITE_LON = -78.863
SITE_NAME = "1296 Wren Lake Drive, Dorset, ON"

# Cloud Ingestion Settings
CLOUD_ENDPOINT = os.environ.get(
    "SOLARIA_CLOUD_ENDPOINT",
    "https://solaria-dashboard-qgcwwot4tq-uc.a.run.app/api/v1/telemetry",
)
CLOUD_TOKEN = os.environ.get("SOLARIA_API_TOKEN", "solaria_cottage_secret_token_2026")

connected_clients = set()
latest_telemetry = {}
latest_device_info = {}
cached_weather = {
    "temperature_c": None,
    "cloud_cover_pct": None,
    "direct_radiation_w_m2": 0,
    "diffuse_radiation_w_m2": 0,
    "is_day": True,
}
last_weather_fetch = 0

rx_buffer = bytearray()


def fetch_dorset_weather():
    """Fetches real-time solar irradiance & cloud cover for Dorset, ON from Open-Meteo"""
    global cached_weather, last_weather_fetch
    now = time.time()
    if now - last_weather_fetch < 300 and cached_weather.get("temperature_c") is not None:
        return cached_weather

    url = (
        f"https://api.open-meteo.com/v1/forecast?latitude={SITE_LAT}&longitude={SITE_LON}"
        f"&current=temperature_2m,cloud_cover,direct_normal_irradiance,global_tilted_irradiance,"
        f"diffuse_radiation,direct_radiation,is_day&timezone=auto"
    )
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "Solaria/1.0"})
        with urllib.request.urlopen(req, timeout=5) as resp:
            if resp.status == 200:
                data = json.loads(resp.read().decode("utf-8"))
                cur = data.get("current", {})
                cached_weather = {
                    "temperature_c": cur.get("temperature_2m"),
                    "cloud_cover_pct": cur.get("cloud_cover"),
                    "direct_radiation_w_m2": cur.get("direct_radiation", 0.0),
                    "diffuse_radiation_w_m2": cur.get("diffuse_radiation", 0.0),
                    "is_day": bool(cur.get("is_day", 0)),
                }
                last_weather_fetch = now
    except Exception:
        pass
    return cached_weather


def classify_sun_condition(metrics: dict, weather: dict) -> str:
    pv_watts = metrics.get("pv_power_w", 0)
    batt_soc = metrics.get("battery_soc_pct", 0)
    charging_state = metrics.get("charging_state", "")

    direct_rad = weather.get("direct_radiation_w_m2") or 0.0
    diffuse_rad = weather.get("diffuse_radiation_w_m2") or 0.0
    total_rad = direct_rad + diffuse_rad
    is_day = weather.get("is_day", False)
    cloud_cover = weather.get("cloud_cover_pct") if weather.get("cloud_cover_pct") is not None else 50

    # 1. Night / Dark Condition: If sun is down, or total solar irradiance is zero/near-zero with no PV harvest
    if not is_day or total_rad < 5.0 or (pv_watts <= 2 and direct_rad < 5.0):
        return "NIGHT"

    # 2. Battery Full / Curtailment / Float Clipping
    if batt_soc >= 99 and ("Float" in charging_state or "Boost" in charging_state):
        return "ABSORPTION_FLOAT_CLIPPED"

    # 3. Daylight Solar Conditions
    if cloud_cover < 25 and direct_rad > 300 and pv_watts > 10:
        return "FULL_SUN"
    elif cloud_cover > 80 or (diffuse_rad > direct_rad and direct_rad < 150):
        return "DIFFUSE_OVERCAST"
    elif 25 <= cloud_cover <= 80:
        return "PARTIAL_SUN_OR_SHADE"
    return "VARIABLE_SUN"


class QuietHTTPHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(STATIC_DIR), **kwargs)

    def log_message(self, format, *args):
        pass


def start_http_server():
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer(("0.0.0.0", HTTP_PORT), QuietHTTPHandler) as httpd:
        print(f"[\033[94mHTTP\033[0m] Solar Web Dashboard: \033[1;32mhttp://localhost:{HTTP_PORT}\033[0m")
        httpd.serve_forever()


def calc_crc16_modbus(data: bytes) -> bytes:
    crc = 0xFFFF
    for pos in data:
        crc ^= pos
        for _ in range(8):
            if crc & 1:
                crc = (crc >> 1) ^ 0xA001
            else:
                crc >>= 1
    return bytes([crc & 0xFF, (crc >> 8) & 0xFF])


def decode_device_info(raw_bytes: bytes) -> dict:
    if len(raw_bytes) < 25 or raw_bytes[1] != 0x03:
        return {}
    try:
        data = raw_bytes[3:-2]
        max_voltage = int.from_bytes(data[0:2], byteorder="big")
        max_current = int.from_bytes(data[2:4], byteorder="big")
        model_bytes = data[4:20]
        model_str = model_bytes.decode("ascii", errors="replace").strip().strip("\x00")
        sw_ver = f"{data[20]}.{data[21]}" if len(data) > 21 else "N/A"
        hw_ver = f"{data[22]}.{data[23]}" if len(data) > 23 else "N/A"

        return {
            "model": model_str or "Renogy Rover/Wanderer",
            "max_voltage_v": max_voltage,
            "max_current_a": max_current,
            "software_version": sw_ver,
            "hardware_version": hw_ver,
        }
    except Exception as e:
        return {"error": str(e)}


def decode_realtime_telemetry(raw_bytes: bytes) -> dict:
    if len(raw_bytes) < 35 or raw_bytes[1] != 0x03:
        return {}
    try:
        data = raw_bytes[3:-2]
        if len(data) < 20:
            return {}

        batt_soc = int.from_bytes(data[0:2], byteorder="big")
        batt_voltage = int.from_bytes(data[2:4], byteorder="big") * 0.1
        batt_charging_current = int.from_bytes(data[4:6], byteorder="big") * 0.01

        ctrl_temp = struct.unpack("b", bytes([data[6]]))[0]
        batt_temp = struct.unpack("b", bytes([data[7]]))[0]

        load_voltage = int.from_bytes(data[8:10], byteorder="big") * 0.1
        load_current = int.from_bytes(data[10:12], byteorder="big") * 0.01
        load_power = int.from_bytes(data[12:14], byteorder="big")

        pv_voltage = int.from_bytes(data[14:16], byteorder="big") * 0.1
        pv_current = int.from_bytes(data[16:18], byteorder="big") * 0.01
        pv_power = int.from_bytes(data[18:20], byteorder="big")

        daily_min_batt_v = (
            int.from_bytes(data[22:24], byteorder="big") * 0.1
            if len(data) >= 24
            else round(batt_voltage, 1)
        )
        daily_max_batt_v = (
            int.from_bytes(data[24:26], byteorder="big") * 0.1
            if len(data) >= 26
            else round(batt_voltage, 1)
        )
        daily_max_chg_curr = int.from_bytes(data[26:28], byteorder="big") * 0.01 if len(data) >= 28 else 0.0
        daily_max_dischg_curr = int.from_bytes(data[28:30], byteorder="big") * 0.01 if len(data) >= 30 else 0.0
        daily_max_pv = int.from_bytes(data[30:32], byteorder="big") if len(data) >= 32 else pv_power
        daily_max_load_w = int.from_bytes(data[32:34], byteorder="big") if len(data) >= 34 else load_power
        daily_chg_ah = int.from_bytes(data[34:36], byteorder="big") if len(data) >= 36 else 0
        daily_dischg_ah = int.from_bytes(data[36:38], byteorder="big") if len(data) >= 38 else 0
        daily_yield_wh = int.from_bytes(data[38:40], byteorder="big") if len(data) >= 40 else 0
        daily_consumed_wh = int.from_bytes(data[40:42], byteorder="big") if len(data) >= 42 else 0
        operating_days = int.from_bytes(data[42:44], byteorder="big") if len(data) >= 44 else 0
        total_overdischg = int.from_bytes(data[44:46], byteorder="big") if len(data) >= 46 else 0
        total_fullchg = int.from_bytes(data[46:48], byteorder="big") if len(data) >= 48 else 0
        total_chg_ah = int.from_bytes(data[48:52], byteorder="big") if len(data) >= 52 else 0
        total_dischg_ah = int.from_bytes(data[52:56], byteorder="big") if len(data) >= 56 else 0
        total_yield_kwh = int.from_bytes(data[56:60], byteorder="big") if len(data) >= 60 else 0
        total_consumed_kwh = int.from_bytes(data[60:64], byteorder="big") if len(data) >= 64 else 0

        load_status = bool(data[64] & 0x80) if len(data) >= 65 else False
        state_code = data[65] if len(data) >= 66 else (data[33] if len(data) > 33 else 0)
        state_map = {
            0x00: "Deactivated",
            0x01: "Activated",
            0x02: "MPPT Charging",
            0x03: "Equalizing Charging",
            0x04: "Boost Charging",
            0x05: "Floating Charging",
            0x06: "Current Limiting",
        }
        charging_state = state_map.get(state_code, f"State 0x{state_code:02X}")

        fault_bits = int.from_bytes(data[66:68], byteorder="big") if len(data) >= 68 else 0
        active_faults = []
        fault_map = {
            0: "Battery Over-Discharge",
            1: "Battery Over-Voltage",
            2: "Battery Under-Voltage Warning",
            3: "Load Short-Circuit",
            4: "Load Over-Current",
            5: "Controller Over-Temp",
            6: "Battery Over-Temp",
            7: "PV Array Over-Power",
            8: "PV Array Short-Circuit",
            9: "PV Array Over-Voltage",
            10: "PV Counter-Current",
            11: "PV Reverse Polarity",
            12: "Battery Reverse Polarity",
            13: "Battery Probe Disconnected",
        }
        for bit_idx, fault_name in fault_map.items():
            if (fault_bits >> bit_idx) & 1:
                active_faults.append(fault_name)
        fault_flags = ", ".join(active_faults) if active_faults else "NORMAL"

        return {
            "pv_power_w": pv_power,
            "pv_voltage_v": round(pv_voltage, 1),
            "pv_current_a": round(pv_current, 2),
            "battery_soc_pct": batt_soc,
            "battery_voltage_v": round(batt_voltage, 1),
            "battery_current_a": round(batt_charging_current, 2),
            "controller_temp_c": ctrl_temp,
            "battery_temp_c": batt_temp,
            "load_power_w": load_power,
            "load_voltage_v": round(load_voltage, 1),
            "load_current_a": round(load_current, 2),
            "load_status": load_status,
            "daily_min_battery_voltage_v": round(daily_min_batt_v, 1),
            "daily_max_battery_voltage_v": round(daily_max_batt_v, 1),
            "daily_max_charging_current_a": round(daily_max_chg_curr, 2),
            "daily_max_discharging_current_a": round(daily_max_dischg_curr, 2),
            "daily_max_pv_w": daily_max_pv,
            "daily_max_load_w": daily_max_load_w,
            "daily_charging_ah": daily_chg_ah,
            "daily_discharging_ah": daily_dischg_ah,
            "daily_generated_wh": daily_yield_wh,
            "daily_consumed_wh": daily_consumed_wh,
            "operating_days": operating_days,
            "total_battery_overdischarge_count": total_overdischg,
            "total_battery_fullcharge_count": total_fullchg,
            "total_charging_ah": total_chg_ah,
            "total_discharging_ah": total_dischg_ah,
            "total_generated_kwh": total_yield_kwh,
            "total_consumed_kwh": total_consumed_kwh,
            "charging_state": charging_state,
            "fault_bits": fault_bits,
            "fault_flags": fault_flags,
        }
    except Exception as e:
        return {"error": str(e)}


def log_telemetry_to_csv(metrics: dict):
    now = datetime.now()
    log_file = LOG_DIR / f"solar_telemetry_{now.strftime('%Y-%m-%d')}.csv"
    is_new = not log_file.exists()

    fieldnames = [
        "timestamp",
        "pv_power_w",
        "pv_voltage_v",
        "pv_current_a",
        "battery_soc_pct",
        "battery_voltage_v",
        "battery_current_a",
        "charging_state",
        "controller_temp_c",
        "battery_temp_c",
        "load_power_w",
        "daily_generated_wh",
        "total_generated_kwh",
    ]

    with open(log_file, "a", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        if is_new:
            writer.writeheader()
        row = {"timestamp": now.isoformat()}
        for k in fieldnames[1:]:
            row[k] = metrics.get(k, "")
        writer.writerow(row)


_id_token_cache = {"token": "", "expires_at": 0}


def get_gcp_bearer_token() -> str:
    global _id_token_cache
    now = time.time()
    if _id_token_cache["token"] and now < _id_token_cache["expires_at"]:
        return _id_token_cache["token"]
    try:
        res = subprocess.run(
            ["gcloud", "auth", "print-identity-token"],
            capture_output=True,
            text=True,
            timeout=4,
        )
        if res.returncode == 0 and res.stdout.strip():
            tok = res.stdout.strip()
            _id_token_cache = {"token": tok, "expires_at": now + 1800}
            return tok
    except Exception:
        pass
    return CLOUD_TOKEN


def upload_to_cloud(record: dict):
    """Asynchronously posts record to Cloud Run ingestion endpoint"""
    if not CLOUD_ENDPOINT:
        return

    def _post():
        try:
            token = get_gcp_bearer_token()
            payload = json.dumps({"batch": [record]}).encode("utf-8")
            req = urllib.request.Request(
                CLOUD_ENDPOINT,
                data=payload,
                headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {token}",
                    "X-API-Key": CLOUD_TOKEN,
                },
                method="POST",
            )
            with urllib.request.urlopen(req, timeout=5) as r:
                if r.status == 200:
                    pass
        except Exception:
            pass

    threading.Thread(target=_post, daemon=True).start()


def process_assembled_frame(frame: bytes):
    if len(frame) < 5 or frame[1] != 0x03:
        return

    byte_count = frame[2]
    timestamp = datetime.now().strftime("%H:%M:%S.%f")[:-3]

    if byte_count >= 60 or len(frame) >= 65:
        metrics = decode_realtime_telemetry(frame)
        if metrics and "error" not in metrics:
            global latest_telemetry
            latest_telemetry = metrics
            log_telemetry_to_csv(metrics)

            weather = fetch_dorset_weather()
            sun_state = classify_sun_condition(metrics, weather)

            array_cap_w = 400
            pv_w = metrics.get("pv_power_w", 0)
            util_pct = round((pv_w / float(array_cap_w)) * 100.0, 1)
            metrics["array_capacity_w"] = array_cap_w
            metrics["array_topology"] = "2S2P (4x100W)"
            metrics["array_utilization_pct"] = util_pct

            rad_total = (weather.get("direct_radiation_w_m2") or 0) + (weather.get("diffuse_radiation_w_m2") or 0)
            perf_ratio = 0.0
            if rad_total > 20:
                expected_w = (rad_total / 1000.0) * float(array_cap_w)
                perf_ratio = round((pv_w / expected_w) * 100.0, 1)
            metrics["performance_ratio_pct"] = perf_ratio

            cloud_record = {
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "site": SITE_NAME,
                "location": {"latitude": SITE_LAT, "longitude": SITE_LON},
                "telemetry": metrics,
                "weather": weather,
                "sun_classification": sun_state,
            }
            upload_to_cloud(cloud_record)

            print(f"\n[\033[1;33m{timestamp} ☀️ RENOGY LIVE TELEMETRY | {sun_state}\033[0m]")
            print(
                f"  ├─ Array (400W 2S2P): \033[1;33m{pv_w} W\033[0m "
                f"({metrics['pv_voltage_v']}V @ {metrics['pv_current_a']}A) | "
                f"Util: {util_pct}% | Peak: {metrics['daily_max_pv_w']}W"
            )
            print(
                f"  ├─ Battery:           \033[1;32m{metrics['battery_voltage_v']} V\033[0m | "
                f"SOC: \033[1;36m{metrics['battery_soc_pct']}%\033[0m | Charge: {metrics['battery_current_a']}A"
            )
            print(
                f"  ├─ State:             \033[1;35m{metrics['charging_state']}\033[0m | "
                f"Health: \033[1;32m{metrics['fault_flags']}\033[0m"
            )
            print(
                f"  ├─ Dorset Wx:         {weather.get('temperature_c')}°C | "
                f"Clouds: {weather.get('cloud_cover_pct')}% | "
                f"Rad: {weather.get('direct_radiation_w_m2')} W/m² (PR: {perf_ratio}%)"
            )
            print(
                f"  ├─ Temps:             Controller {metrics['controller_temp_c']}°C | "
                f"Battery {metrics['battery_temp_c']}°C"
            )
            print(
                f"  └─ Daily Yield:       \033[1;32m{metrics['daily_generated_wh']} Wh\033[0m | "
                f"Lifetime: {metrics['total_generated_kwh']} kWh"
            )

    elif byte_count >= 20:
        dev_info = decode_device_info(frame)
        if dev_info and "error" not in dev_info:
            global latest_device_info
            latest_device_info = dev_info
            print(f"\n[\033[1;36m{timestamp} ℹ️ RENOGY CONTROLLER INFO\033[0m]")
            print(
                f"  ├─ Model:     \033[1m{dev_info['model']}\033[0m "
                f"({dev_info['max_voltage_v']}V / {dev_info['max_current_a']}A)"
            )
            print(f"  └─ Firmware:  SW v{dev_info['software_version']} | HW v{dev_info['hardware_version']}")


async def handle_ws_client(websocket):
    global rx_buffer
    client_addr = websocket.remote_address
    connected_clients.add(websocket)
    print(f"[\033[92mWS\033[0m] Browser connected: {client_addr}")

    try:
        async for raw_msg in websocket:
            try:
                data = json.loads(raw_msg)
                event_type = data.get("type", "unknown")
                timestamp = datetime.now().strftime("%H:%M:%S.%f")[:-3]

                if event_type == "device_selected":
                    dev_name = data.get("name") or "Renogy Device"
                    dev_id = data.get("id", "Unknown")
                    print(
                        f"\n[\033[1;33m{timestamp} ☀️ RENOGY BLE SELECTED\033[0m] "
                        f"Name: \033[1m{dev_name}\033[0m | ID: {dev_id}"
                    )

                elif event_type == "gatt_connected":
                    dev_name = data.get("name") or "Renogy Device"
                    rx_buffer = bytearray()
                    print(
                        f"[\033[92m{timestamp} GATT READY\033[0m] "
                        f"Connected to \033[1m{dev_name}\033[0m (Channels: FFD1 TX, FFF1 RX)"
                    )

                elif event_type in ("notification", "characteristic_value"):
                    raw_chunk = bytes(data.get("bytes", []))
                    rx_buffer.extend(raw_chunk)

                    if len(rx_buffer) >= 3 and rx_buffer[1] == 0x03:
                        expected_len = 3 + rx_buffer[2] + 2
                        if len(rx_buffer) >= expected_len:
                            full_frame = bytes(rx_buffer[:expected_len])
                            rx_buffer = rx_buffer[expected_len:]
                            process_assembled_frame(full_frame)
                    elif len(rx_buffer) > 100:
                        rx_buffer = bytearray()

                elif event_type == "device_disconnected":
                    rx_buffer = bytearray()
                    print(f"[\033[91m{timestamp} BLE DISCONNECTED\033[0m] Renogy device disconnected.")

                elif event_type == "error":
                    print(f"[\033[91m{timestamp} ERROR\033[0m] {data.get('message')}")

            except json.JSONDecodeError:
                pass

    except websockets.exceptions.ConnectionClosed:
        pass
    finally:
        connected_clients.remove(websocket)
        print(f"[\033[90mWS\033[0m] Browser disconnected: {client_addr}")


async def run_server():
    http_thread = threading.Thread(target=start_http_server, daemon=True)
    http_thread.start()

    print(f"[\033[94mWS\033[0m] WebSocket Gateway listening on: \033[1;32mws://localhost:{WS_PORT}\033[0m")
    print("=" * 75)
    print("☀️  \033[1mRENOGY BT-1 / BT-2 SOLAR & BLE LINUX GATEWAY\033[0m")
    print("=" * 75)
    print(f"  • Site: {SITE_NAME} ({SITE_LAT}°N, {SITE_LON}°W)")
    print("  • Service UUID 0xFFD0 (Write Commands to FFD1)")
    print("  • Service UUID 0xFFF0 (Receive Telemetry from FFF1)")
    print("  • Automatic Stream Chunk Reassembly & CRC16 Engine")
    print(f"  • CSV Logging saved automatically to: \033[36m{LOG_DIR}/\033[0m")
    print("-" * 75)
    print(f"Open Dashboard: \033[1;32mhttp://localhost:{HTTP_PORT}\033[0m in Chrome on your Chromebook")
    print("=" * 75)

    async with websockets.serve(handle_ws_client, "0.0.0.0", WS_PORT):
        await asyncio.Future()


def main():
    try:
        asyncio.run(run_server())
    except KeyboardInterrupt:
        print("\n[INFO] Server stopped.")


if __name__ == "__main__":
    main()
