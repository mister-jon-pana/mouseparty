package apps

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// playOut: one player joins and starts; every turn is the bots' (theirs timing out
// too), until the game is over; every view must encode and hide the others' hands
func playOut(t *testing.T, name string, phase func() string, act Activity, ticks int) {
	t.Helper()
	r, _ := testRoom(ann, bob)
	act.Message(r, ann, Msg{"kind": "join"})
	act.Message(r, ann, Msg{"kind": "start"})
	if phase() == "lobby" {
		t.Fatalf("%s didn't start", name)
	}
	for i := 0; i < ticks && phase() != "over"; i++ {
		act.Tick(r, later(time.Hour))
		for _, v := range []Viewer{ann, bob} {
			if _, err := json.Marshal(act.View(v)); err != nil {
				t.Fatalf("%s: a view won't encode: %v", name, err)
			}
		}
	}
	if phase() != "over" {
		t.Fatalf("%s: still %s after %d turns", name, phase(), ticks)
	}
	act.Tick(r, later(time.Hour))
	if phase() != "lobby" {
		t.Fatalf("%s: not back to the lobby", name)
	}
}

func TestHearts(t *testing.T) {
	r, _ := testRoom(ann)
	g := newHearts(r).(*hearts)
	playOut(t, "hearts", func() string { return g.phase }, g, 20000)
	total := 0
	for _, s := range g.seats {
		total += s.Score
	}
	if total%26 != 0 || total == 0 {
		t.Fatal("hearts' points don't add up:", total)
	}

	// a hand by hand: the two of clubs leads, and suit must be followed
	g = newHearts(r).(*hearts)
	g.Message(r, ann, Msg{"kind": "join"})
	g.Message(r, ann, Msg{"kind": "start"})
	me := seatOf(g.seats, ann.ID)
	if g.phase != "pass" || len(g.hands[me]) != 13 {
		t.Fatal(g.phase)
	}
	g.Message(r, ann, Msg{"kind": "pass", "cards": []any{g.hands[me][0], g.hands[me][0], g.hands[me][1]}})
	if g.passing[me] != nil {
		t.Fatal("passed one card twice")
	}
	g.Message(r, ann, Msg{"kind": "pass", "cards": []any{g.hands[me][0], g.hands[me][1], g.hands[me][2]}})
	if g.phase != "play" || len(g.hands[me]) != 13 {
		t.Fatal("the pass didn't go round:", g.phase)
	}
	if !contains(g.hands[g.turn], "2C") || len(g.legal(g.turn)) != 1 {
		t.Fatal("the two of clubs doesn't lead")
	}
	if _, ok := g.View(bob).(map[string]any)["hand"]; ok {
		t.Fatal("a watcher sees a hand")
	}
}

func TestColorMatch(t *testing.T) {
	for range 20 {
		r, _ := testRoom(ann)
		g := newColorMatch(r).(*colorMatch)
		playOut(t, "colormatch", func() string { return g.phase }, g, 5000)
	}
	r, _ := testRoom(ann)
	g := newColorMatch(r).(*colorMatch)
	g.Message(r, ann, Msg{"kind": "join"})
	g.Message(r, ann, Msg{"kind": "start"})
	if len(g.seats) != 2 || len(g.hands[0]) != 7 || g.top()[0] == 'W' {
		t.Fatal("not dealt")
	}
	g.pile = append(g.pile, "R5")
	g.color = 'R'
	if !g.playable("R9") || !g.playable("G5") || g.playable("G9") || !g.playable("W4") {
		t.Fatal("matching's wrong")
	}
}

