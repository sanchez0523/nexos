#!/usr/bin/env bash
#
# Register a new MQTT account so a device can publish to its own subtopic.
#
# Writes the credential into broker/passwd then signals mosquitto to reload
# (via docker kill -SIGHUP) so the change takes effect without dropping the
# existing ingestion connection.

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

PASSWD_FILE="broker/passwd"

color() { printf "\033[%sm%s\033[0m" "$1" "$2"; }
ok()    { color "32" "✔ $*"; printf "\n"; }
warn()  { color "33" "⚠ $*"; printf "\n"; }
die()   { color "31" "✘ $*"; printf "\n" >&2; exit 1; }

[ -f "$PASSWD_FILE" ] || die "$PASSWD_FILE not found — run ./scripts/setup.sh first"

device_id=${1:-}
if [ -z "$device_id" ]; then
  read -r -p "Device ID (alphanumeric + dashes, matches MQTT username): " device_id
fi
[[ "$device_id" =~ ^[a-zA-Z0-9_.-]+$ ]] || die "device_id must match [a-zA-Z0-9_.-]+"

# Reject reserved internal accounts so users can't accidentally overwrite the
# credentials that Nexos itself depends on.
case "$device_id" in
  ingestion-worker|nexos-health)
    die "$device_id is reserved for Nexos internals"
    ;;
esac

password=${2:-}
if [ -z "$password" ]; then
  # Offer to auto-generate so users don't have to think of a value.
  read -r -p "Password (blank = generate): " password
  if [ -z "$password" ]; then
    password=$(openssl rand -base64 32 | tr -d '\n/+=' | head -c 24)
    echo "generated password: $password"
  fi
fi

docker run --rm -v "$ROOT_DIR/broker:/mosquitto" eclipse-mosquitto:2 \
  mosquitto_passwd -b /mosquitto/passwd "$device_id" "$password" >/dev/null
ok "account $device_id added to broker/passwd"

# SIGHUP reloads passwd + acl without dropping active connections. The broker
# container may not be running yet; surface that clearly rather than failing.
if docker compose ps --status running broker 2>/dev/null | grep -q broker; then
  docker compose kill -s SIGHUP broker >/dev/null
  ok "broker reloaded"
else
  warn "broker is not running yet — it will pick up the new account on next start"
fi

cat <<EOF

Connection parameters for $device_id:
  Host:     $(color 36 "mqtts://<your-host>:8883")
  Username: $(color 36 "$device_id")
  Password: $(color 36 "$password")
  CA cert:  $(color 36 "broker/certs/ca.crt")
  Topic:    $(color 36 "devices/$device_id/<sensor>")
  Payload:  $(color 36 '{"value": 23.5}')

Only topics prefixed with devices/$device_id/ are permitted for this account.
EOF
