// One-off: rasterize the Sparrow social card SVG to public/og.png (1200x630).
// Palette from src/styles/sparrow-tokens.css (paper/amber light theme).
// Run: node scripts/gen-og.mjs
import sharp from 'sharp';
import { fileURLToPath } from 'node:url';

const out = fileURLToPath(new URL('../public/og.png', import.meta.url));

const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <rect width="1200" height="630" fill="#faf9f7"/>
  <rect x="0" y="0" width="1200" height="12" fill="#f2a93b"/>
  <g font-family="'Space Grotesk', ui-sans-serif, system-ui, sans-serif">
    <circle cx="150" cy="180" r="46" fill="#f2a93b"/>
    <text x="150" y="198" font-size="52" font-weight="700" fill="#1a1204" text-anchor="middle">S</text>
    <text x="222" y="200" font-size="96" font-weight="700" fill="#1a1a1a">Sparrow</text>
    <text x="150" y="320" font-size="52" font-weight="600" fill="#55524c">Self-hosted webhook delivery</text>
    <text x="150" y="410" font-size="30" font-weight="500" fill="#b06a10">At-least-once delivery · classified retries · HMAC &amp; Ed25519 signing</text>
    <text x="150" y="560" font-size="26" font-weight="500" fill="#8a867e">github.com/sarathsp06/sparrow</text>
  </g>
</svg>`;

await sharp(Buffer.from(svg)).png().toFile(out);
console.log('wrote', out);
