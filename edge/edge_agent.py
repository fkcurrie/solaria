#!/usr/bin/env python3
"""
Solaria Edge Daemon for Linux & Raspberry Pi
Connects to Renogy BT-1/BT-2 over BLE, polls telemetry every 10s,
correlates with Open-Meteo hyper-local weather in Dorset, ON,
buffers locally in SQLite, and uploads to Cloud Run (GCP gca-gke-2025).
"""

import argparse
import asyncio
import json
import logging
import sqlite3
import struct
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

import requests
import yaml

# BLE Imports (Graceful fallback if running in bridge mode)
try:
    from bleak import BleakClient, BleakScanner
    BLEAK_AVAILABLE = True
except ImportError:
    BLEAK_AVAILABLE = False

logging.basicConfig(
    level=logging.INFO,
    format="[%(asctime)s] [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger("SolariaEdge")


class TelemetrySpooler:
    """Local SQLite spooler to guarantee zero data loss during Internet dropouts"""

    def __init__(self, db_path: str):
        self.db_path = db_path
        self._init_db()

    def _init_db(self):
        with sqlite3.connect(self.db_path) as conn:
            conn.execute("""
                CREATE TABLE IF NOT EXISTS telemetry_spool (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    timestamp TEXT,
                    payload_json TEXT,
                    uploaded INTEGER DEFAULT 0
                )
            """)
            conn.commit()

    def store(self, payload: dict):
        with sqlite3.connect(self.db_path) as conn:
            conn.execute(
                "INSERT INTO telemetry_spool (timestamp, payload_json, uploaded) VALUES (?, ?, 0)",
                (payload.get("timestamp"), json.dumps(payload)),
            )
            conn.commit()

    def fetch_unuploaded(self, limit=20):
        with sqlite3.connect(self.db_path) as conn:
            cursor = conn.execute(
                "SELECT id, payload_json FROM telemetry_spool WHERE uploaded = 0 ORDER BY id ASC LIMIT ?",
                (limit,),
            )
            return cursor.fetchall()

    def mark_uploaded(self, ids: list):
        if not ids:
            return
        with sqlite3.connect(self.db_path) as conn:
            placeholders = ",".join("?" for _ in ids)
            conn.execute(f"UPDATE telemetry_spool SET uploaded = 1 WHERE id IN ({placeholders})", ids)
            conn.commit()

    def purge_old_records(self, days=30):
        with sqlite3.connect(self.db_path) as conn:
            conn.execute(
                "DELETE FROM telemetry_spool WHERE uploaded = 1 AND timestamp < datetime('now', ?)",
                (f"-{days} days",),
            )
            conn.commit()


class WeatherProvider:
    """Fetches high-resolution solar irradiance & cloud metrics for Dorset, ON"""

    def __init__(self, lat: float, lon: float, poll_interval_sec: int = 300):
        self.lat = lat
        self.lon = lon
        self.poll_interval = poll_interval_sec
        self.last_fetch = 0
        self.cached_weather = {
            "temperature_c": None,
            "cloud_cover_pct": None,
            "ghi_w_m2": None,
            "dni_w_m2": None,
            "dhi_w_m2": None,
            "direct_radiation_w_m2": None,
            "diffuse_radiation_w_m2": None,
            "sun_condition": "UNKNOWN",
        }

    def get_weather(self) -> dict:
        now = time.time()
        if now - self.last_fetch > self.poll_interval:
            self._fetch_open_meteo()
            self.last_fetch = now
        return self.cached_weather

    def _fetch_open_meteo(self):
        url = "https://api.open-meteo.com/v1/forecast"
        params = {
            "latitude": self.lat,
            "longitude": self.lon,
            "current": [
                "temperature_2m",
                "cloud_cover",
                "direct_normal_irradiance",
                "global_tilted_irradiance",
                "diffuse_radiation",
                "direct_radiation",
                "is_day",
            ],
            "timezone": "auto",
        }
        try:
            resp = requests.get(url, params=params, timeout=8)
            if resp.status_code == 200:
                cur = resp.json().get("current", {})
                self.cached_weather = {
                    "temperature_c": cur.get("temperature_2m"),
                    "cloud_cover_pct": cur.get("cloud_cover"),
                    "ghi_w_m2": cur.get("global_tilted_irradiance"),
                    "dni_w_m2": cur.get("direct_normal_irradiance"),
                    "direct_radiation_w_m2": cur.get("direct_radiation"),
                    "diffuse_radiation_w_m2": cur.get("diffuse_radiation"),
                    "is_day": bool(cur.get("is_day", 1)),
                }
                logger.info(
                    f"☀️ Weather Updated (Dorset, ON): {cur.get('temperature_2m')}°C | "
                    f"Clouds: {cur.get('cloud_cover')}% | Irradiance: {cur.get('direct_radiation')} W/m²"
                )
        except Exception as e:
            logger.warning(f"Weather fetch failed: {e}")


def classify_sun_condition(
    pv_watts: float,
    batt_soc: int,
    charge_state: str,
    weather: dict,
    panel_rating_w: float = 400.0,
) -> str:
    """Classifies ambient solar condition by fusing PV metrics with atmospheric readings"""
    if pv_watts <= 2.0 and not weather.get("is_day", True):
        return "NIGHT"

    if batt_soc >= 99 and "Float" in charge_state:
        return "ABSORPTION_FLOAT_CLIPPED"

    cloud_cover = weather.get("cloud_cover_pct") or 50
    direct_rad = weather.get("direct_radiation_w_m2") or 0
    diffuse_rad = weather.get("diffuse_radiation_w_m2") or 0

    # Ratio of harvest to expected maximum
    harvest_ratio = pv_watts / max(1.0, panel_rating_w)

    if cloud_cover < 25 and direct_rad > 300:
        return "FULL_SUN"
    elif cloud_cover > 80 or (diffuse_rad > direct_rad and direct_rad < 150):
        return "DIFFUSE_OVERCAST"
    elif 25 <= cloud_cover <= 80 or harvest_ratio < 0.6:
        return "PARTIAL_SUN_OR_SHADE"
    else:
        return "VARIABLE_SUN"


def calc_crc16(data: bytes) -> bytes:
    crc = 0xFFFF
    for b in data:
        crc ^= b
        for _ in range(8):
            if crc & 1:
                crc = (crc >> 1) ^ 0xA001
            else:
                crc >>= 1
    return bytes([crc & 0xFF, (crc >> 8) & 0xFF])


def decode_modbus_telemetry(raw_bytes: bytes) -> dict:
    """Decodes 34 holding registers from Renogy Rover / Wanderer"""
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

        daily_max_pv = int.from_bytes(data[30:32], byteorder="big") if len(data) >= 32 else pv_power
        daily_yield = int.from_bytes(data[38:40], byteorder="big") if len(data) >= 40 else 0
        total_yield_kwh = int.from_bytes(data[56:60], byteorder="big") if len(data) >= 60 else 0

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

        return {
            "pv_power_w": pv_power,
            "pv_voltage_v": round(pv_voltage, 2),
            "pv_current_a": round(pv_current, 2),
            "battery_soc_pct": batt_soc,
            "battery_voltage_v": round(batt_voltage, 2),
            "battery_current_a": round(batt_charging_current, 2),
            "controller_temp_c": ctrl_temp,
            "battery_temp_c": batt_temp,
            "load_power_w": load_power,
            "load_voltage_v": round(load_voltage, 2),
            "load_current_a": round(load_current, 2),
            "daily_max_pv_w": daily_max_pv,
            "daily_generated_wh": daily_yield,
            "total_generated_kwh": total_yield_kwh,
            "charging_state": charging_state,
        }
    except Exception as e:
        logger.error(f"Modbus decode error: {e}")
        return {}


class CloudUploader:
    """Asynchronously ships buffered telemetry records to Cloud Run on GCP"""

    def __init__(self, endpoint: str, token: str, spooler: TelemetrySpooler):
        self.endpoint = endpoint
        self.token = token
        self.spooler = spooler

    def upload_pending(self):
        records = self.spooler.fetch_unuploaded(limit=25)
        if not records:
            return

        batch_ids = [r[0] for r in records]
        batch_payloads = [json.loads(r[1]) for r in records]

        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.token}",
        }

        try:
            resp = requests.post(
                self.endpoint,
                json={"batch": batch_payloads},
                headers=headers,
                timeout=10,
            )
            if resp.status_code in (200, 201):
                self.spooler.mark_uploaded(batch_ids)
                logger.info(f"☁️ Successfully synced {len(batch_ids)} telemetry record(s) to Cloud Run!")
            else:
                logger.warning(f"Cloud Ingest rejected payload (Status {resp.status_code}): {resp.text}")
        except Exception as e:
            logger.warning(f"Cloud upload offline (cached locally in SQLite): {e}")


