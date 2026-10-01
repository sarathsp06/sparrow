import React from "react";
import { linearTiming, TransitionSeries } from "@remotion/transitions";
import { slide } from "@remotion/transitions/slide";
import { Audio } from "@remotion/media";
import { AbsoluteFill, interpolate, Sequence, staticFile } from "remotion";
import { Backdrop } from "../components/Backdrop";
import { blurFade } from "../components/blurFade";
import { FlyBy } from "../components/FlyBy";
import { Captions } from "./Captions";
import { FPS, FRAMES, sceneFrames, sceneStart, TOTAL_FRAMES, XFADE } from "./timing";
import { SCENES } from "./scenes";

// "Three nights of one webhook" — a faceless explainer built with the
// faceless-explainer method (see explainer/STORYBOARD.md). Scene lengths come
// from the generated narration (timing.json), so re-running the voiceover
// script re-times the whole film.

// Seam per scene index (the transition INTO scene i). The chapter run pushes
// left on one stage; everything else blur-fades. Scene 1 slides up (promo).
const PUSH = new Set([3, 4, 6, 7]);

// Scene 0 sits on the dark ground; captions switch skin there.
const DARK_UNTIL = sceneStart(1) + XFADE / 2;

const TAKEAWAY = FRAMES.findIndex((f) => f.id === "12-takeaway");

export const SparrowExplainer: React.FC = () => (
  <AbsoluteFill>
    <Backdrop />
    <TransitionSeries>
      {SCENES.flatMap((Scene, i) => [
        ...(i === 0
          ? []
          : [
              i === 1 ? (
                <TransitionSeries.Transition key={`t${i}`} presentation={slide({ direction: "from-bottom" })} timing={linearTiming({ durationInFrames: XFADE })} />
              ) : PUSH.has(i) ? (
                <TransitionSeries.Transition key={`t${i}`} presentation={slide({ direction: "from-right" })} timing={linearTiming({ durationInFrames: XFADE })} />
              ) : (
                <TransitionSeries.Transition key={`t${i}`} presentation={blurFade()} timing={linearTiming({ durationInFrames: XFADE })} />
              ),
            ]),
        <TransitionSeries.Sequence key={FRAMES[i].id} durationInFrames={sceneFrames(i)} name={FRAMES[i].id}>
          <Scene />
        </TransitionSeries.Sequence>,
      ])}
    </TransitionSeries>

    {/* The sparrow rides into the explanation, and out of it. */}
    <Sequence from={sceneStart(2) - XFADE} durationInFrames={26} name="Sparrow fly-by">
      <FlyBy y={220} />
    </Sequence>
    <Sequence from={sceneStart(TAKEAWAY) - XFADE} durationInFrames={26} name="Sparrow fly-by">
      <FlyBy y={700} reverse />
    </Sequence>

    <Captions dark={(f) => f < DARK_UNTIL} />

    {/* Narration: one clip per cue, placed on the cue timeline. */}
    {FRAMES.flatMap((f, i) =>
      f.cues.map((c) => (
        <Sequence key={c.src} from={sceneStart(i) + Math.round(c.start * FPS)} name={`VO ${f.id}`}>
          <Audio src={staticFile(c.src)} volume={1} />
        </Sequence>
      )),
    )}

    {/* Bed + transition whooshes */}
    <Audio
      src={staticFile("bgm/pad.mp3")}
      volume={(f) =>
        interpolate(f, [0, 45, TOTAL_FRAMES - 60, TOTAL_FRAMES], [0, 0.13, 0.13, 0], {
          extrapolateLeft: "clamp",
          extrapolateRight: "clamp",
        })
      }
      name="Ambient pad"
    />
    {FRAMES.slice(1).map((f, k) => (
      <Sequence key={f.id} from={sceneStart(k + 1) - 4} name="Whoosh">
        <Audio src={staticFile("sfx/whoosh-short.mp3")} volume={0.18} />
      </Sequence>
    ))}
  </AbsoluteFill>
);

export const EXPLAINER_DURATION = TOTAL_FRAMES;
