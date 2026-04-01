#!/bin/sh
set -e

TURN_USER="${TURN_USER:-bromedia}"
TURN_PASSWORD="${TURN_PASSWORD:-bromedia-secret}"
TURN_REALM="${TURN_REALM:-bromedia.local}"

CONF_FILE="/etc/coturn/turnserver.conf"

# Patch the config with runtime environment variables
sed -i "s/^user=.*/user=${TURN_USER}:${TURN_PASSWORD}/" "$CONF_FILE"
sed -i "s/^realm=.*/realm=${TURN_REALM}/" "$CONF_FILE"

echo "[entrypoint] Starting coturn with user=${TURN_USER}, realm=${TURN_REALM}"
exec turnserver -c "$CONF_FILE"
