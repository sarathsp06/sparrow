import { loadFont as loadInter } from "@remotion/google-fonts/Inter";
import { loadFont as loadMono } from "@remotion/google-fonts/JetBrainsMono";
import { loadFont as loadSerif } from "@remotion/google-fonts/InstrumentSerif";

const inter = loadInter("normal", { weights: ["400", "500", "600", "700"], subsets: ["latin"] });
const mono = loadMono("normal", { weights: ["400", "500", "700"], subsets: ["latin"] });
const serif = loadSerif("normal", { weights: ["400"], subsets: ["latin"] });

export const C = {
  cream: "#F4F2ED", // warm paper — matches the dashboard's light theme (web/src/app.css)
  ink: "#0B0F14",
  navy: "#181615",
  navyElev: "#252220",
  teal: "#00ADD8",
  tealDeep: "#0E7490",
  coral: "#F97316",
  coralDeep: "#C2410C",
  coralText: "#DC6513", // AA-contrast coral for small text on cream
  red: "#E5484D",
} as const;

export const F = {
  display: `${inter.fontFamily}, -apple-system, "Helvetica Neue", Arial, sans-serif`,
  mono: `${mono.fontFamily}, Menlo, Consolas, monospace`,
  serif: `${serif.fontFamily}, Georgia, serif`,
} as const;
