#!/usr/bin/env node
// Generates the explainer narration from src/explainer/script.json, one clip
// per cue, and writes src/explainer/timing.json — the cue timeline every
// scene reveals against and the captions read from.
//
// Engines (script.json "tts.engine", or TTS env):
//   edge — Microsoft neural voices via `uvx edge-tts` (default; needs network)
//   say  — macOS `say` (offline, robotic; fallback)
// Audition voices via env:
//   VOICE=en-US-AvaMultilingualNeural node scripts/voiceover.mjs
//   TTS=say VOICE=Samantha RATE=178 node scripts/voiceover.mjs
import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const root = new URL("..", import.meta.url).pathname;
const script = JSON.parse(readFileSync(join(root, "src/explainer/script.json"), "utf8"));
const engine = process.env.TTS ?? script.tts.engine;
const voice = process.env.VOICE ?? (engine === script.tts.engine ? script.tts.voice : "Samantha");
const rate = String(process.env.RATE ?? (engine === script.tts.engine ? script.tts.rate : 178));

// Synthesize `text` into `file` (any container ffmpeg can read).
const synth = (text, file) => {
  if (engine === "say") {
    execFileSync("say", ["-v", voice, "-r", rate, "-o", file, text]);
    return;
  }
  execFileSync("uvx", ["edge-tts", "--voice", voice, `--rate=${rate}`, "--text", text, "--write-media", file], {
    stdio: ["ignore", "ignore", "inherit"],
  });
};
const outDir = join(root, "public/vo");

rmSync(outDir, { recursive: true, force: true });
mkdirSync(outDir, { recursive: true });

const probe = (file) =>
  Number(
    execFileSync("ffprobe", ["-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", file])
      .toString()
      .trim(),
  );

const round = (n) => Math.round(n * 1000) / 1000;

const frames = script.frames.map((f) => {
  let t = script.leadIn;
  const cues = f.cues.map((c, i) => {
    const aiff = join(outDir, `${f.id}-${i}.${engine === "say" ? "aiff" : "mp3"}`);
    const wav = join(outDir, `${f.id}-${i}.wav`);
    synth(c.say ?? c.text, aiff);
    // Trim the leading/trailing silence `say` pads with, normalize loudness.
    execFileSync("ffmpeg", [
      "-y", "-loglevel", "error", "-i", aiff,
      "-af",
      "silenceremove=start_periods=1:start_threshold=-50dB,areverse,silenceremove=start_periods=1:start_threshold=-50dB,areverse,loudnorm=I=-16:TP=-1.5:LRA=11",
      "-ar", "48000", "-ac", "1", wav,
    ]);
    rmSync(aiff);
    const dur = probe(wav);
    const cue = { text: c.text, src: `vo/${f.id}-${i}.wav`, start: round(t), dur: round(dur) };
    t += dur + script.cueGap;
    return cue;
  });
  const duration = round(t - script.cueGap + script.tail);
  console.log(`${f.id.padEnd(12)} ${duration.toFixed(2)}s  (${cues.length} cues)`);
  return { id: f.id, duration, cues };
});

const total = frames.reduce((s, f) => s + f.duration, 0);
console.log(`total narration span ≈ ${total.toFixed(1)}s (before transition overlap)`);
writeFileSync(
  join(root, "src/explainer/timing.json"),
  JSON.stringify({ engine, voice, rate, frames }, null, 2) + "\n",
);
