// Built from web/viewer/apps/cards.ts by `npm run build` (vite.config.ts): change it there.
import { h as h$1 } from "./lib.js";
const h = h$1;
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
const style = document.createElement("style");
style.textContent = css;
document.head.appendChild(style);
const SUITS = {
	C: "♣",
	D: "♦",
	S: "♠",
	H: "♥"
};
const RANKS = { T: "10" };
const face = (c) => c === "JK" ? "★" : `${RANKS[c[0]] ?? c[0]}${SUITS[c[1]] ?? ""}`;
const card = (c, opts = {}) => {
	const cls = `card${c[1] === "D" || c[1] === "H" ? " red" : ""}${opts.on ? " on" : ""}${opts.small ? " small" : ""}`;
	const kids = c === "JK" ? [h("span", {}, "JOKER".slice(0, 2)), h("span", { class: "s" }, "★")] : [h("span", {}, RANKS[c[0]] ?? c[0]), h("span", { class: "s" }, SUITS[c[1]] ?? "")];
	return opts.onclick ? h("button", {
		class: cls,
		disabled: opts.disabled,
		title: face(c),
		on: { click: opts.onclick }
	}, kids) : h("span", {
		class: cls,
		title: face(c)
	}, kids);
};
const back = (small = true) => h("span", { class: `card back${small ? " small" : ""}` });
const row = (cs, small = false) => h("div", { class: "cards" }, (cs || []).map((c) => card(c, { small })));
const hand = (cs, pick, on, ok = () => true) => h("div", { class: "cards" }, cs.map((c, i) => card(c, {
	onclick: () => pick(i),
	on: on(i),
	disabled: !ok(c)
})));
const seats = (names, me, turn, counts, extra) => h("div", { class: "seatrow" }, names.map((n, i) => h("div", { class: `seat${i === turn ? " now" : ""}${i === me ? " me" : ""}` }, i === me ? `${n} (you)` : n, counts ? h("span", { class: "muted" }, ` · ${counts[i]}`) : null, extra ? extra(i) : null)));
const picks = () => {
	let seen = "";
	let set = /* @__PURE__ */ new Set();
	return {
		sync(cs) {
			const k = (cs || []).join(",");
			if (k !== seen) {
				seen = k;
				set = /* @__PURE__ */ new Set();
			}
		},
		toggle(i, most = Infinity) {
			if (set.has(i)) set.delete(i);
			else if (set.size < most) set.add(i);
			else if (most === 1) set = /* @__PURE__ */ new Set([i]);
		},
		has: (i) => set.has(i),
		size: () => set.size,
		cards: (cs) => [...set].sort((a, b) => a - b).map((i) => cs[i]),
		clear() {
			set = /* @__PURE__ */ new Set();
		}
	};
};
export { back, card, face, hand, picks, row, seats };
