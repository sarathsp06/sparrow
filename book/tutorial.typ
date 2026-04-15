// ==========================================================================
// Svelte 5 Tutorial — The Disappearing Framework
// A backend engineer's guide to high-performance reactivity.
// All code examples from the Sparrow webhook platform.
// ==========================================================================

#import "template.typ": *

#let version = "v1.3.1"

#show: tutorial-doc.with(
  title: "Svelte 5 Tutorial",
  author: "Sparrow Project",
  version: version,
)

// ═══════════════════════════════════════════════════════════════════════════
// COVER PAGE — O'Reilly style
// ═══════════════════════════════════════════════════════════════════════════

#cover-page(
  title: "Svelte 5 Tutorial",
  subtitle: "The Disappearing Framework",
  tagline: [A backend engineer's guide to high-performance reactivity
    and compiler-driven UI. All examples from a real production codebase.],
  version: version,
  meta: [16 Lessons · Built by & for Backend Engineers · #version],
)


// ═══════════════════════════════════════════════════════════════════════════
// INTRODUCTION + TABLE OF CONTENTS
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()

= Introduction

This tutorial is an attempt to learn and teach Svelte 5 through the lens of a real-world project. All examples are drawn from *Sparrow*, an open-source webhook delivery platform built using an AI-assisted development workflow.

By exploring how Svelte 5 is used to build Sparrow's management UI, you'll see how runes like \$state and \$derived solve actual engineering problems---from handling complex delivery filters to visualizing real-time batch progress.

The Sparrow source code is available on GitHub: #link-text[#underline[https://github.com/sarathsp06/sparrow]]

#v(8pt)
#hr()
#v(8pt)

#outline(
  title: [Table of Contents],
  indent: auto,
  depth: 1,
)

#v(12pt)

*Audience:* Backend engineers who know basic JavaScript/TypeScript and have the Sparrow repo checked out locally. No prior React, Vue, or Svelte experience required.

*Approach:* Each lesson teaches a Svelte feature with real Sparrow code, then shows what the compiler does with it under the hood. You learn the API and the internals together---not in separate sections. Conceptual compiled-output illustrations are clearly labeled as such.

*Source Code:* https://github.com/sarathsp06/sparrow


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 1: $state()
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("1", "Reactive State with $state()")

== The Problem

In a web app, data changes constantly: a user clicks a button, an API returns results, a timer fires. The UI needs to update to reflect these changes. In vanilla JavaScript, you'd manually find DOM elements and update their text content. Svelte automates this: you declare a variable as reactive, change it, and the UI updates itself.

== The Svelte 5 Solution: \$state()

The `$state()` rune tells the Svelte compiler to track a variable. When that variable's value changes (via plain assignment), every part of the UI that reads it re-renders automatically.

#source-code("web/src/routes/health/+page.svelte, lines 15-21")[
```
let loading = $state(true);
let error = $state('');
let healthSummary = $state<HealthSummary | undefined>();
let unhealthyWebhooks = $state<RegisteredWebhook[]>([]);
let degradedWebhooks = $state<RegisteredWebhook[]>([]);
let webhookMetrics = $state(new Map<string, HealthMetrics>());
let namespaceStats = $state<NamespaceStats[]>([]);
```
]

== Line by Line

- `let loading = $state(true)` --- `$state(true)` creates a reactive variable initialized to `true`. While data is loading, the UI shows a skeleton placeholder. When loading completes, `loading = false` triggers the UI to swap to real content.

- `let error = $state('')` --- An empty string means no error. If a fetch fails, `error = 'Something went wrong'` makes an error banner appear.

- `$state<HealthSummary | undefined>()` --- The angle brackets are a generic type parameter (TypeScript). `HealthSummary | undefined` is a union type: the variable is either a HealthSummary object or undefined. Empty parentheses `()` mean no initial value = undefined.

- `$state<RegisteredWebhook[]>([])` --- `RegisteredWebhook[]` means "an array of RegisteredWebhook objects". The `[]` inside parens is the initial value: empty array.

- `$state(new Map<string, HealthMetrics>())` --- A Map is like a dictionary. `Map<string, HealthMetrics>` = keys are strings, values are HealthMetrics objects. Initialized empty.

== How Reactivity Triggers

Whether a mutation triggers a UI update depends on the *type* of value wrapped by `$state()`. This is the single most important table in this tutorial:

#ref-table(
  columns: (1in, 1.8in, 0.9in, 2.8in),

  table.header(th[Type], th[Mutation], th[Detected?], th[Why]),

  td[Primitive\ (number, string, bool)],
  tc[count = 5],
  td[Yes],
  td[Compiler emits \$.set() which notifies dependents],

  td[Plain object],
  tc[obj.name = 'new'],
  td[Yes],
  td[Proxy set trap fires, updates per-property signal],

  td[Array],
  tc[arr.push(item)\ arr\[0\] = x],
  td[Yes\ Yes],
  td[Proxy set trap fires on index and length changes],

  td[Map],
  tc[map.set('k', v)],
  td[*NO*],
  td[.set() uses internal slots that bypass Proxy traps],

  td[Set],
  tc[set.add(v)],
  td[*NO*],
  td[Same: .add() bypasses Proxy traps],

  td[Date],
  tc[date.setHours(12)],
  td[*NO*],
  td[Same: mutation methods operate on internal slots],
)

For Maps, Sets, and Dates, you must *reassign the entire variable* to trigger an update:

#code-block[
```
// Primitives -- plain assignment:
loading = false;              // $.set() fires

// Objects -- property assignment:
webhook.url = newUrl;         // Proxy set trap fires

// Arrays -- mutation works:
items.push(newItem);          // Proxy set trap fires
items = [...items, newItem];  // Also works (reassignment)

// Maps -- MUST reassign:
webhookMetrics.set(id, m);    // SILENT. No UI update.
webhookMetrics = new Map(     // Correct: reassign
  [...webhookMetrics, [id, m]]
);

// Sets -- MUST reassign:
tags.add("new");              // SILENT. No UI update.
tags = new Set([...tags, "new"]);  // Correct
```
]

The key insight: `$state()` is a compiler directive, not a regular function. At runtime, the variable holds the plain value (a boolean, a string, an array), not a wrapper object. Svelte's compiler rewrites your code during build to inject change tracking behind the scenes.

== Under the Hood: Svelte Is a Compiler

This is the most important thing to understand about Svelte: it is not a runtime library. React, Vue, and Angular ship a runtime that interprets your components in the browser. Svelte is a *compiler* that runs at build time and emits optimized JavaScript---like the difference between CPython (interpreter) and Go (ahead-of-time compiler). The heavy lifting happens once, so the browser does less.

The Svelte compiler processes each `.svelte` file through three phases, similar to `go build` (parse → type-check → codegen):

#code-block[
```
// sveltejs/svelte — packages/svelte/src/compiler/index.js
export function compile(source, options) {
  // Phase 1: PARSE — source text -> AST      (like go/parser)
  const parsed = _parse(source);

  // Phase 2: ANALYZE — resolve bindings       (like go/types)
  const analysis = analyze_component(
    parsed, source, combined_options
  );

  // Phase 3: TRANSFORM — AST -> optimized JS  (codegen)
  return transform_component(
    analysis, source, combined_options
  );
}
```
]
#source-ref("sveltejs/svelte -- packages/svelte/src/compiler/index.js")

== What \$state Actually Compiles To

When the compiler sees `$state()`, it replaces it with a *signal*---a small reactive container object. The rune syntax is just syntactic sugar that disappears at build time:

