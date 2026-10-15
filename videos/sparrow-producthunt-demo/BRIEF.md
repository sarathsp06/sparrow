---
workflow: product-launch-video
flow: automation
storyboard: no
message: "Self-hosted webhook delivery you can run, see, and trust in minutes"
destination: youtube
aspect: 1920x1080
language: en
audience: developers and platform teams browsing Product Hunt
length: 40s
angle: show-it-as-is product demo
vo_mode: none
---

## Intent

A 30–45 second Product Hunt demo video for Sparrow, a self-hosted, Postgres-only
webhook delivery server. Show the product as it is: real dashboard screens
captured from the dev instance at http://localhost:8080 plus terminal scenes
built from the real `sparrow` CLI output. Dark, confident, developer-native. No
narration; music bed only if available, otherwise silent with on-screen copy.

## Assets

- capture/* — screenshots of the live dashboard at localhost:8080 (deliveries,
  webhooks with health badges, template editor with AI drawer, portal view).
- web/src/lib/assets/favicon.svg — Sparrow logo mark (repo).

## Customizations

- Terminal scenes rendered as styled terminal windows (registry block when one
  fits) carrying the real CLI commands and output.
- Title card and closing card per the user's script; GitHub URL
  github.com/sarathsp06/sparrow and docs URL on the close.

## Notes

- User script (verbatim) is saved in `user_script.txt`; it is scene direction,
  not voice-over. Keep its scene order: title → start server → register and
  deliver → transform and route → health and portal → closing.
- Only show real things: Grafana panel is optional and is skipped unless real.
- Format 16:9 for Product Hunt's video embed (YouTube).
