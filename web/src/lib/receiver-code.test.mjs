import assert from "node:assert/strict";
import test from "node:test";
import { crc32 as nodeCrc32 } from "node:zlib";

import { highlight, languageOf } from "./highlight.ts";
import { crc32, makeZip } from "./zip.ts";

// readZip parses a stored-only archive through its central directory.
function readZip(buf) {
  const view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  const eocd = buf.length - 22;
  assert.equal(view.getUint32(eocd, true), 0x06054b50, "end of central directory");
  const count = view.getUint16(eocd + 10, true);
  let at = view.getUint32(eocd + 16, true);
  const dec = new TextDecoder();
  const out = [];
  for (let n = 0; n < count; n++) {
    assert.equal(view.getUint32(at, true), 0x02014b50, "central header");
    const crc = view.getUint32(at + 16, true);
    const size = view.getUint32(at + 24, true);
    const nameLen = view.getUint16(at + 28, true);
    const mode = view.getUint32(at + 38, true) >>> 16;
    const local = view.getUint32(at + 42, true);
    const name = dec.decode(buf.subarray(at + 46, at + 46 + nameLen));
    assert.equal(view.getUint32(local, true), 0x04034b50, "local header");
    const localNameLen = view.getUint16(local + 26, true);
    const data = buf.subarray(local + 30 + localNameLen, local + 30 + localNameLen + size);
    out.push({ name, mode: mode & 0o777, crc, content: dec.decode(data), dataCrc: nodeCrc32(data) >>> 0 });
    at += 46 + nameLen;
  }
  return out;
}

test("crc32 matches zlib", () => {
  for (const s of ["", "a", "hello world", "ünïcødé ✓"]) {
    const bytes = new TextEncoder().encode(s);
    assert.equal(crc32(bytes), nodeCrc32(bytes) >>> 0, s);
  }
});

test("makeZip stores every file with its mode under the prefix", () => {
  const zip = makeZip(
    [
      { path: "main.go", content: "package main\n", mode: 0o644 },
      { path: "scripts/run.sh", content: "#!/bin/sh\necho ✓\n", mode: 0o755 },
      { path: "empty.txt", content: "" },
    ],
    "receiver/",
  );
  const entries = readZip(zip);
  assert.deepEqual(
    entries.map((e) => [e.name, e.mode, e.content]),
    [
      ["receiver/main.go", 0o644, "package main\n"],
      ["receiver/scripts/run.sh", 0o755, "#!/bin/sh\necho ✓\n"],
      ["receiver/empty.txt", 0o644, ""],
    ],
  );
  for (const e of entries) assert.equal(e.crc, e.dataCrc, e.name);
});

test("makeZip is deterministic", () => {
  const files = [{ path: "a", content: "x" }];
  assert.deepEqual(makeZip(files), makeZip(files));
});

test("languageOf picks the highlighter from the path", () => {
  assert.equal(languageOf("receiver.go"), "go");
  assert.equal(languageOf("go.mod"), "go");
  assert.equal(languageOf("app/receiver.py"), "python");
  assert.equal(languageOf("src/receiver.ts"), "ts");
  assert.equal(languageOf("testdata/spec.json"), "json");
  assert.equal(languageOf(".env.example"), "sh");
  assert.equal(languageOf("README.md"), "");
});

test("highlight keeps the text and classifies tokens", () => {
  const src = 'func main() { // go\n\tx := "a \\" b" + `raw` + 42 /* c */\n}';
  const tokens = highlight(src, "go");
  assert.equal(tokens.map((t) => t.text).join(""), src);
  const kinds = (kind) => tokens.filter((t) => t.kind === kind).map((t) => t.text);
  assert.deepEqual(kinds("keyword"), ["func"]);
  assert.deepEqual(kinds("comment"), ["// go", "/* c */"]);
  assert.deepEqual(kinds("string"), ['"a \\" b"', "`raw`"]);
  assert.deepEqual(kinds("number"), ["42"]);
});

test("highlight handles Python docstrings and comments", () => {
  const src = 'def f(x1):\n    """Doc "quoted"."""\n    return None  # done\n';
  const tokens = highlight(src, "python");
  assert.equal(tokens.map((t) => t.text).join(""), src);
  assert.deepEqual(tokens.filter((t) => t.kind === "string").map((t) => t.text), ['"""Doc "quoted"."""']);
  assert.deepEqual(tokens.filter((t) => t.kind === "keyword").map((t) => t.text), ["def", "return", "None"]);
  assert.ok(!tokens.some((t) => t.kind === "number"), "x1 is not a number");
});

test("highlight without a language is one plain token", () => {
  assert.deepEqual(highlight("# Title", ""), [{ text: "# Title", kind: "plain" }]);
});
