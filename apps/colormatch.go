package apps

// Color Match: the shedding card game, two to four seats (a bot joins one player).
// Seven cards each; play one that matches the top card's colour or number, or
// a wild, which names the colour. Skip, Reverse and Draw Two do what they say,
// and Wild Draw Four too. Can't or won't play: draw one, then play it or pass.
// First to empty their hand wins what's left in everyone else's.
//
// Cards are two letters: a colour (R Y G B) then 0-9, S (skip), R (reverse) or
// D (draw two); "WW" is a wild, "W4" a wild draw four.

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const matchColors = "RYGB"

type colorMatch struct {
	Base
	room  *Room
	lobby *Lobby
	phase string // lobby, play, over
	seats []*seat
	hands [][]string
	stock []string
	pile  []string // the top is last
	color byte     // the colour to match (a wild's choice)
	turn  int
	dir   int
	drawn string // the card drawn this turn, which may be played or passed on
	last  string
	wait  pacer
	over  Timer
}

func newColorMatch(r *Room) Activity {
	return &colorMatch{room: r, lobby: NewLobby(1, 4, 20*time.Second), phase: "lobby"}
}

func matchDeck() []string {
	var d []string
	for _, c := range matchColors {
		d = append(d, string(c)+"0")
		for _, v := range "123456789SRD" {
			d = append(d, string(c)+string(v), string(c)+string(v))
		}
	}
	for range 4 {
		d = append(d, "WW", "W4")
	}
	return d
}

func matchPoints(c string) int {
	switch {
	case c[0] == 'W':
		return 50
	case c[1] >= '0' && c[1] <= '9':
		return int(c[1] - '0')
	}
	return 20
}

func (g *colorMatch) start(r *Room) {
	g.seats = sitDown(r, g.lobby, 2, 4)
	g.stock = matchDeck()
	Shuffle(r, g.stock)
	g.hands = make([][]string, len(g.seats))
	for i := range g.seats {
		g.hands[i] = slices.Clone(g.stock[:7])
		g.stock = g.stock[7:]
	}
	g.pile = nil
	for { // the first card turned up is a number
		c := g.stock[0]
		g.stock = g.stock[1:]
		g.pile = append(g.pile, c)
		if c[1] >= '0' && c[1] <= '9' && c[0] != 'W' {
			break
		}
	}
	g.color = g.top()[0]
	g.turn, g.dir, g.drawn = r.Rand.IntN(len(g.seats)), 1, ""
	g.last = g.seats[g.turn].Name + " goes first."
	g.phase = "play"
	g.wait.begin(g.seats[g.turn])
}

func (g *colorMatch) top() string { return g.pile[len(g.pile)-1] }

func (g *colorMatch) playable(c string) bool {
	return c[0] == 'W' || c[0] == g.color || c[1] == g.top()[1] && g.top()[0] != 'W'
}

func (g *colorMatch) next(k int) int {
	n := len(g.seats)
	return ((g.turn+k*g.dir)%n + n) % n
}

// deal1 takes the top of the stock, the pile shuffled back in when it runs out
func (g *colorMatch) deal1() (string, bool) {
	if len(g.stock) == 0 {
		if len(g.pile) < 2 {
			return "", false
		}
		g.stock = slices.Clone(g.pile[:len(g.pile)-1])
		g.pile = g.pile[len(g.pile)-1:]
		Shuffle(g.room, g.stock)
	}
	c := g.stock[0]
	g.stock = g.stock[1:]
	return c, true
}

func (g *colorMatch) give(i, n int) {
	for range n {
		if c, ok := g.deal1(); ok {
			g.hands[i] = append(g.hands[i], c)
		}
	}
}

// play puts down c for the seat whose turn it is (it's in their hand, and playable)
func (g *colorMatch) play(c string, color byte) {
	i := g.turn
	take(&g.hands[i], c)
	g.pile = append(g.pile, c)
	g.color, g.drawn = c[0], ""
	if c[0] == 'W' {
		g.color = color
	}
	name := g.seats[i].Name
	g.last = fmt.Sprintf("%s played %s.", name, matchName(c, g.color))
	if len(g.hands[i]) == 0 {
		won := 0
		for k, h := range g.hands {
			if k != i {
				for _, x := range h {
					won += matchPoints(x)
				}
			}
		}
		g.seats[i].Score += won
		if p := g.lobby.Players.Get(g.seats[i].ID); p != nil {
			p.Score += won
		}
		g.last = fmt.Sprintf("%s went out, and wins %d!", name, won)
		g.phase = "over"
		g.over.Set(cardsOver)
		return
	}
	step := 1
	switch {
	case c[1] == 'S':
		step = 2
	case c[1] == 'R':
		if g.dir = -g.dir; len(g.seats) == 2 {
			step = 2
		}
	case c[1] == 'D':
		g.give(g.next(1), 2)
		step = 2
	case c == "W4":
		g.give(g.next(1), 4)
		step = 2
	}
	g.turn = g.next(step)
	g.wait.begin(g.seats[g.turn])
}