#code-block[
```
// What you write:
let count = $state(0);

// What the compiler emits (conceptual illustration):
import * as $ from 'svelte/internal/client';
let count = $.state(0);   // creates a signal object

// Template: <p>{count}</p>
// Compiles to:
var p = $.template('<p> </p>');
var text = p.firstChild;
$.render_effect(() => {
  $.set_text(text, $.get(count));  // direct DOM update
});

// No virtual DOM tree. No diffing algorithm.
// One render_effect per dynamic expression.
// Cost = O(changed signals), not O(template size).
```
]

Contrast this with React, where the entire component function re-executes on every state change, returns a new virtual DOM tree, and React diffs the old and new trees to find what changed. Svelte skips all of that---the compiler already knows which DOM node to update.

== The Signal Object

Under the hood, `$.state(0)` creates a signal---a plain JavaScript object with a value, a version counter, and a list of who depends on it. If you've worked with the observer pattern (event emitters, pub/sub), this will feel familiar:

#source-code("sveltejs/svelte -- packages/svelte/src/internal/client/reactivity/sources.js")[
```
// sveltejs/svelte — internal/client/reactivity/sources.js
export function source(v) {
  return {
    f: 0,              // flags (DIRTY, CLEAN, etc.)
    v,                 // current value
    reactions: null,   // effects/deriveds that read this
    equals,            // equality check function
    rv: 0,             // read version
    wv: 0              // write version
  };
}
```
]

When you assign to a `$state` variable, the compiler emits `$.set(signal, newValue)`. This checks equality, updates the value, increments the write version, and walks the signal's reaction list marking each dependent as *DIRTY*---like a database trigger notifying all subscribed queries that a column changed:

#code-block[
```
// sources.js -- set() (simplified)
export function set(source, value) {
  if (source.equals(value, source.v)) return;

  source.v = value;              // update value
  source.wv++;                   // increment write version
  mark_reactions(source, DIRTY); // notify all dependents
  schedule_effects();            // batch DOM updates
}
```
]

== Deep Reactivity: The Proxy System

When `$state` wraps an object or array, Svelte creates an ES6 Proxy with a *per-property signal Map*. Each property access creates (or reads) a dedicated signal. Changing `obj.name` only re-renders expressions that read `obj.name`, not those that read `obj.age`:

#code-block[
```
// proxy.js (simplified)
export function proxy(value) {
  const sources = new Map();  // property -> signal

  return new Proxy(value, {
    get(target, prop) {
      let s = sources.get(prop);
      if (!s) {
        s = source(target[prop]);
        sources.set(prop, s);
      }
      return get(s);  // registers dependency
    },
    set(target, prop, val) {
      let s = sources.get(prop);
      set(s, proxy(val));  // recursively proxy nested
      return true;
    }
  });
}
```
]

== Why Maps, Sets, and Dates Are Not Reactive

The proxy system works by intercepting property access (`get` trap) and property assignment (`set` trap). For plain objects and arrays, `obj.name = x` triggers the proxy's `set` trap, which updates the per-property signal. But Map, Set, and Date use a different mechanism.

When you call `map.set('key', value)`, JavaScript does this:

#code-block[
```
// What happens at the JS engine level:
//
// 1. map.set  ->  Proxy GET trap fires (reading 'set')
//                 Returns the native Map.prototype.set fn
//
// 2. map.set('key', value)
//    -> Map.prototype.set runs on the RAW Map
//    -> Mutates the Map's [[MapData]] internal slot
//    -> Proxy SET trap NEVER fires
//       (no property was assigned on the proxy object)
//
// The proxy can see you LOOKED UP .set (step 1),
// but cannot see the MUTATION (step 2) because
// it happens inside the engine's C++ implementation,
// not through a property assignment.
//
// Same for Set.add(), Set.delete(), Date.setHours(),
// and all other built-in methods that operate on
// internal slots rather than ordinary properties.
```
]

Think of it like a Go struct with unexported fields: you can call methods on it, but an external wrapper cannot intercept what those methods do internally. The Proxy is the external wrapper; the Map's internal storage is the unexported field.

*Vue 3 handles this differently*---its `reactive()` wraps Map/Set methods (like `.set()`) with custom functions that manually trigger reactivity. Svelte chose not to do this, keeping the proxy implementation simpler and more predictable. The trade-off is that you must remember to reassign Maps and Sets.

== \$state.raw: Opting Out of Deep Tracking

The proxy system has overhead: every property access goes through a trap, and each property gets its own signal. For large arrays or data you never mutate in place (like API responses you display read-only), use `$state.raw` to skip deep tracking. The variable itself is still reactive---swapping the whole value triggers updates---but individual properties are not tracked.

#source-code("web/src/routes/deliveries/+page.svelte, lines 5, 15, 72-87")[
```
// Current code (deliveries/+page.svelte, line 15):
let deliveries: WebhookDelivery[] = $state([]);

// The fetch replaces the entire array (lines 77-79):
const res = await deliveryClient
  .listDeliveries(req);
deliveries = res.deliveries || [];

// $state([]) deep-proxies every WebhookDelivery object.
// But we never mutate deliveries[0].status in place --
// we always replace the whole array. Wasted overhead.

// Optimized alternative:
let deliveries = $state.raw<WebhookDelivery[]>([]);
// No per-property signals. Reassigning the array still
// triggers updates. Cheaper for read-only display data.
```
]

== Performance Tiers

Think of reactive state in three tiers, cheapest to most expensive:

- *Tier 1 -- Primitive \$state*: `$state(0)`, `$state("")`. A single signal. Cheapest possible.
- *Tier 2 -- \$state.raw(obj)*: One signal for the reference. No per-property tracking. Use for API responses and large read-only data.
- *Tier 3 -- \$state(obj)*: Proxy + per-property signals. Full deep reactivity. Use when you mutate individual properties.

#gotcha(1, "Mutating arrays")[
  If you push to an array with `.push()`, Svelte 5 does detect the mutation (unlike Svelte 4 which required reassignment). However, reassignment is still the clearest pattern: `items = [...items, newItem]`.
]

#gotcha(2, "\$state() is not a regular function")[
  You cannot do `const x = condition ? $state(1) : $state(2)`. The compiler must see `$state()` as a direct initializer in a `let` declaration at the top level of a component's script block.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 2: $derived()
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("2", "Computed Values with $derived()")

== The Problem

You have some state, and you need a value that's calculated from that state. For example: you have an array of webhooks and need a filtered list showing only the active ones. You want to declare the computation once and have it automatically stay in sync.

== Simple Form: \$derived(expression)

#source-code("web/src/lib/components/CopyableId.svelte, lines 11-13")[
```
const display = $derived(
  truncate > 0 && id.length > truncate
    ? id.substring(0, truncate) + '...'
    : id
);
```
]

This creates a `display` value that automatically recomputes whenever `truncate` or `id` changes. The ternary operator (`? :`) checks: if truncation is enabled and the ID is longer than the limit, show a shortened version; otherwise show the full ID.

== Complex Form: \$derived.by(() => \{ ... \})

#source-code("web/src/routes/webhooks/+page.svelte, lines 45-57")[
```
let filteredWebhooks = $derived.by(() => {
  let result = webhooks;
  if (healthFilter) {
    result = result.filter(
      (wh) => wh.health === healthFilter
    );
  }
  if (urlSearch.trim()) {
    const q = urlSearch.toLowerCase();
    result = result.filter((wh) =>
      wh.url.toLowerCase().includes(q) ||
      wh.description?.toLowerCase().includes(q)
    );
  }
  return result;
});
```
]

