# Portal Tokens -- Minted Tokens Grant Scoped Gateway Access
Tags: portal

A portal token (a consumer token from POST /v1/tokens) must actually authenticate against
the live portal gateway and be scoped to the minting consumer's own data. The
gateway's internal path mapping and denials are unit-tested; this proves the
end-to-end wire path: mint -> use -> see own webhook, reject the anonymous
caller, and stop working once revoked.

## A Minted Token Sees Only Its Consumer's Webhooks
* Use consumer "portal"
* Start target "receiver"
* Register event type "portal.thing.happened"
* Register webhook "receiver" in current consumer subscribed to "portal.thing.happened"
* Mint portal token for current consumer
* Portal GET "webhooks" should return status "200"
* Portal webhooks list should contain webhook "receiver"
* Portal GET "webhooks" without token should return status "401"

## A Revoked Portal Token Stops Working
* Use consumer "portal-revoke"
* Mint portal token for current consumer
* Portal GET "webhooks" should return status "200"
* Revoke the portal token
* Portal GET "webhooks" should return status "401"

## Minting With An External Id Returns The Valid Token
* Use consumer "portal-idem"
* Mint portal token for current consumer with external id "user-1"
* Mint portal token for current consumer with external id "user-1" should reuse the previous token
* Revoke the portal token
* Mint portal token for current consumer with external id "user-1" should mint a new token
