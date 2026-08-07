// WebSocket live-score feed. ULTRA tier only.
//
// The feed pushes a "score" frame whenever a subscribed match's score changes
// (the payload nests under "score" and carries the ULTRA model fields
// win_probability_p1 and danger), plus a "ping" heartbeat roughly every 15s.
// With signals=["break_point"] it also pushes a "break_point" frame the instant
// a break point arises and a "break_point_result" frame when it resolves.
//
// This starter keeps a single connection for clarity — a clear "here's the bot
// loop", not a full SDK (the server allows at most 2 concurrent connections
// per key). Production code should add reconnect-with-backoff; the official
// Python and JS SDKs do this for you.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coder/websocket"
)

const defaultBaseURL = "https://api.livetennisapi.com/api/public/v1"

// baseURL is the REST base; override with LIVETENNISAPI_BASE_URL for staging.
func baseURL() string {
	if v := strings.TrimRight(os.Getenv("LIVETENNISAPI_BASE_URL"), "/"); v != "" {
		return v
	}
	return defaultBaseURL
}

// wsURL derives the wss:// endpoint from the REST base: https->wss, http->ws,
// append /ws, and carry the key as the ?token= query param. (The browser
// WebSocket API can't set headers, so the server accepts the key in the query;
// over TLS it's encrypted in transit but can land in logs — prefer a scoped
// key for streaming.)
func wsURL(base, key string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/ws"
	q := url.Values{}
	if key != "" {
		q.Set("token", key)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// feedError is a server-sent "error" frame that reconnecting can't fix
// (a bad key, a tier without WebSocket access, the service being disabled).
type feedError struct {
	code string
	msg  string
}

func (e *feedError) Error() string { return e.msg }

func frameError(raw []byte) error {
	var f struct {
		Error string `json:"error"`
		Hint  string `json:"hint"`
	}
	_ = json.Unmarshal(raw, &f)
	code := f.Error
	if code == "" {
		code = "error"
	}
	msg := "live feed error: " + code
	if f.Hint != "" {
		msg += " — " + f.Hint
	}
	return &feedError{code: code, msg: msg}
}

// stream opens the feed, subscribes, and routes every frame to the handler
// until the context is cancelled (Ctrl-C) or the socket closes.
func stream(ctx context.Context, key string, h handler) error {
	target, err := wsURL(baseURL(), key)
	if err != nil {
		return err
	}

	c, _, err := websocket.Dial(ctx, target, &websocket.DialOptions{
		HTTPHeader: http.Header{"User-Agent": {userAgent}},
	})
	if err != nil {
		return fmt.Errorf("could not open the live feed: %w", err)
	}
	defer c.CloseNow()
	c.SetReadLimit(1 << 20)

	// Subscribe immediately — the server closes the socket if no subscribe
	// frame arrives within ~15s. The frame is just {"topics": [...]} plus the
	// optional "signals" list; anything else in it is ignored. Swap topics to
	// ["match:<id>"] to follow one match.
	sub, _ := json.Marshal(map[string]any{
		"topics":  []string{"live-scores"},
		"signals": []string{"break_point"},
	})
	if err := c.Write(ctx, websocket.MessageText, sub); err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	for {
		_, raw, err := c.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown on Ctrl-C
			}
			return fmt.Errorf("live feed closed: %w", err)
		}

		var env envelope
		if json.Unmarshal(raw, &env) != nil {
			continue
		}
		switch env.Type {
		case "subscribed":
			log.Println("subscribed — waiting for score + break-point frames …")
		case "error":
			return frameError(raw)
		case "ping":
			// heartbeat — ignore
		default:
			dispatch(raw, h)
		}
	}
}