func TestCribbage(t *testing.T) {
	for _, c := range []struct {
		hand    []string
		starter string
		crib    bool
		want    int
	}{
		{[]string{"5H", "5C", "5S", "JD"}, "5D", false, 29},
		{[]string{"4H", "5H", "6H", "7H"}, "KS", false, 4 + 2*2 + 4}, // a run of 4, 4+5+6 and 5+... fifteens, flush
		{[]string{"2C", "3C", "4C", "6C"}, "TD", true, 4 + 3},        // the crib's flush wants the starter too
		{[]string{"AH", "AS", "2D", "3C"}, "3H", false, 4 + 12},      // two pairs, double-double run
	} {
		got, why := handScore(c.hand, c.starter, c.crib)
		if got != c.want {
			t.Errorf("%v + %s: %d (%v), not %d", c.hand, c.starter, got, why, c.want)
		}
	}
	if p, _ := pegPoints([]string{"5H", "TD"}, 15); p != 2 {
		t.Fatal("fifteen pegs", p)
	}
	if p, _ := pegPoints([]string{"7H", "5D", "6C"}, 18); p != 3 {
		t.Fatal("a run pegs", p)
	}
	for n := range 3 {
		here := []Viewer{ann, bob, cat}[:n+1]
		r, _ := testRoom(here...)
		g := newCribbage(r).(*cribbage)
		for _, v := range here {
			g.Message(r, v, Msg{"kind": "join"})
		}
		g.Message(r, ann, Msg{"kind": "start"})
		for i := 0; i < 100000 && g.phase != "over"; i++ {
			g.Tick(r, later(time.Hour))
		}
		if g.phase != "over" {
			t.Fatal("cribbage never ends with", len(g.seats))
		}
	}
	r, _ := testRoom(ann, bob, cat, dan)
	g := newCribbage(r).(*cribbage)
	for _, v := range []Viewer{ann, bob, cat, dan} {
		g.Message(r, v, Msg{"kind": "join"})
	}
	g.Message(r, ann, Msg{"kind": "start"})
	if len(g.seats) != 4 || len(g.crib) != 0 || len(g.hands[0]) != 5 {
		t.Fatal("four aren't dealt five", len(g.hands[0]))
	}
	for i := 0; i < 100000 && g.phase != "over"; i++ {
		g.Tick(r, later(time.Hour))
	}
	if g.seats[0].Score != g.seats[2].Score || g.seats[1].Score != g.seats[3].Score {
		t.Fatal("partners don't share")
	}
}

func TestCanasta(t *testing.T) {
	for range 3 {
		r, _ := testRoom(ann)
		g := newCanasta(r).(*canasta)
		playOut(t, "canasta", func() string { return g.phase }, g, 200000)
		if max(g.team[0], g.team[1]) < canastaTo {
			t.Fatal("over before anyone won")
		}
	}
	if meldable('7', nil, []string{"7H", "2C", "JK"}) == "" || meldable('7', nil, []string{"7H", "7C", "JK"}) != "" ||
		meldable('7', []string{"7H", "7C", "JK", "2D"}, []string{"2S"}) != "" && meldable('7', []string{"7H", "7C", "JK", "2D"}, []string{"2S", "JK"}) == "" {
		t.Fatal("melds are wrong")
	}
}

func TestMahjong(t *testing.T) {
	if !mjComplete([]string{"1p", "2p", "3p", "4s", "5s", "6s", "7m", "8m", "9m", "Rd", "Rd", "Rd", "Ew", "Ew"}, 0) {
		t.Fatal("four sets and a pair don't win")
	}
	if !mjComplete([]string{"1p", "1p", "3p", "3p", "4s", "4s", "7m", "7m", "9m", "9m", "Rd", "Rd", "Ew", "Ew"}, 0) {
		t.Fatal("seven pairs don't win")
	}
	if mjComplete([]string{"1p", "2p", "4p", "4s", "5s", "6s", "7m", "8m", "9m", "Rd", "Rd", "Rd", "Ew", "Ew"}, 0) {
		t.Fatal("a gap wins")
	}
	if !mjComplete([]string{"4s", "5s", "6s", "Ew", "Ew"}, 3) {
		t.Fatal("three shown sets, a set and a pair don't win")
	}
	for range 5 {
		r, _ := testRoom(ann)
		g := newMahjong(r).(*mahjong)
		playOut(t, "mahjong", func() string { return g.phase }, g, 100000)
		if g.played != mjHands {
			t.Fatal("hands played:", g.played)
		}
	}
	r, _ := testRoom(ann, bob)
	g := newMahjong(r).(*mahjong)
	g.Message(r, ann, Msg{"kind": "join"})
	g.Message(r, bob, Msg{"kind": "join"})
	g.Message(r, ann, Msg{"kind": "start"})
	tiles := 0
	for k := range 4 {
		tiles += len(g.hands[k])
	}
	if tiles != 53 || len(g.wall) != 136-53 {
		t.Fatal("not dealt:", tiles)
	}
	if v := g.View(bob).(map[string]any); fmt.Sprint(v["tiles"]) == fmt.Sprint(g.hands[seatOf(g.seats, ann.ID)]) {
		t.Fatal("Bob sees Ann's tiles")
	}
}

func contains(s []string, x string) bool {
	for _, y := range s {
		if y == x {
			return true
		}
	}
	return false
}
