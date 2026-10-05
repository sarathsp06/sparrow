package client

import (
	"strings"

	"github.com/google/uuid"
)

// ListenScheme is the URL scheme of a listen session webhook
// (sparrow-cli://<webhook id>). It is never dialed: the delivery worker hands
// such deliveries to the CLI that polls for them. Webhook URL validation only
// accepts http and https, so the API cannot register one directly.
const ListenScheme = "sparrow-cli"

// ListenURL is the URL of the listen session webhook webhookID.
func ListenURL(webhookID uuid.UUID) string {
	return ListenScheme + "://" + webhookID.String()
}

// IsListenURL reports whether rawURL is a listen session webhook URL.
func IsListenURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, ListenScheme+"://")
}
