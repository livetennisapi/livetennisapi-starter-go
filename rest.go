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