== Line by Line

- `$derived.by(() => { ... })` --- When the computation needs multiple statements, use `$derived.by()` with a function body.

- `.filter((wh) => wh.health === healthFilter)` --- `.filter()` creates a new array keeping only elements where the callback returns true.

- `wh.description?.toLowerCase()` --- The `?.` is optional chaining. `description` might be undefined. Without `?.`, calling `.toLowerCase()` on undefined would crash.

== \$derived vs \$state: When to Use Which

Use `$state` for data you set directly (API responses, user input). Use `$derived` for data computed from other state. If you find yourself writing code that sets a variable every time another changes, use `$derived`.

== Under the Hood: Derived Signals

The compiler transforms `$derived(expr)` into `$.derived(() => expr)`. A derived signal is *lazy*---it doesn't recompute immediately when a dependency changes. Instead, the runtime marks it as *MAYBE_DIRTY* and only recalculates when something actually reads its value. Think of it as a lazy database view vs. a materialized one.

#code-block[
```
// What happens when a source signal changes:
//
// 1. $.set(source, newValue)
//    -> mark_reactions(source, DIRTY)
//
// 2. For each dependent:
//    - If it's a render_effect -> mark DIRTY (will re-run)
//    - If it's a derived -> mark MAYBE_DIRTY
//      (might not need recalc if other deps cancel out)
//
// 3. Derived only recalculates when $.get() is called.
//    If the source value didn't actually change its
//    equality check, the derived stays clean.
//
// This is push-DIRTY, pull-VALUE:
//   "I know something changed" (push)
//   "Let me check if I need to recompute" (pull)
```
]

React's `useMemo` requires you to manually list dependencies in an array---get it wrong and you have stale values or infinite loops. Vue's `computed()` auto-tracks at runtime (closer to Svelte). Svelte's compiler resolves the dependency graph at build time, so there is no dependency array to get wrong and no runtime cost for dependency tracking.

#gotcha(1, "Only reactive sources are tracked")[
  `$derived` only tracks variables from `$state`, `$derived`, or `$props`. A plain `let x = 5` is not reactive.
]

#gotcha(2, "Don't mutate inside \$derived")[
  A `$derived` computation should be pure. Never modify other `$state` variables inside. That's what `$effect` is for.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 3: $props()
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("3", "Components & Props with $props()")

== The Problem

As your UI grows, you need to break it into reusable pieces. A health badge, a copyable ID display, a confirmation dialog. Each piece needs to accept data from its parent. In Svelte 5, components receive data through `$props()`.

== Defining Props with TypeScript

#source-code("web/src/lib/components/CopyableId.svelte, lines 2-8")[
```
interface Props {
  id: string;
  href?: string;
  truncate?: number;
}

let { id, href, truncate = 8 }: Props = $props();
```
]

== Line by Line

- `interface Props { ... }` --- A TypeScript interface defines the shape of an object.
- `id: string` --- Required prop. Missing it = TypeScript compile error.
- `href?: string` --- The `?` makes this optional. When omitted, the value is undefined.
- `let { id, href, truncate = 8 }: Props = $props()` --- Destructures props. `truncate = 8` is a default value.

== Callback Props (Functions as Props)

#source-code("web/src/lib/components/ConfirmDialog.svelte, lines 1-11")[
```
interface Props {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'danger' | 'warning' | 'info';
  onconfirm: () => void;
  oncancel: () => void;
}
```
]

- `onconfirm: () => void` --- This prop expects a function. The parent passes a callback. When the user clicks "Confirm", the dialog calls `onconfirm()`.

This is the standard pattern for child-to-parent communication in Svelte 5: the parent passes a function down, the child calls it when something happens.

== Under the Hood: What \$props Compiles To

The compiler transforms each destructured prop into a signal read via `$.prop()`. When a parent changes a prop value, only the DOM expressions that read _that specific prop_ update---not the entire child component. In React, every prop change re-runs the entire component function.

#source-code("web/src/lib/components/Pagination.svelte, lines 2-25")[
```
// Real Sparrow code (Pagination.svelte, lines 2-25):
interface Props {
  currentPage: number;
  totalPages: number;
  totalCount: number;
  pageSize: number;
  onPageChange: (pageNum: number) => void;
  itemLabel?: string;
}

let { currentPage, totalPages, totalCount,
      pageSize, onPageChange,
      itemLabel = 'items' }: Props = $props();

function nextPage() {
  if (currentPage < totalPages) {
    onPageChange(currentPage + 1);
  }
}
function previousPage() {
  if (currentPage > 1) {
    onPageChange(currentPage - 1);
  }
}
```
]

The compiler transforms this into signal reads. Below is a *conceptual illustration* of the compiled output (not literal Sparrow code):

#code-block[
```
// Conceptual compiled output (illustrative):
// 1. Props become signal reads via $.prop()
let currentPage = $.prop($$props, 'currentPage');
let totalPages  = $.prop($$props, 'totalPages');
let onPageChange = $.prop($$props, 'onPageChange');
let itemLabel = $.prop($$props, 'itemLabel', 8, 'items');

// 2. Template text {currentPage} becomes:
$.render_effect(() => {
  $.set_text(text_node, $.get(currentPage));
});

// 3. disabled={currentPage === 1} becomes:
$.render_effect(() => {
  $.set_attribute(btn, 'disabled',
    $.get(currentPage) === 1);
});

// Each render_effect targets exactly one DOM node.
// No tree diffing. No re-rendering sibling elements.
```
]

#gotcha(1, "Props are read-only")[
  You cannot reassign a prop inside the child. Props flow one direction: parent to child.
]

#gotcha(2, "Default values only apply when undefined")[
  If parent passes `truncate={0}`, default 8 does NOT apply. 0 is a valid value.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 4: {#if}
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("4", "Conditional Rendering with {#if}")

== The Problem

Most UI isn't static. You need to show a loading spinner while data loads, an error message when something fails, and actual content when data arrives.

== Three-State Pattern: Loading / Error / Content

#source-code("web/src/routes/health/+page.svelte, lines 97-146")[
```
{#if loading}
  <!-- Skeleton placeholders -->
  <div class="animate-pulse">...</div>
{:else if error}
  <!-- Error message -->
  <div class="text-red-600">{error}</div>
{:else}
  <!-- Actual content -->
  <div>{healthSummary.totalWebhooks}</div>
{/if}
```
]

== How It Works

- `{#if loading}` --- If `loading` is truthy, this branch renders.
- `{:else if error}` --- Check error. An empty string is falsy (no error).
- `{:else}` --- The fallback. Show the real content.
- `{/if}` --- Closes the block.

== DOM Destruction vs CSS Hiding

`{#if}` removes elements from the DOM when the condition is false. It doesn't hide them with CSS. Any internal state (scroll position, input values) is lost.

*Why destroy instead of hide?* Svelte could set `display: none` and keep elements in the DOM, but that has real costs: hidden DOM nodes still consume memory, their event listeners remain active, and any `$effect` inside the branch keeps running. Destroying the branch means zero cost when it's not shown---no DOM nodes, no event listeners, no effects. For a webhook dashboard with hundreds of conditional sections, this adds up. If you need to preserve internal state across visibility toggles, use CSS (`class:hidden`) instead.

== Inline Ternaries in Attributes

#code-block[
```
<span class="{wh.active ? 'bg-green-500' : 'bg-gray-300'}">
  {wh.active ? 'Active' : 'Paused'}
</span>
```
]

