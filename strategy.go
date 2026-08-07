// Your break-point strategy — the one file you are meant to edit.
//
// OnBreakPoint is where the headline signal lands. The default implementation
// decides on a paper action and hands it to placePaperOrder, which only logs.
// Nothing in this file places a real bet.
//
// The seam for real execution is Strategy.execute. It is deliberately left
// returning an error and is deliberately NOT called, so this starter can never
// move money. Wire your own exchange or venue there when you are ready — see the
// clearly marked block in placePaperOrder.
package main

import (
	"errors"
	"log"
)

// PaperOrder is an intended (paper) order. Logged, never sent.
type PaperOrder struct {
	MatchID int
	Side    int // which player (1 or 2) this order backs
	Stake   float64
	Reason  string
}

// Strategy is a minimal, illustrative break-point reactor.
//
// The logic here is intentionally simple — it exists to show *where* your code
// goes, not to be a profitable model. Replace decide with your own edge.
type Strategy struct {
	BaseStake       float64
	BreakPointsSeen int
}

// NewStrategy returns a strategy with the default 10-unit base stake.
func NewStrategy() *Strategy {
	return &Strategy{BaseStake: 10}
}

// -- event handlers ----------------------------------------------------------

// OnScore handles a routine score change. Kept quiet (DEBUG only) so break
// points stand out in the log.
func (s *Strategy) OnScore(f ScoreFrame) {
	if debugEnabled() {
		log.Printf("score  match=%d  sets=%v  win_prob_p1=%s", f.MatchID, f.Score.Sets, prob(f.Score.WinProbP1))
	}
}

// OnBreakPoint handles the headline signal: a break point is on the board.
func (s *Strategy) OnBreakPoint(f BreakPointFrame) {
	s.BreakPointsSeen++
	log.Printf(
		"BREAK POINT  match=%d  p%d serving, p%d holds %d break point(s)  swing=%s",
		f.MatchID, f.Server, f.Returner, f.BreakPoints, prob(f.ProbSwing),
	)
	if order := s.decide(f); order != nil {
		s.placePaperOrder(*order)
	}
}

// OnBreakPointResult handles a break point resolving (held or broken).
func (s *Strategy) OnBreakPointResult(f BreakPointResultFrame) {
	log.Printf(
		"  -> break point %s  match=%d  p1 win prob now %s",
		f.Outcome, f.MatchID, prob(f.WinProbP1After),
	)
}

// -- decision ----------------------------------------------------------------

// decide turns a break-point event into an intended paper order, or nil.
//
// Illustrative rule: when the serving side is *not* favoured to hold, back the
// returner to convert. Stake scales with how many break points are live. This
// is a placeholder — put your real edge here.
func (s *Strategy) decide(f BreakPointFrame) *PaperOrder {
	if f.ServerSideFavoured {
		return nil
	}
	if f.Returner != 1 && f.Returner != 2 {
		return nil
	}
	n := f.BreakPoints
	if n < 1 {
		n = 1
	}
	if n > 3 {
		n = 3
	}
	return &PaperOrder{
		MatchID: f.MatchID,
		Side:    f.Returner,
		Stake:   s.BaseStake * float64(n),
		Reason:  "break point(s) against an unfavoured server",
	}
}

// -- execution (paper only) --------------------------------------------------

// placePaperOrder logs the intended order. It places NO real bet.
func (s *Strategy) placePaperOrder(o PaperOrder) {
	log.Printf(
		"PAPER ORDER  back p%d on match %d for %.2f  (%s)",
		o.Side, o.MatchID, o.Stake, o.Reason,
	)

	// ================= WIRE YOUR OWN EXCHANGE / VENUE HERE =================
	// This starter intentionally stops at logging. To go live, implement
	// `execute` against your venue's API and call it here. It is left
	// uncalled on purpose so a fresh clone can never move real money.
	//
	//     if err := s.execute(o); err != nil { log.Println(err) }
	// ======================================================================
}

// execute is the seam for real order placement. Unimplemented by design: it
// returns an error and is never called, so a fresh clone places no real bets.
func (s *Strategy) execute(o PaperOrder) error {
	return errors.New("wire your real exchange/venue here; this starter places NO real bets")
}
