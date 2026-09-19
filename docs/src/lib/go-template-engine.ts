export interface EventContext {
  event_id: string;
  event_name: string;
  timestamp: string;
  attempt: number;
  payload: any;
}

export interface RenderResult {
  result: string;
  jsonObj?: any;
  error?: string;
}

/**
 * Replaces {{param "key"}} or {{ param "key" }} tokens in strings
 */
export function substituteParams(template: string, params: Record<string, string>): string {
  if (!template) return '';
  return template.replace(/\{\{\s*param\s+"([^"]+)"(?:\s*\|\s*([a-zA-Z0-9_]+))?\s*\}\}/g, (_, key, filter) => {
    let val = params[key] ?? '';
    if (filter === 'urlencode') {
      val = encodeURIComponent(val);
    }
    return val;
  });
}

/**
 * Format string helper (Go printf %v, %s, %d, %j, %q)
 */
function goPrintf(formatStr: string, ...args: any[]): string {
  let idx = 0;
  return formatStr.replace(/%(?:[vsdjq]|(\.\d+f)|f)/g, (match) => {
    if (idx >= args.length) return match;
    const arg = args[idx++];
    if (match === '%j') return JSON.stringify(arg);
    if (match === '%q') return JSON.stringify(String(arg));
    if (typeof arg === 'object' && arg !== null) {
      return JSON.stringify(arg);
    }
    return String(arg);
  });
}

/**
 * Evaluates a single Go expression with functions and pipelines (| json, | upper, etc.)
 */
function evaluateExpression(exprStr: string, scope: Record<string, any>): any {
  exprStr = exprStr.trim();
  if (!exprStr) return '';

  // Handle pipeline splits: e.g. .payload | json or printf "..." .payload.x | json
  // Careful: pipelines inside quotes shouldn't be split.
  const pipelineParts = splitPipeline(exprStr);
  if (pipelineParts.length > 1) {
    let currentVal = evaluateExpression(pipelineParts[0], scope);
    for (let i = 1; i < pipelineParts.length; i++) {
      const stage = pipelineParts[i].trim();
      currentVal = applyFilterStage(stage, currentVal, scope);
    }
    return currentVal;
  }

  // Expression might be a literal, variable, or function call
  // Function call: e.g. printf "format" arg1 arg2
  // or eq .event_name "sparrow.webhook.health_changed"
  const tokens = parseTokens(exprStr);
  if (tokens.length === 0) return '';

  const firstToken = tokens[0];

  // Functions
  if (firstToken === 'printf') {
    const args = tokens.slice(1).map(t => evaluateToken(t, scope));
    const format = String(args[0] ?? '');
    return goPrintf(format, ...args.slice(1));
  }

  if (firstToken === 'eq') {
    const args = tokens.slice(1).map(t => evaluateToken(t, scope));
    return args[0] === args[1];
  }

  if (firstToken === 'ne') {
    const args = tokens.slice(1).map(t => evaluateToken(t, scope));
    return args[0] !== args[1];
  }

  if (firstToken === 'json') {
    const arg = evaluateToken(tokens[1], scope);
    return JSON.stringify(arg);
  }

  if (firstToken === 'upper') {
    const arg = evaluateToken(tokens[1], scope);
    return String(arg ?? '').toUpperCase();
  }

  if (firstToken === 'lower') {
    const arg = evaluateToken(tokens[1], scope);
    return String(arg ?? '').toLowerCase();
  }

  if (firstToken === 'urlencode') {
    const arg = evaluateToken(tokens[1], scope);
    return encodeURIComponent(String(arg ?? ''));
  }

  if (firstToken === 'dict') {
    const args = tokens.slice(1).map(t => evaluateToken(t, scope));
    const obj: Record<string, any> = {};
    for (let i = 0; i < args.length; i += 2) {
      if (i + 1 < args.length) {
        obj[String(args[i])] = args[i + 1];
      }
    }
    return obj;
  }

  if (firstToken === 'list') {
    return tokens.slice(1).map(t => evaluateToken(t, scope));
  }

  if (firstToken === 'dig') {
    const args = tokens.slice(1).map(t => evaluateToken(t, scope));
    if (args.length < 2) return '';
    const mapObj = args[args.length - 1];
    const defaultVal = args[args.length - 2];
    const keys = args.slice(0, args.length - 2);
    let curr = mapObj;
    for (const key of keys) {
      if (curr && typeof curr === 'object' && key in curr) {
        curr = curr[key];
      } else {
        return defaultVal;
      }
    }
    return curr ?? defaultVal;
  }

  if (tokens.length === 1) {
    return evaluateToken(tokens[0], scope);
  }

  // Fallback for expression evaluation
  return evaluateToken(tokens[0], scope);
}

