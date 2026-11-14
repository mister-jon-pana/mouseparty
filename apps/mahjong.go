package apps

// Mahjong, Hong Kong style, simplified: four seats (bots in the empty ones),
// 136 tiles (three suits of 1-9, four winds, three dragons, four of each; no
// flowers). Thirteen tiles each; on your turn draw one and discard one. Anyone
// may claim a discard to win; to make a pung (three alike) or a kong (four);
// the next player alone to make a chow (three in a row of a suit). The win
// beats a pung or kong, which beat a chow. A winning hand is four sets and a
// pair, or seven pairs. Fan: 1, and 1 for a self-drawn tile, 1 for a concealed
// hand, 1 for each dragon pung, 3 for all pungs, 3 for one suit with honours,
// 6 for one suit alone, 4 for seven pairs. Four hands, each seat dealing once;
// the most fan wins.
//
// Tiles are two letters: "1p".."9p" dots, "1s".."9s" bamboo, "1m".."9m"
// characters; winds "Ew" "Sw" "Ww" "Nw"; dragons "Rd" "Gd" "Wd".

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	mjHands = 4
	mjClaim = 8 * time.Second // to claim a discard
	mjShow  = 12 * time.Second
)

var mjHonors = []string{"Ew", "Sw", "Ww", "Nw", "Rd", "Gd", "Wd"}

type mjMeld struct {
	Tiles     []string `json:"tiles"`
	Concealed bool     `json:"concealed"`
}

type mahjong struct {
	Base
	room     *Room
	lobby    *Lobby
	phase    string // lobby, turn, claim, show, over
	seats    []*seat
	hands    [4][]string
	melds    [4][]mjMeld
	discards [4][]string
	wall     []string
	dealer   int
	played   int // hands played
	turn     int
	drawn    string // the tile just drawn (it can win by itself)
	discard  string // the tile out for claiming
	from     int    // who threw it
	claims   [4]string
	chows    [4][]string // a chow claimed: the two tiles from the hand
	result   string
	winner   int
	last     string
	wait     pacer
	over     Timer
}

func newMahjong(r *Room) Activity {
	return &mahjong{room: r, lobby: NewLobby(1, 4, 20*time.Second), phase: "lobby"}
}

func mjTiles() []string {
	var t []string
	for _, s := range "psm" {
		for n := '1'; n <= '9'; n++ {
			t = append(t, string(n)+string(s))
		}
	}
	return append(t, mjHonors...)
}

func mjSuited(t string) bool {
	return len(t) == 2 && t[0] >= '1' && t[0] <= '9' && strings.IndexByte("psm", t[1]) >= 0
}
func mjTile(t string) bool { return slices.Contains(mjTiles(), t) }

// mjSort: suits in order, then honours
func mjSort(h []string) {
	order := mjTiles()
	slices.SortFunc(h, func(a, b string) int { return slices.Index(order, a) - slices.Index(order, b) })
}

func mjCount(h []string, t string) int {
	n := 0
	for _, x := range h {
		if x == t {
			n++
		}
	}
	return n
}

func (g *mahjong) start(r *Room) {
	g.seats = sitDown(r, g.lobby, 4, 4)
	g.dealer, g.played = r.Rand.IntN(4), 0
	g.deal()
}

func (g *mahjong) deal() {
	g.wall = nil
	for range 4 {
		g.wall = append(g.wall, mjTiles()...)
	}
	Shuffle(g.room, g.wall)
	for i := range 4 {
		g.hands[i] = slices.Clone(g.wall[:13])
		g.wall = g.wall[13:]
		mjSort(g.hands[i])
		g.melds[i], g.discards[i] = nil, nil
	}
	g.result, g.winner = "", -1
	g.turn = g.dealer
	g.last = g.seats[g.dealer].Name + " deals."
	g.draw()
}

// draw: the seat whose turn it is takes a tile from the wall; an empty wall ends the hand drawn
func (g *mahjong) draw() {
	if len(g.wall) == 0 {
		g.endHand(-1, "The wall is empty: nobody wins this hand.")
		return
	}
	g.drawn = g.wall[0]
	g.wall = g.wall[1:]
	g.hands[g.turn] = append(g.hands[g.turn], g.drawn)
	g.phase = "turn"
	g.wait.begin(g.seats[g.turn])
}