#gotcha(1, "Equality checks")[
  JavaScript's `==` does type coercion. Always use `===` for strict equality.
]

#gotcha(2, "Empty arrays are truthy")[
  An empty array `[]` is truthy. Use `{#if myArray.length > 0}` instead.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 5: {#each}
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("5", "Rendering Lists with {#each}")

== The Problem

You have an array of data and need to render each item.

== Basic Usage: Table Rows

#source-code("web/src/routes/webhooks/+page.svelte, lines 484-512")[
```
{#each filteredWebhooks as wh}
  <tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>
    <td>{wh.url}</td>
    <td>
      <CopyableId id={wh.webhookId}
        href="/webhooks/{wh.webhookId}" />
    </td>
    {#each wh.events.slice(0, 2) as event}
      <span class="badge">{event}</span>
    {/each}
    {#if wh.events.length > 2}
      <span>+{wh.events.length - 2}</span>
    {/if}
  </tr>
{/each}
```
]

== Key Points

- `{#each filteredWebhooks as wh}` --- `wh` is the loop variable holding one webhook per iteration.
- `wh.events.slice(0, 2)` --- Nested `{#each}`. "Show first 2 + overflow count" pattern.

== Skeleton Placeholders with Array(n)

#source-code("web/src/routes/health/+page.svelte, lines 104-109")[
```
{#each Array(4) as _}
  <div class="bg-white animate-pulse">
    <div class="h-8 bg-gray-200 rounded"></div>
  </div>
{/each}
```
]

- `Array(4)` --- Standard idiom for "repeat N times". `as _` means "I don't use this value."

#gotcha(1, "Keyed vs Unkeyed Lists")[
  For reorderable lists, add a key: `{#each items as item (item.id)}`. *Why?* Without a key, Svelte matches DOM nodes by _index_. If you remove item 2 from a list of 5, Svelte updates items 2--4 in place and destroys item 5---even though only item 2 was removed. With a key, Svelte matches by identity: it destroys only the removed item's DOM node and leaves the rest untouched. This matters when items have internal state (focused inputs, animations, component instances). Think of it like a database primary key: without it, the DB can't tell which row you mean.
]

#gotcha(2, "Destructuring in the loop")[
  You can destructure: `{#each webhooks as { url, webhookId }}`.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 6: {@const}
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("6", "Inline Constants with {@const}")

== The Problem

Inside an `{#each}` loop or `{#if}` block, you often need to compute a value. Without `{@const}`, you'd duplicate the expression everywhere.

== Map Lookup

#source-code("web/src/routes/health/+page.svelte, line 206")[
```
{#snippet webhookCard(wh: RegisteredWebhook)}
  {@const metrics = webhookMetrics.get(wh.webhookId)}
  <a href="/webhooks/{wh.webhookId}">
    {#if metrics}
      <span>{(metrics.successRate * 100).toFixed(1)}%</span>
    {/if}
  </a>
{/snippet}
```
]

- `{@const metrics = webhookMetrics.get(wh.webhookId)}` --- Creates a block-scoped constant. Without it, you'd repeat `.get()` 5+ times.

== Computed Value for Rendering

#source-code("web/src/routes/health/+page.svelte, line 228")[
```
{@const totalErrors = (metrics.clientErrors || 0)
  + (metrics.serverErrors || 0)
  + (metrics.timeoutErrors || 0)
  + (metrics.networkErrors || 0)
  + (metrics.unexpectedStatusErrors || 0)}
```
]

== Function Call Result

#source-code("web/src/routes/deliveries/[deliveryId]/+page.svelte, line 136")[
```
{#if delivery.errorCategory !== 'success'}
  {@const cat = getCategoryDisplay(delivery.errorCategory)}
  <span class="{cat.bgColor} {cat.color}">
    {cat.label}
  </span>
{/if}
```
]

Calls the function once, stores the result, then uses `cat.bgColor`, `cat.color`, `cat.label`.

#gotcha(1, "Must be at the top of a block")[
  `{@const}` must be first in its block.
]

#gotcha(2, "Truly const")[
  Cannot reassign it. Mutable values belong in the script section as `$state`.
]

#gotcha(3, "No async")[
  `{@const}` is synchronous only. Cannot use `await`.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 7: Event Handling
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("7", "Event Handling")

== The Problem

Users click buttons, type in inputs, press Enter, select dropdowns. In Svelte 5, event handlers are plain HTML attributes---no special syntax.

== Inline Handler

#source-code("web/src/routes/webhooks/+page.svelte, line 487")[
```
<tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>
```
]

- `onclick` --- Standard HTML attribute, lowercase. Different from Svelte 4's `on:click` (deprecated).

== Handler with Event Object

#source-code("web/src/routes/webhooks/+page.svelte, line 548")[
```
<button onclick={(e) => toggleActive(wh, e)}>
```
]

== Named Function Reference

#source-code("web/src/lib/components/CopyableId.svelte, lines 15-25 & 42")[
```
async function copyId(e: Event) {
  e.stopPropagation();
  e.preventDefault();
  try {
    await navigator.clipboard.writeText(id);
    copied = true;
    setTimeout(() => { copied = false; }, 1500);
  } catch { /* noop */ }
}

// In template:
<button onclick={copyId}>
```
]

== Keyboard Events

#source-code("web/src/routes/deliveries/+page.svelte, line 267")[
```
<input
  bind:value={namespaceFilter}
  onkeydown={(e) => e.key === 'Enter' && applyFilters()}
/>
```
]

== Form Submission

#source-code("web/src/routes/events/[eventName]/update/+page.svelte, line 131")[
```
<form onsubmit={updateEvent}>
  <input bind:value={name} disabled />
  <textarea bind:value={description}></textarea>
  <button type="submit">Update Event</button>
</form>
```
]

== Change Events on Selects

#source-code("web/src/routes/deliveries/+page.svelte, line 276")[
```
<select bind:value={statusFilter} onchange={applyFilters}>
  <option value="">All</option>
</select>
```
]

#gotcha(1, "handler vs handler()")[
  `onclick={handler}` passes the function. `onclick={handler()}` CALLS it immediately. Almost always a bug.
]

#gotcha(2, "Svelte 5 vs Svelte 4 syntax")[
  Svelte 5 uses `onclick`. Svelte 4 used `on:click`. Both work, but `on:` is deprecated.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 8: $effect()
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("8", "Side Effects with $effect()")

== The Problem

Sometimes when state changes, you need to do something beyond just updating the UI: start a timer, set up a DOM observer, log analytics. These are side effects. `$effect()` runs a function whenever its reactive dependencies change, and optionally cleans up when dependencies change again or the component is destroyed.

== Auto-Dismiss Timer

#source-code("web/src/lib/components/BatchProgress.svelte, lines 64-74")[
```
let autoDismissTimer: ReturnType<typeof setTimeout> | undefined;

$effect(() => {
  if (isTerminal && !dismissed) {
    ondone?.();
    autoDismissTimer = setTimeout(() => {
      dismissed = true;
    }, 5000);
  }
  return () => {
    if (autoDismissTimer) clearTimeout(autoDismissTimer);
  };
});
```
]

== Line by Line

- `$effect(() => { ... })` --- Runs after every render where its tracked dependencies change. Here it tracks `isTerminal` (a `$derived`) and `dismissed` (a `$state`).

- `ondone?.()` --- Optional chaining on a function call. If `ondone` was passed as a prop, call it. If not, do nothing.

- `setTimeout(() => { dismissed = true }, 5000)` --- After 5 seconds, auto-dismiss the progress bar.

