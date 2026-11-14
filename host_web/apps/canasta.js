// Built from web/viewer/apps/canasta.ts by `npm run build` (vite.config.ts): change it there.
import { button, h as h$1, lobby, mount, timer } from "./lib.js";
import { back, card, face, hand, picks, row, seats } from "./cards.js";
const h = h$1;
const canasta = (ctx, view) => {
	const pick = picks();
	const m = mount(ctx, "cover", (v, kit) => {
		if (v.phase === "lobby") return lobby(v.lobby, kit, "Canasta", "Four seats as two teams, partners across (bots fill the empty ones). Meld three or more of a rank; jokers and twos are wild. Seven make a canasta. First team to 2,500.");
		pick.sync(v.hand);
		const mine = v.phase === "play" && v.turn === v.me;
		const myTeam = v.me >= 0 ? v.me % 2 : -1;
		const chosen = v.hand ? pick.cards(v.hand) : [];
		const team = (t) => h("div", { style: "flex:1;min-width:14em" }, h("h3", {}, `${v.seats[t]} & ${v.seats[t + 2]}${t === myTeam ? " (you)" : ""}: ${v.team[t]}`), (v.threes[t] || []).length ? h("div", { class: "muted" }, "Red threes ", row(v.threes[t], true)) : null, v.melds[t].length ? v.melds[t].map((ml) => h("div", {
			style: `display:flex;align-items:center;gap:0.4em${t === myTeam && mine && v.drew && chosen.length ? ";cursor:pointer" : ""}`,
			on: { click: () => {
				if (t !== myTeam || !mine || !v.drew || !chosen.length) return;
				const natural = ml.find((c) => c !== "JK" && c[0] !== "2");
				kit.send({
					kind: "meld",
					cards: chosen,
					rank: natural ? natural[0] : ""
				});
			} }
		}, row(ml, true), ml.length >= 7 ? h("b", {}, ml.some((c) => c === "JK" || c[0] === "2") ? "canasta" : "natural canasta") : null)) : h("p", { class: "muted" }, "No melds yet."));
		let acts;
		if (!mine) acts = h("p", { class: "muted" }, v.phase === "play" ? `${v.seats[v.turn]} to play` : "");
		else if (!v.drew) acts = h("p", {}, h("b", {}, "Your turn: "), button("Draw two", () => kit.send({ kind: "draw" })), " ", button(`Take the pile with ${chosen.length ? chosen.map(face).join(" ") : "…"}`, () => kit.send({
			kind: "pile",
			cards: chosen
		}), { disabled: !v.top || !v.pile }), " ", timer(v.left), h("div", {
			class: "muted",
			style: "font-size:0.8em"
		}, "To take the pile, pick the cards from your hand that meld with its top card first."));
		else acts = h("p", {}, h("b", {}, "Your turn: "), button("Meld", () => kit.send({
			kind: "meld",
			cards: chosen,
			rank: ""
		}), { disabled: chosen.length < 1 }), " ", button("Discard", () => kit.send({
			kind: "discard",
			card: chosen[0]
		}), { disabled: chosen.length !== 1 }), " ", timer(v.left), h("div", {
			class: "muted",
			style: "font-size:0.8em"
		}, "Pick cards, then Meld (or click one of your team's melds to add to it)."));
		return [
			h("h2", {}, v.phase === "over" ? "Game over" : "Canasta"),
			seats(v.seats, v.me, v.phase === "play" ? v.turn : -1, v.counts),
			h("div", {
				class: "cards",
				style: "align-items:center;gap:1em"
			}, h("span", {}, "Pile ", v.top ? card(v.top) : back(false), h("span", { class: "muted" }, ` ${v.pile}${v.frozen ? ", frozen" : ""}`)), h("span", { class: "muted" }, `${v.stock} in the stock`)),
			v.last ? h("p", {}, v.last) : null,
			h("div", { class: "cols" }, team(0), team(1)),
			acts,
			v.hand ? [h("h3", {}, `Your hand${v.melded ? "" : ` (your team's first meld must be worth ${v.min})`}`), hand(v.hand, (i) => {
				pick.toggle(i);
				m.redraw();
			}, (i) => pick.has(i))] : h("p", { class: "muted" }, "You are watching.")
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
export { canasta as default };
