// Canasta (apps/canasta.go): draw two, or pick the cards to take the pile with;
// pick cards and Meld; pick one and Discard to end your turn.
import type { Start } from '../types.ts';
import * as L from './lib.ts';
import * as C from './cards.ts';

const h = L.h;

// apps/canasta.go View
interface View {
    phase: 'lobby' | 'play' | 'over';
    lobby: L.Lobby;
    seats: string[];
    counts: number[];
    me: number;
    turn: number;
    drew: boolean;
    melds: string[][][]; // each team's, a meld a list
    threes: (string[] | null)[];
    team: number[];
    to: number;
    top: string;
    pile: number;
    frozen: boolean;
    stock: number;
    last: string;
    left: number;
    board: L.Player[];
    hand?: string[];
    min?: number;
    melded?: boolean;
}
type Event = { kind: 'nope'; why: string };
type Out = L.LobbyMsg | { kind: 'draw' } | { kind: 'pile'; cards: string[] } | { kind: 'meld'; cards: string[]; rank: string } | { kind: 'discard'; card: string };

const canasta: Start<View, Event, Out> = (ctx, view) => {
    const pick = C.picks();
    const m = L.mount<View, Out>(ctx, 'cover', (v, kit) => {
        if (v.phase === 'lobby')
            return L.lobby(
                v.lobby,
                kit,
                'Canasta',
                'Four seats as two teams, partners across (bots fill the empty ones). Meld three or more of a rank; jokers and twos are wild. Seven make a canasta. First team to 2,500.',
            );
        pick.sync(v.hand);
        const mine = v.phase === 'play' && v.turn === v.me;
        const myTeam = v.me >= 0 ? v.me % 2 : -1;
        const chosen = v.hand ? pick.cards(v.hand) : [];
        const team = (t: number) =>
            h(
                'div',
                { style: 'flex:1;min-width:14em' },
                h('h3', {}, `${v.seats[t]} & ${v.seats[t + 2]}${t === myTeam ? ' (you)' : ''}: ${v.team[t]}`),
                (v.threes[t] || []).length ? h('div', { class: 'muted' }, 'Red threes ', C.row(v.threes[t], true)) : null,
                v.melds[t].length
                    ? v.melds[t].map((ml) =>
                          h(
                              'div',
                              {
                                  style: `display:flex;align-items:center;gap:0.4em${t === myTeam && mine && v.drew && chosen.length ? ';cursor:pointer' : ''}`,
                                  on: {
                                      click: () => {
                                          if (t !== myTeam || !mine || !v.drew || !chosen.length) return;
                                          const natural = ml.find((c) => c !== 'JK' && c[0] !== '2');
                                          kit.send({ kind: 'meld', cards: chosen, rank: natural ? natural[0] : '' });
                                      },
                                  },
                              },
                              C.row(ml, true),
                              ml.length >= 7 ? h('b', {}, ml.some((c) => c === 'JK' || c[0] === '2') ? 'canasta' : 'natural canasta') : null,
                          ),
                      )
                    : h('p', { class: 'muted' }, 'No melds yet.'),
            );
        let acts: L.Kid;
        if (!mine) acts = h('p', { class: 'muted' }, v.phase === 'play' ? `${v.seats[v.turn]} to play` : '');
        else if (!v.drew)
            acts = h(
                'p',
                {},
                h('b', {}, 'Your turn: '),
                L.button('Draw two', () => kit.send({ kind: 'draw' })),
                ' ',
                L.button(`Take the pile with ${chosen.length ? chosen.map(C.face).join(' ') : '…'}`, () => kit.send({ kind: 'pile', cards: chosen }), {
                    disabled: !v.top || !v.pile,
                }),
                ' ',
                L.timer(v.left),
                h('div', { class: 'muted', style: 'font-size:0.8em' }, 'To take the pile, pick the cards from your hand that meld with its top card first.'),
            );
        else
            acts = h(
                'p',
                {},
                h('b', {}, 'Your turn: '),
                L.button('Meld', () => kit.send({ kind: 'meld', cards: chosen, rank: '' }), { disabled: chosen.length < 1 }),
                ' ',
                L.button('Discard', () => kit.send({ kind: 'discard', card: chosen[0] }), { disabled: chosen.length !== 1 }),
                ' ',
                L.timer(v.left),
                h('div', { class: 'muted', style: 'font-size:0.8em' }, "Pick cards, then Meld (or click one of your team's melds to add to it)."),
            );
        return [
            h('h2', {}, v.phase === 'over' ? 'Game over' : 'Canasta'),
            C.seats(v.seats, v.me, v.phase === 'play' ? v.turn : -1, v.counts),
            h(
                'div',
                { class: 'cards', style: 'align-items:center;gap:1em' },
                h('span', {}, 'Pile ', v.top ? C.card(v.top) : C.back(false), h('span', { class: 'muted' }, ` ${v.pile}${v.frozen ? ', frozen' : ''}`)),
                h('span', { class: 'muted' }, `${v.stock} in the stock`),
            ),
            v.last ? h('p', {}, v.last) : null,
            h('div', { class: 'cols' }, team(0), team(1)),
            acts,
            v.hand
                ? [
                      h('h3', {}, `Your hand${v.melded ? '' : ` (your team's first meld must be worth ${v.min})`}`),
                      C.hand(
                          v.hand,
                          (i) => {
                              pick.toggle(i);
                              m.redraw();
                          },
                          (i) => pick.has(i),
                      ),
                  ]
                : h('p', { class: 'muted' }, 'You are watching.'),
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
export default canasta;
