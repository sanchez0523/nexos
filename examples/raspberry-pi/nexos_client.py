"""Nexos Raspberry Pi client.

Publishes the Pi's CPU temperature and memory-used percentage to a Nexos
broker over TLS once every 10 seconds.

Usage:
    pip install -r requirements.txt
    python3 nexos_client.py \\
        --host 192.168.1.100 \\
        --device-id rpi-lab \\
        --password "<from add-device.sh>" \\
        --ca-cert ./ca.crt

The CA cert should be copied from the Nexos install's broker/certs/ca.crt so
the device can verify the broker's TLS handshake.
"""

from __future__ import annotations

import argparse
import json
import logging
import signal
import socket
import ssl
import sys
import time
from pathlib import Path

import paho.mqtt.client as mqtt
import psutil

logger = logging.getLogger("nexos-rpi")

# ── sensors ─────────────────────────────────────────────────────────────────
def read_cpu_temperature_c() -> float | None:
    """Return CPU temperature in °C, or None if the platform doesn't expose it.

    Raspberry Pi OS exposes temperatures via /sys/class/thermal. Fall back to
    psutil on other systems so the script is useful during development on a
    laptop.
    """
    thermal = Path("/sys/class/thermal/thermal_zone0/temp")
    if thermal.exists():
        raw = thermal.read_text().strip()
        return int(raw) / 1000.0
    try:
        temps = psutil.sensors_temperatures()  # type: ignore[attr-defined]
    except (AttributeError, NotImplementedError):
        return None
    for readings in temps.values():
        if readings:
            return readings[0].current
    return None


def read_memory_percent() -> float:
    return psutil.virtual_memory().percent


# ── MQTT wiring ──────────────────────────────────────────────────────────────
def build_client(device_id: str, password: str, ca_cert: Path) -> mqtt.Client:
    client_id = f"nexos-rpi-{socket.gethostname()}-{device_id}"
    client = mqtt.Client(client_id=client_id, clean_session=False)
    client.username_pw_set(device_id, password)

    context = ssl.create_default_context(cafile=str(ca_cert))
    # Nexos uses CN=broker for the server cert; users may hit the broker via
    # an IP or other hostname. Disable hostname verification here and rely on
    # CA-signed trust, matching the setup.sh SAN list.
    context.check_hostname = False
    context.verify_mode = ssl.CERT_REQUIRED
    client.tls_set_context(context)

    def on_connect(_c, _u, _f, rc: int) -> None:
        if rc == 0:
            logger.info("connected to broker")
        else:
            logger.error("connect failed rc=%s", rc)

    def on_disconnect(_c, _u, rc: int) -> None:
        if rc != 0:
            logger.warning("unexpected disconnect rc=%s; paho will retry", rc)

    client.on_connect = on_connect
    client.on_disconnect = on_disconnect
    # Built-in exponential backoff matches Nexos' ingestion subscriber.
    client.reconnect_delay_set(min_delay=1, max_delay=30)
    return client


def publish(client: mqtt.Client, device_id: str, sensor: str, value: float) -> None:
    topic = f"devices/{device_id}/{sensor}"
    payload = json.dumps({"value": value})
    result = client.publish(topic, payload, qos=0)
    if result.rc == mqtt.MQTT_ERR_SUCCESS:
        logger.info("pub %s = %s", topic, payload)
    else:
        logger.warning("pub failed %s rc=%s", topic, result.rc)


# ── main ─────────────────────────────────────────────────────────────────────
def main() -> int:
    parser = argparse.ArgumentParser(description="Nexos Raspberry Pi sample client")
    parser.add_argument("--host", required=True, help="Nexos broker host (IP or DNS)")
    parser.add_argument("--port", type=int, default=8883)
    parser.add_argument("--device-id", required=True, help="MQTT username and topic prefix")
    parser.add_argument("--password", required=True, help="MQTT password (see add-device.sh)")
    parser.add_argument(
        "--ca-cert",
        type=Path,
        required=True,
        help="Path to the Nexos CA certificate (broker/certs/ca.crt)",
    )
    parser.add_argument("--interval", type=float, default=10.0, help="seconds between publishes")
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    if not args.ca_cert.exists():
        logger.error("CA cert not found: %s", args.ca_cert)
        return 1

    client = build_client(args.device_id, args.password, args.ca_cert)
    client.connect(args.host, args.port, keepalive=30)
    client.loop_start()

    stop = False
    def handle_stop(*_):
        nonlocal stop
        stop = True
    signal.signal(signal.SIGINT, handle_stop)
    signal.signal(signal.SIGTERM, handle_stop)

    try:
        while not stop:
            temp = read_cpu_temperature_c()
            if temp is not None:
                publish(client, args.device_id, "cpu_temperature", temp)
            publish(client, args.device_id, "memory_used_pct", read_memory_percent())
            for _ in range(int(args.interval * 10)):
                if stop:
                    break
                time.sleep(0.1)
    finally:
        client.loop_stop()
        client.disconnect()
    return 0


if __name__ == "__main__":
    sys.exit(main())
