// Minimal static host for the split-deployment tests: serves the built UI
// (internal/ui/dist) with SPA fallback, like nginx `try_files $uri /index.html`.
// It knows nothing about Sparrow — no config injection, no API proxying.
//
//   node tests/split/static-server.mjs <dir> <port>
import { createReadStream, statSync } from "node:fs";
import { createServer } from "node:http";
import { extname, join, normalize } from "node:path";

const [root, port] = [process.argv[2], Number(process.argv[3] ?? 14173)];
const types = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json", ".svg": "image/svg+xml", ".txt": "text/plain" };

createServer((req, res) => {
  const path = normalize(decodeURIComponent(new URL(req.url, "http://x").pathname)).replace(/^(\.\.[/\\])+/, "");
  let file = join(root, path);
  try {
    if (!statSync(file).isFile()) throw new Error("dir");
  } catch {
    file = join(root, "index.html");
  }
  res.writeHead(200, { "Content-Type": types[extname(file)] ?? "application/octet-stream" });
  createReadStream(file).pipe(res);
}).listen(port, () => console.log(`static UI on http://localhost:${port}`));
