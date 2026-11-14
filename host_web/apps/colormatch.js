// Built from web/viewer/apps/colormatch.ts by `npm run build` (vite.config.ts): change it there.
import { board, button, h as h$1, lobby, mount, timer } from "./lib.js";
import { seats } from "./cards.js";
const h = h$1;
const HUE = {
	R: "#e8323c",
	Y: "#f5c400",
	G: "#2fa84f",
	B: "#1f6fd6",
	W: "#222"
};
const NAME = {
	R: "Red",
	Y: "Yellow",
	G: "Green",
	B: "Blue"
};
const SIGN = {
	S: "⊘",
	R: "⇄",
	D: "+2"
};
const face = (c, opts = {}) => {
	const label = c === "WW" ? "W" : c === "W4" ? "+4" : SIGN[c[1]] ?? c[1];
	const style = `background:${c[0] === "W" ? `conic-gradient(${HUE.R} 0 25%, ${HUE.Y} 0 50%, ${HUE.G} 0 75%, ${HUE.B} 0)` : HUE[c[0]]};color:#fff;text-shadow:0 1px 2px #000;font-size:${opts.big ? "1.5em" : "1.1em"};border:0.15em solid #fff`;
	return opts.onclick ? h("button", {
		class: "card",
		style,
		disabled: opts.disabled,
		on: { click: opts.onclick }
	}, label) : h("span", {
		class: "card",
		style
	}, label);
};
const colorMatch = (ctx, view) => {
	let wild = null;
	const m = mount(ctx, "cover", (v, kit) => {
		if (v.phase === "lobby") return lobby(v.lobby, kit, "Color Match", "Two to four players (a bot joins if you are alone). Match the top card by colour or number; wilds pick the colour. First to empty their hand wins the points left in everyone else’s.");
		const mine = v.phase === "play" && v.me === v.turn;
		const legal = new Set(v.legal || []);
		if (!mine) wild = null;
		const hand = v.hand ? h("div", { class: "cards" }, v.hand.map((c) => face(c, {
			disabled: !legal.has(c),
			onclick: () => {
				if (c[0] === "W") {
					wild = c;
					m.redraw();
				} else kit.send({
					kind: "play",
					card: c,
					color: ""
				});
			}
		}))) : null;
		const acts = !mine ? h("p", { class: "muted" }, v.phase === "play" ? `${v.seats[v.turn]} to play` : "") : wild ? h("p", {}, "Choose a colour: ", Object.keys(NAME).map((k) => h("button", {
			style: `background:${HUE[k]}`,
			on: { click: () => {
				kit.send({
					kind: "play",
					card: wild,
					color: k
				});
				wild = null;
			} }
		}, NAME[k])), button("Cancel", () => {
			wild = null;
			m.redraw();
		})) : h("p", {}, h("b", {}, "Your turn. "), v.drawn ? [button("Keep it", () => kit.send({ kind: "pass" })), " or play the card you drew. "] : button("Draw a card", () => kit.send({ kind: "draw" })), " ", timer(v.left));
		return [
			h("h2", {}, v.phase === "over" ? "Game over" : "Color Match"),
			seats(v.seats, v.me, v.phase === "play" ? v.turn : -1, v.counts),
			h("p", { class: "muted" }, v.dir > 0 ? "Play goes →" : "Play goes ←"),
			h("div", {
				class: "cards",
				style: "align-items:center;gap:1em"
			}, face(v.top, { big: true }), h("span", {}, "Colour: ", h("b", { style: `color:${HUE[v.color]}` }, NAME[v.color] ?? v.color)), h("span", { class: "muted" }, `${v.stock} to draw`)),
			v.last ? h("p", {}, v.last) : null,
			acts,
			hand ? [h("h3", {}, "Your hand"), hand] : h("p", { class: "muted" }, "You are watching."),
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
export { colorMatch as default };
