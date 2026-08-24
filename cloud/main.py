#!/usr/bin/env python3
"""
Solaria Cloud Run Microservice
Ingestion API & Live Analytics Dashboard for Renogy Solar & Dorset Weather Intelligence
Project: gca-gke-2025
"""

import os
from datetime import datetime, timezone
from typing import Any

from fastapi import Depends, FastAPI, Header, HTTPException, Request
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.templating import Jinja2Templates
from pydantic import BaseModel

# Cloud Firestore integration (optional fallback to memory store for local testing)
try:
    from google.cloud import firestore
    db = firestore.Client(project=os.environ.get("GCP_PROJECT", "gca-gke-2025"))
    FIRESTORE_ENABLED = True
except Exception:
    db = None
    FIRESTORE_ENABLED = False

app = FastAPI(
    title="Solaria Cloud Engine",
    description="Renogy Solar Telemetry & Environmental Intelligence Platform",
    version="1.0.0",
)

templates = Jinja2Templates(directory="templates")

API_TOKEN = os.environ.get("SOLARIA_API_TOKEN", "solaria_cottage_secret_token_2026")

# In-memory circular buffer for fast dashboard serving (holds last 1440 points = 4 hours @ 10s)
TELEMETRY_BUFFER: list[dict[str, Any]] = []
LATEST_TELEMETRY: dict[str, Any] = {
    "timestamp": datetime.now(timezone.utc).isoformat(),
    "site": "Dorset Solar Station",
    "location": {"latitude": 45.186, "longitude": -78.863},
    "telemetry": {
        "pv_power_w": 0,
        "pv_voltage_v": 13.0,
        "pv_current_a": 0.0,
        "battery_soc_pct": 80,
        "battery_voltage_v": 12.7,
        "battery_current_a": 0.0,
        "controller_temp_c": 27,
        "battery_temp_c": 29,
        "load_power_w": 0,
        "daily_max_pv_w": 15,
        "daily_generated_wh": 0,
        "total_generated_kwh": 8363,
        "charging_state": "MPPT Charging",
    },
    "weather": {
        "temperature_c": 18.5,
        "cloud_cover_pct": 20,
        "direct_radiation_w_m2": 0,
        "diffuse_radiation_w_m2": 0,
        "is_day": False,
    },
    "sun_classification": "NIGHT",
}


class TelemetryItem(BaseModel):
    timestamp: str
    site: str | None = "Dorset Station"
    location: dict[str, float] | None = None
    telemetry: dict[str, Any]
    weather: dict[str, Any] | None = None
    sun_classification: str | None = "UNKNOWN"


class TelemetryBatch(BaseModel):
    batch: list[TelemetryItem]


def verify_token(authorization: str | None = Header(None)):
    if not authorization:
        raise HTTPException(status_code=401, detail="Missing Authorization Header")
    token = authorization.replace("Bearer ", "").strip()
    if token != API_TOKEN:
        raise HTTPException(status_code=403, detail="Invalid API Token")
    return token


@app.get("/", response_class=HTMLResponse)
async def dashboard_view(request: Request):
    """Renders the main Solar & Environmental Intelligence Dashboard"""
    return templates.TemplateResponse(
        "index.html",
        {
            "request": request,
            "site_name": "1296 Wren Lake Drive (Dorset, ON)",
            "lat": 45.186,
            "lon": -78.863,
        },
    )


@app.post("/api/v1/telemetry")
async def ingest_telemetry(batch_data: TelemetryBatch, token: str = Depends(verify_token)):
    """High-throughput streaming ingestion endpoint for edge nodes"""
    global LATEST_TELEMETRY

    items = [item.model_dump() for item in batch_data.batch]
    if not items:
        return {"status": "ok", "ingested": 0}

    # Update in-memory buffer
    for item in items:
        TELEMETRY_BUFFER.append(item)
        if len(TELEMETRY_BUFFER) > 2000:
            TELEMETRY_BUFFER.pop(0)

    LATEST_TELEMETRY = items[-1]

    # Write to Firestore if connected
    if FIRESTORE_ENABLED and db:
        try:
            batch_write = db.batch()
            col = db.collection("solar_telemetry")
            for item in items:
                doc_ref = col.document()
                batch_write.set(doc_ref, item)
            batch_write.commit()
        except Exception as e:
            print(f"[WARN] Firestore write failed: {e}")

    return {"status": "ok", "ingested": len(items), "latest_ts": LATEST_TELEMETRY.get("timestamp")}


@app.get("/api/v1/live")
async def get_live_data():
    """Returns the latest telemetry point, battery status, and weather condition"""
    return JSONResponse(LATEST_TELEMETRY)


@app.get("/api/v1/history")
async def get_history(limit: int = 100):
    """Returns recent time-series points for live Chart.js rendering"""
    return JSONResponse(TELEMETRY_BUFFER[-limit:])


@app.get("/api/v1/analysis")
async def get_analysis():
    """Computes daily peak hours, sun condition breakdowns, and generation metrics"""
    if not TELEMETRY_BUFFER:
        return {"summary": "No data points collected yet"}

    sun_counts = {
        "FULL_SUN": 0,
        "PARTIAL_SUN_OR_SHADE": 0,
        "DIFFUSE_OVERCAST": 0,
        "ABSORPTION_FLOAT_CLIPPED": 0,
        "NIGHT": 0,
    }
    max_watts = 0
    total_wh = LATEST_TELEMETRY.get("telemetry", {}).get("daily_generated_wh", 0)

    for pt in TELEMETRY_BUFFER:
        s = pt.get("sun_classification", "NIGHT")
        if s in sun_counts:
            sun_counts[s] += 1
        w = pt.get("telemetry", {}).get("pv_power_w", 0)
        max_watts = max(max_watts, w)

    total_pts = max(1, len(TELEMETRY_BUFFER))
    sun_percentages = {k: round((v / total_pts) * 100, 1) for k, v in sun_counts.items()}

    return {
        "site": "1296 Wren Lake Drive, Dorset, ON",
        "latest_soc": LATEST_TELEMETRY.get("telemetry", {}).get("battery_soc_pct"),
        "latest_voltage": LATEST_TELEMETRY.get("telemetry", {}).get("battery_voltage_v"),
        "peak_power_w": max_watts,
        "daily_yield_wh": total_wh,
        "sun_condition_distribution_pct": sun_percentages,
    }


if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", "8080"))
    uvicorn.run("main:app", host="0.0.0.0", port=port, reload=True)
