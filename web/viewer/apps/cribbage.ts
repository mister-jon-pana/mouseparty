// Cribbage (apps/cribbage.go): pick the card(s) for the crib, then click a card
// to peg it. Goes, and the counting, are done for you.
import type { Start } from '../types.ts';
import * as L from './lib.ts';
import * as C from './cards.ts';

const h = L.h;

// apps/cribbage.go View
interface View {
    phase: 'lobby' | 'crib' | 'peg' | 'show' | 'over';
    lobby: L.Lobby;
    seats: string[];
    counts: number[];
    me: number;
    turn: number;
    dealer: number;
    scores: number[];
    to: number;
    count: number;
    run: string[] | null;
    last: string;
    board: L.Player[];
    left: number;
    toCrib: number;
    teams: boolean;
    starter?: string;
    shows?: string[];
    hand?: string[];
    gave?: boolean;
    legal?: string[];
}
type Event = { kind: 'nope'; why: string };
type Out = L.LobbyMsg | { kind: 'crib'; cards: string[] } | { kind: 'peg'; card: string };

const cribbage: Start<View, Event, Out> = (ctx, view) => {
    const pick = C.picks();
    const m = L.mount<View, Out>(ctx, 'cover', (v, kit) => {
        if (v.phase === 'lobby')
            return L.lobby(
                v.lobby,
                kit,
                'Cribbage',
                'Two to four players (a bot joins if you are alone; four play as partners across). Peg to 31 for fifteens, pairs and runs, then count your hand with the starter. First to 121.',
            );
        pick.sync(v.hand);
        const mine = v.phase === 'peg' && v.turn === v.me;
        const legal = new Set(v.legal || []);
        const peg = (i: number) =>
            h(
                'span',
                {},
                ` · ${v.scores[i]}`,
                h('span', {
                    style: `display:inline-block;width:4em;height:0.35em;margin-left:0.4em;background:linear-gradient(90deg,var(--lime) ${(100 * Math.min(v.scores[i], v.to)) / v.to}%,rgb(255 255 255 / .15) 0);border-radius:1em`,
                }),
                i === v.dealer ? h('span', { class: 'muted' }, ' (dealer)') : null,
            );
        let acts: L.Kid = null;
        if (v.phase === 'crib' && v.hand) {
            acts = v.gave
                ? h('p', { class: 'muted' }, 'Waiting for the others… ', L.timer(v.left))
                : h(
                      'p',
                      {},
                      `Pick ${v.toCrib} for ${v.dealer === v.me ? 'your' : `${v.seats[v.dealer]}'s`} crib. `,
                      L.button('Put in the crib', () => kit.send({ kind: 'crib', cards: pick.cards(v.hand!) }), { disabled: pick.size() !== v.toCrib }),
                      ' ',
                      L.timer(v.left),
                  );
        } else if (v.phase === 'peg') {
            acts = mine ? h('p', {}, h('b', {}, 'Your turn: '), 'click a card. ', L.timer(v.left)) : h('p', { class: 'muted' }, `${v.seats[v.turn]} to play`);
        }
        const hand = v.hand
            ? C.hand(
                  v.hand,
                  (i) => {
                      if (v.phase === 'crib' && !v.gave) {
                          pick.toggle(i, v.toCrib);
                          m.redraw();
                      } else if (mine) kit.send({ kind: 'peg', card: v.hand![i] });
                  },
                  (i) => pick.has(i),
                  (c) => (v.phase === 'crib' && !v.gave) || (mine && legal.has(c)),
              )
            : null;
        return [
            h('h2', {}, v.phase === 'over' ? 'Game over' : 'Cribbage'),
            C.seats(v.seats, v.me, v.phase === 'peg' ? v.turn : -1, v.counts, peg),
            v.teams ? h('p', { class: 'muted' }, 'Partners sit across: their points are yours.') : null,
            h(
                'div',
                { class: 'cards', style: 'align-items:center;gap:1em' },
                v.starter ? h('span', {}, 'Starter ', C.card(v.starter)) : null,
                v.phase === 'peg' ? h('span', { style: 'font-size:1.6em' }, h('b', {}, v.count)) : null,
                v.phase === 'peg' ? C.row(v.run, true) : null,
            ),
            v.last ? h('p', {}, v.last) : null,
            v.shows
                ? h(
                      'div',
                      {},
                      v.shows.map((s) => h('p', {}, s)),
                  )
                : null,
            acts,
            hand ? [h('h3', {}, 'Your hand'), hand] : h('p', { class: 'muted' }, 'You are watching.'),
            L.board(v.board),
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
export default cribbage;
