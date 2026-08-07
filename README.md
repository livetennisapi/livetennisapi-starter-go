# Live Tennis API — Break-Point Starter (Go)

[![ci](https://github.com/livetennisapi/livetennisapi-starter-go/actions/workflows/ci.yml/badge.svg)](https://github.com/livetennisapi/livetennisapi-starter-go/actions/workflows/ci.yml)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

A tiny, runnable app that reacts to **break points** on the
[Live Tennis API](https://livetennisapi.com) live feed — ATP, WTA, Challenger,
ITF and juniors. It shows exactly where your trading logic goes — and it places
**no real bets**.

There is no official Go SDK, so this starter talks to the API directly: it opens
the ULTRA WebSocket feed with `signals: ["break_point"]`, routes every frame to
a `Strategy`, and logs the paper action the strategy would take. A `-rest` flag
runs the polling counterpart. Its only dependency is a WebSocket library
([`github.com/coder/websocket`](https://github.com/coder/websocket)).

> **Requires an ULTRA key.** The WebSocket feed and the break-point signals are
> ULTRA-tier only. Get a key at <https://livetennisapi.com/#pricing>.
> No key yet? A **FREE** key (no card — <https://livetennisapi.com/subscribe/free>)
> lets you explore the REST endpoints first, but this starter's WebSocket feed
> still needs ULTRA.
>
> Needs **Go 1.23+**.

## Run it

```bash
cp .env.example .env          # then put your ULTRA key in .env
go run .                      # live WebSocket stream (break-point reactor)
go run . -rest                # REST example: list live matches + win probability
```

You'll see output like:

```
15:04:05 connecting to the live feed with signals=[break_point] …
15:04:05 subscribed — waiting for score + break-point frames …
15:04:12 BREAK POINT  match=18953  p1 serving, p2 holds 2 break point(s)  swing=0.22
15:04:12 PAPER ORDER  back p2 on match 18953 for 20.00  (break point(s) against an unfavoured server)
15:04:29   -> break point broken  match=18953  p1 win prob now 0.63
```

Stop it with Ctrl-C.

## Where your code goes

Everything you edit is in **`strategy.go`**:

- `OnBreakPoint(f)` — the headline signal lands here.
- `decide(f)` — return a `*PaperOrder` (or `nil`). Put your edge here.
- `placePaperOrder(o)` — logs the intended order. **Nothing real happens.**

The rest is plumbing: `stream.go` (WebSocket connect + subscribe), `frames.go`
(frame types + routing), `rest.go` (the REST example), `main.go` (wiring).

### Wiring a real venue

`placePaperOrder` contains a clearly marked block:

```
// ================= WIRE YOUR OWN EXCHANGE / VENUE HERE =================
```

The real-execution seam is `Strategy.execute`. It is **left returning an error
and never called on purpose**, so a fresh clone can never move money. To go
live, implement `execute` against your venue's API and call it from
`placePaperOrder`. That choice — and any risk it carries — is entirely yours.

## Test

```bash
go test ./...     # no network: canned frames through the real dispatch path
```

The tests pump canned frames through the same routing the live app uses, and
assert the safety invariant: the execution seam refuses to place a real bet.

## How it maps to the API

REST base: `https://api.livetennisapi.com/api/public/v1`
WebSocket: `wss://api.livetennisapi.com/api/public/v1/ws?token=<key>`

Auth: `Authorization: Bearer twjp_…` (preferred) or an `X-API-Key` header on
REST; the WebSocket carries the key as `?token=` because browser WebSocket
clients cannot set headers.

The subscribe frame this starter sends after connecting:

```json
{ "topics": ["live-scores"], "signals": ["break_point"] }
```

The server keys off `topics` (plus the optional `signals` list) — nothing else
belongs in the frame. Swap the topic to `["match:<id>"]` to follow one match.
Frames it reacts to: `score`, `break_point`, `break_point_result` (the ~15s
`ping` heartbeat and the `subscribed` ack are ignored). Every `score` frame
nests its payload under `score` and carries the ULTRA model fields
`win_probability_p1` and `danger` — a `null` there means the model had no
output for that state, not that the feed withheld it. The feed allows at most
**2 concurrent connections per key**. See the
[WebSocket section of the API reference](https://docs.livetennisapi.com/reference.html#websocket).

The REST example (`go run . -rest`) calls `GET /matches?status=live` and prints
each live match's set score and (ULTRA) win probability — poll an endpoint like
that on an interval for a bot that doesn't need the WebSocket. On a FREE key
(100 requests/day) poll no faster than every 15 minutes; an always-on dashboard
should run on BASIC or above.

## Error handling

- `401 unauthorized` — the key is wrong; check `LIVETENNISAPI_KEY`.
- `403 upgrade_required` — the endpoint (or the WebSocket) is above your tier;
  the body carries the upgrade URL. Never a silent empty result.
- `429 rate_limited` — over the per-minute or per-day cap. Honour `Retry-After`;
  a daily 429 also carries `resets_at`, the exact UTC instant your quota resets.
- `429 abuse_throttled` — a ~24-hour block for clients that chronically ignore
  their caps; the body's `retry_at_epoch` says when it lifts. Fix the polling or
  retry loop rather than retrying harder — `rest.go` shows how to tell the two
  429s apart.

Every REST response carries `X-RateLimit-Limit` / `-Remaining` / `-Reset`
headers.

## Quotas

| Tier | Rate limit | Price |
|---|---|---|
| FREE | 30/min · 100/day | $0 — no card |
| BASIC | 60/min · 1,000/day | $9.99/mo |
| PRO | 300/min · 10,000/day | $29.99/mo |
| ULTRA | 600/min · 500,000/day | $99.99/mo |

The WebSocket feed and the break-point signals need ULTRA; `/usage` reports
your key's consumption without counting against it.

## Links

[Documentation](https://docs.livetennisapi.com) ·
[Free API key](https://livetennisapi.com/subscribe/free) ·
[Discord](https://discord.gg/f8WUZHgDm6) ·
[GitHub org](https://github.com/livetennisapi)

## License

MIT — see [LICENSE](./LICENSE).

## Affiliate program

Know developers who need tennis data? The [affiliate program](https://affiliates.livetennisapi.com/program) pays 51% recurring commission for the life of every referred subscription — 30-day cookie, and the people you refer get 10% off.
