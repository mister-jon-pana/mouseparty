package apps

// Cribbage: two to four seats (a bot joins one player; four play as two teams,
// partners across). Six cards each with two players (five with three or four);
// each puts two (or one) in the dealer's crib, and a starter is cut (a jack: two
// for the dealer). Then pegging: in turn, a card at a time, the count never past
// 31; fifteen and thirty-one are two, pairs two (three of a kind six, four
// twelve), runs one a card, the last card one. Can't play: a go, made for you.
// Then the show: each hand with the starter counts fifteens, pairs, runs,
// flushes and his nobs; then the dealer's crib. First to 121 wins.

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	cribTo   = 121
	cribShow = 15 * time.Second
)

type cribbage struct {
	Base
	room    *Room
	lobby   *Lobby
	phase   string // lobby, crib, peg, show, over
	seats   []*seat
	dealer  int
	hands   [][]string // after the crib: the four kept (shown at the show)
	left    [][]string // what's left to peg
	crib    []string
	gave    []bool // has put their cards in the crib
	starter string
	count   int
	run     []string // played since the count was last reset
	played  []string // every card pegged this hand, in order
	gone    []bool   // has said go since the reset
	lastBy  int      // who played last
	turn    int
	shows   []string // the show, line by line
	last    string
	wait    pacer
	over    Timer
}

func newCribbage(r *Room) Activity {
	return &cribbage{room: r, lobby: NewLobby(1, 4, 20*time.Second), phase: "lobby"}
}

// cribRank: ace 1 up to king 13
func cribRank(c string) int {
	if rank(c) == 12 {
		return 1
	}
	return rank(c) + 2
}

func cribValue(c string) int { return min(cribRank(c), 10) }

func (g *cribbage) n() int { return len(g.seats) }

// points: what seat i scored (a partner too, with four)
func (g *cribbage) points(i, p int, why string) {
	if p <= 0 || g.phase == "over" {
		return
	}
	g.seats[i].Score += p
	if g.n() == 4 {
		g.seats[(i+2)%4].Score += p
	}
	g.last = fmt.Sprintf("%s: %s for %d.", g.seats[i].Name, why, p)
	if g.seats[i].Score >= cribTo {
		g.last += fmt.Sprintf(" %s wins!", g.seats[i].Name)
		if g.n() == 4 {
			g.last = fmt.Sprintf("%s: %s for %d. %s and %s win!", g.seats[i].Name, why, p, g.seats[i].Name, g.seats[(i+2)%4].Name)
		}
		for k, s := range g.seats {
			if k == i || (g.n() == 4 && k == (i+2)%4) {
				if pl := g.lobby.Players.Get(s.ID); pl != nil {
					pl.Score++
				}
			}
		}
		g.phase = "over"
		g.over.Set(cardsOver)
	}
}

func (g *cribbage) start(r *Room) {
	g.seats = sitDown(r, g.lobby, 2, 4)
	g.dealer = r.Rand.IntN(g.n())
	g.deal()
}

func (g *cribbage) deal() {
	d := deck(1)
	Shuffle(g.room, d)
	each := 6
	if g.n() > 2 {
		each = 5
	}
	g.hands, g.left, g.gave = make([][]string, g.n()), make([][]string, g.n()), make([]bool, g.n())
	for i := range g.seats {
		g.hands[i] = slices.Clone(d[:each])
		sortCards(g.hands[i])
		d = d[each:]
	}
	g.crib = nil
	if g.n() == 3 {
		g.crib = append(g.crib, d[0])
		d = d[1:]
	}
	g.starter = d[0]
	g.shows, g.played = nil, nil
	g.phase = "crib"
	g.wait.Set(cardTurn)
	for i, s := range g.seats {
		if s.bot() {
			g.give(i, g.botCrib(i))
		}
	}
}

// toCrib: how many each puts in
func (g *cribbage) toCrib() int {
	if g.n() == 2 {
		return 2
	}
	return 1
}