class RenogyBlePoller:
    """Manages continuous BLE connection, chunk reassembly, and periodic Modbus polling"""

    def __init__(self, config: dict, spooler: TelemetrySpooler, weather: WeatherProvider, uploader: CloudUploader):
        self.config = config
        self.spooler = spooler
        self.weather = weather
        self.uploader = uploader
        self.rx_buffer = bytearray()
        self.client = None
        self.is_running = True

    async def run(self):
        logger.info("Starting Solaria Edge Daemon loop...")
        while self.is_running:
            try:
                if not BLEAK_AVAILABLE:
                    logger.error("Bleak library is not installed. Please install bleak or use local bridge mode.")
                    await asyncio.sleep(10)
                    continue

                ble_name = self.config["device"]["ble_name"]
                logger.info(f"Scanning for Renogy BLE device: {ble_name}...")
                device = await BleakScanner.find_device_by_name(ble_name, timeout=10.0)

                if not device:
                    logger.warning(f"Device '{ble_name}' not found in scan. Retrying in 5s...")
                    await asyncio.sleep(5)
                    continue

                async with BleakClient(device, timeout=self.config["device"].get("connect_timeout_sec", 15)) as client:
                    self.client = client
                    logger.info(f"✅ Connected to {device.name} [{device.address}]")

                    notify_uuid = self.config["device"]["ble_char_notify"]
                    write_uuid = self.config["device"]["ble_char_write"]

                    # Hook notification handler
                    await client.start_notify(notify_uuid, self._on_ble_notification)
                    logger.info("Subscribed to BLE Modbus notifications.")

                    # Main Polling Loop
                    poll_interval = self.config["device"].get("poll_interval_sec", 10)
                    query_frame = bytes([0xFF, 0x03, 0x01, 0x00, 0x00, 0x22])
                    full_query = query_frame + calc_crc16(query_frame)

                    while client.is_connected and self.is_running:
                        # Send Modbus Read Holding Registers (0x0100 - 0x0122)
                        await client.write_gatt_char(write_uuid, full_query, response=False)

                        # Trigger cloud upload of any buffered records
                        if self.config.get("cloud", {}).get("enabled"):
                            self.uploader.upload_pending()

                        await asyncio.sleep(poll_interval)

            except Exception as e:
                logger.error(f"BLE connection error: {e}. Reconnecting in 5s...")
                await asyncio.sleep(5)

    def _on_ble_notification(self, sender, data: bytearray):
        self.rx_buffer.extend(data)

        # Reassemble Modbus frames
        if len(self.rx_buffer) >= 3 and self.rx_buffer[1] == 0x03:
            expected_len = 3 + self.rx_buffer[2] + 2
            if len(self.rx_buffer) >= expected_len:
                full_frame = bytes(self.rx_buffer[:expected_len])
                self.rx_buffer = self.rx_buffer[expected_len:]
                self._process_modbus_frame(full_frame)
        elif len(self.rx_buffer) > 100:
            self.rx_buffer = bytearray()

    def _process_modbus_frame(self, frame: bytes):
        metrics = decode_modbus_telemetry(frame)
        if not metrics:
            return

        now_iso = datetime.now(timezone.utc).isoformat()
        current_weather = self.weather.get_weather()
        sun_state = classify_sun_condition(
            metrics["pv_power_w"],
            metrics["battery_soc_pct"],
            metrics["charging_state"],
            current_weather,
            self.config.get("site", {}).get("panel_rated_watts", 400),
        )

        record = {
            "timestamp": now_iso,
            "site": self.config.get("site", {}).get("name", "Dorset Station"),
            "location": {
                "latitude": self.config.get("site", {}).get("latitude"),
                "longitude": self.config.get("site", {}).get("longitude"),
            },
            "telemetry": metrics,
            "weather": current_weather,
            "sun_classification": sun_state,
        }

        logger.info(
            f"☀️ [{sun_state}] Solar: {metrics['pv_power_w']}W ({metrics['pv_voltage_v']}V) | "
            f"Batt: {metrics['battery_voltage_v']}V ({metrics['battery_soc_pct']}%) | "
            f"Clouds: {current_weather.get('cloud_cover_pct')}%"
        )

        # Store in local SQLite spooler
        self.spooler.store(record)