- `return () => { ... }` --- The cleanup function. Called when (1) the effect re-runs or (2) the component is destroyed. Clears the timer to prevent memory leaks.

== IntersectionObserver Pattern

#source-code("web/src/lib/components/FloatingAction.svelte, lines 7-22")[
```
let visible = $state(false);

$effect(() => {
  const target = document.querySelector(targetSelector);
  if (!target) return;

  const observer = new IntersectionObserver(
    ([entry]) => {
      visible = !entry.isIntersecting;
    },
    { threshold: 0 }
  );

  observer.observe(target);

  return () => observer.disconnect();
});
```
]

== How It Works

- `IntersectionObserver` --- A browser API that fires a callback when an element enters or exits the viewport.
- `visible = !entry.isIntersecting` --- When the target scrolls out of view, show the floating action button.
- `return () => observer.disconnect()` --- Cleanup: stop observing when the component unmounts.

== \$effect vs \$derived

Use `$derived` for pure computations (input → output). Use `$effect` for anything that touches the outside world: DOM manipulation, timers, network requests, browser APIs.

== Under the Hood: How \$effect Tracks Dependencies

The runtime maintains a global variable called `active_reaction`. When an effect runs, it sets itself as the active reaction. Any signal read via `$.get(signal)` during execution registers the active reaction as a dependent. This is how dependencies are tracked automatically---no manual subscription, no dependency arrays like React's `useEffect([deps])`.

#code-block[
```
// runtime.js (simplified)
let active_reaction = null;  // global: who is running?

export function get(signal) {
  if (active_reaction !== null) {
    // Record: active_reaction depends on this signal
    signal.reactions.push(active_reaction);
  }
  return signal.v;  // return current value
}

// When an effect runs:
function run_effect(effect) {
  const prev = active_reaction;
  active_reaction = effect;   // set ourselves as active
  effect.fn();                // any $.get() calls register us
  active_reaction = prev;     // restore previous
}
```
]
#source-ref("sveltejs/svelte -- packages/svelte/src/internal/client/runtime.js")

So when the BatchProgress effect runs, it calls `$.get(isTerminal)` and `$.get(dismissed)`. Both signals now have this effect in their `reactions` list. When either signal changes, the effect is marked DIRTY and scheduled to re-run. The old cleanup function runs first, then the effect body executes again.

The compiler transforms `$effect(() => {...})` into `$.user_effect(() => {...})`. The runtime distinguishes between *user effects* (your code, runs after DOM updates) and *render effects* (compiler-generated, updates the DOM). You never create render effects directly---the compiler creates them from your template expressions.

#gotcha(1, "Don't use \$effect for derived state")[
  Never write `$effect(() => { count = items.length })`. Use `$derived` instead. *Why?* Effects that set `$state` variables create a hidden dependency cycle: the effect reads a signal (triggering registration), sets another signal (marking dependents dirty), which may trigger more effects. In the worst case, this causes an infinite loop. Even when it works, it creates an unnecessary intermediate state: the UI renders once with the stale value, then again with the updated one (two renders instead of one). `$derived` avoids both problems---it computes the value lazily, inline, with no intermediate state.
]

#gotcha(2, "Cleanup is critical")[
  If your effect creates timers, observers, or event listeners, always return a cleanup function. Forgetting cleanup causes memory leaks.
]

#gotcha(3, "\$effect runs after render")[
  `$effect` runs asynchronously after the DOM updates. If you need to read DOM measurements, `$effect` is the right place.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 9: Snippets & Render
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("9", "Snippets & Render")

== The Problem

You need to reuse a chunk of HTML template within a component, or pass template content from a parent to a child. In Svelte 4 this was done with slots. Svelte 5 replaces slots with snippets---explicit, typed, and more powerful.

== Declaring a Snippet in a Template

#source-code("web/src/routes/health/+page.svelte, lines 205-259")[
```
{#snippet webhookCard(wh: RegisteredWebhook)}
  {@const metrics = webhookMetrics.get(wh.webhookId)}
  <a href="/webhooks/{wh.webhookId}">
    <span>{wh.description || 'Webhook'}</span>
    <HealthBadge health={wh.health} size="sm" />
    {#if metrics}
      <span>{(metrics.successRate * 100).toFixed(1)}%</span>
    {/if}
  </a>
{/snippet}

<!-- Used twice: -->
{#each unhealthyWebhooks as wh}
  {@render webhookCard(wh)}
{/each}
{#each degradedWebhooks as wh}
  {@render webhookCard(wh)}
{/each}
```
]

== How It Works

- `{#snippet webhookCard(wh: RegisteredWebhook)}` --- Declares a reusable template block. Think of it like a function that returns HTML.
- `{@render webhookCard(wh)}` --- Calls the snippet, passing the current webhook. Rendered inline at the call site.
- Defined once, rendered in two loops (unhealthy + degraded). Without snippets, you'd copy the entire card HTML twice.

== Snippets as Props (Replacing Slots)

#source-code("web/src/lib/components/EmptyState.svelte, lines 1-25")[
```
// EmptyState.svelte:
import type { Snippet } from 'svelte';

interface Props {
  icon?: string;
  title: string;
  description?: string;
  action?: Snippet;  // <-- Snippet type!
}

let { icon, title, description, action } = $props();

// In template:
{#if action}
  {@render action()}
{/if}
```
]

== Passing a Snippet from Parent

#source-code("web/src/lib/components/SubscriptionManager.svelte, lines 425-433")[
```
<EmptyState
  icon="link"
  title="No subscriptions yet"
  description="Create subscriptions to define..."
>
  {#snippet action()}
    <button onclick={openCreateModal}>
      Create First Subscription
    </button>
  {/snippet}
</EmptyState>
```
]

- `action?: Snippet` --- The `Snippet` type represents a renderable template block. Optional because not every empty state needs a button.
- `{#snippet action()}` inside parent --- Defines snippet content inline. The name must match the prop name.

== Layout Children Pattern

#source-code("web/src/routes/+layout.svelte, lines 6 & 60")[
```
let { children } = $props();

<!-- At the bottom of the layout: -->
{@render children?.()}
```
]

`children` is a built-in snippet prop in SvelteKit that contains the page content. The `?.` guards against undefined during SSR.

#gotcha(1, "Snippets replace slots")[
  Svelte 4's `<slot />` is deprecated. Use `{#snippet}` + `{@render}` instead.
]

#gotcha(2, "Snippet scope")[
  Snippets defined inside a component access all variables in the component's scope (closures).
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 10: Two-Way Binding
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("10", "Two-Way Binding")

== The Problem

Form inputs need to both display a value and update it when the user types. Svelte's `bind:` directive automates this two-way sync.

== bind:value on Inputs

#source-code("web/src/lib/components/SubscriptionManager.svelte, lines 674-680")[
```
<input
  type="text"
  bind:value={form.eventName}
  oninput={() => handleEventChange(form.eventName)}
  placeholder="Type event name"
/>
```
]

- `bind:value={form.eventName}` --- Two-way sync: the input displays the value, and typing updates it automatically.
- The `oninput` handler here is for additional logic (fetching event details), not for the binding itself.

== bind:value on Selects

#source-code("web/src/routes/deliveries/+page.svelte, line 276")[
```
<select bind:value={statusFilter} onchange={applyFilters}>
  <option value="">All</option>
  <option value="delivered">Delivered</option>
  <option value="failed">Failed</option>
</select>
```
]

== \$bindable: Two-Way Props

