# Access Tokens and Invites
Tags: auth, access

Named access tokens and one-time invites (pkg/access) against a live server
with SPARROW_API_KEY set. A tenant-wide token works exactly like the master
key; a consumer token only works through the portal gateway; invites are
single use; revocation and cancellation take effect; and rotating the master
key does not sign anyone out.

## A Tenant-Wide Token Works Like The Master Key
* Start authed sparrow server
* Create access token "e2e-ci" with the master key
* GET "/v1/event-types" with access token "e2e-ci" should return status "200"
* Whoami with access token "e2e-ci" should report name "e2e-ci"

## A Revoked Token Is Rejected With Reason Revoked
* Start authed sparrow server
* Create access token "e2e-revoke" with the master key
* Revoke access token "e2e-revoke"
* GET "/v1/event-types" with access token "e2e-revoke" should be rejected with reason "revoked"

## An Invite Is Single Use
* Start authed sparrow server
* Create invite "e2e-alice" with the master key
* Redeem invite "e2e-alice" should succeed as token "e2e-alice-token"
* GET "/v1/event-types" with access token "e2e-alice-token" should return status "200"
* Redeem invite "e2e-alice" should fail

## A Cancelled Invite Cannot Be Redeemed
* Start authed sparrow server
* Create invite "e2e-bob" with the master key
* Cancel invite "e2e-bob"
* Redeem invite "e2e-bob" should fail

## A Consumer Token Is Pinned To Its Consumer's Portal
* Start authed sparrow server
* Create access token "e2e-portal" for consumer "e2e-acme" with the master key
* GET "/v1/event-types" with access token "e2e-portal" should return status "403"
* GET "/portal/api/webhooks" with access token "e2e-portal" should return status "200"
* Create access token "e2e-full" with the master key
* GET "/portal/api/webhooks" with access token "e2e-full" should return status "403"

## Rotating The Master Key Keeps Tokens Working
* Start authed sparrow server
* Create access token "e2e-survivor" with the master key
* Restart authed sparrow server with master key "e2e-rotated-key"
* GET "/v1/event-types" with access token "e2e-survivor" should return status "200"
* GET "/v1/event-types" on authed server with key "e2e-secret-key" should return status "401"
* Stop authed sparrow server
