# Live Tennis API — Break-Point Starter (Go)

A tiny, runnable app that reacts to **break points** on the
[Live Tennis API](https://livetennisapi.com) live feed. It shows exactly where
your trading logic goes — and it places **no real bets**.

There is no official Go SDK, so this starter talks to the API directly: it opens
the ULTRA WebSocket feed with `signals: ["break_point"]`, routes every frame to
a `Strategy`, and logs the paper action the strategy would take. A `-rest` flag
runs the polling counterpart. Its only dependency is a WebSocket library
([`github.com/coder/websocket`](https://github.com/coder/websocket)).

> **Requires an ULTRA key.** The WebSocket feed and the break-point signals are
> ULTRA-tier only. Get a key at <https://livetennisapi.com/#pricing>.
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

The subscribe frame this starter sends after connecting:

```json
{ "action": "subscribe", "topics": ["live-scores"], "signals": ["break_point"] }
```

Swap the topic to `["match:<id>"]` to follow one match. Frames it reacts to:
`score`, `break_point`, `break_point_result` (`ping` and `subscribed` are
ignored). See the
[WebSocket section of the API reference](https://docs.livetennisapi.com/reference.html#websocket).

The REST example (`go run . -rest`) calls `GET /matches?status=live` and prints
each live match's set score and (ULTRA) win probability — poll an endpoint like
that on an interval for a bot that doesn't need the WebSocket.

## License

MIT — see [LICENSE](./LICENSE).