func matchName(c string, color byte) string {
	colors := map[byte]string{'R': "red", 'Y': "yellow", 'G': "green", 'B': "blue"}
	switch c {
	case "WW":
		return "a wild (" + colors[color] + ")"
	case "W4":
		return "a wild draw four (" + colors[color] + ")"
	}
	v := map[byte]string{'S': "skip", 'R': "reverse", 'D': "draw two"}[c[1]]
	if v == "" {
		v = string(c[1])
	}
	return colors[c[0]] + " " + v
}

// draw: one card for the seat whose turn it is; they may play it, or else it's the next's turn
func (g *colorMatch) draw() {
	c, ok := g.deal1()
	if !ok {
		g.pass()
		return
	}
	g.hands[g.turn] = append(g.hands[g.turn], c)
	g.last = g.seats[g.turn].Name + " drew a card."
	if !g.playable(c) {
		g.pass()
		return
	}
	g.drawn = c
	g.wait.begin(g.seats[g.turn])
}

func (g *colorMatch) pass() {
	g.drawn = ""
	g.turn = g.next(1)
	g.wait.begin(g.seats[g.turn])
}

// bestColor: the colour seat i holds most of
func (g *colorMatch) bestColor(i int) byte {
	best, n := byte('R'), -1
	for _, c := range []byte(matchColors) {
		k := 0
		for _, x := range g.hands[i] {
			if x[0] == c {
				k++
			}
		}
		if k > n {
			best, n = c, k
		}
	}
	return best
}

// botMove: a playable card (a coloured one before a wild), else draw; the drawn card if it plays
func (g *colorMatch) botMove() {
	i := g.turn
	if g.drawn != "" {
		g.play(g.drawn, g.bestColor(i))
		return
	}
	var wild string
	for _, c := range g.hands[i] {
		if !g.playable(c) {
			continue
		}
		if c[0] == 'W' {
			wild = c
			continue
		}
		g.play(c, 0)
		return
	}
	if wild != "" {
		g.play(wild, g.bestColor(i))
		return
	}
	g.draw()
}

func (g *colorMatch) Message(r *Room, from Viewer, m Msg) {
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
	switch m.Kind() {
	case "play":
		c := strings.ToUpper(m.Str("card", 2))
		color := strings.ToUpper(m.Str("color", 1))
		if !slices.Contains(g.hands[i], c) || !g.playable(c) || (g.drawn != "" && c != g.drawn) {
			r.To(from.ID, Msg{"kind": "nope", "why": "That doesn't match."})
			return
		}
		if c[0] == 'W' && (len(color) != 1 || !strings.Contains(matchColors, color)) {
			return
		}
		var col byte
		if len(color) == 1 {
			col = color[0]
		}
		g.play(c, col)
	case "draw":
		if g.drawn != "" {
			return
		}
		g.draw()
	case "pass":
		if g.drawn == "" {
			return
		}
		g.last = g.seats[i].Name + " kept the card."
		g.pass()
	default:
		return
	}
	r.Changed()
}

func (g *colorMatch) Tick(r *Room, now time.Time) {
	switch g.phase {
	case "lobby":
		if g.lobby.Tick(now) {
			g.start(r)
			r.Changed()
		}
	case "play":
		if g.wait.Done(now) {
			g.botMove()
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

func (g *colorMatch) Left(r *Room, v Viewer) {
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

func (g *colorMatch) View(v Viewer) any {
	view := map[string]any{"phase": g.phase, "lobby": g.lobby.View(v)}
	if g.phase == "lobby" {
		return view
	}
	i := seatOf(g.seats, v.ID)
	counts := []int{}
	for _, h := range g.hands {
		counts = append(counts, len(h))
	}
	view["seats"], view["counts"], view["me"], view["turn"], view["dir"] = seatNames(g.seats), counts, i, g.turn, g.dir
	view["top"], view["color"], view["stock"], view["last"], view["left"] = g.top(), string(g.color), len(g.stock), g.last, g.wait.Left()
	view["board"] = seatBoard(g.seats, false)
	if i >= 0 {
		view["hand"] = g.hands[i]
		if g.phase == "play" && g.turn == i {
			ok := []string{}
			for _, c := range g.hands[i] {
				if g.playable(c) && (g.drawn == "" || c == g.drawn) {
					ok = append(ok, c)
				}
			}
			view["legal"], view["drawn"] = ok, g.drawn
		}
	}
	return view
}