#source-code("web/src/lib/components/SubscriptionManager.svelte, lines 19-29")[
```
let {
  webhookId,
  namespace,
  subscriptions = $bindable([]),
  onRefresh,
}: {
  webhookId: string;
  namespace: string;
  subscriptions: EventSubscription[];
  onRefresh?: () => void;
} = $props();
```
]

== How \$bindable Works

- `subscriptions = $bindable([])` --- Declares that this prop can be two-way bound. The parent uses `bind:subscriptions={myList}` and changes flow both directions.
- Without `$bindable`, props are read-only (Lesson 3). `$bindable` explicitly opts into two-way communication.
- `$bindable([])` provides a default value if the parent doesn't bind.

== When to Use Each Pattern

*bind:value* --- For native HTML elements (input, select, textarea).

*\$bindable* --- For component props needing two-way flow. Rare---prefer callback props.

*Callback props* --- For most child-to-parent communication (Lesson 3). More explicit.

#gotcha(1, "bind: is two-way")[
  Changes flow both directions. Setting the variable updates the input; typing updates the variable.
]

#gotcha(2, "\$bindable is opt-in")[
  By default, props are read-only. You must explicitly mark a prop as `$bindable`.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 11: Lifecycle Hooks
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("11", "Lifecycle Hooks")

== The Problem

Components need to do things at specific moments: fetch data when first rendered, clean up timers when removed.

== onMount: Fetching Data on Load

#source-code("web/src/routes/health/+page.svelte, line 70")[
```
import { onMount } from 'svelte';

let loading = $state(true);
let error = $state('');

async function fetchData() {
  loading = true;
  error = '';
  try {
    const res = await healthClient.getHealthSummary({});
    healthSummary = res.summary;
  } catch (e: any) {
    error = formatAPIError(e, 'Failed to load');
  } finally {
    loading = false;
  }
}

onMount(fetchData);
```
]

== How It Works

- `onMount(fetchData)` --- Calls `fetchData` once, after the component is first rendered into the DOM. NOT during SSR.
- The `try/catch/finally` pattern ensures `loading` is always cleared, even on failure.

== onDestroy: Cleaning Up

#source-code("web/src/routes/deliveries/+page.svelte, lines 4, 44-46")[
```
import { onDestroy } from 'svelte';

let pollingTimer: ReturnType<typeof setInterval> | undefined;

onDestroy(() => {
  if (pollingTimer) clearInterval(pollingTimer);
});
```
]

- `onDestroy` --- Runs when the component is removed from the DOM. Use it to clean up intervals, event listeners, subscriptions.

== onMount vs \$effect

*onMount* --- Runs once after first render. Best for initial data fetching.

*\$effect* --- Runs after every render where dependencies change. Best for reactive side effects.

Use `onMount` for one-time setup. Use `$effect` when you need to react to state changes.

#gotcha(1, "onMount doesn't run on the server")[
  `onMount` is browser-only. During SSR, the component renders without calling it.
]

#gotcha(2, "Don't forget cleanup")[
  If `onMount` starts an interval, pair it with `onDestroy`. Or use `$effect` with return cleanup.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 12: Async Data Fetching
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("12", "Async Data Fetching")

== The Problem

Every page loads data from a backend API. You need to handle loading state, display errors, and sometimes fetch multiple resources in parallel.

== The API Client Layer

#source-code("web/src/lib/services.ts, lines 41-50")[
```
import { createClient } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { WebhookService, EventService } from '../proto/webhook_pb.js';

const transport = createConnectTransport({
  baseUrl: PUBLIC_API_URL || '/',
  interceptors,  // API key injection
});

export const webhookClient = createClient(WebhookService, transport);
export const eventClient = createClient(EventService, transport);
export const deliveryClient = createClient(DeliveryService, transport);
export const healthClient = createClient(HealthService, transport);
```
]

== How It Works

- `createConnectTransport` --- Creates an HTTP transport that speaks the Connect protocol.
- `createClient(WebhookService, transport)` --- Creates a typed client. You call `webhookClient.listWebhooks({...})` and get back typed responses.
- `interceptors` --- Middleware that attaches the `X-API-Key` header when configured.

== Parallel Fetching with Promise.all

#source-code("web/src/routes/health/+page.svelte, lines 27-38")[
```
const [summaryRes, statsRes, unhealthyRes, degradedRes] =
  await Promise.all([
    healthClient.getHealthSummary({}),
    webhookClient.getNamespaceStats({ namespace: '' }),
    healthClient.listWebhooksByHealth({
      health: WebhookHealth.HEALTH_UNHEALTHY,
      pagination: { limit: 20, offset: 0 },
    }),
    healthClient.listWebhooksByHealth({
      health: WebhookHealth.HEALTH_DEGRADED,
      pagination: { limit: 20, offset: 0 },
    }),
  ]);
```
]

- `Promise.all([...])` --- Runs all four API calls concurrently. Waits for all to complete.
- `const [a, b, c, d] = await ...` --- Array destructuring on the result.

== The Loading/Error/Content Pattern

#code-block[
```
let loading = $state(true);
let error = $state('');
let data = $state<DataType | undefined>();

async function fetchData() {
  loading = true;
  error = '';
  try {
    const res = await client.getData({...});
    data = res.data;
  } catch (e: any) {
    error = formatAPIError(e, 'Failed to load');
  } finally {
    loading = false;
  }
}

onMount(fetchData);
```
]

The template then uses the three-state `{#if}` pattern from Lesson 4.

== Error Formatting

#source-code("web/src/lib/utils.ts, lines 217-241")[
```
function formatAPIError(err: unknown, contextPrefix?: string): string {
  let msg = (err as any)?.message ?? String(err);
  // Strip gRPC code prefix: "[internal] ..."
  msg = msg.replace(/^\[\w+\]\s*/, '');
  if (!contextPrefix) return msg;
  return `${contextPrefix}: ${msg}`;
}
```
]

#gotcha(1, "Always set loading = false")[
  Use `finally { loading = false }`. Without `finally`, an error leaves the page stuck on the skeleton.
]

#gotcha(2, "Promise.all is all-or-nothing")[
  If one call fails, you lose all results. Use `Promise.allSettled()` for partial results.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 13: Layout & Navigation
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("13", "Layout & Navigation")

== The Problem

Every page shares a navigation header. You don't want to copy this HTML into every page. SvelteKit's layout system lets you define shared UI once.

== The Layout Component

#source-code("web/src/routes/+layout.svelte, all 60 lines")[
```
<script lang="ts">
  import { page } from "$app/state";
  import "../app.css";

  let { children } = $props();

  const titles: Record<string, string> = {
    "/webhooks": "Webhooks",
    "/events": "Events",
    "/health": "Health",
  };

  function getTitle(): string {
    const path = page.route.id?.toString() || "/";
    return titles[path] || "";
  }
</script>

<header class="sticky top-0 ...">
  <h2>{getTitle()}</h2>
  <nav>
    <a href="/webhooks">Webhooks</a>
    <a href="/events">Events</a>
    <a href="/deliveries">Deliveries</a>
    <a href="/health">Health</a>
  </nav>
</header>

{@render children?.()}
```
]

== How It Works

- `let { children } = $props()` --- SvelteKit passes a `children` snippet containing the current page's content.
- `{@render children?.()}` --- Renders the page content below the header.
- `page.route.id` --- SvelteKit's reactive page state gives the current route pattern for dynamic titles.
- `import "../app.css"` --- Global CSS (Tailwind) imported in layout, available everywhere.

== SvelteKit Navigation: goto()

