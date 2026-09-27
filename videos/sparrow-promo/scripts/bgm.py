#!/usr/bin/env python3
"""Synthesizes a soft, royalty-free ambient pad for the explainer bed.

Four slow chords (Am9 - Fmaj7 - Cmaj9 - G6), detuned sine voices with gentle
crossfades, a one-pole low-pass and a faint shimmer. Deterministic output.

    python3 scripts/bgm.py 140   # seconds -> public/bgm/pad.wav
"""
import math
import os
import struct
import sys
import wave

SR = 32000
dur = float(sys.argv[1]) if len(sys.argv) > 1 else 140.0
CHORD_LEN = 8.0
XF = 2.5  # crossfade seconds

def hz(n):  # midi -> hz
    return 440.0 * 2 ** ((n - 69) / 12)

CHORDS = [
    [45, 57, 60, 64, 67, 71],  # Am9
    [41, 53, 57, 60, 64, 69],  # Fmaj7
    [48, 55, 59, 62, 64, 67],  # Cmaj9
    [43, 55, 59, 62, 64, 71],  # G6/add9
]

n = int(dur * SR)
out = [0.0] * n
for ci in range(int(dur / CHORD_LEN) + 2):
    chord = CHORDS[ci % len(CHORDS)]
    start = ci * CHORD_LEN - XF / 2
    end = start + CHORD_LEN + XF
    s0, s1 = max(0, int(start * SR)), min(n, int(end * SR))
    for vi, note in enumerate(chord):
        f = hz(note)
        amp = (0.10 if note < 50 else 0.055) * (0.85 if vi % 2 else 1.0)
        for det in (-0.12, 0.12):
            w = 2 * math.pi * f * (1 + det / 100)
            ph = vi * 1.7 + det
            for i in range(s0, s1):
                t = i / SR
                lt = t - start
                env = min(1.0, lt / XF, (end - t) / XF)
                env = env * env * (3 - 2 * env)  # smoothstep
                out[i] += amp * env * math.sin(w * t + ph)

# one-pole low-pass + slow global swell + faint shimmer
y = 0.0
a = 1 - math.exp(-2 * math.pi * 1400 / SR)
peak = 1e-9
for i in range(n):
    t = i / SR
    y += a * (out[i] - y)
    swell = 0.85 + 0.15 * math.sin(2 * math.pi * t / 23.0)
    shimmer = 0.012 * math.sin(2 * math.pi * hz(88) * t) * (0.5 + 0.5 * math.sin(2 * math.pi * t / 5.3))
    out[i] = (y * swell) + shimmer
    peak = max(peak, abs(out[i]))

fade = 3.0
os.makedirs("public/bgm", exist_ok=True)
with wave.open("public/bgm/pad.wav", "wb") as wf:
    wf.setnchannels(1)
    wf.setsampwidth(2)
    wf.setframerate(SR)
    frames = bytearray()
    for i, v in enumerate(out):
        t = i / SR
        g = min(1.0, t / fade, (dur - t) / fade)
        frames += struct.pack("<h", int(max(-1, min(1, v / peak * 0.8 * g)) * 32767))
    wf.writeframes(bytes(frames))
print(f"wrote public/bgm/pad.wav ({dur:.0f}s)")
