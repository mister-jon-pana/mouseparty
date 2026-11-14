// Hearts (apps/hearts.go): pick three cards to pass, then play a card a trick.
// Cards you can't play now are greyed.
import type { Start } from '../types.ts';
import * as L from './lib.ts';
import * as C from './cards.ts';

const h = L.h;

// apps/hearts.go View
interface View {
    phase: 'lobby' | 'pass' | 'play' | 'over';
    lobby: L.Lobby;
    seats: string[];
    counts: number[];
    me: number; // your seat, -1 watching
    scores: number[];
    taken: number[];
    board: L.Player[];
    trick: string[] | null;
    leader: number;
    turn: number;
    last: string;
    lastTrick: string[] | null;
    left: number;
    passTo: number; // 1 left, 3 right, 2 across
    hand?: string[];
    passed?: boolean;
    legal?: string[];
}
type Event = { kind: 'nope'; why: string };
type Out = L.LobbyMsg | { kind: 'pass'; cards: string[] } | { kind: 'play'; card: string };

const WAY: Record<number, string> = { 1: 'to your left', 3: 'to your right', 2: 'across' };

const hearts: Start<View, Event, Out> = (ctx, view) => {
    const pick = C.picks();
    const m = L.mount<View, Out>(ctx, 'cover', (v, kit) => {
        if (v.phase === 'lobby')
            return L.lobby(
                v.lobby,
                kit,
                'Hearts',
                'Four seats (bots fill the empty ones). Avoid hearts, a point each, and the queen of spades, 13. Take them all and everyone else gets 26. Lowest score wins at 50.',
            );
        pick.sync(v.hand);
        const mine = v.me >= 0 && v.turn === v.me && v.phase === 'play';
        const table = (v.trick || []).map((c, k) => h('div', { style: 'text-align:center' }, C.card(c), h('div', { class: 'muted', style: 'font-size:0.7em' }, v.seats[(v.leader + k) % 4])));
        let acts: L.Kid = null;
        if (v.phase === 'pass' && v.hand) {
            acts = v.passed
                ? h('p', { class: 'muted' }, 'Passed. Waiting for the others… ', L.timer(v.left))
                : h(
                      'p',
                      {},
                      `Pick three cards to pass ${WAY[v.passTo]}. `,
                      L.button(`Pass ${pick.size()}/3`, () => kit.send({ kind: 'pass', cards: pick.cards(v.hand!) }), { disabled: pick.size() !== 3 }),
                      ' ',
                      L.timer(v.left),
                  );
        } else if (v.phase === 'play') {
            acts = mine ? h('p', {}, h('b', {}, 'Your turn: '), 'click a card. ', L.timer(v.left)) : h('p', { class: 'muted' }, `${v.seats[v.turn]} to play`);
        }
        const hand = v.hand
            ? C.hand(
                  v.hand,
                  (i) => {
                      if (v.phase === 'pass' && !v.passed) {
                          pick.toggle(i, 3);
                          m.redraw();
                      } else if (mine) kit.send({ kind: 'play', card: v.hand![i] });
                  },
                  (i) => pick.has(i),
                  (c) => v.phase === 'pass' || (mine && (v.legal || []).includes(c)),
              )
            : null;
        return [
            h('h2', {}, v.phase === 'over' ? 'Game over' : 'Hearts'),
            C.seats(v.seats, v.me, v.phase === 'play' ? v.turn : -1, v.counts, (i) => h('span', { class: 'muted' }, ` · ${v.scores[i]} (+${v.taken[i]})`)),
            v.last ? h('p', {}, v.last) : null,
            h('div', { class: 'cards', style: 'min-height:4.6em;gap:0.8em' }, table.length ? table : v.lastTrick ? h('div', { class: 'muted' }, 'Last trick: ', C.row(v.lastTrick, true)) : null),
            acts,
            hand ? [h('h3', {}, 'Your hand'), hand] : h('p', { class: 'muted' }, 'You are watching.'),
            L.board(v.board, 'Scores (lowest wins)'),
        ];
    });
    m.draw(view);
    return {
        view: m.draw,
        message(e) {
            if (e.kind === 'nope') m.kit.note(e.why);
        },
    };
};
export default hearts;
