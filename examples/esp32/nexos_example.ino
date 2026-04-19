// Nexos ESP32 example
//
// Publishes a synthetic temperature + humidity reading every 5 seconds to a
// Nexos MQTT broker over TLS.
//
// Requirements:
//   - Arduino IDE / PlatformIO with the ESP32 board package
//   - Library: PubSubClient (by Nick O'Leary)
//
// Before flashing, fill in the CONFIG block below. The CA certificate comes
// from the Nexos install that ran `scripts/setup.sh` — copy the contents of
// broker/certs/ca.crt between the R"(...)" markers.

#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <PubSubClient.h>

// ── CONFIG ──────────────────────────────────────────────────────────────────
#define WIFI_SSID       "YOUR_WIFI_SSID"
#define WIFI_PASSWORD   "YOUR_WIFI_PASSWORD"

#define MQTT_HOST       "192.168.1.100"   // host/IP running Nexos
#define MQTT_PORT       8883
#define DEVICE_ID       "esp32-living"    // must match the username from add-device.sh
#define MQTT_PASSWORD   "CHANGE_ME"       // the password add-device.sh printed

#define PUBLISH_INTERVAL_MS 5000

// Paste the contents of broker/certs/ca.crt here. Keep the R"EOF( ... )EOF"
// delimiters so the preprocessor treats it as a raw string literal.
static const char* NEXOS_CA_CERT = R"EOF(
-----BEGIN CERTIFICATE-----
PASTE_CA_CERT_HERE
-----END CERTIFICATE-----
)EOF";
// ────────────────────────────────────────────────────────────────────────────

WiFiClientSecure  netClient;
PubSubClient      mqtt(netClient);

unsigned long lastPublishMs = 0;

void connectWiFi() {
  Serial.printf("[wifi] connecting to %s", WIFI_SSID);
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  Serial.printf(" ok, ip=%s\n", WiFi.localIP().toString().c_str());
}

void connectMqtt() {
  netClient.setCACert(NEXOS_CA_CERT);
  mqtt.setServer(MQTT_HOST, MQTT_PORT);

  // Each attempt gets a unique client ID so stale broker sessions can't
  // reject us; the paho-on-device approach mirrors what the Go subscriber
  // does on the ingestion side.
  String clientId = String("nexos-esp32-") + String((uint32_t)ESP.getEfuseMac(), HEX);

  while (!mqtt.connected()) {
    Serial.printf("[mqtt] connecting as %s...", DEVICE_ID);
    if (mqtt.connect(clientId.c_str(), DEVICE_ID, MQTT_PASSWORD)) {
      Serial.println(" ok");
    } else {
      Serial.printf(" rc=%d, retrying in 5s\n", mqtt.state());
      delay(5000);
    }
  }
}

// Sensor simulation — replace with real reads (DHT22, BME280, …) when you wire
// actual hardware. The payload contract is a flat object with a single numeric
// `value` field, OR a bare number.
float readTemperature() {
  return 20.0 + (float)random(-200, 200) / 100.0;
}
float readHumidity() {
  return 50.0 + (float)random(-500, 500) / 100.0;
}

void publishMetric(const char* sensor, float value) {
  char topic[128];
  snprintf(topic, sizeof(topic), "devices/%s/%s", DEVICE_ID, sensor);

  char payload[64];
  // Keep the payload minimal — Nexos ignores unknown fields but the smaller
  // the frame, the happier ESP32's TLS stack is.
  snprintf(payload, sizeof(payload), "{\"value\":%.2f}", value);

  mqtt.publish(topic, payload);
  Serial.printf("[pub] %s = %s\n", topic, payload);
}

void setup() {
  Serial.begin(115200);
  delay(500);
  connectWiFi();
  connectMqtt();
}

void loop() {
  if (!mqtt.connected()) connectMqtt();
  mqtt.loop();

  unsigned long now = millis();
  if (now - lastPublishMs >= PUBLISH_INTERVAL_MS) {
    lastPublishMs = now;
    publishMetric("temperature", readTemperature());
    publishMetric("humidity",    readHumidity());
  }
}
