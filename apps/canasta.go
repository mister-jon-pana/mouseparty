package apps

// Canasta: four seats as two teams, partners across (bots in the empty seats).
// Two decks and four jokers; jokers and twos are wild; red threes are laid out
// at once for 100 each (all four 800); black threes block the pile. Eleven
// cards each. On your turn draw two from the stock, or take the whole pile by
// melding its top card with two of that rank from your hand (or one and a
// wild, or onto your team's meld of it, once your team has melded and the
// pile holds no wild); then meld (three or more of a rank, at least two of
// them natural and at most three wild) and discard one. A team's first meld
// must be worth canastaMin. Seven in a meld is a canasta: 500 with no wilds,
// 300 with. You can go out only once your team has a canasta (100 more).
// Each hand scores the melds' cards, the canastas and red threes, less what's
// left in hand. First team to canastaTo wins.

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const canastaTo = 2500

type canasta struct {
	Base
	room   *Room
	lobby  *Lobby
	phase  string // lobby, play, over
	seats  []*seat
	hands  [4][]string
	stock  []string
	pile   []string             // the top is last
	melds  [2]map[byte][]string // each team's, by rank
	threes [2][]string          // red threes laid out
	team   [2]int               // each team's score
	turn   int
	drew   bool // the seat whose turn it is has drawn
	out    int  // the team that went out, or -1
	last   string
	wait   pacer
	over   Timer
}

func newCanasta(r *Room) Activity {
	return &canasta{room: r, lobby: NewLobby(1, 4, 20*time.Second), phase: "lobby"}
}

func wild(c string) bool     { return c == "JK" || rank(c) == 0 }
func redThree(c string) bool { return c == "3D" || c == "3H" }
func blackThree(c string) bool {
	return c == "3C" || c == "3S"
}

func canastaValue(c string) int {
	switch {
	case c == "JK":
		return 50
	case rank(c) == 0 || rank(c) == 12:
		return 20
	case rank(c) >= 6:
		return 10
	}
	return 5
}

// canastaMin: what a team's first meld must be worth, by its score
func (g *canasta) canastaMin(t int) int {
	switch {
	case g.team[t] >= 1500:
		return 90
	case g.team[t] < 0:
		return 15
	}
	return 50
}

func (g *canasta) start(r *Room) {
	g.seats = sitDown(r, g.lobby, 4, 4)
	g.team = [2]int{}
	g.turn = r.Rand.IntN(4)
	g.deal()
}

func (g *canasta) deal() {
	g.stock = deck(2)
	g.stock = append(g.stock, "JK", "JK", "JK", "JK")
	Shuffle(g.room, g.stock)
	g.melds = [2]map[byte][]string{{}, {}}
	g.threes = [2][]string{}
	for i := range 4 {
		g.hands[i] = slices.Clone(g.stock[:11])
		g.stock = g.stock[11:]
		g.layThrees(i)
	}
	g.pile = nil
	for { // the upcard: turned until it's an ordinary card
		c := g.stock[0]
		g.stock = g.stock[1:]
		g.pile = append(g.pile, c)
		if !wild(c) && !redThree(c) && !blackThree(c) {
			break
		}
	}
	g.drew, g.out, g.phase = false, -1, "play"
	g.last = g.seats[g.turn].Name + " goes first."
	g.wait.begin(g.seats[g.turn])
}

// layThrees: seat i's red threes out on the table, each replaced from the stock
func (g *canasta) layThrees(i int) {
	for {
		k := slices.IndexFunc(g.hands[i], redThree)
		if k < 0 {
			sortCanasta(g.hands[i])
			return
		}
		g.threes[i%2] = append(g.threes[i%2], g.hands[i][k])
		g.hands[i] = slices.Delete(g.hands[i], k, k+1)
		if len(g.stock) > 0 {
			g.hands[i] = append(g.hands[i], g.stock[0])
			g.stock = g.stock[1:]
		}
	}
}

