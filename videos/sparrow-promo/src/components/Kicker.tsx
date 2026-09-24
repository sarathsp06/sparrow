import React from "react";
import { interpolate, random, useCurrentFrame } from "remotion";
import { C, F } from "../theme";

const GLYPHS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#/$%";

// Section label in the top-left corner. Letters resolve from scrambled glyphs,
// left to right, like a terminal decoding the heading.
export const Kicker: React.FC<{ text: string; delay?: number; color?: string; instant?: boolean }> = ({
  text,
  delay = 0,
  color = C.ink,
  instant = false,
}) => {
  const frame = useCurrentFrame();
  const resolved = instant
    ? text.length
    : interpolate(frame - delay, [0, 22], [0, text.length], {
        extrapolateLeft: "clamp",
        extrapolateRight: "clamp",
      });
  const shown = text
    .split("")
    .map((ch, i) => {
      if (i < resolved || ch === " ") return ch;
      if (i > resolved + 6) return " ";
      return GLYPHS[Math.floor(random(`k-${text}-${i}-${Math.floor(frame / 2)}`) * GLYPHS.length)];
    })
    .join("");

  return (
    <div
      style={{
        position: "absolute",
        top: 64,
        left: 72,
        display: "flex",
        alignItems: "center",
        gap: 14,
        fontFamily: F.mono,
        fontSize: 22,
        fontWeight: 500,
        letterSpacing: "0.16em",
        color,
        whiteSpace: "pre",
        opacity: instant ? 1 : interpolate(frame - delay, [0, 6], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
      }}
    >
      <span
        style={{
          width: 12,
          height: 12,
          borderRadius: 3,
          background: C.coral,
          rotate: `${frame * 2}deg`,
          boxShadow: "0 0 12px rgba(249,115,22,0.6)",
        }}
      />
      {shown}
    </div>
  );
};