def main():
    parser = argparse.ArgumentParser(description="Solaria Renogy BLE Edge Daemon")
    parser.add_argument("--config", default="config.yaml", help="Path to config YAML file")
    args = parser.parse_args()

    config_path = Path(args.config)
    if not config_path.exists():
        logger.error(f"Configuration file not found: {config_path}")
        sys.exit(1)

    with open(config_path, "r", encoding="utf-8") as f:
        config = yaml.safe_load(f)

    db_path = config.get("cloud", {}).get("spool_db_path", "spool.db")
    spooler = TelemetrySpooler(db_path)

    site_cfg = config.get("site", {})
    weather = WeatherProvider(
        lat=site_cfg.get("latitude", 45.186),
        lon=site_cfg.get("longitude", -78.863),
        poll_interval_sec=config.get("weather", {}).get("poll_interval_sec", 300),
    )

    cloud_cfg = config.get("cloud", {})
    uploader = CloudUploader(
        endpoint=cloud_cfg.get("endpoint", "http://localhost:8080/api/v1/telemetry"),
        token=cloud_cfg.get("api_token", ""),
        spooler=spooler,
    )

    poller = RenogyBlePoller(config, spooler, weather, uploader)
    try:
        asyncio.run(poller.run())
    except KeyboardInterrupt:
        logger.info("Daemon stopped by user.")


if __name__ == "__main__":
    main()