func (g *cribbage) give(i int, cs []string) {
	if g.gave[i] || !takeAll(&g.hands[i], cs) {
		return
	}
	g.crib = append(g.crib, cs...)
	g.gave[i] = true
	if slices.Contains(g.gave, false) || g.phase != "crib" {
		return
	}
	g.last = fmt.Sprintf("The starter is %s.", g.starter)
	g.phase = "peg"
	if rank(g.starter) == 9 { // a jack: his heels
		g.points(g.dealer, 2, "his heels")
	}
	if g.phase == "over" {
		return
	}
	for i := range g.seats {
		g.left[i] = slices.Clone(g.hands[i])
	}
	g.count, g.run, g.gone, g.lastBy = 0, nil, make([]bool, g.n()), -1
	g.turn = (g.dealer + 1) % g.n()
	g.step()
}

// canPlay: seat i has a card that keeps the count to 31
func (g *cribbage) canPlay(i int) bool {
	for _, c := range g.left[i] {
		if g.count+cribValue(c) <= 31 {
			return true
		}
	}
	return false
}

// step: on to whoever can play next; goes said for those who can't, the count reset when nobody can
func (g *cribbage) step() {
	for range 2 * g.n() {
		if g.phase != "peg" {
			return
		}
		if g.canPlay(g.turn) {
			g.wait.begin(g.seats[g.turn])
			return
		}
		g.gone[g.turn] = true
		anyone := false
		for i := range g.seats {
			if !g.gone[i] && g.canPlay(i) {
				anyone = true
			}
		}
		if !anyone { // nobody can: the last card, and the count starts again
			if g.count > 0 && g.count < 31 && g.lastBy >= 0 {
				g.points(g.lastBy, 1, "last card")
			}
			if g.phase != "peg" {
				return
			}
			done := true
			for i := range g.seats {
				if len(g.left[i]) > 0 {
					done = false
				}
			}
			if done {
				g.show()
				return
			}
			g.count, g.run, g.gone = 0, nil, make([]bool, g.n())
			if g.lastBy >= 0 {
				g.turn = (g.lastBy + 1) % g.n()
			}
			continue
		}
		g.turn = (g.turn + 1) % g.n()
	}
	g.wait.begin(g.seats[g.turn])
}

// pegPoints: what the card just put on run scores, and why
func pegPoints(run []string, count int) (int, []string) {
	p, why := 0, []string{}
	if count == 15 {
		p, why = p+2, append(why, "fifteen")
	}
	if count == 31 {
		p, why = p+2, append(why, "thirty-one")
	}
	last := run[len(run)-1]
	same := 1
	for k := len(run) - 2; k >= 0 && cribRank(run[k]) == cribRank(last); k-- {
		same++
	}
	switch same {
	case 2:
		p, why = p+2, append(why, "a pair")
	case 3:
		p, why = p+6, append(why, "three of a kind")
	case 4:
		p, why = p+12, append(why, "four of a kind")
	}
	for k := len(run); k >= 3; k-- { // the longest run among the last k cards
		ranks := []int{}
		for _, c := range run[len(run)-k:] {
			ranks = append(ranks, cribRank(c))
		}
		slices.Sort(ranks)
		ok := true
		for j := 1; j < len(ranks); j++ {
			if ranks[j] != ranks[j-1]+1 {
				ok = false
			}
		}
		if ok {
			p, why = p+k, append(why, fmt.Sprintf("a run of %d", k))
			break
		}
	}
	return p, why
}

func (g *cribbage) peg(i int, c string) {
	take(&g.left[i], c)
	g.count += cribValue(c)
	g.run = append(g.run, c)
	g.played = append(g.played, c)
	g.lastBy = i
	g.last = fmt.Sprintf("%s played %s: %d.", g.seats[i].Name, c, g.count)
	if p, why := pegPoints(g.run, g.count); p > 0 {
		g.points(i, p, strings.Join(why, " and "))
	}
	if g.phase != "peg" {
		return
	}
	if g.count == 31 {
		g.count, g.run, g.gone = 0, nil, make([]bool, g.n())
	}
	g.turn = (i + 1) % g.n()
	g.step()
}