function applyFilterStage(stageStr: string, inputVal: any, scope: Record<string, any>): any {
  const tokens = parseTokens(stageStr);
  if (tokens.length === 0) return inputVal;
  const funcName = tokens[0];

  if (funcName === 'json') {
    return JSON.stringify(inputVal);
  }
  if (funcName === 'upper') {
    return String(inputVal ?? '').toUpperCase();
  }
  if (funcName === 'lower') {
    return String(inputVal ?? '').toLowerCase();
  }
  if (funcName === 'urlencode') {
    return encodeURIComponent(String(inputVal ?? ''));
  }
  if (funcName === 'len') {
    if (Array.isArray(inputVal)) return inputVal.length;
    if (typeof inputVal === 'object' && inputVal) return Object.keys(inputVal).length;
    return String(inputVal ?? '').length;
  }
  if (funcName === 'ellipsis') {
    const maxLen = Number(evaluateToken(tokens[1], scope) ?? 30);
    const str = String(inputVal ?? '');
    return str.length > maxLen ? str.slice(0, Math.max(0, maxLen - 3)) + '...' : str;
  }
  if (funcName === 'printf') {
    const args = tokens.slice(1).map(t => evaluateToken(t, scope));
    const formatStr = String(args[0] ?? '%v');
    return goPrintf(formatStr, inputVal, ...args.slice(1));
  }
  return inputVal;
}