// complete: the concealed tiles, with n sets already shown, make a winning hand
func mjComplete(tiles []string, shown int) bool {
	if len(tiles)+3*shown != 14 {
		return false
	}
	counts := map[string]int{}
	for _, t := range tiles {
		counts[t]++
	}
	if shown == 0 { // seven pairs
		pairs := 0
		for _, n := range counts {
			if n == 2 {
				pairs++
			}
		}
		if pairs == 7 {
			return true
		}
	}
	for t, n := range counts {
		if n >= 2 {
			counts[t] -= 2
			if mjSets(counts) {
				counts[t] += 2
				return true
			}
			counts[t] += 2
		}
	}
	return false
}

// mjSets: the counts are all pungs and chows
func mjSets(counts map[string]int) bool {
	for _, t := range mjTiles() {
		if counts[t] == 0 {
			continue
		}
		if counts[t] >= 3 {
			counts[t] -= 3
			ok := mjSets(counts)
			counts[t] += 3
			if ok {
				return true
			}
		}
		if mjSuited(t) && t[0] <= '7' {
			a, b := string(t[0]+1)+t[1:], string(t[0]+2)+t[1:]
			if counts[a] > 0 && counts[b] > 0 {
				counts[t]--
				counts[a]--
				counts[b]--
				ok := mjSets(counts)
				counts[t]++
				counts[a]++
				counts[b]++
				if ok {
					return true
				}
			}
		}
		return false
	}
	return true
}

// fan: what seat i's winning hand is worth, and why
func (g *mahjong) fan(i int, self bool) (int, []string) {
	all := slices.Clone(g.hands[i])
	concealed := true
	for _, m := range g.melds[i] {
		all = append(all, m.Tiles...)
		if !m.Concealed {
			concealed = false
		}
	}
	f, why := 1, []string{"a win 1"}
	if self {
		f, why = f+1, append(why, "self-drawn 1")
	}
	if concealed {
		f, why = f+1, append(why, "concealed 1")
	}
	counts := map[string]int{}
	for _, t := range all {
		counts[t]++
	}
	sevenPairs := len(g.melds[i]) == 0 && func() bool {
		for _, n := range counts {
			if n != 2 && n != 4 {
				return false
			}
		}
		return true
	}() && !mjSetsAfterPair(g.hands[i])
	if sevenPairs {
		f, why = f+4, append(why, "seven pairs 4")
	}
	for _, d := range []string{"Rd", "Gd", "Wd"} {
		if counts[d] >= 3 {
			f, why = f+1, append(why, "dragon pung 1")
		}
	}
	allPungs := !sevenPairs
	for _, m := range g.melds[i] {
		if m.Tiles[0] != m.Tiles[1] {
			allPungs = false
		}
	}
	if allPungs && len(g.melds[i]) < 4 { // the concealed part must be pungs and a pair
		pairs := 0
		for t, n := range mjCountsOf(g.hands[i]) {
			_ = t
			switch n {
			case 2:
				pairs++
			case 3, 4:
			default:
				allPungs = false
			}
		}
		allPungs = allPungs && pairs == 1
	}
	if allPungs {
		f, why = f+3, append(why, "all pungs 3")
	}
	suits, honors := map[byte]bool{}, false
	for _, t := range all {
		if mjSuited(t) {
			suits[t[1]] = true
		} else {
			honors = true
		}
	}
	switch {
	case len(suits) == 1 && !honors:
		f, why = f+6, append(why, "one suit 6")
	case len(suits) == 1:
		f, why = f+3, append(why, "one suit and honours 3")
	}
	return min(f, 13), why
}

func mjCountsOf(h []string) map[string]int {
	c := map[string]int{}
	for _, t := range h {
		c[t]++
	}
	return c
}

// mjSetsAfterPair: the hand is a pair and sets too (so it isn't counted as seven pairs)
func mjSetsAfterPair(h []string) bool {
	counts := mjCountsOf(h)
	for t, n := range counts {
		if n >= 2 {
			counts[t] -= 2
			if mjSets(counts) {
				return true
			}
			counts[t] += 2
		}
	}
	return false
}

