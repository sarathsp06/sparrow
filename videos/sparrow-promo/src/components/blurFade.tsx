import React from "react";
import { AbsoluteFill } from "remotion";
import type {
  TransitionPresentation,
  TransitionPresentationComponentProps,
} from "@remotion/transitions";

type Props = Record<string, never>;

// Crossfade with a little depth: the outgoing scene softens and drifts back,
// the incoming one sharpens into place.
const BlurFadePresentation: React.FC<TransitionPresentationComponentProps<Props>> = ({
  children,
  presentationDirection,
  presentationProgress: p,
}) => {
  const entering = presentationDirection === "entering";
  const style: React.CSSProperties = entering
    ? { opacity: p, filter: `blur(${(1 - p) * 10}px)`, scale: String(1.03 - p * 0.03) }
    : { opacity: 1 - p, filter: `blur(${p * 10}px)`, scale: String(1 - p * 0.03) };
  return <AbsoluteFill style={style}>{children}</AbsoluteFill>;
};

export const blurFade = (): TransitionPresentation<Props> => ({
  component: BlurFadePresentation,
  props: {},
});
