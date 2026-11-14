package apps

// Hearts: four seats, bots in any nobody takes. Thirteen cards each; three passed
// left, then right, then across, then kept. The two of clubs leads; follow suit
// if you can. Each heart taken is a point and the queen of spades thirteen;
// hearts can't be led until one has been thrown, and nothing worth points goes
// on the first trick unless that's all you have. Take all 26 and everyone else
// gets them instead. Lowest score wins once anyone reaches heartsTo.

import (
	"fmt"
	"slices"
	"time"
)

const heartsTo = 50

type hearts struct {
	Base
	room      *Room
	lobby     *Lobby
	phase     string // lobby, pass, play, over
	seats     []*seat
	hands     [4][]string
	passing   [4][]string // what each seat has chosen to pass
	hand      int         // hands played this game: which way the pass goes
	trick     []string    // cards on the table, led by leader
	leader    int
	turn      int
	taken     [4]int // points taken this hand
	broken    bool   // a heart has been thrown
	first     bool   // the first trick of the hand
	last      string
	wait      pacer
	over      Timer
	lastTrick []string
}

func newHearts(r *Room) Activity {
	return &hearts{room: r, lobby: NewLobby(1, 4, 20*time.Second), phase: "lobby"}
}

// passTo: 1 left, 3 right, 2 across, 0 kept, hand by hand
func (g *hearts) passTo() int { return []int{1, 3, 2, 0}[g.hand%4] }

func (g *hearts) start(r *Room) {
	g.seats = sitDown(r, g.lobby, 4, 4)
	g.hand = 0
	g.deal(r)
}

func (g *hearts) deal(r *Room) {
	d := deck(1)
	Shuffle(r, d)
	for i := range 4 {
		g.hands[i] = slices.Clone(d[i*13 : i*13+13])
		sortCards(g.hands[i])
		g.passing[i] = nil
	}
	g.taken, g.broken, g.trick, g.lastTrick = [4]int{}, false, nil, nil
	if g.passTo() == 0 {
		g.play()
		return
	}
	g.phase = "pass"
	g.wait.Set(cardTurn)
	for i, s := range g.seats {
		if s.bot() {
			g.passing[i] = g.botPass(i)
		}
	}
}

// play: the passes made, the two of clubs leads
func (g *hearts) play() {
	if to := g.passTo(); to != 0 {
		var got [4][]string
		for i := range 4 {
			got[(i+to)%4] = g.passing[i]
			for _, c := range g.passing[i] {
				take(&g.hands[i], c)
			}
		}
		for i := range 4 {
			g.hands[i] = append(g.hands[i], got[i]...)
			sortCards(g.hands[i])
		}
	}
	g.phase, g.first = "play", true
	for i := range 4 {
		if slices.Contains(g.hands[i], "2C") {
			g.leader, g.turn = i, i
		}
	}
	g.wait.begin(g.seats[g.turn])
}

func heartsPoints(c string) int {
	switch {
	case c == "QS":
		return 13
	case suit(c) == 'H':
		return 1
	}
	return 0
}

// legal: the cards seat i may play now
func (g *hearts) legal(i int) []string {
	hand := g.hands[i]
	var ok []string
	if len(g.trick) == 0 {
		if g.first {
			return []string{"2C"}
		}
		for _, c := range hand {
			if suit(c) != 'H' || g.broken {
				ok = append(ok, c)
			}
		}
		if len(ok) == 0 {
			return slices.Clone(hand)
		}
		return ok
	}
	led := suit(g.trick[0])
	for _, c := range hand {
		if suit(c) == led {
			ok = append(ok, c)
		}
	}
	if len(ok) > 0 {
		return ok
	}
	if g.first { // nothing worth points on the first trick, if there's anything else
		for _, c := range hand {
			if heartsPoints(c) == 0 {
				ok = append(ok, c)
			}
		}
		if len(ok) > 0 {
			return ok
		}
	}
	return slices.Clone(hand)
}

// botPass: the three highest, spades queen and up first
func (g *hearts) botPass(i int) []string {
	h := slices.Clone(g.hands[i])
	slices.SortFunc(h, func(a, b string) int {
		va, vb := rank(a)+heartsPoints(a)*2, rank(b)+heartsPoints(b)*2
		return vb - va
	})
	return h[:3]
}

// botCard: the highest card that won't win the trick, or else the lowest; when void, the worst card to hold
func (g *hearts) botCard(i int) string {
	ok := g.legal(i)
	if len(g.trick) == 0 {
		slices.SortFunc(ok, func(a, b string) int { return rank(a) - rank(b) })
		return ok[0]
	}
	led := suit(g.trick[0])
	high := -1
	for _, c := range g.trick {
		if suit(c) == led && rank(c) > high {
			high = rank(c)
		}
	}
	if suit(ok[0]) != led { // void: throw points, else the highest
		if slices.Contains(ok, "QS") {
			return "QS"
		}
		slices.SortFunc(ok, func(a, b string) int {
			return (rank(b) + 13*heartsPoints(b)) - (rank(a) + 13*heartsPoints(a))
		})
		return ok[0]
	}
	best := ""
	for _, c := range ok {
		if rank(c) < high && (best == "" || rank(c) > rank(best)) {
			best = c
		}
	}
	if best != "" {
		return best
	}
	slices.SortFunc(ok, func(a, b string) int { return rank(a) - rank(b) })
	if len(g.trick) == 3 { // last to play and taking it anyway: the highest
		return ok[len(ok)-1]
	}
	return ok[0]
}