func (g *mahjong) canWin(i int, t string) bool {
	return mjComplete(append(slices.Clone(g.hands[i]), t), len(g.melds[i]))
}

// win: seat i wins (self-drawn, or on the discard)
func (g *mahjong) win(i int, self bool) {
	if !self {
		g.hands[i] = append(g.hands[i], g.discard)
		g.discards[g.from] = g.discards[g.from][:len(g.discards[g.from])-1]
	}
	f, why := g.fan(i, self)
	g.seats[i].Score += f
	how := "self-drawn"
	if !self {
		how = "on " + g.seats[g.from].Name + "'s discard"
	}
	g.endHand(i, fmt.Sprintf("%s wins %s: %s = %d fan.", g.seats[i].Name, how, strings.Join(why, ", "), f))
}

func (g *mahjong) endHand(winner int, result string) {
	g.result, g.winner, g.last = result, winner, result
	g.played++
	g.phase = "show"
	g.wait.Set(mjShow)
}

// throw: seat i discards t, and the others may claim it
func (g *mahjong) throw(i int, t string) {
	take(&g.hands[i], t)
	mjSort(g.hands[i])
	g.discards[i] = append(g.discards[i], t)
	g.discard, g.from, g.drawn = t, i, ""
	g.claims, g.chows = [4]string{}, [4][]string{}
	g.last = fmt.Sprintf("%s discarded %s.", g.seats[i].Name, mjName(t))
	g.phase = "claim"
	g.claims[i] = "pass"
	for k, s := range g.seats {
		if k != i && (s.bot() || len(g.options(k)) == 0) {
			g.claims[k] = g.botClaim(k)
		}
	}
	g.wait.Set(mjClaim)
	g.settle(false)
}

func mjName(t string) string {
	names := map[string]string{"Ew": "East wind", "Sw": "South wind", "Ww": "West wind", "Nw": "North wind", "Rd": "Red dragon", "Gd": "Green dragon", "Wd": "White dragon"}
	if n, ok := names[t]; ok {
		return n
	}
	return string(t[0]) + " of " + map[byte]string{'p': "dots", 's': "bamboo", 'm': "characters"}[t[1]]
}

// options: what seat k may do with the discard: win, pung, kong, chow
func (g *mahjong) options(k int) []string {
	if g.phase != "claim" || k == g.from {
		return nil
	}
	var o []string
	if g.canWin(k, g.discard) {
		o = append(o, "win")
	}
	n := mjCount(g.hands[k], g.discard)
	if n >= 2 {
		o = append(o, "pung")
	}
	if n >= 3 {
		o = append(o, "kong")
	}
	if k == (g.from+1)%4 && len(g.chowsFor(k)) > 0 {
		o = append(o, "chow")
	}
	return o
}

// chowsFor: the pairs of tiles in seat k's hand that make a chow with the discard
func (g *mahjong) chowsFor(k int) [][]string {
	t := g.discard
	if !mjSuited(t) {
		return nil
	}
	var out [][]string
	for _, d := range [][2]int{{-2, -1}, {-1, 1}, {1, 2}} {
		a, b := int(t[0]-'0')+d[0], int(t[0]-'0')+d[1]
		if a < 1 || b > 9 {
			continue
		}
		x, y := fmt.Sprintf("%d%c", a, t[1]), fmt.Sprintf("%d%c", b, t[1])
		if slices.Contains(g.hands[k], x) && slices.Contains(g.hands[k], y) {
			out = append(out, []string{x, y})
		}
	}
	return out
}

// botClaim: win when it can; pung an honour it holds a pair of; else pass
func (g *mahjong) botClaim(k int) string {
	o := g.options(k)
	switch {
	case slices.Contains(o, "win"):
		return "win"
	case slices.Contains(o, "pung") && !mjSuited(g.discard):
		return "pung"
	}
	return "pass"
}

