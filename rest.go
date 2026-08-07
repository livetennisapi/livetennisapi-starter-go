// REST example — the polling counterpart to the streaming loop.
//
// Run with `go run . -rest`. It lists currently-live matches and prints each
// one's set score and (ULTRA) win probability. Poll an endpoint like this on an
// interval to build a bot that doesn't need the WebSocket.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Only the documented fields we actually print are modelled here; the decoder
// ignores everything else, so new server-side fields never break this client.
type restPlayer struct {
	Name string `json:"name"`
}

type restScore struct {
	Sets      []int    `json:"sets"`  // [sets_p1, sets_p2]
	Games     [][]int  `json:"games"` // player-major: [[p1 per set], [p2 per set]]
	Server    int      `json:"server"`
	WinProbP1 *float64 `json:"win_probability_p1"` // ULTRA-only; nil otherwise
}

type restMatch struct {
	ID         int                   `json:"id"`
	Tournament string                `json:"tournament"`
	Status     string                `json:"status"`
	Players    map[string]restPlayer `json:"players"`
	Score      *restScore            `json:"score"`
}

// matchPage is the list envelope: {"data": [...], "meta": {...}}.
type matchPage struct {
	Data []restMatch `json:"data"`
}

func runREST(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, baseURL()+"/matches?status=live&limit=20", nil,
	)
	if err != nil {
		return err
	}
	// The API accepts either a Bearer token or an X-API-Key header.
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach the API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("the API rejected this key (401); check LIVETENNISAPI_KEY")
	case resp.StatusCode == http.StatusTooManyRequests:
		return rateLimitError(resp, body)
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("GET /matches returned %d: %s", resp.StatusCode, body)
	}

	var page matchPage
	if err := json.Unmarshal(body, &page); err != nil {
		return fmt.Errorf("could not decode response: %w", err)
	}
	if len(page.Data) == 0 {
		log.Println("no live matches right now — try again during play")
		return nil
	}

	log.Printf("%d live match(es):", len(page.Data))
	for _, m := range page.Data {
		line := fmt.Sprintf("  [%d] %s: %s vs %s",
			m.ID, m.Tournament, m.Players["p1"].Name, m.Players["p2"].Name)
		if m.Score != nil {
			line += fmt.Sprintf("  sets=%v  win_prob_p1=%s", m.Score.Sets, prob(m.Score.WinProbP1))
		}
		log.Println(line)
	}
	return nil
}

// rateLimitError turns a 429 into an actionable message. Two flavours share
// the status code, so switch on the body's error field: "rate_limited" is an
// ordinary per-minute or per-day cap (wait it out — honour Retry-After; a
// daily 429 also carries resets_at, the exact UTC instant the quota resets),
// while "abuse_throttled" is a ~24h block for clients that chronically ignore
// their caps — fix the polling/retry loop, don't retry harder.
func rateLimitError(resp *http.Response, body []byte) error {
	var e struct {
		Error        string `json:"error"`
		Scope        string `json:"scope"`     // "day" on a daily-cap 429
		ResetsAt     string `json:"resets_at"` // daily 429 only
		RetryAtEpoch int64  `json:"retry_at_epoch"`
	}
	_ = json.Unmarshal(body, &e)
	if e.Error == "abuse_throttled" {
		return fmt.Errorf(
			"abuse_throttled (429): this key is blocked for ~24h for chronically exceeding its caps; fix the polling/retry loop (retry_at_epoch=%d)",
			e.RetryAtEpoch)
	}
	msg := "rate_limited (429): over the per-minute cap"
	if e.Scope == "day" {
		msg = "rate_limited (429): the daily quota is used up"
		if e.ResetsAt != "" {
			msg += "; it resets at " + e.ResetsAt
		}
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		msg += " — retry after " + ra + "s"
	}
	return fmt.Errorf("%s", msg)
}
