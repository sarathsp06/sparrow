// ==========================================================================
// Svelte 5 Tutorial — Typst Template
// Reusable styles, functions, and layout for the tutorial book.
// ==========================================================================

// ---------------------------------------------------------------------------
// Colors
// ---------------------------------------------------------------------------
#let primary       = rgb("#1a1a2e")
#let svelte-orange = rgb("#ff3e00")
#let accent        = svelte-orange
#let code-bg       = rgb("#f8f9fa")
#let code-border   = rgb("#dee2e6")
#let blue          = rgb("#2563eb")
#let gray-600      = rgb("#4b5563")
#let gray-400      = rgb("#9ca3af")
#let green         = rgb("#059669")
#let warn-orange   = rgb("#d97706")
#let dark-bg       = rgb("#0f172a")
#let slate-700     = rgb("#334155")

// ---------------------------------------------------------------------------
// Reusable Components
// ---------------------------------------------------------------------------

/// Lesson header block: tag + title. Placed after a pagebreak by the caller.
#let lesson-header(number, title) = {
  if number != "" {
    text(
      font: "Inter", weight: "bold", size: 11pt,
      fill: accent,
    )[LESSON #number]
    v(2pt)
  }
  heading(level: 1, title)
}

/// Section heading (level 2).
#let section-head(title) = heading(level: 2, title)

/// Source reference — italic blue text.
#let source-ref(ref-text) = {
  v(4pt)
  text(font: "Inter", style: "italic", size: 9pt, fill: blue)[Source: #ref-text]
  v(2pt)
}

/// Code block — monospace on gray background.
#let code-block(code-text, lang: none) = {
  block(
    width: 100%,
    fill: code-bg,
    stroke: 0.5pt + code-border,
    inset: 8pt,
    radius: 2pt,
    below: 8pt,
    text(font: "Fira Code", size: 9pt, code-text),
  )
}

/// Source reference immediately followed by a code block, kept together.
#let source-code(ref-text, code-text) = {
  block(breakable: false)[
    #source-ref(ref-text)
    #code-block(code-text)
  ]
}

/// Body paragraph (justified Inter).
#let body-text(content) = {
  text(font: "Inter", size: 10.5pt, content)
}

/// Bullet item.
#let bullet-item(content) = {
  list.item(content)
}

/// Gotcha box — warning-colored heading + gray body, kept together.
#let gotcha(number, title, body) = {
  block(breakable: false, above: 10pt, below: 6pt)[
    #text(
      font: "Inter", weight: "bold", size: 10.5pt,
      fill: warn-orange,
    )[Gotcha #number: #title]
    #v(2pt)
    #text(font: "Inter", size: 10pt, fill: gray-600, body)
  ]
}

/// Horizontal rule.
#let hr() = line(length: 100%, stroke: 0.5pt + code-border)

// ---------------------------------------------------------------------------
// Document Setup Function (applied via #show: ...)
// ---------------------------------------------------------------------------
#let tutorial-doc(
  title: "Svelte 5 Tutorial",
  author: "Sparrow Project",
  version: "v0.0.0",
  body,
) = {
  // -- Page layout --
  set page(
    paper: "us-letter",
    margin: (top: 0.6in, bottom: 0.6in, left: 0.75in, right: 0.75in),
    footer: context {
      // No footer on the first page (cover)
      if counter(page).get().first() > 1 {
        set text(font: "Inter", size: 8pt, fill: gray-400)
        grid(
          columns: (1fr, 1fr, 1fr),
          align: (left, center, right),
          [Svelte 5 Tutorial],
          [#counter(page).display()],
          [Sparrow Codebase Examples],
        )
      }
    },
  )

  // Start page counter from 1 on cover
  counter(page).update(1)

  // -- Fonts --
  set text(font: "Inter", size: 10.5pt, fill: black)

  // -- Paragraphs --
  set par(justify: true, leading: 0.65em)

  // -- Headings --
  show heading.where(level: 1): it => {
    set text(font: "Inter", weight: "bold", size: 22pt, fill: primary)
    block(above: 0pt, below: 12pt, it.body)
  }

  show heading.where(level: 2): it => {
    set text(font: "Inter", weight: "bold", size: 14pt, fill: primary)
    block(above: 16pt, below: 6pt, breakable: false, it.body)
  }

  // -- Lists --
  set list(marker: [•], indent: 6pt, body-indent: 8pt, spacing: 6pt)

  // -- Raw/code inline --
  show raw.where(block: false): it => {
    box(
      text(font: "Fira Code", size: 9.5pt, it),
    )
  }

  // -- Document metadata --
  set document(title: title, author: author)

  body
}
