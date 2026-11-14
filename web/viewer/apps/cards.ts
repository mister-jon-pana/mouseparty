// What the card and tile games share (apps/table.go is the host's half): cards and
// tiles drawn as faces, a hand you pick from, the table's seats round you, and
// the scores. Cards are text, rank then suit: "QH", "TC", "2S"; "JK" a joker.
import * as L from './lib.ts';

const h = L.h;

const css = `
.ty .cards { display: flex; flex-wrap: wrap; gap: 0.25em; margin: 0.3em 0; }
.ty .card { width: 2.6em; height: 3.7em; padding: 0.15em 0.2em; margin: 0; border-radius: 0.35em; background: #fdfaf2; color: #1d1d1f;
            text-shadow: none; border: 1px solid rgb(0 0 0 / 0.25); display: inline-flex; flex-direction: column; align-items: center;
            justify-content: center; line-height: 1.05; font-weight: 800; box-shadow: 0 0.12em 0.3em rgb(0 0 0 / 0.35); }
.ty .card.red { color: #d0021b; }
.ty .card .s { font-size: 1.35em; }
.ty .card.back { background: repeating-linear-gradient(45deg, #5b3fd1 0 0.3em, #7a5cf0 0.3em 0.6em); border-color: #fff; }
.ty button.card:hover:not(:disabled) { background: #fff; transform: translateY(-0.3em); }
.ty button.card.on { background: #ffe28a; box-shadow: 0 0 0.8em var(--yellow); transform: translateY(-0.45em); }
.ty button.card:disabled { opacity: 0.45; }
.ty .card.small { width: 1.9em; height: 2.7em; font-size: 0.85em; }
.ty .seatrow { display: flex; gap: 0.6em; flex-wrap: wrap; margin: 0.3em 0; }
.ty .seat { padding: 0.25em 0.7em; border-radius: 0.6em; background: rgb(255 255 255 / 0.08); border: 1px solid rgb(255 255 255 / 0.18); }
.ty .seat.now { border-color: var(--yellow); box-shadow: 0 0 0.6em rgb(255 210 63 / 0.5); }
.ty .seat.me { background: rgb(62 224 255 / 0.18); }`;
const style = document.createElement('style');
style.textContent = css;
document.head.appendChild(style);

const SUITS: Record<string, string> = { C: '♣', D: '♦', S: '♠', H: '♥' };
const RANKS: Record<string, string> = { T: '10' };

// face: what a card says, "10♥"
export const face = (c: string) => (c === 'JK' ? '★' : `${RANKS[c[0]] ?? c[0]}${SUITS[c[1]] ?? ''}`);

// card: a card's face; with onclick, a button
export const card = (c: string, opts: { onclick?: () => void; on?: boolean; disabled?: boolean; small?: boolean } = {}) => {
    const red = c[1] === 'D' || c[1] === 'H';
    const cls = `card${red ? ' red' : ''}${opts.on ? ' on' : ''}${opts.small ? ' small' : ''}`;
    const kids = c === 'JK' ? [h('span', {}, 'JOKER'.slice(0, 2)), h('span', { class: 's' }, '★')] : [h('span', {}, RANKS[c[0]] ?? c[0]), h('span', { class: 's' }, SUITS[c[1]] ?? '')];
    return opts.onclick ? h('button', { class: cls, disabled: opts.disabled, title: face(c), on: { click: opts.onclick } }, kids) : h('span', { class: cls, title: face(c) }, kids);
};

// back: a face-down card, for how many someone holds
export const back = (small = true) => h('span', { class: `card back${small ? ' small' : ''}` });

// row: cards in a row, none clickable
export const row = (cs: string[] | null | undefined, small = false) =>
    h(
        'div',
        { class: 'cards' },
        (cs || []).map((c) => card(c, { small })),
    );

// hand: your cards; pick(i) when one is clicked, on(i) says which are picked, ok(c) which may be
export const hand = (cs: string[], pick: (i: number) => void, on: (i: number) => boolean, ok: (c: string) => boolean = () => true) =>
    h(
        'div',
        { class: 'cards' },
        cs.map((c, i) => card(c, { onclick: () => pick(i), on: on(i), disabled: !ok(c) })),
    );

// seats: who sits where, whose turn it is, and how many cards each holds
export const seats = (names: string[], me: number, turn: number, counts?: number[], extra?: (i: number) => L.Kid) =>
    h(
        'div',
        { class: 'seatrow' },
        names.map((n, i) =>
            h(
                'div',
                { class: `seat${i === turn ? ' now' : ''}${i === me ? ' me' : ''}` },
                i === me ? `${n} (you)` : n,
                counts ? h('span', { class: 'muted' }, ` · ${counts[i]}`) : null,
                extra ? extra(i) : null,
            ),
        ),
    );

// picks: a set of hand indexes kept across redraws, cleared when the hand changes
export const picks = () => {
    let seen = '';
    let set = new Set<number>();
    return {
        sync(cs: string[] | null | undefined) {
            const k = (cs || []).join(',');
            if (k !== seen) {
                seen = k;
                set = new Set();
            }
        },
        toggle(i: number, most = Infinity) {
            if (set.has(i)) set.delete(i);
            else if (set.size < most) set.add(i);
            else if (most === 1) set = new Set([i]);
        },
        has: (i: number) => set.has(i),
        size: () => set.size,
        cards: (cs: string[]) => [...set].sort((a, b) => a - b).map((i) => cs[i]),
        clear() {
            set = new Set();
        },
    };
};
