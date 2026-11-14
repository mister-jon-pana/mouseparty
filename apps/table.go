package apps

// What the card and tile games share (Hearts, Color Match, Cribbage, Canasta,
// Mahjong): a table of seats, each a viewer or a bot. The lobby's players sit
// down in a random order and bots fill the seats a game needs; a player who
// leaves mid-game hands their seat to a bot, so the game goes on. A bot plays
// after a moment (botWait), and a player whose turn runs out has the bot's
// move made for them. Cards are text: rank then suit, "QH", "TC", "2S".

import (
	"slices"
	"strings"
	"time"
)

const (
	botWait   = 1200 * time.Millisecond // how long a bot thinks
	cardTurn  = time.Minute             // a player's turn, before the bot plays it for them
	cardsOver = 20 * time.Second        // the result shown, before the lobby again
)

var botNames = []string{"Robo Rita", "Robo Ray", "Robo Rex", "Robo Rose"}

// seat is a place at the table: a viewer's (ID > 0) or a bot's (ID < 0)
type seat struct {
	ID    int
	Name  string
	Score int
}

func (s *seat) bot() bool { return s.ID < 0 }

// sitDown: the lobby's players in a random order, and bots to make at least n seats
// (and at most max, players past it left standing)
func sitDown(r *Room, l *Lobby, n, max int) []*seat {
	ids := l.Players.IDs()
	Shuffle(r, ids)
	var seats []*seat
	for _, id := range ids {
		if len(seats) < max {
			seats = append(seats, &seat{ID: id, Name: l.Players.Name(id)})
		}
	}
	for b := 0; len(seats) < n; b++ {
		seats = append(seats, &seat{ID: -(b + 1), Name: botNames[b%len(botNames)]})
	}
	return seats
}

// seatOf: the seat of viewer id, or -1
func seatOf(seats []*seat, id int) int {
	if id <= 0 {
		return -1
	}
	return slices.IndexFunc(seats, func(s *seat) bool { return s.ID == id })
}

// standUp: a viewer who left hands their seat to a bot; true if they had one
func standUp(seats []*seat, id int) bool {
	k := seatOf(seats, id)
	if k < 0 {
		return false
	}
	seats[k].ID = -100 - k
	seats[k].Name += " (bot)"
	return true
}

// humans: how many seats a viewer holds
func humans(seats []*seat) int {
	n := 0
	for _, s := range seats {
		if !s.bot() {
			n++
		}
	}
	return n
}

// seatBoard: the seats' scores, for lib.ts board (best first, or lowest first for Hearts)
func seatBoard(seats []*seat, lowWins bool) []Player {
	b := []Player{}
	for _, s := range seats {
		b = append(b, Player{ID: s.ID, Name: s.Name, Score: s.Score})
	}
	slices.SortStableFunc(b, func(x, y Player) int {
		if lowWins {
			return x.Score - y.Score
		}
		return y.Score - x.Score
	})
	return b
}

// seatNames: who sits where, for a view
func seatNames(seats []*seat) []string {
	out := []string{}
	for _, s := range seats {
		out = append(out, s.Name)
	}
	return out
}

// pacer times a turn: a bot's moment to think, or a player's minute
type pacer struct{ Timer }

// begin: the seat whose turn it is now
func (p *pacer) begin(s *seat) {
	if s.bot() {
		p.Set(botWait)
	} else {
		p.Set(cardTurn)
	}
}

// ---------- cards ----------

const (
	cardRanks = "23456789TJQKA"
	cardSuits = "CDSH" // clubs, diamonds, spades, hearts
)

// deck: the 52 cards, copies times over
func deck(copies int) []string {
	var d []string
	for range copies {
		for _, s := range cardSuits {
			for _, r := range cardRanks {
				d = append(d, string(r)+string(s))
			}
		}
	}
	return d
}

// rank: 0 for a two, up to 12 for an ace; -1 for anything else (a joker)
func rank(c string) int {
	if len(c) != 2 {
		return -1
	}
	return strings.IndexByte(cardRanks, c[0])
}

func suit(c string) byte {
	if len(c) != 2 {
		return 0
	}
	return c[1]
}

// isCard: a card of a 52-card deck
func isCard(c string) bool {
	return len(c) == 2 && rank(c) >= 0 && strings.IndexByte(cardSuits, c[1]) >= 0
}

// take removes one c from hand; false if it isn't there
func take(hand *[]string, c string) bool {
	k := slices.Index(*hand, c)
	if k < 0 {
		return false
	}
	*hand = slices.Delete(*hand, k, k+1)
	return true
}

// takeAll removes every one of cs from hand, or none of them
func takeAll(hand *[]string, cs []string) bool {
	h := slices.Clone(*hand)
	for _, c := range cs {
		if !take(&h, c) {
			return false
		}
	}
	*hand = h
	return true
}

// sortCards: by suit, then rank
func sortCards(h []string) {
	slices.SortFunc(h, func(a, b string) int {
		if sa, sb := strings.IndexByte(cardSuits, suit(a)), strings.IndexByte(cardSuits, suit(b)); sa != sb {
			return sa - sb
		}
		return rank(a) - rank(b)
	})
}

// strs: a message's list of texts (cards, tiles), each cut to max; false if it isn't one
func strs(m Msg, key string, max, most int) ([]string, bool) {
	raw, ok := m[key].([]any)
	if !ok || len(raw) > most {
		return nil, false
	}
	var out []string
	for _, x := range raw {
		s, ok := x.(string)
		if !ok {
			return nil, false
		}
		out = append(out, cut(s, max))
	}
	return out, true
}
