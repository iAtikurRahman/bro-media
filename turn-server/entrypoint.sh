#!/bin/sh
set -e

TURN_USER="${TURN_USER:-bromedia}"
TURN_PASSWORD="${TURN_PASSWORD:-bromedia-secret}"
TURN_REALM="${TURN_REALM:-bromedia.local}"

TEMPLATE_FILE="/etc/coturn/turnserver.conf"
RUNTIME_CONF="/tmp/turnserver.conf"

# Create a writable runtime config from the read-only template.
cp "$TEMPLATE_FILE" "$RUNTIME_CONF"

# Patch config with runtime environment variables
sed -i "s/^user=.*/user=${TURN_USER}:${TURN_PASSWORD}/" "$RUNTIME_CONF"
sed -i "s/^realm=.*/realm=${TURN_REALM}/" "$RUNTIME_CONF"

echo "[entrypoint] Starting coturn with user=${TURN_USER}, realm=${TURN_REALM}"
exec turnserver -c "$RUNTIME_CONF"
