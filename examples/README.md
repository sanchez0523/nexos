# Nexos Examples

Three ready-to-run clients that publish to a Nexos broker. Use them to verify
your install, record demo material, or as a starting point for your own
integrations.

Every example requires one Mosquitto account per device. Create accounts with:

```bash
./scripts/add-device.sh <device_id>
```

All examples authenticate over TLS using the self-signed CA at
`broker/certs/ca.crt`. Copy it to the device before running.

## esp32/ — Arduino sketch

Single-file `.ino` that connects an ESP32 over Wi-Fi + MQTT TLS and publishes
synthetic temperature / humidity every 5 seconds.

1. Open `examples/esp32/nexos_example.ino` in the Arduino IDE.
2. Install the `PubSubClient` library (Library Manager → PubSubClient by Nick O'Leary).
3. Fill in the CONFIG block at the top.
4. Paste the contents of `broker/certs/ca.crt` into `NEXOS_CA_CERT`.
5. Flash and open the serial monitor — a new card appears on the dashboard
   within seconds.

Replace `readTemperature` / `readHumidity` with your real sensors when you
wire them up. The payload contract is `{"value": N}` or a bare number.

## raspberry-pi/ — Python script

Publishes CPU temperature (`/sys/class/thermal`) and memory usage every 10
seconds.

```bash
cd examples/raspberry-pi
pip install -r requirements.txt

python3 nexos_client.py \
  --host 192.168.1.100 \
  --device-id rpi-lab \
  --password "<from add-device.sh>" \
  --ca-cert /path/to/broker/certs/ca.crt
```

Works on any Linux host (macOS too, via `psutil` temperatures). Use it as a
starting point for headless Pi deployments.

## simulator/ — Go data generator

Drives N virtual devices publishing realistic sine-wave + noise data. Use
this when you don't have real hardware handy or when recording the demo GIF
for the README.

```bash
# Create MQTT accounts for sim-1 .. sim-3 first:
for i in 1 2 3; do ./scripts/add-device.sh "sim-$i" "s!mpw$i"; done

# Then run the simulator:
cd examples/simulator
go run . \
  -broker mqtts://localhost:8883 \
  -ca ../../broker/certs/ca.crt \
  -devices 3 \
  -passwords "s!mpw1,s!mpw2,s!mpw3"
```

Each device publishes `temperature`, `humidity`, `pressure` by default.
Override with `-sensors "foo,bar"`.
