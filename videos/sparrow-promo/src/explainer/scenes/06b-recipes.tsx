import React from "react";
import {
  AbsoluteFill,
  Audio,
  Img,
  Sequence,
  staticFile,
  useCurrentFrame,
} from "remotion";
import { C, F } from "../../theme";
import { cueFrame, useReveal, ease, frameTiming } from "../timing";
import { StepRail, card, Terminal } from "../ui";
import { Wire, Comet, Ripple } from "../../components/Wire";

const ID = "06b-recipes";

// Recipe tiles — PagerDuty is the hero (first position, highlighted at c2).
// Slack is just one tile among seven, never leading.
const RECIPES = [
  { label: "PagerDuty", icon: "icons/pagerduty.svg", fmt: "Events API v2" },
  { label: "Twilio", icon: "icons/twilio.svg", fmt: "Twilio Messages" },
  { label: "SendGrid", icon: "icons/sendgrid.svg", fmt: "SendGrid v3" },
  { label: "ClickHouse", icon: "icons/clickhouse.svg", fmt: "JSONEachRow" },
  { label: "Discord", icon: "icons/discord.svg", fmt: "embeds" },
  { label: "ntfy", icon: "icons/ntfy.svg", fmt: "ntfy topic" },
  { label: "Slack", icon: "icons/slack.svg", fmt: "Block Kit" },
];

// c1 text: "PagerDuty, Twilio, SendGrid, ClickHouse and more, one YAML file each."
// Words: PagerDuty,(0) Twilio,(1) SendGrid,(2) ClickHouse(3) and(4) more,(5) one(6) YAML(7) file(8) each.(9)
const C1_DUR = frameTiming(ID).cues[1].dur;
// First four recipes land on their spoken names; last three ("and more") land together
const TILE_OFFSETS = [
  0,                      // PagerDuty — first word
  (1 / 9) * C1_DUR,      // Twilio
  (2 / 9) * C1_DUR,      // SendGrid
  (3 / 9) * C1_DUR,      // ClickHouse
  (5 / 9) * C1_DUR,      // Discord — "more"
  (5 / 9) * C1_DUR + 0.15, // ntfy — slight stagger
  (5 / 9) * C1_DUR + 0.3,  // Slack — slight stagger
];

// Grid layout: 4 top row + 3 bottom row
const GRID_LEFT = 520;
const GRID_TOP = 300;
const TILE_W = 280;
const TILE_H = 80;
const TILE_GAP_X = 16;
const TILE_GAP_Y = 16;

const tilePos = (i: number) => {
  const row = i < 4 ? 0 : 1;
  const col = i < 4 ? i : i - 4;
  // Centre the bottom row (3 items in a 4-col grid)
  const xOffset = row === 1 ? (TILE_W + TILE_GAP_X) / 2 : 0;
  return {
    x: GRID_LEFT + col * (TILE_W + TILE_GAP_X) + xOffset,
    y: GRID_TOP + row * (TILE_H + TILE_GAP_Y),
  };
};

// PagerDuty recipe YAML content (mirrors satellites/recipes/pagerduty.yaml)
const RECIPE_YAML = [
  { indent: 0, key: "webhook", value: "" },
  { indent: 1, key: "url", value: "" },
  { indent: 1, key: "", value: "https://events.pagerduty.com/v2/enqueue", small: true },
  { indent: 1, key: "headers", value: "" },
  { indent: 2, key: "Content-Type", value: "application/json" },
  { indent: 0, key: "subscription", value: "" },
  { indent: 1, key: "transform_template", value: "|" },
  { indent: 2, key: "", value: '"event_action": "trigger",' },
  { indent: 2, key: "", value: '"dedup_key": {{.event_id | json}},' },
  { indent: 2, key: "", value: '"payload": { "summary": ... }' },
];

