# Portal Tokens -- Minted Tokens Grant Scoped Gateway Access
Tags: portal

A portal token minted through the REST API must actually authenticate against
the live portal gateway and be scoped to the minting consumer's own data. The
gateway's internal path mapping and denials are unit-tested; this proves the
end-to-end wire path: mint -> use -> see own webhook, and reject the anonymous
caller.

## A Minted Token Sees Only Its Consumer's Webhooks
* Use consumer "portal"
* Start target "receiver"
* Register event type "portal.thing.happened"
* Register webhook "receiver" in current consumer subscribed to "portal.thing.happened"
* Mint portal token for current consumer
* Portal GET "webhooks" should return status "200"
* Portal webhooks list should contain webhook "receiver"
* Portal GET "webhooks" without token should return status "401"