// put plays c for seat i (it's legal)
func (g *hearts) put(i int, c string) {
	take(&g.hands[i], c)
	g.trick = append(g.trick, c)
	if suit(c) == 'H' {
		g.broken = true
	}
	if len(g.trick) < 4 {
		g.turn = (g.turn + 1) % 4
		g.wait.begin(g.seats[g.turn])
		return
	}
	led, win, high := suit(g.trick[0]), 0, -1
	pts := 0
	for k, c := range g.trick {
		pts += heartsPoints(c)
		if suit(c) == led && rank(c) > high {
			win, high = k, rank(c)
		}
	}
	w := (g.leader + win) % 4
	g.taken[w] += pts
	g.last = fmt.Sprintf("%s took the trick", g.seats[w].Name)
	if pts > 0 {
		g.last += fmt.Sprintf(" (%d)", pts)
	}
	g.last += "."
	g.lastTrick, g.trick, g.first = g.trick, nil, false
	g.leader, g.turn = w, w
	if len(g.hands[0]) > 0 {
		g.wait.begin(g.seats[g.turn])
		return
	}
	g.score()
}

func (g *hearts) score() {
	moon := slices.Index(g.taken[:], 26)
	for i, s := range g.seats {
		switch {
		case moon >= 0 && i != moon:
			s.Score += 26
		case moon < 0:
			s.Score += g.taken[i]
		}
	}
	if moon >= 0 {
		g.last = g.seats[moon].Name + " shot the moon!"
	}
	g.hand++
	top := 0
	for _, s := range g.seats {
		top = max(top, s.Score)
	}
	if top < heartsTo {
		g.deal(g.room)
		return
	}
	b := seatBoard(g.seats, true)
	g.last += fmt.Sprintf(" Game over: %s wins.", b[0].Name)
	for _, s := range g.seats {
		if p := g.lobby.Players.Get(s.ID); p != nil && s.Name == b[0].Name {
			p.Score++
		}
	}
	g.phase = "over"
	g.over.Set(cardsOver)
}

func (g *hearts) Message(r *Room, from Viewer, m Msg) {
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
	case m.Kind() == "pass" && g.phase == "pass":
		cs, ok := strs(m, "cards", 2, 3)
		if !ok || len(cs) != 3 || g.passing[i] != nil {
			return
		}
		h := slices.Clone(g.hands[i])
		if !takeAll(&h, cs) {
			return
		}
		g.passing[i] = cs
		g.passed()
	case m.Kind() == "play" && g.phase == "play" && g.turn == i:
		c := m.Str("card", 2)
		if !slices.Contains(g.legal(i), c) {
			r.To(from.ID, Msg{"kind": "nope", "why": "You can't play that now."})
			return
		}
		g.put(i, c)
	default:
		return
	}
	r.Changed()
}

func (g *hearts) passed() {
	for i := range 4 {
		if g.passing[i] == nil {
			return
		}
	}
	g.play()
}

func (g *hearts) Tick(r *Room, now time.Time) {
	switch g.phase {
	case "lobby":
		if g.lobby.Tick(now) {
			g.start(r)
			r.Changed()
		}
	case "pass":
		if g.wait.Done(now) {
			for i := range 4 {
				if g.passing[i] == nil {
					g.passing[i] = g.botPass(i)
				}
			}
			g.passed()
			r.Changed()
		}
	case "play":
		if g.wait.Done(now) {
			g.put(g.turn, g.botCard(g.turn))
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

func (g *hearts) Left(r *Room, v Viewer) {
	if g.phase != "lobby" && standUp(g.seats, v.ID) {
		for k, s := range g.seats {
			if s.bot() && g.phase == "pass" && g.passing[k] == nil {
				g.passing[k] = g.botPass(k)
			}
		}
		if g.phase == "pass" {
			g.passed()
		}
		if g.phase == "play" && g.seats[g.turn].bot() {
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

func (g *hearts) View(v Viewer) any {
	view := map[string]any{"phase": g.phase, "lobby": g.lobby.View(v)}
	if g.phase == "lobby" {
		return view
	}
	i := seatOf(g.seats, v.ID)
	counts := []int{}
	for k := range 4 {
		counts = append(counts, len(g.hands[k]))
	}
	view["seats"], view["counts"], view["me"] = seatNames(g.seats), counts, i
	view["scores"], view["taken"], view["board"] = seatScores(g.seats), g.taken, seatBoard(g.seats, true)
	view["trick"], view["leader"], view["turn"], view["last"], view["lastTrick"] = g.trick, g.leader, g.turn, g.last, g.lastTrick
	view["left"], view["passTo"] = g.wait.Left(), g.passTo()
	if i >= 0 {
		view["hand"] = g.hands[i]
		view["passed"] = g.passing[i] != nil
		if g.phase == "play" && g.turn == i {
			view["legal"] = g.legal(i)
		}
	}
	return view
}

func seatScores(seats []*seat) []int {
	out := []int{}
	for _, s := range seats {
		out = append(out, s.Score)
	}
	return out
}