// handScore: fifteens, pairs, runs, flush and nobs of four cards and the starter
func handScore(hand []string, starter string, crib bool) (int, []string) {
	all := append(slices.Clone(hand), starter)
	p, why := 0, []string{}
	fifteens := 0
	for mask := 1; mask < 1<<len(all); mask++ {
		sum := 0
		for k, c := range all {
			if mask&(1<<k) != 0 {
				sum += cribValue(c)
			}
		}
		if sum == 15 {
			fifteens++
		}
	}
	if fifteens > 0 {
		p, why = p+2*fifteens, append(why, fmt.Sprintf("fifteen %d", 2*fifteens))
	}
	counts := map[int]int{}
	for _, c := range all {
		counts[cribRank(c)]++
	}
	pairs := 0
	for _, n := range counts {
		pairs += n * (n - 1) / 2
	}
	if pairs > 0 {
		p, why = p+2*pairs, append(why, fmt.Sprintf("pairs %d", 2*pairs))
	}
	for lo := 1; lo <= 13; lo++ { // runs: each longest stretch of ranks, times its duplicates
		if counts[lo] == 0 || counts[lo-1] > 0 {
			continue
		}
		n, ways := 0, 1
		for r := lo; counts[r] > 0; r++ {
			n, ways = n+1, ways*counts[r]
		}
		if n >= 3 {
			p, why = p+n*ways, append(why, fmt.Sprintf("runs %d", n*ways))
		}
	}
	flush := true
	for _, c := range hand {
		if suit(c) != suit(hand[0]) {
			flush = false
		}
	}
	if flush && len(hand) == 4 {
		switch {
		case suit(starter) == suit(hand[0]):
			p, why = p+5, append(why, "flush 5")
		case !crib:
			p, why = p+4, append(why, "flush 4")
		}
	}
	for _, c := range hand {
		if rank(c) == 9 && suit(c) == suit(starter) {
			p, why = p+1, append(why, "his nobs 1")
		}
	}
	return p, why
}

func (g *cribbage) show() {
	g.phase, g.shows = "show", nil
	for k := 1; k <= g.n(); k++ {
		i := (g.dealer + k) % g.n()
		p, why := handScore(g.hands[i], g.starter, false)
		g.shows = append(g.shows, fmt.Sprintf("%s: %s = %d", g.seats[i].Name, strings.Join(append([]string{strings.Join(g.hands[i], " ")}, why...), ", "), p))
		g.points(i, p, "the show")
		if g.phase == "over" {
			return
		}
	}
	p, why := handScore(g.crib, g.starter, true)
	g.shows = append(g.shows, fmt.Sprintf("Crib (%s): %s = %d", g.seats[g.dealer].Name, strings.Join(append([]string{strings.Join(g.crib, " ")}, why...), ", "), p))
	g.points(g.dealer, p, "the crib")
	if g.phase == "show" {
		g.phase = "show"
		g.last = "The show."
		g.wait.Set(cribShow)
	}
}

// botCrib: what to put in the crib: what keeps the best hand (counted without a starter's help)
func (g *cribbage) botCrib(i int) []string {
	h := g.hands[i]
	k := g.toCrib()
	best, bestScore := []string(nil), -1
	for a := range h {
		for b := a + 1; b <= len(h); b++ {
			if k == 1 && b != len(h) || k == 2 && b == len(h) {
				continue
			}
			out := []string{h[a]}
			if k == 2 {
				out = append(out, h[b])
			}
			keep := slices.DeleteFunc(slices.Clone(h), func(c string) bool { return slices.Contains(out, c) })
			s, _ := handScore(keep[:len(keep)-1], keep[len(keep)-1], false)
			if s > bestScore {
				best, bestScore = out, s
			}
		}
	}
	return best
}

// botPeg: the card that scores most now; else the highest that doesn't make 5 or 21
func (g *cribbage) botPeg(i int) string {
	best, bestP := "", -1
	for _, c := range g.left[i] {
		if g.count+cribValue(c) > 31 {
			continue
		}
		p, _ := pegPoints(append(slices.Clone(g.run), c), g.count+cribValue(c))
		n := g.count + cribValue(c)
		if p == 0 && (n == 5 || n == 21) {
			p = -1
		}
		if p > bestP || p == bestP && cribValue(c) > cribValue(best) {
			best, bestP = c, p
		}
	}
	return best
}

