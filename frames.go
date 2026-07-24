// Frame types and routing for the live feed.
//
// Every WebSocket message is a JSON object with a "type" discriminator. The
// break-point and score fields sit inline on the frame (there is no nested
// score object on the stream), so each frame type is a flat struct. Decoding is
// deliberately tolerant: unknown fields are ignored, absent fields stay at their
// zero value — the server ships additive changes within v1, so an old client
// must never choke on a new field.
package main

import "encoding/json"

// envelope peeks only at the discriminator so we can route to the right struct.
type envelope struct {
	Type string `json:"type"`
}

// ScoreFrame is one "score" frame: a subscribed match's score changed.
// win_probability_p1 and danger are ULTRA-only and may be absent (nil).
type ScoreFrame struct {
	Type      string   `json:"type"`
	MatchID   int      `json:"match_id"`
	Sets      []int    `json:"sets"`   // [sets_p1, sets_p2]
	Games     [][]int  `json:"games"`  // player-major: [[p1 per set], [p2 per set]]
	Points    []string `json:"points"` // [points_p1, points_p2], e.g. ["40","30"]
	Server    int      `json:"server"` // player currently serving (1 or 2)
	WinProbP1 *float64 `json:"win_probability_p1"`
	Danger    *float64 `json:"danger"`
}

// BreakPointFrame is one "break_point" frame — a break point is on the board.
// server is the player serving; returner holds the break point(s). break_points
// is how many are live at once (1-3). prob_swing is the same quantity the REST
// score calls "danger". Opt-in signal; ULTRA-only.
type BreakPointFrame struct {
	Type               string   `json:"type"`
	MatchID            int      `json:"match_id"`
	Server             int      `json:"server"`
	Returner           int      `json:"returner"`
	BreakPoints        int      `json:"break_points"`
	Set                int      `json:"set"`
	Game               int      `json:"game"`
	Point              string   `json:"point"`
	WinProbP1          *float64 `json:"win_probability_p1"`
	ProbSwing          *float64 `json:"prob_swing"`
	ServerSideFavoured bool     `json:"server_side_favoured"`
	TS                 string   `json:"ts"`
}

// BreakPointResultFrame is one "break_point_result" frame — a break point just
// resolved. outcome is "held" (server saved it) or "broken" (returner
// converted). WinProbP1After is p1's win probability once the game closed.
// Opt-in signal; ULTRA-only.
type BreakPointResultFrame struct {
	Type           string   `json:"type"`
	MatchID        int      `json:"match_id"`
	Server         int      `json:"server"`
	Outcome        string   `json:"outcome"`
	WinProbP1After *float64 `json:"win_probability_p1_after"`
	TS             string   `json:"ts"`
}

// handler is the seam the strategy implements. Kept as an interface so dispatch
// can be tested against a spy with no socket and no strategy logic.
type handler interface {
	OnScore(ScoreFrame)
	OnBreakPoint(BreakPointFrame)
	OnBreakPointResult(BreakPointResultFrame)
}

// dispatch routes one raw JSON message to the matching handler. "ping" and
// "subscribed" are protocol noise and are ignored, as is anything unknown.
// A message that is not valid JSON is dropped rather than fatal.
func dispatch(raw []byte, h handler) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return
	}
	switch env.Type {
	case "score":
		var f ScoreFrame
		if json.Unmarshal(raw, &f) == nil {
			h.OnScore(f)
		}
	case "break_point":
		var f BreakPointFrame
		if json.Unmarshal(raw, &f) == nil {
			h.OnBreakPoint(f)
		}
	case "break_point_result":
		var f BreakPointResultFrame
		if json.Unmarshal(raw, &f) == nil {
			h.OnBreakPointResult(f)
		}
	}
}

// prob renders an optional probability for logging: "0.63" or "n/a" (FREE/PRO
// keys don't receive win_probability, so nil is expected, not an error).
func prob(p *float64) string {
	if p == nil {
		return "n/a"
	}
	return trimFloat(*p)
}
