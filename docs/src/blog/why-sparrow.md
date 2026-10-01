---
title: Keep the record close
kicker: On self-hosting
author: Sparrow team
description: Why Sparrow is self-hosted first. A partner's endpoint goes dark overnight, and the next morning the only thing that matters is whether you can say what they missed.
pubDate: 2026-09-30
tags: [architecture, self-hosting]
---

At 02:10 on a Tuesday, a partner's webhook endpoint stops answering. Nobody notices. Orders keep coming in, your system keeps emitting `order.created`, and somewhere a request quietly times out every few seconds.

At 09:00 the partner's operations lead sends you one line: "Did we miss anything last night?"

It is a polite question, and it is also a test. Not of your code, exactly, but of your word. Every webhook is a small promise: *when this happens, I will tell you.* Most of the time nobody checks. Then one morning somebody does, and the only thing that matters is whether you can answer with evidence instead of reassurance.

## A promise you can't check is a hope

People who carried important messages have always kept a ledger. The courier who signs for a parcel, the clerk who stamps the date on a letter, the ship's log that records what was loaded and where. None of that paperwork moves anything. It exists so that later, when someone asks "did it arrive?", there is an answer that does not depend on anyone's memory.

Sending HTTP requests is the easy part of a webhook system. The ledger is the hard part. Which events went out, which ones failed, why they failed, which ones are still waiting. That record is what turns delivery from a hope into a promise.

## Keep it where the events live

If that ledger lives in someone else's system, answering the partner means filing a ticket, exporting logs from a vendor dashboard, or stitching together two half-stories that never quite agree.

Sparrow keeps the ledger next to the thing it describes. Every event is written to PostgreSQL before any delivery starts. Each attempt, its response code, and its error category are written there too. When the partner's endpoint went dark at 02:10, nothing was dropped on the floor. The deliveries held their place in the queue, backed off, and kept trying.

So the 09:00 answer is not "I think we're fine." It is: here are the deliveries that failed overnight, here is the timeout each one hit, and here is when each one finally landed. If any ran out of retries, you retry them by ID and watch them arrive while the partner is still on the call.

## Own only what you can carry

There is a quieter idea underneath self-hosting: you should not own more than you can look after. A system you run yourself has to be one you can actually understand at 02:10. So Sparrow stays small on purpose:

- one Go server;
- one PostgreSQL database;
- no Redis, no message broker, no object storage.

The job queue ([River](https://riverqueue.com)) runs inside PostgreSQL. There is one database to back up, one thing to monitor, one place to look when something goes wrong. Fewer moving parts means fewer places for the truth to hide.

## Some things should stay home

Order events carry names, amounts, and addresses. They belong to your customers, and you are holding them in trust. Handing them to a third-party relay means one more company holds that data, one more vendor review, one more place a breach can start.

Self-hosting keeps the payloads, the retry policy, and the history inside the network that already owns them. Sparrow is built for teams that run it behind their own VPN, for internal and partner delivery. It is deliberately not a multi-tenant SaaS, and that narrow focus is what lets it stay simple.

The next time someone asks "did we miss anything last night?", the honest answer should take one query and a few minutes. That is what it means to keep a promise in software: not that nothing ever fails, but that you can always say what happened. It is why Sparrow is self-hosted first.