func (g *cribbage) Message(r *Room, from Viewer, m Msg) {
	if g.phase == "lobby" {
		if g.lobby.Message(r, from, m) {
			g.start(r)
		}
		r.Changed()
		return
	}
	i := seatOf(g.seats, from.ID)
	if i < 0 {
		return
	}
	switch {
	case m.Kind() == "crib" && g.phase == "crib":
		cs, ok := strs(m, "cards", 2, 2)
		if !ok || len(cs) != g.toCrib() || (len(cs) == 2 && cs[0] == cs[1]) {
			return
		}
		g.give(i, cs)
	case m.Kind() == "peg" && g.phase == "peg" && g.turn == i:
		c := strings.ToUpper(m.Str("card", 2))
		if !slices.Contains(g.left[i], c) || g.count+cribValue(c) > 31 {
			r.To(from.ID, Msg{"kind": "nope", "why": "That would go past 31."})
			return
		}
		g.peg(i, c)
	default:
		return
	}
	r.Changed()
}

func (g *cribbage) Tick(r *Room, now time.Time) {
	switch g.phase {
	case "lobby":
		if g.lobby.Tick(now) {
			g.start(r)
			r.Changed()
		}
	case "crib":
		if g.wait.Done(now) {
			for i := range g.seats {
				g.give(i, g.botCrib(i))
			}
			r.Changed()
		}
	case "peg":
		if g.wait.Done(now) {
			if c := g.botPeg(g.turn); c != "" {
				g.peg(g.turn, c)
			} else {
				g.step()
			}
			r.Changed()
		}
	case "show":
		if g.wait.Done(now) {
			g.dealer = (g.dealer + 1) % g.n()
			g.deal()
			r.Changed()
		}
	case "over":
		if g.over.Done(now) {
			g.phase = "lobby"
			g.lobby.Reopen(r)
			r.Changed()
		}
	}
}

func (g *cribbage) Left(r *Room, v Viewer) {
	if g.phase != "lobby" && g.phase != "over" && standUp(g.seats, v.ID) {
		for i, s := range g.seats {
			if s.bot() && g.phase == "crib" && !g.gave[i] {
				g.give(i, g.botCrib(i))
			}
		}
		if g.phase == "peg" && g.seats[g.turn].bot() {
			g.wait.Hurry(botWait)
		}
		if humans(g.seats) == 0 {
			g.phase = "over"
			g.over.Set(time.Second)
		}
	}
	g.lobby.Left(v)
	r.Changed()
}

func (g *cribbage) View(v Viewer) any {
	view := map[string]any{"phase": g.phase, "lobby": g.lobby.View(v)}
	if g.phase == "lobby" {
		return view
	}
	i := seatOf(g.seats, v.ID)
	counts := []int{}
	for k := range g.seats {
		if g.phase == "crib" {
			counts = append(counts, len(g.hands[k]))
		} else {
			counts = append(counts, len(g.left[k]))
		}
	}
	view["seats"], view["counts"], view["me"], view["turn"], view["dealer"] = seatNames(g.seats), counts, i, g.turn, g.dealer
	view["scores"], view["to"], view["count"], view["run"], view["last"] = seatScores(g.seats), cribTo, g.count, g.run, g.last
	view["board"], view["left"], view["toCrib"], view["teams"] = seatBoard(g.seats, false), g.wait.Left(), g.toCrib(), g.n() == 4
	if g.phase != "crib" {
		view["starter"] = g.starter
	}
	if g.phase == "show" || g.phase == "over" {
		view["shows"] = g.shows
	}
	if i >= 0 {
		switch g.phase {
		case "crib":
			view["hand"], view["gave"] = g.hands[i], g.gave[i]
		case "peg":
			view["hand"] = g.left[i]
			if g.turn == i {
				ok := []string{}
				for _, c := range g.left[i] {
					if g.count+cribValue(c) <= 31 {
						ok = append(ok, c)
					}
				}
				view["legal"] = ok
			}
		default:
			view["hand"] = g.hands[i]
		}
	}
	return view
}