#source-code("web/src/routes/webhooks/+page.svelte, line 487")[
```
import { goto } from '$app/navigation';

<tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>
```
]

- `goto(url)` --- Programmatic navigation. No full page reload---SvelteKit does client-side navigation.

== Dynamic Routes

#code-block[
```
web/src/routes/
  webhooks/
    +page.svelte          -> /webhooks
    register/+page.svelte -> /webhooks/register
    [webhookId]/+page.svelte -> /webhooks/abc123
  events/
    [eventName]/
      reports/+page.svelte -> /events/user.created/reports
```
]

- `[webhookId]` --- A dynamic segment. The value is available via `page.params.webhookId`.

== svelte:head

#source-code("web/src/routes/health/+page.svelte, lines 73-75")[
```
<svelte:head>
  <title>Health | Sparrow</title>
</svelte:head>
```
]

`<svelte:head>` injects elements into the document head. Each page sets its own title.

#gotcha(1, "Layout wraps all pages")[
  Everything in `+layout.svelte` appears on every page in that directory (and subdirectories).
]

#gotcha(2, "goto() vs <a>")[
  Prefer `<a href="...">` for regular links. Use `goto()` for programmatic navigation.
]


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 14: Component Composition
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("14", "Component Composition")

== The Problem

Real applications need reusable components that work together: a pagination control, a progress bar, a confirmation dialog. This lesson shows how to design them, combining Lessons 1--13.

== Pattern 1: Callback-Driven Component

#source-code("web/src/lib/components/Pagination.svelte, all 72 lines")[
```
interface Props {
  currentPage: number;
  totalPages: number;
  totalCount: number;
  pageSize: number;
  onPageChange: (pageNum: number) => void;
  itemLabel?: string;
}

let { currentPage, totalPages, totalCount,
  pageSize, onPageChange, itemLabel = 'items'
}: Props = $props();

// In template:
<button onclick={() => onPageChange(pageNum)}>
  {pageNum}
</button>
```
]

Pagination is "dumb"---it displays page numbers and calls `onPageChange` when clicked. The parent owns the state and data-fetching logic.

== Pattern 2: Self-Contained with \$effect

#source-code("web/src/lib/components/BatchProgress.svelte, lines 24-74")[
```
let { batch, label = 'Batch', oncancel, ondone } = $props();

let dismissed = $state(false);

let progressPercent = $derived(
  batch && batch.total > 0
    ? Math.round(((batch.processed + batch.failed)
        / batch.total) * 100)
    : 0
);

let isTerminal = $derived(
  batch?.status === 'completed' ||
  batch?.status === 'failed'
);

let statusColor = $derived.by(() => {
  switch (batch.status) {
    case 'completed': return 'bg-green-500';
    case 'failed':    return 'bg-red-500';
    case 'processing': return 'bg-blue-500';
    default:           return 'bg-yellow-500';
  }
});

$effect(() => {
  if (isTerminal && !dismissed) {
    ondone?.();
    autoDismissTimer = setTimeout(
      () => { dismissed = true; }, 5000);
  }
  return () => clearTimeout(autoDismissTimer);
});
```
]

== How It Combines Lessons

- *\$props()* (Lesson 3) --- Receives batch status and callbacks.
- *\$state* (Lesson 1) --- Local `dismissed` state.
- *\$derived* (Lesson 2) --- `progressPercent`, `isTerminal`, `statusColor`.
- *\$derived.by()* (Lesson 2) --- Switch statement needs block form.
- *\$effect* (Lesson 8) --- Auto-dismiss timer with cleanup.
- *Callback props* (Lesson 3) --- `oncancel`, `ondone`.

== Pattern 3: Snippet Slot Component

#source-code("web/src/lib/components/EmptyState.svelte")[
```
import type { Snippet } from 'svelte';

interface Props {
  icon?: string;
  title: string;
  description?: string;
  action?: Snippet;  // Slot replacement
}

// Template:
{#if action}
  {@render action()}
{/if}
```
]

EmptyState provides structure (icon, title) and a snippet slot for custom content. The parent decides what action to show; the child decides where.

== Design Guidelines

- *Keep components focused* --- Each component has one job.
- *Parent owns state, child reports events* --- Use callback props.
- *Use snippets for flexible content* --- Accept `Snippet` props for custom HTML.
- *Type everything* --- `interface Props` for every component.


// ═══════════════════════════════════════════════════════════════════════════
// LESSON 15: The Full Stack
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("15", "The Full Stack")

This final lesson traces a complete path through Sparrow's stack---from protobuf definition to compiled Svelte component, embedded in a Go binary, served to the browser. Every technology choice at each layer exists for a concrete reason.

== The Journey of a Type

#code-block[
```
+--------------+   buf generate   +------------------+
| webhook.proto |---------------> | Go structs       |
| (source of   |                  | (proto/webhook.  |
|  truth)      |                  |  pb.go)          |
|              |   protoc-gen-es  +------------------+
| message      |---------------> | JS classes + .dts|
| Webhook {    |                  | (proto/webhook_  |
|   id, url,   |                  |  pb.js/d.ts)     |
|   secret ... |                  +--------+---------+
| }            |                           |
+--------------+                           | import
                                           v
                                    +--------------+
                                    | services.ts  |
                                    | (typed       |
                                    |  RPC client) |
                                    +------+-------+
                                           |
                                           v
                                    +--------------+
                                    | Svelte       |
                                    | components   |
                                    | (type-safe)  |
                                    +--------------+
```
]

== Layer 1: Protobuf --- Single Source of Truth

The `webhook.proto` file defines all types and RPC services once. The `buf generate` command produces Go server code, Go clients, JS/TS clients, and Python clients from this single definition. No type duplication across languages.

#source-code("buf.gen.yaml")[
```
# buf.gen.yaml -- generates code for all languages
plugins:
  # Go server (protobuf + gRPC + Connect-RPC)
  - plugin: buf.build/protocolbuffers/go
    out: .
  - plugin: buf.build/connectrpc/go
    out: .

  # Web UI (Connect-RPC / @bufbuild/protobuf v2)
  - name: es
    path: web/node_modules/.bin/protoc-gen-es
    out: .
    opt:
      - target=js+dts  # JS classes + TypeScript types
```
]

== Layer 2: Connect-RPC Transport

Connect-RPC is a protocol-compatible alternative to gRPC-Web that works over standard HTTP. The Svelte frontend creates a typed transport with an interceptor that attaches the API key header to every request.

#source-code("web/src/lib/services.ts, lines 1-53")[
```
// Real Sparrow code (services.ts):
import { PUBLIC_API_URL }
  from '$env/static/public';
import { createClient }
  from '@connectrpc/connect';
import { createConnectTransport }
  from '@connectrpc/connect-web';
import type { Interceptor }
  from '@connectrpc/connect';
import {
  WebhookService, EventService,
  SubscriptionService, DeliveryService,
  HealthService
} from '../../../proto/webhook_pb.js';

// Runtime config injected by Go server
const runtimeConfig: SparrowConfig =
  (typeof window !== 'undefined'
    && window.__SPARROW_CONFIG__) || {};
const apiKey = runtimeConfig.apiKey || '';

// Interceptor attaches API key header
const apiKeyInterceptor: Interceptor =
  (next) => async (req) => {
    req.header.set('X-API-Key', apiKey);
    return next(req);
  };

const transport = createConnectTransport({
  baseUrl: PUBLIC_API_URL || '/',
  interceptors: apiKey
    ? [apiKeyInterceptor] : [],
});

// Typed clients (full autocomplete on RPCs)
export const webhookClient =
  createClient(WebhookService, transport);
export const eventClient =
  createClient(EventService, transport);
export const deliveryClient =
  createClient(DeliveryService, transport);
export const healthClient =
  createClient(HealthService, transport);
```
]

