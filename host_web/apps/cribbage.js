// Built from web/viewer/apps/cribbage.ts by `npm run build` (vite.config.ts): change it there.
import { board, button, h as h$1, lobby, mount, timer } from "./lib.js";
import { card, hand, picks, row, seats } from "./cards.js";
const h = h$1;
const cribbage = (ctx, view) => {
	const pick = picks();
	const m = mount(ctx, "cover", (v, kit) => {
		if (v.phase === "lobby") return lobby(v.lobby, kit, "Cribbage", "Two to four players (a bot joins if you are alone; four play as partners across). Peg to 31 for fifteens, pairs and runs, then count your hand with the starter. First to 121.");
		pick.sync(v.hand);
		const mine = v.phase === "peg" && v.turn === v.me;
		const legal = new Set(v.legal || []);
		const peg = (i) => h("span", {}, ` · ${v.scores[i]}`, h("span", { style: `display:inline-block;width:4em;height:0.35em;margin-left:0.4em;background:linear-gradient(90deg,var(--lime) ${100 * Math.min(v.scores[i], v.to) / v.to}%,rgb(255 255 255 / .15) 0);border-radius:1em` }), i === v.dealer ? h("span", { class: "muted" }, " (dealer)") : null);
		let acts = null;
		if (v.phase === "crib" && v.hand) acts = v.gave ? h("p", { class: "muted" }, "Waiting for the others… ", timer(v.left)) : h("p", {}, `Pick ${v.toCrib} for ${v.dealer === v.me ? "your" : `${v.seats[v.dealer]}'s`} crib. `, button("Put in the crib", () => kit.send({
			kind: "crib",
			cards: pick.cards(v.hand)
		}), { disabled: pick.size() !== v.toCrib }), " ", timer(v.left));
		else if (v.phase === "peg") acts = mine ? h("p", {}, h("b", {}, "Your turn: "), "click a card. ", timer(v.left)) : h("p", { class: "muted" }, `${v.seats[v.turn]} to play`);
		const hand$1 = v.hand ? hand(v.hand, (i) => {
			if (v.phase === "crib" && !v.gave) {
				pick.toggle(i, v.toCrib);
				m.redraw();
			} else if (mine) kit.send({
				kind: "peg",
				card: v.hand[i]
			});
		}, (i) => pick.has(i), (c) => v.phase === "crib" && !v.gave || mine && legal.has(c)) : null;
		return [
			h("h2", {}, v.phase === "over" ? "Game over" : "Cribbage"),
			seats(v.seats, v.me, v.phase === "peg" ? v.turn : -1, v.counts, peg),
			v.teams ? h("p", { class: "muted" }, "Partners sit across: their points are yours.") : null,
			h("div", {
				class: "cards",
				style: "align-items:center;gap:1em"
			}, v.starter ? h("span", {}, "Starter ", card(v.starter)) : null, v.phase === "peg" ? h("span", { style: "font-size:1.6em" }, h("b", {}, v.count)) : null, v.phase === "peg" ? row(v.run, true) : null),
			v.last ? h("p", {}, v.last) : null,
			v.shows ? h("div", {}, v.shows.map((s) => h("p", {}, s))) : null,
			acts,
			hand$1 ? [h("h3", {}, "Your hand"), hand$1] : h("p", { class: "muted" }, "You are watching."),
			board(v.board)
		];
	});
	m.draw(view);
	return {
		view: m.draw,
		message(e) {
			if (e.kind === "nope") m.kit.note(e.why);
		}
	};
};
export { cribbage as default };
