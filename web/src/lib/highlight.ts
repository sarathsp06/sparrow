// A small syntax highlighter for previewing generated receiver code. It
// returns tokens (rendered as text, never HTML) for comments, strings,
// numbers and keywords; anything it does not recognise stays plain. Enough
// for reading a file, not a parser.

export type TokenKind = "plain" | "comment" | "string" | "number" | "keyword";

export interface Token {
  text: string;
  kind: TokenKind;
}

const KEYWORDS: Record<string, string[]> = {
  go: "break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var nil true false".split(" "),
  python: "False None True and as assert async await break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield".split(" "),
  ts: "as async await break case catch class const continue default else export extends false finally for from function if import in instanceof interface let new null of return switch this throw true try type typeof undefined var void while".split(" "),
  json: ["true", "false", "null"],
  sh: "case do done elif else esac exit export fi for function if in then while".split(" "),
};

/** The highlighter language for a file path, or "" for plain text. */
export function languageOf(path: string): string {
  const base = path.split("/").pop() ?? path;
  if (base.endsWith(".go") || base === "go.mod") return "go";
  if (base.endsWith(".py")) return "python";
  if (base.endsWith(".ts") || base.endsWith(".js") || base.endsWith(".mjs")) return "ts";
  if (base.endsWith(".json")) return "json";
  if (base.endsWith(".sh") || base.startsWith(".env") || base === ".gitignore" || base.endsWith(".txt")) return "sh";
  return "";
}

/** Splits source into tokens; joined, they are exactly source. */
export function highlight(source: string, language: string): Token[] {
  if (!language) return [{ text: source, kind: "plain" }];
  const lineComment = language === "python" || language === "sh" ? "#" : language === "json" ? "" : "//";
  const keywords = new Set(KEYWORDS[language] ?? []);
  const tokens: Token[] = [];
  let plain = "";
  const push = (text: string, kind: TokenKind) => {
    if (plain) {
      tokens.push({ text: plain, kind: "plain" });
      plain = "";
    }
    tokens.push({ text, kind });
  };

  let i = 0;
  while (i < source.length) {
    const rest = source.slice(i);
    let m: RegExpMatchArray | null;
    if (lineComment && rest.startsWith(lineComment)) {
      const end = source.indexOf("\n", i);
      const stop = end === -1 ? source.length : end;
      push(source.slice(i, stop), "comment");
      i = stop;
    } else if ((language === "go" || language === "ts") && rest.startsWith("/*")) {
      const end = source.indexOf("*/", i + 2);
      const stop = end === -1 ? source.length : end + 2;
      push(source.slice(i, stop), "comment");
      i = stop;
    } else if (language === "python" && (m = rest.match(/^("""[\s\S]*?"""|'''[\s\S]*?''')/))) {
      push(m[0], "string");
      i += m[0].length;
    } else if ((m = rest.match(/^("(?:[^"\\\n]|\\.)*"|'(?:[^'\\\n]|\\.)*')/)) || ((language === "go" || language === "ts") && (m = rest.match(/^`[^`]*`/)))) {
      push(m[0], "string");
      i += m[0].length;
    } else if ((m = rest.match(/^\d[\d_]*(?:\.\d+)?(?:[eE][+-]?\d+)?/)) && !/[\w$]/.test(source[i - 1] ?? "")) {
      push(m[0], "number");
      i += m[0].length;
    } else if ((m = rest.match(/^[A-Za-z_$][\w$]*/))) {
      if (keywords.has(m[0])) push(m[0], "keyword");
      else plain += m[0];
      i += m[0].length;
    } else {
      plain += source[i];
      i++;
    }
  }
  if (plain) tokens.push({ text: plain, kind: "plain" });
  return tokens;
}