== Layer 3: Svelte Components (Type-Safe)

Components call the typed RPC clients and display results. TypeScript ensures the response types match the proto definitions. The Svelte compiler then turns these components into optimized DOM update code---no virtual DOM, just direct mutations as described in Lessons 1 and 3.

#source-code("web/src/routes/webhooks/+page.svelte, lines 2-17, 62-98")[
```
// Real Sparrow code (webhooks/+page.svelte):
import { webhookClient as client,
         healthClient } from '$lib/services';
import { onMount } from 'svelte';
import type { RegisteredWebhook }
  from '../../../../proto/webhook_pb.js';

let webhooks: RegisteredWebhook[] = $state([]);
let loading = $state(true);
let error = $state('');

// Pagination state
let limit = $state(25);
let offset = $state(0);
let totalCount = $state(0);

async function fetchWebhooks() {
  loading = true;
  error = '';
  try {
    const res = await client.listWebhooks({
      namespace: '',
      pagination: { limit, offset },
    });
    // res.webhooks is RegisteredWebhook[]
    // TypeScript enforces field access:
    //   res.webhooks[0].url       OK
    //   res.webhooks[0].bogus     compile error
    webhooks = res.webhooks || [];
    totalCount =
      res.pagination?.totalCount || 0;
  } catch (e: any) {
    error = formatAPIError(e,
      'Failed to load webhooks');
  } finally {
    loading = false;
  }
}

onMount(fetchWebhooks);  // fetch on page load
```
]

== Layer 4: Build + Embed

The SvelteKit build produces static HTML/JS/CSS in `internal/ui/dist/`. The Go compiler's `embed` directive bakes these files into the binary. The result is a single executable with no external dependencies---no Node runtime, no file server, no CDN required.

#code-block[
```
// The complete build chain:
$ cd web && npm run build
  # Vite -> Svelte compiler -> Tailwind -> tree-shake
  # -> adapter-static -> ../internal/ui/dist/

$ go build ./cmd/server
  # go:embed all:dist -> embed FS in binary
  # Output: single static binary (~15MB)

$ ./server
  # :8080 -- Connect-RPC API + embedded SPA
  # :50051 -- gRPC (direct)
  # No Node. No nginx. No CDN. One process.
```
]

== Why This Architecture

Each technology at each layer was chosen for a specific reason:

- *Protobuf* --- Single source of truth for types across Go, TypeScript, and Python. Schema evolution without breaking changes.
- *Connect-RPC* --- Works over standard HTTP (no gRPC-Web proxy needed). Browser-compatible. Same proto definitions as gRPC.
- *Svelte 5* --- Compiled to vanilla JS. No runtime framework to ship. \~5--8 KB tree-shaken runtime vs \~44 KB for React. Runes provide type-safe reactivity.
- *adapter-static* --- Pure static files that can be embedded. No SSR server needed. SPA mode for client-side routing.
- *go:embed* --- Single binary deployment. No filesystem dependencies. Copy one file, run it. Works in distroless containers.
- *Distroless* --- No shell, no package manager. \~5MB base image. Minimal attack surface for a self-hosted webhook platform.

== The End-to-End Type Safety Chain

The most important property of this architecture is the unbroken type chain. A field added to `webhook.proto` automatically appears in the Go server struct, the TypeScript client, and the Svelte component---with compile-time errors if any layer gets it wrong. There is no manually-maintained API contract, no JSON schema to keep in sync, no runtime type assertion. The proto file IS the contract, and code generation enforces it everywhere.


// ═══════════════════════════════════════════════════════════════════════════
// QUICK REFERENCE CARD
// ═══════════════════════════════════════════════════════════════════════════

#pagebreak()
#lesson-header("", "Quick Reference Card")

#ref-table(
  columns: (1.3in, 2.7in, 2.5in),

  // Header row
  table.header(th[Concept], th[Syntax], th[Use When]),

  // Data rows
  tb[Reactive state],
  tc[\$state(initialValue)],
  td[Data you set directly],

  tb[Computed value],
  tc[\$derived(expr)\ \$derived.by(() => \{ \})],
  td[Value calculated from state],

  tb[Side effects],
  tc[\$effect(() => \{\ ...\ return () => cleanup;\ \})],
  td[Timers, DOM, browser APIs],

  tb[Component props],
  tc[let \{ a, b \} = \$props()],
  td[Receiving data from parent],

  tb[Two-way prop],
  tc[x = \$bindable(\[\])],
  td[Parent reads/writes child state],

  tb[Conditional],
  tc(raw("{#if cond}  {:else if}  {:else}  {/if}")),
  td[Show/hide DOM sections],

  tb[List rendering],
  tc(raw("{#each arr as item}  {#each arr as x (key)}")),
  td[Rendering arrays],

  tb[Inline constant],
  tc(raw("{@const x = expr}")),
  td[Avoid repeated expressions],

  tb[Snippet (define)],
  tc(raw("{#snippet name(params)}  HTML...  {/snippet}")),
  td[Reusable template blocks],

  tb[Snippet (render)],
  tc(raw("{@render name(args)}")),
  td[Calling a snippet],

  tb[Event handler],
  tc[onclick=\{handler\}\ onclick=\{(e) => \{\}\}],
  td[User interactions],

  tb[Two-way binding],
  tc[bind:value=\{x\}],
  td[Form inputs],

  tb[Lifecycle],
  tc[onMount(fn)\ onDestroy(fn)],
  td[Setup/cleanup],

  tb[Head elements],
  tc(raw("<svelte:head>  <title>...</title>  </svelte:head>")),
  td[Per-page title/meta],
)

#v(0.3in)

== Key TypeScript Syntax Explained in This Tutorial

#ref-table(
  columns: (1.5in, 2.2in, 2.8in),

  // Header
  table.header(th[Syntax], th[Meaning], th[Example]),

  // Data rows
  tc[x: string],
  td[Type annotation],
  tc[let name: string = 'hi'],

  tc[x?: string],
  td[Optional property],
  tc[interface \{ href?: string \}],

  tc[A | B],
  td[Union type (A or B)],
  tc[string | undefined],

  tc[T\[\]],
  td[Array of type T],
  tc[RegisteredWebhook\[\]],

  tc[Map\<K, V\>],
  td[Map with key/value types],
  tc[Map\<string, Metrics\>],

  tc[() => void],
  td[Fn, no args, no return],
  tc[onconfirm: () => void],

  tc[x?.y],
  td[Optional chaining],
  tc[desc?.toLowerCase()],

  tc[\<T\>()],
  td[Generic type parameter],
  tc[\$state\<Webhook\[\]\>(\[\])],

  tc[interface],
  td[Object shape definition],
  tc[interface Props \{ ... \}],

  tc[Snippet],
  td[Renderable template type],
  tc[action?: Snippet],

  tc[ReturnType\<T\>],
  td[Extract return type],
  tc[ReturnType\<typeof setTimeout\>],

  tc[Record\<K,V\>],
  td[Object with typed keys],
  tc[Record\<string, string\>],

  tc[as any],
  td[Type assertion],
  tc[(err as any)?.message],
)

#v(0.5in)
#hr()
#v(0.1in)

All code examples from the Sparrow webhook platform. \
#link-text[https://github.com/sarathsp06/sparrow]
