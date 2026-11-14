// Color Match (apps/colormatch.go): click a card that matches to play it; a wild
// asks for its colour. Draw when you can't (or won't), then play it or keep it.
import type { Start } from '../types.ts';
import * as L from './lib.ts';
import * as C from './cards.ts';

const h = L.h;

// apps/colormatch.go View
interface View {
    phase: 'lobby' | 'play' | 'over';
    lobby: L.Lobby;
    seats: string[];
    counts: number[];
    me: number;
    turn: number;
    dir: number;
    top: string;
    color: string;
    stock: number;
    last: string;
    left: number;
    board: L.Player[];
    hand?: string[];
    legal?: string[];
    drawn?: string;
}
type Event = { kind: 'nope'; why: string };
type Out = L.LobbyMsg | { kind: 'play'; card: string; color: string } | { kind: 'draw' } | { kind: 'pass' };

const HUE: Record<string, string> = { R: '#e8323c', Y: '#f5c400', G: '#2fa84f', B: '#1f6fd6', W: '#222' };
const NAME: Record<string, string> = { R: 'Red', Y: 'Yellow', G: 'Green', B: 'Blue' };
const SIGN: Record<string, string> = { S: '⊘', R: '⇄', D: '+2' };

// a card's face: its colour, its number or sign
const face = (c: string, opts: { onclick?: () => void; disabled?: boolean; big?: boolean } = {}) => {
    const label = c === 'WW' ? 'W' : c === 'W4' ? '+4' : (SIGN[c[1]] ?? c[1]);
    const wild = c[0] === 'W';
    const bg = wild ? `conic-gradient(${HUE.R} 0 25%, ${HUE.Y} 0 50%, ${HUE.G} 0 75%, ${HUE.B} 0)` : HUE[c[0]];
    const style = `background:${bg};color:#fff;text-shadow:0 1px 2px #000;font-size:${opts.big ? '1.5em' : '1.1em'};border:0.15em solid #fff`;
    return opts.onclick ? h('button', { class: 'card', style, disabled: opts.disabled, on: { click: opts.onclick } }, label) : h('span', { class: 'card', style }, label);
};

const colorMatch: Start<View, Event, Out> = (ctx, view) => {
    let wild: string | null = null; // a wild picked, waiting for its colour
    const m = L.mount<View, Out>(ctx, 'cover', (v, kit) => {
        if (v.phase === 'lobby')
            return L.lobby(
                v.lobby,
                kit,
                'Color Match',
                'Two to four players (a bot joins if you are alone). Match the top card by colour or number; wilds pick the colour. First to empty their hand wins the points left in everyone else’s.',
            );
        const mine = v.phase === 'play' && v.me === v.turn;
        const legal = new Set(v.legal || []);
        if (!mine) wild = null;
        const hand = v.hand
            ? h(
                  'div',
                  { class: 'cards' },
                  v.hand.map((c) =>
                      face(c, {
                          disabled: !legal.has(c),
                          onclick: () => {
                              if (c[0] === 'W') {
                                  wild = c;
                                  m.redraw();
                              } else kit.send({ kind: 'play', card: c, color: '' });
                          },
                      }),
                  ),
              )
            : null;
        const acts = !mine
            ? h('p', { class: 'muted' }, v.phase === 'play' ? `${v.seats[v.turn]} to play` : '')
            : wild
              ? h(
                    'p',
                    {},
                    'Choose a colour: ',
                    Object.keys(NAME).map((k) =>
                        h(
                            'button',
                            {
                                style: `background:${HUE[k]}`,
                                on: {
                                    click: () => {
                                        kit.send({ kind: 'play', card: wild!, color: k });
                                        wild = null;
                                    },
                                },
                            },
                            NAME[k],
                        ),
                    ),
                    L.button('Cancel', () => {
                        wild = null;
                        m.redraw();
                    }),
                )
              : h(
                    'p',
                    {},
                    h('b', {}, 'Your turn. '),
                    v.drawn ? [L.button('Keep it', () => kit.send({ kind: 'pass' })), ' or play the card you drew. '] : L.button('Draw a card', () => kit.send({ kind: 'draw' })),
                    ' ',
                    L.timer(v.left),
                );
        return [
            h('h2', {}, v.phase === 'over' ? 'Game over' : 'Color Match'),
            C.seats(v.seats, v.me, v.phase === 'play' ? v.turn : -1, v.counts),
            h('p', { class: 'muted' }, v.dir > 0 ? 'Play goes →' : 'Play goes ←'),
            h(
                'div',
                { class: 'cards', style: 'align-items:center;gap:1em' },
                face(v.top, { big: true }),
                h('span', {}, 'Colour: ', h('b', { style: `color:${HUE[v.color]}` }, NAME[v.color] ?? v.color)),
                h('span', { class: 'muted' }, `${v.stock} to draw`),
            ),
            v.last ? h('p', {}, v.last) : null,
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
export default colorMatch;