// settle: once everyone has answered (or time's up), the best claim takes the discard
func (g *mahjong) settle(timeUp bool) {
	if g.phase != "claim" {
		return
	}
	for k := range 4 {
		if g.claims[k] == "" {
			if !timeUp {
				return
			}
			g.claims[k] = "pass"
		}
	}
	for _, want := range []string{"win", "kong", "pung", "chow"} {
		for d := 1; d < 4; d++ {
			k := (g.from + d) % 4
			if g.claims[k] != want {
				continue
			}
			switch want {
			case "win":
				g.win(k, false)
				return
			case "kong", "pung":
				n := 3
				if want == "kong" {
					n = 4
				}
				for range n - 1 {
					take(&g.hands[k], g.discard)
				}
				g.melds[k] = append(g.melds[k], mjMeld{Tiles: slices.Repeat([]string{g.discard}, n)})
			case "chow":
				two := g.chows[k]
				if two == nil {
					two = g.chowsFor(k)[0]
				}
				take(&g.hands[k], two[0])
				take(&g.hands[k], two[1])
				tiles := []string{two[0], two[1], g.discard}
				mjSort(tiles)
				g.melds[k] = append(g.melds[k], mjMeld{Tiles: tiles})
			}
			g.discards[g.from] = g.discards[g.from][:len(g.discards[g.from])-1]
			g.last = fmt.Sprintf("%s claimed %s for a %s.", g.seats[k].Name, mjName(g.discard), want)
			g.turn = k
			if want == "kong" {
				g.draw()
				return
			}
			g.phase = "turn"
			g.wait.begin(g.seats[k])
			return
		}
	}
	g.turn = (g.from + 1) % 4
	g.draw()
}

// botDiscard: the tile least use to the hand: lone honours and far-off tiles first
func (g *mahjong) botDiscard(i int) string {
	h := g.hands[i]
	best, bestU := h[0], 1<<30
	for _, t := range h {
		u := 4 * (mjCount(h, t) - 1)
		if mjSuited(t) {
			for _, d := range []int{-2, -1, 1, 2} {
				n := int(t[0]-'0') + d
				if n >= 1 && n <= 9 && slices.Contains(h, fmt.Sprintf("%d%c", n, t[1])) {
					u += 3 - max(d, -d)
				}
			}
			if t[0] == '1' || t[0] == '9' {
				u--
			}
		} else {
			u--
		}
		if u < bestU {
			best, bestU = t, u
		}
	}
	return best
}

// selfKong: a tile seat i holds all four of, concealed
func (g *mahjong) selfKong(i int) string {
	for _, t := range g.hands[i] {
		if mjCount(g.hands[i], t) == 4 {
			return t
		}
	}
	return ""
}

func (g *mahjong) Message(r *Room, from Viewer, m Msg) {
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
	switch g.phase {
	case "turn":
		if i != g.turn {
			return
		}
		switch m.Kind() {
		case "discard":
			t := m.Str("tile", 2)
			if !slices.Contains(g.hands[i], t) {
				return
			}
			g.throw(i, t)
		case "win":
			if !mjComplete(g.hands[i], len(g.melds[i])) {
				r.To(from.ID, Msg{"kind": "nope", "why": "That isn't a winning hand."})
				return
			}
			g.win(i, true)
		case "kong":
			t := m.Str("tile", 2)
			if !mjTile(t) || mjCount(g.hands[i], t) != 4 {
				return
			}
			for range 4 {
				take(&g.hands[i], t)
			}
			g.melds[i] = append(g.melds[i], mjMeld{Tiles: []string{t, t, t, t}, Concealed: true})
			g.last = g.seats[i].Name + " declared a kong."
			g.draw()
		default:
			return
		}
	case "claim":
		if g.claims[i] != "" {
			return
		}
		k := m.Kind()
		switch {
		case k == "pass":
		case k == "chow" && slices.Contains(g.options(i), "chow"):
			two, ok := strs(m, "tiles", 2, 2)
			if !ok || len(two) != 2 || !slices.ContainsFunc(g.chowsFor(i), func(p []string) bool { return slices.Equal(p, two) || slices.Equal(p, []string{two[1], two[0]}) }) {
				return
			}
			g.chows[i] = two
		case (k == "win" || k == "pung" || k == "kong") && slices.Contains(g.options(i), k):
		default:
			return
		}
		g.claims[i] = k
		g.settle(false)
	default:
		return
	}
	r.Changed()
}

