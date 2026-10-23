#!/usr/bin/env sh
# Seeds a demo consumer through the admin ("VPN") side and prints a portal
# link for the public side. Run after `docker compose up -d`.
#
#   ./try.sh [consumer]        default consumer: acme
#
# Creates two webhooks on the demo receiver (once): /hook answers 200, /fail
# answers 410. Pushes three order.created events so the portal has
# successful and failed deliveries to show, then mints a
# consumer token. Your app does that last step (POST /v1/tokens) behind its
# own login; the admin key never leaves the server.
set -eu

CONSUMER="${1:-acme}"
ADMIN="http://localhost:${SPARROW_ADMIN_PORT:-8180}"
PORTAL="http://localhost:${PORTAL_PORT:-8181}"
KEY="${SPARROW_API_KEY:-local-admin-key}"

api() { # api METHOD PATH [JSON]
	curl -fsS -X "$1" "$ADMIN$2" -H "X-API-Key: $KEY" -H "Content-Type: application/json" ${3:+-d "$3"}
}

# Talk only to this stack's Sparrow, never to another server on the port.
cd "$(dirname "$0")"
if ! docker compose ps --status running --services 2>/dev/null | grep -qx sparrow; then
	echo "This stack's sparrow service is not running: docker compose up -d" >&2
	exit 1
fi

printf 'Waiting for Sparrow'
until curl -fsS "$ADMIN/ready" >/dev/null 2>&1; do printf '.'; sleep 1; done
echo

hooks=$(api GET "/v1/consumers/$CONSUMER/webhooks")
for url in http://receiver/hook http://receiver/fail; do
	case "$hooks" in *"\"$url\""*) continue ;; esac
	api POST "/v1/consumers/$CONSUMER/webhooks" \
		"{\"url\":\"$url\",\"description\":\"Demo receiver\",\"events\":[\"order.created\"]}" >/dev/null
	echo "Created webhook $url for $CONSUMER"
done

for i in 1 2 3; do
	api POST "/v1/consumers/$CONSUMER/events?event=order.created" "{\"payload\":{\"order_id\":$i}}" >/dev/null
done
echo "Pushed 3 order.created events: delivered to /hook, failed (410) at /fail"

path=$(api POST /v1/tokens "{\"name\":\"try.sh\",\"consumer\":\"$CONSUMER\",\"ttl_seconds\":86400}" |
	sed -n 's/.*"portal_path": *"\([^"]*\)".*/\1/p' | sed 's/\\u0026/\&/g')
if [ -z "$path" ]; then
	echo "Could not read portal_path from POST /v1/tokens" >&2
	exit 1
fi
echo
echo "Portal for $CONSUMER (valid 24h):"
echo "  $PORTAL$path"
echo
echo "Through the proxy only the portal answers; these are 404 from Caddy:"
echo "  $PORTAL/  $PORTAL/v1/webhooks  $PORTAL/docs"
