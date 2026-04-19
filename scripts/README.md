# Nexos Scripts

These two scripts are the entirety of Nexos's operational surface area. They
intentionally stay small — anything more complex belongs in the Go service or
the Makefile.

## `setup.sh` — interactive install

Run once, immediately after `git clone`:

```bash
./scripts/setup.sh
```

What it generates:

| File                     | Contents                                            |
|--------------------------|-----------------------------------------------------|
| `.env`                   | admin creds + randomly generated secrets (chmod 600) |
| `broker/certs/ca.crt`    | self-signed root CA (10-year, `CN=Nexos Local CA`)   |
| `broker/certs/server.*`  | broker TLS cert, 365-day, `SAN=broker,localhost,127.0.0.1` |
| `broker/passwd`          | hashed credentials for `ingestion-worker` + `nexos-health` |

### Options
- `--certs-only` regenerate TLS material without touching `.env` or passwd.
- Re-running with an existing `.env` reuses its values (delete the file first
  if you want fresh secrets).

Requires `openssl` and `docker` on the host. `mosquitto_passwd` is invoked via
the `eclipse-mosquitto:2` image so you don't need mosquitto installed locally.

## `add-device.sh` — register a new MQTT account

```bash
./scripts/add-device.sh esp32-living           # prompts for password
./scripts/add-device.sh sim-1 "s!mpw1"         # non-interactive
```

Appends the account to `broker/passwd` and sends `SIGHUP` to the running
broker so the change is picked up without dropping existing connections.

The script rejects reserved names (`ingestion-worker`, `nexos-health`) to
prevent you from clobbering the internal accounts that `setup.sh` creates.

The printed connection summary covers the minimum a device needs: host,
credentials, CA cert location, and topic prefix. Each account is restricted
by the broker ACL so a device can only publish to `devices/<its-id>/#`.
