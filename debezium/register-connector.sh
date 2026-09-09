#!/bin/sh

set -e

DEBEZIUM_URL="http://${DEBEZIUM_HOST:-debezium}:${DEBEZIUM_PORT:-8083}"

echo "Waiting for Debezium Connect to become ready at ${DEBEZIUM_URL}..."

while true; do
  status_code=$(curl -s -o /dev/null -w "%{http_code}" "${DEBEZIUM_URL}/connectors" || true)
  if [ "$status_code" = "200" ]; then
    echo "Debezium Connect is ready!"
    break
  fi
  echo "Debezium not ready yet (HTTP $status_code). Retrying in 3 seconds..."
  sleep 3
done

echo "Registering Archon Postgres Outbox CDC Connector..."

CONNECTOR_CONFIG=$(cat <<EOF
{
  "name": "archon-postgres-outbox-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "tasks.max": "1",
    "plugin.name": "pgoutput",
    "database.hostname": "${DB_HOST:-db}",
    "database.port": "${DB_PORT:-5432}",
    "database.user": "${DB_USER}",
    "database.password": "${DB_PASSWORD}",
    "database.dbname": "${DB_NAME}",
    "database.server.name": "archon_pg",
    "topic.prefix": "archon_cdc",
    "table.include.list": "public.outbox_events",
    "tombstones.on.delete": "false",
    "publication.autocreate.mode": "all_tables",
    "slot.name": "debezium_archon_outbox_slot",
    "transforms": "outbox",
    "transforms.outbox.type": "io.debezium.transforms.outbox.EventRouter",
    "transforms.outbox.route.topic.replacement": "ai.requests",
    "transforms.outbox.table.fields.additional.placement": "type:header:eventType"
  }
}
EOF
)

RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  --data "${CONNECTOR_CONFIG}" \
  "${DEBEZIUM_URL}/connectors")

echo "Response from Debezium: $RESPONSE"

echo "Checking connector status..."
curl -s "${DEBEZIUM_URL}/connectors/archon-postgres-outbox-connector/status"
echo ""
echo "Connector registered successfully."
