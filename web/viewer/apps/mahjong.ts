// Mahjong (apps/mahjong.go): on your turn click a tile to discard it (or Win,
// or Kong); when someone discards, claim it (Win, Pung, Kong, Chow) or pass.
import type { Start } from '../types.ts';
import * as L from './lib.ts';
import * as C from './cards.ts';

const h = L.h;

interface Meld {
    tiles: string[];
    concealed: boolean;
}
// apps/mahjong.go View
interface View {
    phase: 'lobby' | 'turn' | 'claim' | 'show' | 'over';
    lobby: L.Lobby;
    seats: string[];
    counts: number[];
    me: number;
    turn: number;
    dealer: number;
    melds: Meld[][];
    discards: string[][];
    wall: number;
    last: string;
    left: number;
    board: L.Player[];
    hand: number;
    of: number;
    result: string;
    winning?: string[];
    discard?: string;
    from?: number;
    tiles?: string[];
    drawn?: string;
    canWin?: boolean;
    kong?: string;
    options?: string[];
    chows?: string[][];
}
type Event = { kind: 'nope'; why: string };
type Out = L.LobbyMsg | { kind: 'discard'; tile: string } | { kind: 'kong'; tile: string } | { kind: 'chow'; tiles: string[] } | { kind: 'win' } | { kind: 'pung' } | { kind: 'pass' };

const SUIT: Record<string, [string, string]> = { p: ['●', '#1f6fd6'], s: ['竹', '#2fa84f'], m: ['萬', '#c62828'] };
const HONOR: Record<string, [string, string]> = {
    Ew: ['東', '#1d1d1f'],
    Sw: ['南', '#1d1d1f'],
    Ww: ['西', '#1d1d1f'],
    Nw: ['北', '#1d1d1f'],
    Rd: ['中', '#c62828'],
    Gd: ['發', '#2fa84f'],
    Wd: ['□', '#1f6fd6'],
};
const NAMES: Record<string, string> = {
    Ew: 'East wind',
    Sw: 'South wind',
    Ww: 'West wind',
    Nw: 'North wind',
    Rd: 'Red dragon',
    Gd: 'Green dragon',
    Wd: 'White dragon',
    p: 'dots',
    s: 'bamboo',
    m: 'characters',
};
const tname = (t: string) => NAMES[t] ?? `${t[0]} of ${NAMES[t[1]]}`;

// tile: a tile's face ("" face down)
const tile = (t: string, opts: { onclick?: () => void; on?: boolean; small?: boolean; disabled?: boolean } = {}) => {
    if (!t) return C.back(opts.small);
    const [mark, color] = HONOR[t] ?? [`${t[0]}${SUIT[t[1]][0]}`, SUIT[t[1]][1]];
    const style = `background:#f6f1e1;color:${color};font-size:${opts.small ? '0.9em' : '1.15em'}`;
    const cls = `card${opts.small ? ' small' : ''}${opts.on ? ' on' : ''}`;
    return opts.onclick ? h('button', { class: cls, style, title: tname(t), disabled: opts.disabled, on: { click: opts.onclick } }, mark) : h('span', { class: cls, style, title: tname(t) }, mark);
};

const mahjong: Start<View, Event, Out> = (ctx, view) => {
    const m = L.mount<View, Out>(ctx, 'cover', (v, kit) => {
        if (v.phase === 'lobby')
            return L.lobby(
                v.lobby,
                kit,
                'Mahjong',
                'Four seats (bots fill the empty ones), Hong Kong style without flowers. Draw and discard to make four sets and a pair (or seven pairs). Four hands; the most fan wins.',
            );
        const mine = v.phase === 'turn' && v.turn === v.me;
        const seat = (i: number) =>
            h(
                'div',
                { class: `seat${i === v.turn && v.phase === 'turn' ? ' now' : ''}${i === v.me ? ' me' : ''}`, style: 'margin:0.2em 0' },
                h('b', {}, v.seats[i]),
                i === v.dealer ? h('span', { class: 'muted' }, ' (dealer)') : null,
                h('span', { class: 'muted' }, ` · ${v.counts[i]} tiles`),
                h(
                    'div',
                    { class: 'cards' },
                    v.melds[i].map((ml) =>
                        h(
                            'span',
                            { style: 'display:inline-flex;gap:0.1em;margin-right:0.4em' },
                            ml.tiles.map((t) => tile(t, { small: true })),
                        ),
                    ),
                ),
                v.discards[i].length
                    ? h(
                          'div',
                          { class: 'cards', style: 'opacity:0.75' },
                          v.discards[i].map((t) => tile(t, { small: true })),
                      )
                    : null,
            );
        let acts: L.Kid = null;
        if (mine)
            acts = h(
                'p',
                {},
                h('b', {}, 'Your turn: '),
                'click a tile to discard it. ',
                v.canWin ? L.button('Mahjong! (win)', () => kit.send({ kind: 'win' }), { on: true }) : null,
                v.kong ? L.button(`Kong of ${tname(v.kong)}`, () => kit.send({ kind: 'kong', tile: v.kong! })) : null,
                ' ',
                L.timer(v.left),
            );
        else if (v.phase === 'claim' && v.options)
            acts = h(
                'p',
                {},
                `${v.seats[v.from!]} discarded `,
                tile(v.discard!),
                ' ',
                v.options.includes('win') ? L.button('Mahjong! (win)', () => kit.send({ kind: 'win' }), { on: true }) : null,
                v.options.includes('kong') ? L.button('Kong', () => kit.send({ kind: 'kong', tile: v.discard! })) : null,
                v.options.includes('pung') ? L.button('Pung', () => kit.send({ kind: 'pung' })) : null,
                (v.chows || []).map((c) => L.button(['Chow ', c.map((t) => tile(t, { small: true }))], () => kit.send({ kind: 'chow', tiles: c }))),
                L.button('Pass', () => kit.send({ kind: 'pass' })),
                ' ',
                L.timer(v.left),
            );
        else if (v.phase === 'claim') acts = h('p', { class: 'muted' }, `${v.seats[v.from!]} discarded `, tile(v.discard!, { small: true }), ' …');
        else if (v.phase === 'turn') acts = h('p', { class: 'muted' }, `${v.seats[v.turn]} to play`);
        return [
            h('h2', {}, v.phase === 'over' ? 'Game over' : `Mahjong · hand ${Math.min(v.hand, v.of)} of ${v.of}`),
            h('p', { class: 'muted' }, `${v.wall} tiles in the wall`),
            v.last ? h('p', {}, v.last) : null,
            v.winning
                ? h(
                      'div',
                      { class: 'cards' },
                      v.winning.map((t) => tile(t)),
                  )
                : null,
            acts,
            v.tiles
                ? [
                      h('h3', {}, 'Your tiles'),
                      h(
                          'div',
                          { class: 'cards' },
                          v.tiles.map((t) => tile(t, { onclick: mine ? () => kit.send({ kind: 'discard', tile: t }) : undefined, on: t === v.drawn })),
                      ),
                  ]
                : h('p', { class: 'muted' }, 'You are watching.'),
            h('div', { class: 'cols' }, [0, 1, 2, 3].map(seat)),
            L.board(v.board, 'Fan'),
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
export default mahjong;