func sortCanasta(h []string) {
	slices.SortFunc(h, func(a, b string) int {
		ra, rb := rank(a), rank(b)
		if a == "JK" {
			ra = 13
		}
		if b == "JK" {
			rb = 13
		}
		if ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
}

func (g *canasta) frozen() bool { return slices.ContainsFunc(g.pile, wild) }

func (g *canasta) hasCanasta(t int) bool {
	for _, m := range g.melds[t] {
		if len(m) >= 7 {
			return true
		}
	}
	return false
}

// meldable: cs (with the meld already there, have) is a good meld of rank r: why not, if it isn't
func meldable(r byte, have, cs []string) string {
	all := append(slices.Clone(have), cs...)
	nat, wilds := 0, 0
	for _, c := range all {
		switch {
		case wild(c):
			wilds++
		case c[0] == r && !redThree(c) && !blackThree(c):
			nat++
		default:
			return "A meld is cards of one rank, and wilds."
		}
	}
	switch {
	case r == '3':
		return "Threes don't meld."
	case len(all) < 3:
		return "A meld is three cards or more."
	case nat < 2:
		return "A meld needs two natural cards."
	case wilds > 3:
		return "A meld has at most three wilds."
	}
	return ""
}

// rankOf: the rank a set of cards melds as (wilds go with the rest), or 0
func rankOf(cs []string, named byte) byte {
	for _, c := range cs {
		if !wild(c) {
			return c[0]
		}
	}
	return named
}

func valueOf(cs []string) int {
	v := 0
	for _, c := range cs {
		v += canastaValue(c)
	}
	return v
}

// meld: seat i lays cs down; why not, if they can't
func (g *canasta) meld(i int, cs []string, named byte) string {
	t := i % 2
	r := rankOf(cs, named)
	if r == 0 || len(cs) == 0 {
		return "Pick the cards to meld."
	}
	if why := meldable(r, g.melds[t][r], cs); why != "" {
		return why
	}
	if len(g.melds[t]) == 0 && valueOf(cs) < g.canastaMin(t) {
		return fmt.Sprintf("Your team's first meld must be worth %d.", g.canastaMin(t))
	}
	h := slices.Clone(g.hands[i])
	if !takeAll(&h, cs) {
		return "Those aren't your cards."
	}
	if len(h) == 0 && !g.hasCanastaAfter(t, r, cs) {
		return "You can't go out without a canasta."
	}
	g.hands[i] = h
	g.melds[t][r] = append(g.melds[t][r], cs...)
	g.last = fmt.Sprintf("%s melded %s.", g.seats[i].Name, strings.Join(cs, " "))
	if len(g.melds[t][r]) >= 7 && len(g.melds[t][r])-len(cs) < 7 {
		g.last = fmt.Sprintf("%s made a canasta of %ss!", g.seats[i].Name, rankName(r))
	}
	if len(g.hands[i]) == 0 {
		g.goOut(t)
	}
	return ""
}

func (g *canasta) hasCanastaAfter(t int, r byte, cs []string) bool {
	return g.hasCanasta(t) || len(g.melds[t][r])+len(cs) >= 7
}

func rankName(r byte) string {
	if r == 'T' {
		return "10"
	}
	return string(r)
}

// takePile: seat i takes the pile, melding its top with cs from their hand
func (g *canasta) takePile(i int, cs []string) string {
	t := i % 2
	if len(g.pile) == 0 {
		return "The pile is empty."
	}
	top := g.pile[len(g.pile)-1]
	if wild(top) || blackThree(top) {
		return "The pile is blocked."
	}
	r := top[0]
	nat := 0
	for _, c := range cs {
		if c[0] == r && !wild(c) {
			nat++
		}
	}
	strict := g.frozen() || len(g.melds[t]) == 0
	switch {
	case strict && nat < 2:
		return "Take the pile with two of its top card's rank from your hand."
	case !strict && len(cs) == 0 && g.melds[t][r] == nil:
		return "Pick cards to meld with the top card."
	}
	h := slices.Clone(g.hands[i])
	if !takeAll(&h, cs) {
		return "Those aren't your cards."
	}
	with := append(slices.Clone(cs), top)
	if why := meldable(r, g.melds[t][r], with); why != "" {
		return why
	}
	if len(g.melds[t]) == 0 && valueOf(with) < g.canastaMin(t) {
		return fmt.Sprintf("Your team's first meld must be worth %d.", g.canastaMin(t))
	}
	g.melds[t][r] = append(g.melds[t][r], with...)
	g.hands[i] = append(h, g.pile[:len(g.pile)-1]...)
	n := len(g.pile)
	g.pile = nil
	g.layThrees(i)
	g.drew = true
	g.last = fmt.Sprintf("%s took the pile (%d cards).", g.seats[i].Name, n)
	return ""
}

func (g *canasta) drawStock(i int) {
	if len(g.stock) < 2 {
		g.goOut(-1)
		return
	}
	g.hands[i] = append(g.hands[i], g.stock[:2]...)
	g.stock = g.stock[2:]
	g.layThrees(i)
	g.drew = true
	g.last = g.seats[i].Name + " drew two."
}

// discard ends seat i's turn
func (g *canasta) discard(i int, c string) string {
	t := i % 2
	h := slices.Clone(g.hands[i])
	if !take(&h, c) {
		return "That isn't your card."
	}
	if len(h) == 0 && !g.hasCanasta(t) {
		return "You can't go out without a canasta."
	}
	g.hands[i] = h
	g.pile = append(g.pile, c)
	g.last = fmt.Sprintf("%s discarded %s.", g.seats[i].Name, c)
	if len(h) == 0 {
		g.goOut(t)
		return ""
	}
	g.turn, g.drew = (i+1)%4, false
	g.wait.begin(g.seats[g.turn])
	return ""
}

// goOut: the hand's over (t -1: the stock ran out); it's scored
func (g *canasta) goOut(t int) {
	g.out = t
	var line []string
	for k := range 2 {
		s := 0
		for _, m := range g.melds[k] {
			s += valueOf(m)
			if len(m) >= 7 {
				if slices.ContainsFunc(m, wild) {
					s += 300
				} else {
					s += 500
				}
			}
		}
		red := 100 * len(g.threes[k])
		if len(g.threes[k]) == 4 {
			red = 800
		}
		if len(g.melds[k]) == 0 {
			red = -red
		}
		s += red
		if k == t {
			s += 100
		}
		s -= valueOf(g.hands[k]) + valueOf(g.hands[k+2])
		g.team[k] += s
		line = append(line, fmt.Sprintf("%s and %s: %+d", g.seats[k].Name, g.seats[k+2].Name, s))
	}
	for k, s := range g.seats {
		s.Score = g.team[k%2]
	}
	g.last = strings.Join(line, "; ") + "."
	if t >= 0 {
		g.last = fmt.Sprintf("%s went out. %s", g.seats[g.turn].Name, g.last)
	}
	if max(g.team[0], g.team[1]) < canastaTo {
		said := g.last
		g.turn = (g.turn + 1) % 4
		g.deal()
		g.last = said + " New hand: " + g.last
		return
	}
	w := 0
	if g.team[1] > g.team[0] {
		w = 1
	}
	g.last += fmt.Sprintf(" %s and %s win!", g.seats[w].Name, g.seats[w+2].Name)
	for k, s := range g.seats {
		if k%2 == w {
			if p := g.lobby.Players.Get(s.ID); p != nil {
				p.Score++
			}
		}
	}
	g.phase = "over"
	g.over.Set(cardsOver)
}

// botTurn: take the pile with a natural pair, else draw; meld what's ready; discard the least useful card
func (g *canasta) botTurn() {
	i, t := g.turn, g.turn%2
	if !g.drew {
		taken := false
		if len(g.pile) > 0 {
			top := g.pile[len(g.pile)-1]
			var pair []string
			for _, c := range g.hands[i] {
				if c[0] == top[0] && !wild(c) && len(pair) < 2 {
					pair = append(pair, c)
				}
			}
			if len(pair) == 2 && len(g.pile) > 2 {
				taken = g.takePile(i, pair) == ""
			}
		}
		if !taken {
			g.drawStock(i)
		}
		if g.phase != "play" {
			return
		}
	}
	// meld every rank with three naturals (the first meld only if it's worth enough), and onto the team's melds
	byRank := map[byte][]string{}
	for _, c := range g.hands[i] {
		if !wild(c) && !blackThree(c) && !redThree(c) {
			byRank[c[0]] = append(byRank[c[0]], c)
		}
	}
	var first [][]string
	for r, cs := range byRank {
		if g.melds[t][r] != nil || len(cs) >= 3 {
			first = append(first, cs)
		}
	}
	slices.SortFunc(first, func(a, b []string) int { return strings.Compare(a[0], b[0]) })
	if len(g.melds[t]) == 0 {
		total := 0
		for _, cs := range first {
			total += valueOf(cs)
		}
		if total < g.canastaMin(t) {
			first = nil
		}
		for _, cs := range first { // melded together, they're worth enough: lay the biggest first
			if len(g.melds[t]) == 0 && valueOf(cs) < g.canastaMin(t) {
				continue
			}
			g.meld(i, cs, 0)
		}
	}
	for _, cs := range first {
		if len(g.hands[i]) <= len(cs)+1 && !g.hasCanastaAfter(t, cs[0][0], cs) {
			continue
		}
		g.meld(i, cs, 0)
		if g.phase != "play" || g.turn != i {
			return
		}
	}
	// a wild onto a meld of six makes a canasta
	for r, m := range g.melds[t] {
		if len(m) == 6 && len(g.hands[i]) > 2 {
			if k := slices.IndexFunc(g.hands[i], wild); k >= 0 {
				g.meld(i, []string{g.hands[i][k]}, r)
			}
		}
	}
	if g.phase != "play" || g.turn != i || len(g.hands[i]) == 0 {
		return
	}
	g.discard(i, g.botDiscard(i))
}

func (g *canasta) botDiscard(i int) string {
	h := g.hands[i]
	if k := slices.IndexFunc(h, blackThree); k >= 0 {
		return h[k]
	}
	count := map[byte]int{}
	for _, c := range h {
		count[c[0]]++
	}
	best, bestScore := h[0], 1<<30
	for _, c := range h {
		s := canastaValue(c) + 30*count[c[0]]
		if wild(c) {
			s += 1000
		}
		if g.melds[i%2][c[0]] != nil {
			s += 500
		}
		if s < bestScore {
			best, bestScore = c, s
		}
	}
	return best
}

func (g *canasta) Message(r *Room, from Viewer, m Msg) {
	if g.phase == "lobby" {
		if g.lobby.Message(r, from, m) {
			g.start(r)
		}
		r.Changed()
		return
	}
	i := seatOf(g.seats, from.ID)
	if g.phase != "play" || i < 0 || i != g.turn {
		return
	}
	why := ""
	okCards := func(cs []string) bool {
		for _, c := range cs {
			if c != "JK" && !isCard(c) {
				return false
			}
		}
		return true
	}
	switch m.Kind() {
	case "draw":
		if g.drew {
			return
		}
		g.drawStock(i)
	case "pile":
		cs, ok := strs(m, "cards", 2, 20)
		if !ok || !okCards(cs) || g.drew {
			return
		}
		why = g.takePile(i, cs)
	case "meld":
		cs, ok := strs(m, "cards", 2, 20)
		rk := strings.ToUpper(m.Str("rank", 1))
		if !ok || !okCards(cs) || !g.drew || (len(rk) == 1 && rank(rk+"C") < 0) {
			return
		}
		var named byte
		if len(rk) == 1 {
			named = rk[0]
		}
		why = g.meld(i, cs, named)
	case "discard":
		if !g.drew {
			return
		}
		why = g.discard(i, strings.ToUpper(m.Str("card", 2)))
	default:
		return
	}
	if why != "" {
		r.To(from.ID, Msg{"kind": "nope", "why": why})
		return
	}
	if g.phase == "play" && g.turn == i {
		g.wait.begin(g.seats[i])
	}
	r.Changed()
}

func (g *canasta) Tick(r *Room, now time.Time) {
	switch g.phase {
	case "lobby":
		if g.lobby.Tick(now) {
			g.start(r)
			r.Changed()
		}
	case "play":
		if g.wait.Done(now) {
			g.botTurn()
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

func (g *canasta) Left(r *Room, v Viewer) {
	if g.phase == "play" && standUp(g.seats, v.ID) {
		if g.seats[g.turn].bot() {
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

func (g *canasta) View(v Viewer) any {
	view := map[string]any{"phase": g.phase, "lobby": g.lobby.View(v)}
	if g.phase == "lobby" {
		return view
	}
	i := seatOf(g.seats, v.ID)
	counts := []int{}
	for k := range 4 {
		counts = append(counts, len(g.hands[k]))
	}
	melds := [2][][]string{}
	for t := range 2 {
		melds[t] = [][]string{}
		for _, r := range "AKQJT98765432" {
			if m := g.melds[t][byte(r)]; m != nil {
				melds[t] = append(melds[t], m)
			}
		}
	}
	top := ""
	if len(g.pile) > 0 {
		top = g.pile[len(g.pile)-1]
	}
	view["seats"], view["counts"], view["me"], view["turn"], view["drew"] = seatNames(g.seats), counts, i, g.turn, g.drew
	view["melds"], view["threes"], view["team"], view["to"] = melds, g.threes, g.team, canastaTo
	view["top"], view["pile"], view["frozen"], view["stock"], view["last"], view["left"] = top, len(g.pile), g.frozen(), len(g.stock), g.last, g.wait.Left()
	view["board"] = seatBoard(g.seats, false)
	if i >= 0 {
		view["hand"], view["min"], view["melded"] = g.hands[i], g.canastaMin(i%2), len(g.melds[i%2]) > 0
	}
	return view
}