func (g *mahjong) Tick(r *Room, now time.Time) {
	switch g.phase {
	case "lobby":
		if g.lobby.Tick(now) {
			g.start(r)
			r.Changed()
		}
	case "turn":
		if g.wait.Done(now) {
			i := g.turn
			switch {
			case mjComplete(g.hands[i], len(g.melds[i])): // a player whose time ran out too: it's a win
				g.win(i, true)
			case g.selfKong(i) != "" && g.seats[i].bot():
				t := g.selfKong(i)
				for range 4 {
					take(&g.hands[i], t)
				}
				g.melds[i] = append(g.melds[i], mjMeld{Tiles: []string{t, t, t, t}, Concealed: true})
				g.draw()
			default:
				g.throw(i, g.botDiscard(i))
			}
			r.Changed()
		}
	case "claim":
		if g.wait.Done(now) {
			g.settle(true)
			r.Changed()
		}
	case "show":
		if g.wait.Done(now) {
			if g.played >= mjHands {
				b := seatBoard(g.seats, false)
				g.last = fmt.Sprintf("%s Game over: %s wins with %d fan.", g.result, b[0].Name, b[0].Score)
				for _, s := range g.seats {
					if s.Score == b[0].Score {
						if p := g.lobby.Players.Get(s.ID); p != nil {
							p.Score++
						}
					}
				}
				g.phase = "over"
				g.over.Set(cardsOver)
			} else {
				g.dealer = (g.dealer + 1) % 4
				g.deal()
			}
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

func (g *mahjong) Left(r *Room, v Viewer) {
	if g.phase != "lobby" && g.phase != "over" && standUp(g.seats, v.ID) {
		if g.phase == "claim" {
			for i, s := range g.seats {
				if s.bot() && g.claims[i] == "" {
					g.claims[i] = g.botClaim(i)
				}
			}
			g.settle(false)
		}
		if g.phase == "turn" && g.seats[g.turn].bot() {
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

func (g *mahjong) View(v Viewer) any {
	view := map[string]any{"phase": g.phase, "lobby": g.lobby.View(v)}
	if g.phase == "lobby" {
		return view
	}
	i := seatOf(g.seats, v.ID)
	counts := []int{}
	for k := range 4 {
		counts = append(counts, len(g.hands[k]))
	}
	melds := [4][]mjMeld{}
	for k := range 4 {
		melds[k] = []mjMeld{}
		for _, m := range g.melds[k] {
			if m.Concealed && g.phase != "show" && g.phase != "over" && k != i {
				m = mjMeld{Tiles: []string{m.Tiles[0], "", "", m.Tiles[0]}, Concealed: true}
			}
			melds[k] = append(melds[k], m)
		}
	}
	discards := [4][]string{}
	for k := range 4 {
		discards[k] = append([]string{}, g.discards[k]...)
	}
	view["seats"], view["counts"], view["me"], view["turn"], view["dealer"] = seatNames(g.seats), counts, i, g.turn, g.dealer
	view["melds"], view["discards"], view["wall"], view["last"], view["left"] = melds, discards, len(g.wall), g.last, g.wait.Left()
	view["board"], view["hand"], view["of"], view["result"] = seatBoard(g.seats, false), g.played+1, mjHands, g.result
	if g.phase == "show" && g.winner >= 0 {
		view["winning"] = g.hands[g.winner]
	}
	if g.phase == "claim" {
		view["discard"], view["from"] = g.discard, g.from
	}
	if i >= 0 {
		view["tiles"] = g.hands[i]
		switch {
		case g.phase == "turn" && g.turn == i:
			view["drawn"] = g.drawn
			view["canWin"] = mjComplete(g.hands[i], len(g.melds[i]))
			view["kong"] = g.selfKong(i)
		case g.phase == "claim" && g.claims[i] == "":
			view["options"] = g.options(i)
			if slices.Contains(g.options(i), "chow") {
				view["chows"] = g.chowsFor(i)
			}
		}
	}
	return view
}