export const Recipes: React.FC = () => {
  const frame = useCurrentFrame();
  const r = useReveal();

  const c0 = cueFrame(ID, 0);
  const c2 = cueFrame(ID, 2);

  // -- c0: compact request card folds into recipe file
  const requestCardPop = r(c0 + 4, 18);
  const c0dur = frameTiming(ID).cues[0].dur;
  const foldStart = c0 + Math.round(c0dur * 15); // fold midway through cue
  const foldProgress = ease(frame, foldStart, foldStart + 16);
  const recipeFilePop = ease(frame, foldStart + 8, foldStart + 22);

  // -- c2: terminal + PagerDuty highlight + wire
  const termPop = r(c2 + 4);
  const pdHighlight = ease(frame, c2 + 20, c2 + 34);

  // PagerDuty tile position for wire target
  const pdPos = tilePos(0);
  const pdCenter = { x: pdPos.x + TILE_W / 2, y: pdPos.y + TILE_H / 2 };

  // Terminal position
  const TERM_LEFT = 520;
  const TERM_TOP = 600;
  const TERM_W = 720;
  const termCenter = { x: TERM_LEFT + TERM_W / 2, y: TERM_TOP };

  return (
    <AbsoluteFill>
      <StepRail active={2} />

      {/* Title block: same upper-left style as 06-sign with RECIPES tag */}
      <div
        style={{
          position: "absolute",
          top: 140,
          left: 120,
          fontFamily: F.display,
          color: C.ink,
          opacity: r(c0),
          transform: `translateY(${(1 - r(c0)) * 18}px)`,
          filter: `blur(${(1 - r(c0)) * 6}px)`,
        }}
      >
        <div
          style={{
            fontFamily: F.mono,
            fontSize: 22,
            letterSpacing: "0.18em",
            color: C.coralText,
            fontWeight: 700,
          }}
        >
          STEP 3
        </div>
        <div
          style={{
            fontSize: 76,
            fontWeight: 700,
            letterSpacing: "-0.03em",
            lineHeight: 1.05,
            display: "flex",
            alignItems: "baseline",
            gap: 20,
          }}
        >
          Build the request
          <span
            style={{
              fontFamily: F.mono,
              fontSize: 18,
              fontWeight: 600,
              letterSpacing: "0.1em",
              color: C.tealDeep,
              background: "rgba(0,173,216,0.10)",
              border: `1.5px solid rgba(0,173,216,0.35)`,
              borderRadius: 999,
              padding: "5px 14px",
              position: "relative",
              top: -6,
            }}
          >
            RECIPES
          </span>
        </div>
      </div>

      {/* c0: compact request card that folds away */}
      {requestCardPop > 0.01 && foldProgress < 1 && (
        <div
          style={{
            position: "absolute",
            left: 120,
            top: 380,
            width: 340,
            opacity: requestCardPop * (1 - foldProgress),
            transform: `translateX(${(1 - requestCardPop) * -20}px) scaleY(${1 - foldProgress * 0.6})`,
            transformOrigin: "top left",
          }}
        >
          <div
            style={{
              ...card,
              padding: "12px 18px",
              fontFamily: F.mono,
              fontSize: 14,
              lineHeight: 1.6,
              color: C.ink,
            }}
          >
            <div style={{ display: "flex", gap: 8, alignItems: "center", marginBottom: 6 }}>
              <span
                style={{
                  color: "#fff",
                  background: C.tealDeep,
                  borderRadius: 5,
                  padding: "1px 8px",
                  fontSize: 12,
                  fontWeight: 700,
                }}
              >
                POST
              </span>
              <span style={{ fontSize: 13, color: "rgba(11,15,20,0.5)" }}>hooks.partner.example</span>
            </div>
            <div style={{ color: "rgba(11,15,20,0.4)", fontSize: 12, letterSpacing: "0.08em", marginBottom: 4 }}>
              HEADERS {"·"} BODY {"·"} SIGNATURE
            </div>
          </div>
        </div>
      )}

      {/* c0: pagerduty.yaml recipe file card (folds in) */}
      {recipeFilePop > 0.01 && (
        <div
          style={{
            position: "absolute",
            left: 120,
            top: 360,
            width: 370,
            opacity: recipeFilePop,
            transform: `translateY(${(1 - recipeFilePop) * 12}px)`,
          }}
        >
          <div
            style={{
              ...card,
              background: C.navy,
              color: C.cream,
              border: "1px solid rgba(232,230,225,0.15)",
              padding: "14px 20px",
              overflow: "hidden",
            }}
          >
            {/* File tab */}
            <div
              style={{
                fontFamily: F.mono,
                fontSize: 14,
                color: C.teal,
                marginBottom: 10,
                display: "flex",
                alignItems: "center",
                gap: 8,
              }}
            >
              <svg width={14} height={16} viewBox="0 0 14 16">
                <path d="M1 1h8l4 4v10H1V1z" fill="none" stroke={C.teal} strokeWidth={1.5} />
                <path d="M9 1v4h4" fill="none" stroke={C.teal} strokeWidth={1.5} />
              </svg>
              pagerduty.yaml
            </div>
            <div style={{ fontFamily: F.mono, fontSize: 15, lineHeight: 1.55 }}>
              {RECIPE_YAML.map((line, i) => {
                const lineAt = foldStart + 12 + i * 3;
                const linePop = ease(frame, lineAt, lineAt + 8);
                return (
                  <div
                    key={i}
                    style={{
                      paddingLeft: line.indent * 16,
                      opacity: linePop,
                      fontSize: "small" in line ? 13 : undefined,
                      whiteSpace: "nowrap",
                    }}
                  >
                    {line.key ? (
                      <>
                        <span style={{ color: C.teal }}>{line.key}</span>
                        {line.value ? `: ${line.value}` : ":"}
                      </>
                    ) : (
                      <span style={{ color: "rgba(232,230,225,0.7)" }}>{line.value}</span>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {/* c1: recipe tiles grid — each lands on its spoken name */}
      {RECIPES.map((recipe, i) => {
        const tileAt = cueFrame(ID, 1, TILE_OFFSETS[i]);
        const pop = r(tileAt);
        const pos = tilePos(i);
        // PagerDuty tile highlights teal at c2
        const isPD = i === 0;
        const highlight = isPD ? pdHighlight : 0;

        return (
          <div
            key={recipe.label}
            style={{
              ...card,
              position: "absolute",
              left: pos.x,
              top: pos.y,
              width: TILE_W,
              height: TILE_H,
              display: "flex",
              alignItems: "center",
              gap: 12,
              padding: "0 18px",
              opacity: pop,
              transform: `translateY(${(1 - pop) * 16}px) scale(${0.92 + pop * 0.08})`,
              borderColor: highlight > 0
                ? `rgba(0,173,216,${0.12 + highlight * 0.6})`
                : "rgba(11,15,20,0.12)",
              borderWidth: highlight > 0 ? 2 : 1,
              boxShadow: highlight > 0
                ? `0 14px 38px rgba(11,15,20,0.08), 0 0 ${highlight * 20}px rgba(0,173,216,${highlight * 0.35})`
                : "0 14px 38px rgba(11,15,20,0.08)",
            }}
          >
            <Img
              src={staticFile(recipe.icon)}
              style={{ width: 28, height: 28, flexShrink: 0 }}
            />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontFamily: F.display,
                  fontSize: 20,
                  fontWeight: 600,
                  color: C.ink,
                }}
              >
                {recipe.label}
              </div>
              <div
                style={{
                  fontFamily: F.mono,
                  fontSize: 14,
                  color: C.tealDeep,
                  marginTop: 1,
                }}
              >
                {recipe.fmt}
              </div>
            </div>
          </div>
        );
      })}

      {/* c2: Terminal */}
      {termPop > 0.01 && (
        <div
          style={{
            position: "absolute",
            left: TERM_LEFT,
            top: TERM_TOP,
            width: TERM_W,
            opacity: termPop,
            transform: `translateY(${(1 - termPop) * 14}px)`,
          }}
        >
          <Terminal title="terminal">
            <div>
              <span style={{ color: C.teal }}>$</span>{" "}
              <TypingText
                text="sparrow use pagerduty --event order.created"
                startFrame={c2 + 8}
                charsPerFrame={1.2}
              />
            </div>
          </Terminal>

          {/* Footnote */}
          <div
            style={{
              fontFamily: F.mono,
              fontSize: 15,
              color: "rgba(11,15,20,0.4)",
              marginTop: 10,
              letterSpacing: "0.04em",
              opacity: ease(frame, c2 + 40, c2 + 54),
            }}
          >
            helper CLI {"·"} satellites/sparrow
          </div>
        </div>
      )}

      {/* c2: wire + comet from terminal to PagerDuty tile */}
      <svg
        width={1920}
        height={1080}
        style={{ position: "absolute", inset: 0, pointerEvents: "none", overflow: "visible" }}
      >
        <Wire
          id="recipe-pagerduty"
          a={termCenter}
          b={pdCenter}
          frame={frame}
          drawStart={c2 + 22}
          drawEnd={c2 + 42}
          color="rgba(0,173,216,0.45)"
          width={2.5}
        />
        <Comet
          a={termCenter}
          b={pdCenter}
          frame={frame}
          start={c2 + 28}
          end={c2 + 44}
          color={C.teal}
          r={7}
        />
        <Ripple
          x={pdCenter.x}
          y={pdCenter.y}
          frame={frame}
          at={c2 + 44}
          color={C.teal}
          size={36}
        />
      </svg>

      {/* SFX */}
      <Sequence from={c2 + 44}>
        <Audio src={staticFile("sfx/ping.mp3")} volume={0.18} />
      </Sequence>
    </AbsoluteFill>
  );
};

// Simple typing animation for terminal commands
const TypingText: React.FC<{
  text: string;
  startFrame: number;
  charsPerFrame: number;
}> = ({ text, startFrame, charsPerFrame }) => {
  const frame = useCurrentFrame();
  const elapsed = Math.max(0, frame - startFrame);
  const chars = Math.min(text.length, Math.floor(elapsed * charsPerFrame));
  const showCursor = frame >= startFrame && chars < text.length;
  return (
    <span>
      {text.slice(0, chars)}
      {showCursor && (
        <span style={{ opacity: 0.7 }}>{"_"}</span>
      )}
    </span>
  );
};
