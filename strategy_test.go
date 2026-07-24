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
