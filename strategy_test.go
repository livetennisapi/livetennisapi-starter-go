package main

import (
	"reflect"
	"strings"
	"testing"
)

// spy records which handler each frame is routed to, with no strategy logic —
// the same shape as the Python/Node starter tests.
type spy struct{ calls []string }

func (s *spy) OnScore(ScoreFrame)           { s.calls = append(s.calls, "score") }
func (s *spy) OnBreakPoint(BreakPointFrame) { s.calls = append(s.calls, "break_point") }
func (s *spy) OnBreakPointResult(BreakPointResultFrame) {
	s.calls = append(s.calls, "break_point_result")
}

func TestDispatchRoutesEachFrameTypeAndIgnoresNoise(t *testing.T) {
	sp := &spy{}
	for _, m := range [][]byte{
		[]byte(`{"type":"score","match_id":1}`),
		[]byte(`{"type":"break_point","match_id":1,"returner":2,"break_points":1}`),
		[]byte(`{"type":"break_point_result","match_id":1,"outcome":"broken"}`),
		[]byte(`{"type":"ping"}`),       // ignored
		[]byte(`{"type":"subscribed"}`), // ignored
		[]byte(`not json`),              // dropped, not fatal
	} {
		dispatch(m, sp)
	}
	want := []string{"score", "break_point", "break_point_result"}
	if !reflect.DeepEqual(sp.calls, want) {
		t.Fatalf("routed %v, want %v", sp.calls, want)
	}
}

// recorder keeps the decoded frames so tests can assert on wire-shape parsing.
type recorder struct {
	scores []ScoreFrame
	bps    []BreakPointFrame
}

func (r *recorder) OnScore(f ScoreFrame)                     { r.scores = append(r.scores, f) }
func (r *recorder) OnBreakPoint(f BreakPointFrame)           { r.bps = append(r.bps, f) }
func (r *recorder) OnBreakPointResult(BreakPointResultFrame) {}

// The score payload nests under "score" and carries the ULTRA model fields —
// exactly what the server sends (live as of 2026-08-07).
func TestScoreFrameParsesNestedPayloadWithModelFields(t *testing.T) {
	rec := &recorder{}
	dispatch([]byte(`{"type":"score","match_id":42,"score":{"sets":[1,0],"games":[[6,2],[4,0]],"points":["40","30"],"server":1,"is_tiebreak":false,"timestamp":"2026-08-07T12:00:00Z","win_probability_p1":0.71,"danger":0.22}}`), rec)
	if len(rec.scores) != 1 {
		t.Fatalf("expected 1 score frame, got %d", len(rec.scores))
	}
	f := rec.scores[0]
	if f.MatchID != 42 || !reflect.DeepEqual(f.Score.Sets, []int{1, 0}) || f.Score.Server != 1 {
		t.Fatalf("nested score payload not decoded: %+v", f)
	}
	if f.Score.WinProbP1 == nil || *f.Score.WinProbP1 != 0.71 || f.Score.Danger == nil {
		t.Fatalf("model fields must ride on the frame: %+v", f.Score)
	}
}

// Regression: "set" and "game" arrive as strings ("1-0", "4-5"). Typing them
// as ints made json.Unmarshal fail, which silently dropped every break_point
// frame in dispatch.
func TestBreakPointFrameParsesStringSetAndGame(t *testing.T) {
	rec := &recorder{}
	dispatch([]byte(`{"type":"break_point","match_id":7,"server":1,"returner":2,"break_points":2,"set":"1-0","game":"4-5","point":"30-40","win_probability_p1":0.41,"prob_swing":0.18,"server_side_favoured":false,"ts":"2026-08-07T12:00:00Z"}`), rec)
	if len(rec.bps) != 1 {
		t.Fatalf("break_point frame was dropped instead of routed")
	}
	f := rec.bps[0]
	if f.Set != "1-0" || f.Game != "4-5" || f.BreakPoints != 2 {
		t.Fatalf("break_point fields not decoded: %+v", f)
	}
}

func TestDecideBacksReturnerWhenServerNotFavoured(t *testing.T) {
	order := NewStrategy().decide(BreakPointFrame{
		MatchID: 5, Returner: 2, BreakPoints: 2, ServerSideFavoured: false,
	})
	if order == nil {
		t.Fatal("expected a paper order")
	}
	if order.Side != 2 || order.MatchID != 5 || order.Stake != 20 { // base 10 * 2 break points
		t.Fatalf("unexpected order %+v", *order)
	}
}

func TestDecideStandsAsideWhenServerFavoured(t *testing.T) {
	if order := NewStrategy().decide(BreakPointFrame{
		Returner: 2, ServerSideFavoured: true, BreakPoints: 1,
	}); order != nil {
		t.Fatalf("expected nil, got %+v", *order)
	}
}

func TestDecideIgnoresAnUnknownReturner(t *testing.T) {
	if order := NewStrategy().decide(BreakPointFrame{Returner: 0, BreakPoints: 1}); order != nil {
		t.Fatalf("expected nil for an unknown returner, got %+v", *order)
	}
}

// The load-bearing safety property: the real-execution seam is never wired.
func TestExecuteSeamRefusesToPlaceARealBet(t *testing.T) {
	err := NewStrategy().execute(PaperOrder{MatchID: 1, Side: 1, Stake: 10})
	if err == nil {
		t.Fatal("execute must not place a real bet")
	}
	if !strings.Contains(err.Error(), "NO real bets") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// wsURL derivation is load-bearing protocol correctness (https->wss, /ws, token).
func TestWSURLDerivation(t *testing.T) {
	got, err := wsURL("https://api.livetennisapi.com/api/public/v1", "twjp_abc")
	if err != nil {
		t.Fatal(err)
	}
	want := "wss://api.livetennisapi.com/api/public/v1/ws?token=twjp_abc"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
