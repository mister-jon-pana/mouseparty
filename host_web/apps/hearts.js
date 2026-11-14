// Built from web/viewer/apps/hearts.ts by `npm run build` (vite.config.ts): change it there.
import { board, button, h as h$1, lobby, mount, timer } from "./lib.js";
import { card, hand, picks, row, seats } from "./cards.js";
const h = h$1;
const WAY = {
	1: "to your left",
	3: "to your right",
	2: "across"
};
const hearts = (ctx, view) => {
	const pick = picks();
	const m = mount(ctx, "cover", (v, kit) => {
		if (v.phase === "lobby") return lobby(v.lobby, kit, "Hearts", "Four seats (bots fill the empty ones). Avoid hearts, a point each, and the queen of spades, 13. Take them all and everyone else gets 26. Lowest score wins at 50.");
		pick.sync(v.hand);
		const mine = v.me >= 0 && v.turn === v.me && v.phase === "play";
		const table = (v.trick || []).map((c, k) => h("div", { style: "text-align:center" }, card(c), h("div", {
			class: "muted",
			style: "font-size:0.7em"
		}, v.seats[(v.leader + k) % 4])));
		let acts = null;
		if (v.phase === "pass" && v.hand) acts = v.passed ? h("p", { class: "muted" }, "Passed. Waiting for the others… ", timer(v.left)) : h("p", {}, `Pick three cards to pass ${WAY[v.passTo]}. `, button(`Pass ${pick.size()}/3`, () => kit.send({
			kind: "pass",
			cards: pick.cards(v.hand)
		}), { disabled: pick.size() !== 3 }), " ", timer(v.left));
		else if (v.phase === "play") acts = mine ? h("p", {}, h("b", {}, "Your turn: "), "click a card. ", timer(v.left)) : h("p", { class: "muted" }, `${v.seats[v.turn]} to play`);
		const hand$1 = v.hand ? hand(v.hand, (i) => {
			if (v.phase === "pass" && !v.passed) {
				pick.toggle(i, 3);
				m.redraw();
			} else if (mine) kit.send({
				kind: "play",
				card: v.hand[i]
			});
		}, (i) => pick.has(i), (c) => v.phase === "pass" || mine && (v.legal || []).includes(c)) : null;
		return [
			h("h2", {}, v.phase === "over" ? "Game over" : "Hearts"),
			seats(v.seats, v.me, v.phase === "play" ? v.turn : -1, v.counts, (i) => h("span", { class: "muted" }, ` · ${v.scores[i]} (+${v.taken[i]})`)),
			v.last ? h("p", {}, v.last) : null,
			h("div", {
				class: "cards",
				style: "min-height:4.6em;gap:0.8em"
			}, table.length ? table : v.lastTrick ? h("div", { class: "muted" }, "Last trick: ", row(v.lastTrick, true)) : null),
			acts,
			hand$1 ? [h("h3", {}, "Your hand"), hand$1] : h("p", { class: "muted" }, "You are watching."),
			board(v.board, "Scores (lowest wins)")
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
export { hearts as default };