function evaluateToken(token: string, scope: Record<string, any>): any {
  if (!token) return '';
  token = token.trim();

  // String literal
  if ((token.startsWith('"') && token.endsWith('"')) || (token.startsWith("'") && token.endsWith("'"))) {
    return token.slice(1, -1).replace(/\\"/g, '"').replace(/\\n/g, '\n');
  }

  // Boolean literal
  if (token === 'true') return true;
  if (token === 'false') return false;
  if (token === 'nil' || token === 'null') return null;

  // Number literal
  if (!isNaN(Number(token))) {
    return Number(token);
  }

  // Sub-expression enclosed in parentheses e.g. (.payload | json)
  if (token.startsWith('(') && token.endsWith(')')) {
    return evaluateExpression(token.slice(1, -1), scope);
  }

  // Scope property lookup (e.g. .payload.alert_recipients, $r.email, .event_id, .)
  if (token === '.') {
    return scope['.'] ?? scope;
  }

  if (token.startsWith('.')) {
    const path = token.slice(1).split('.');
    let curr = scope['.'] !== undefined ? scope['.'] : scope;
    for (const p of path) {
      if (!p) continue;
      if (curr && typeof curr === 'object' && p in curr) {
        curr = curr[p];
      } else {
        return undefined;
      }
    }
    return curr;
  }

  if (token.startsWith('$')) {
    const parts = token.split('.');
    const varName = parts[0];
    let curr = scope[varName];
    for (let i = 1; i < parts.length; i++) {
      const p = parts[i];
      if (curr && typeof curr === 'object' && p in curr) {
        curr = curr[p];
      } else {
        return undefined;
      }
    }
    return curr;
  }

  return scope[token];
}

function parseTokens(str: string): string[] {
  const tokens: string[] = [];
  let current = '';
  let inQuotes = false;
  let quoteChar = '';
  let parenDepth = 0;

  for (let i = 0; i < str.length; i++) {
    const char = str[i];
    if ((char === '"' || char === "'") && (i === 0 || str[i - 1] !== '\\')) {
      if (inQuotes && char === quoteChar) {
        inQuotes = false;
        quoteChar = '';
      } else if (!inQuotes) {
        inQuotes = true;
        quoteChar = char;
      }
      current += char;
    } else if (!inQuotes && char === '(') {
      parenDepth++;
      current += char;
    } else if (!inQuotes && char === ')') {
      parenDepth--;
      current += char;
    } else if (!inQuotes && parenDepth === 0 && /\s/.test(char)) {
      if (current.trim()) {
        tokens.push(current.trim());
        current = '';
      }
    } else {
      current += char;
    }
  }
  if (current.trim()) {
    tokens.push(current.trim());
  }
  return tokens;
}

function splitPipeline(expr: string): string[] {
  const parts: string[] = [];
  let current = '';
  let inQuotes = false;
  let quoteChar = '';
  let parenDepth = 0;

  for (let i = 0; i < expr.length; i++) {
    const char = expr[i];
    if ((char === '"' || char === "'") && (i === 0 || expr[i - 1] !== '\\')) {
      if (inQuotes && char === quoteChar) {
        inQuotes = false;
      } else if (!inQuotes) {
        inQuotes = true;
        quoteChar = char;
      }
      current += char;
    } else if (!inQuotes && char === '(') {
      parenDepth++;
      current += char;
    } else if (!inQuotes && char === ')') {
      parenDepth--;
      current += char;
    } else if (!inQuotes && parenDepth === 0 && char === '|') {
      parts.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }
  if (current.trim()) {
    parts.push(current.trim());
  }
  return parts;
}

/**
 * Full Go Template rendering engine for Sparrow templates
 */
export function renderGoTemplate(template: string, ctx: EventContext, params: Record<string, string> = {}): RenderResult {
  try {
    // 1. Substitute {{param "name"}}
    let processed = substituteParams(template, params);

    // Initial scope setup
    const rootScope: Record<string, any> = {
      '.': ctx,
      event_id: ctx.event_id,
      event_name: ctx.event_name,
      timestamp: ctx.timestamp,
      attempt: ctx.attempt,
      payload: ctx.payload,
    };

    // Parse and evaluate Go template tags
    const rendered = processTemplateBlock(processed, rootScope);

    let jsonObj: any = null;
    try {
      jsonObj = JSON.parse(rendered.trim());
    } catch {
      // Might not be JSON (e.g. ntfy text or twilio urlencoded string)
    }

    return {
      result: rendered,
      jsonObj,
    };
  } catch (err: any) {
    return {
      result: '',
      error: err.message || String(err),
    };
  }
}

function processTemplateBlock(tmpl: string, scope: Record<string, any>): string {
  let output = '';
  let cursor = 0;

  while (cursor < tmpl.length) {
    const openIdx = tmpl.indexOf('{{', cursor);
    if (openIdx === -1) {
      output += tmpl.slice(cursor);
      break;
    }

    output += tmpl.slice(cursor, openIdx);
    const closeIdx = tmpl.indexOf('}}', openIdx);
    if (closeIdx === -1) {
      output += tmpl.slice(openIdx);
      break;
    }

    const tagContent = tmpl.slice(openIdx + 2, closeIdx).trim();

    // Range block
    if (tagContent.startsWith('range ')) {
      const { blockContent, nextCursor } = extractMatchingEnd(tmpl, closeIdx + 2, 'range');
      const rangeExpr = tagContent.slice(6).trim();

      // Range syntax: range $i, $r := .payload.alert_recipients OR range .payload.items
      let varIdx = '$i';
      let varItem = '$r';
      let targetExpr = rangeExpr;

      if (rangeExpr.includes(':=')) {
        const parts = rangeExpr.split(':=');
        const vars = parts[0].split(',').map(v => v.trim());
        if (vars.length === 2) {
          varIdx = vars[0];
          varItem = vars[1];
        } else if (vars.length === 1) {
          varItem = vars[0];
        }
        targetExpr = parts[1].trim();
      }

      const listVal = evaluateExpression(targetExpr, scope);
      if (Array.isArray(listVal)) {
        listVal.forEach((item, idx) => {
          const itemScope = {
            ...scope,
            '.': item,
            [varIdx]: idx,
            [varItem]: item,
          };
          output += processTemplateBlock(blockContent, itemScope);
        });
      }
      cursor = nextCursor;
      continue;
    }

    // If / Else block
    if (tagContent.startsWith('if ')) {
      const { ifBlock, elseBlock, nextCursor } = extractIfElseBlocks(tmpl, closeIdx + 2);
      const condExpr = tagContent.slice(3).trim();
      const condResult = Boolean(evaluateExpression(condExpr, scope));

      if (condResult) {
        output += processTemplateBlock(ifBlock, scope);
      } else if (elseBlock !== null) {
        output += processTemplateBlock(elseBlock, scope);
      }
      cursor = nextCursor;
      continue;
    }

    // Standalone expression tag e.g. {{ .event_name | json }} or {{ $i }}
    const val = evaluateExpression(tagContent, scope);
    if (val !== undefined && val !== null) {
      output += String(val);
    }

    cursor = closeIdx + 2;
  }

  return output;
}

function extractMatchingEnd(tmpl: string, startIdx: number, keyword: string): { blockContent: string; nextCursor: number } {
  let depth = 1;
  let cursor = startIdx;

  while (cursor < tmpl.length) {
    const openIdx = tmpl.indexOf('{{', cursor);
    if (openIdx === -1) break;
    const closeIdx = tmpl.indexOf('}}', openIdx);
    if (closeIdx === -1) break;

    const tag = tmpl.slice(openIdx + 2, closeIdx).trim();
    if (tag.startsWith(keyword + ' ') || tag.startsWith('if ')) {
      depth++;
    } else if (tag === 'end') {
      depth--;
      if (depth === 0) {
        return {
          blockContent: tmpl.slice(startIdx, openIdx),
          nextCursor: closeIdx + 2,
        };
      }
    }
    cursor = closeIdx + 2;
  }

  return { blockContent: tmpl.slice(startIdx), nextCursor: tmpl.length };
}

function extractIfElseBlocks(tmpl: string, startIdx: number): { ifBlock: string; elseBlock: string | null; nextCursor: number } {
  let depth = 1;
  let cursor = startIdx;
  let elseIdx: number | null = null;

  while (cursor < tmpl.length) {
    const openIdx = tmpl.indexOf('{{', cursor);
    if (openIdx === -1) break;
    const closeIdx = tmpl.indexOf('}}', openIdx);
    if (closeIdx === -1) break;

    const tag = tmpl.slice(openIdx + 2, closeIdx).trim();
    if (tag.startsWith('if ') || tag.startsWith('range ')) {
      depth++;
    } else if (tag === 'else' && depth === 1) {
      elseIdx = openIdx;
    } else if (tag === 'end') {
      depth--;
      if (depth === 0) {
        if (elseIdx !== null) {
          return {
            ifBlock: tmpl.slice(startIdx, elseIdx),
            elseBlock: tmpl.slice(elseIdx + tmpl.slice(elseIdx).indexOf('}}') + 2, openIdx),
            nextCursor: closeIdx + 2,
          };
        } else {
          return {
            ifBlock: tmpl.slice(startIdx, openIdx),
            elseBlock: null,
            nextCursor: closeIdx + 2,
          };
        }
      }
    }
    cursor = closeIdx + 2;
  }

  return { ifBlock: tmpl.slice(startIdx), elseBlock: null, nextCursor: tmpl.length };
}
