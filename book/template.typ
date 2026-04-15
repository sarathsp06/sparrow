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
#let table-header-bg = rgb("#f3f4f6")

// Cover-specific shades (derived from svelte-orange)
#let cover-mid    = rgb("#e8846b")
#let cover-light  = rgb("#d4a08a")
#let cover-rule   = rgb("#cccccc")
#let cover-title  = rgb("#333333")
#let cover-sub    = rgb("#555555")
#let cover-body   = rgb("#666666")
#let cover-meta   = rgb("#888888")
#let cover-series = rgb("#666666")

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

/// Code block — thin wrapper; visual styling is handled by the
/// `show raw.where(block: true)` rule in `tutorial-doc`.
#let code-block(code-text, lang: none) = {
  code-text
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

/// Styled link / URL text.
#let link-text(body) = text(fill: blue, body)

// ---------------------------------------------------------------------------
// Reference Table Helpers
// ---------------------------------------------------------------------------

/// Styled table header cell — bold text on light gray background.
#let th(body) = {
  table.cell(fill: table-header-bg, text(weight: "bold", fill: primary, size: 9pt, body))
}

/// Table cell with monospace font for code/syntax.
#let tc(body) = {
  text(font: "Fira Code", size: 8.5pt, body)
}

/// Table cell with regular body text.
#let td(body) = {
  text(size: 9pt, body)
}

/// Table cell with bold label text.
#let tb(body) = {
  text(weight: "bold", size: 9pt, body)
}

/// Reference table — consistent styling for Quick Reference / Cheat Sheet tables.
#let ref-table(columns: (), ..cells) = {
  table(
    columns: columns,
    align: (left,) * columns.len(),
    stroke: 0.5pt + code-border,
    inset: 6pt,
    ..cells,
  )
}

// ---------------------------------------------------------------------------
// Cover Page
// ---------------------------------------------------------------------------

/// O'Reilly-style cover page with dissolving code blocks illustration.
#let cover-page(
  title: "Svelte 5 Tutorial",
  subtitle: "The Disappearing Framework",
  tagline: none,
  series: "SPARROW ENGINEERING SERIES",
  topics: "Svelte 5 Runes · SvelteKit · TypeScript",
  meta: none,
  publisher: "SPARROW",
  version: "v0.0.0",
) = {
  page(
    margin: 0pt,
    footer: none,
  )[
    #set text(font: "Inter")

    // White background
    #place(top + left, rect(width: 100%, height: 100%, fill: white))

    // Top colored band
    #place(top + left, rect(width: 100%, height: 12pt, fill: svelte-orange))

    // Series branding
    #place(top + left, dx: 48pt, dy: 36pt)[
      #text(fill: cover-series, size: 9pt, weight: "bold", tracking: 1pt)[#series]
    ]

    // Rule below branding
    #place(top + left, dx: 48pt, dy: 56pt,
      rect(width: 516pt, height: 0.5pt, fill: cover-rule),
    )

    // ── Illustration: dissolving code blocks ──
    #place(top + left, dx: 80pt, dy: 100pt)[
      #box(width: 460pt, height: 280pt)[
        // Source blocks (solid)
        #place(left + top, dx: 0pt, dy: 20pt,
          rect(width: 60pt, height: 18pt, fill: svelte-orange, radius: 2pt))
        #place(left + top, dx: 0pt, dy: 44pt,
          rect(width: 90pt, height: 18pt, fill: cover-mid, radius: 2pt))
        #place(left + top, dx: 10pt, dy: 68pt,
          rect(width: 70pt, height: 18pt, fill: cover-light, radius: 2pt))
        #place(left + top, dx: 0pt, dy: 92pt,
          rect(width: 50pt, height: 18pt, fill: svelte-orange, radius: 2pt))
        #place(left + top, dx: 0pt, dy: 116pt,
          rect(width: 80pt, height: 18pt, fill: cover-mid, radius: 2pt))
        #place(left + top, dx: 10pt, dy: 140pt,
          rect(width: 55pt, height: 18pt, fill: cover-light, radius: 2pt))
        #place(left + top, dx: 0pt, dy: 164pt,
          rect(width: 45pt, height: 18pt, fill: svelte-orange, radius: 2pt))

        // Fragments (breaking apart)
        #place(left + top, dx: 140pt, dy: 30pt,
          rect(width: 40pt, height: 14pt, fill: rgb(255, 62, 0, 180), radius: 2pt))
        #place(left + top, dx: 155pt, dy: 55pt,
          rect(width: 30pt, height: 12pt, fill: rgb(255, 62, 0, 140), radius: 2pt))
        #place(left + top, dx: 145pt, dy: 78pt,
          rect(width: 35pt, height: 13pt, fill: rgb(255, 62, 0, 120), radius: 2pt))
        #place(left + top, dx: 160pt, dy: 105pt,
          rect(width: 25pt, height: 11pt, fill: rgb(255, 62, 0, 100), radius: 2pt))
        #place(left + top, dx: 148pt, dy: 130pt,
          rect(width: 28pt, height: 12pt, fill: rgb(255, 62, 0, 80), radius: 2pt))
        #place(left + top, dx: 155pt, dy: 155pt,
          rect(width: 22pt, height: 10pt, fill: rgb(255, 62, 0, 60), radius: 2pt))

        // Particles (dissolving)
        #place(left + top, dx: 220pt, dy: 25pt,
          rect(width: 18pt, height: 8pt, fill: rgb(255, 62, 0, 80), radius: 1pt))
        #place(left + top, dx: 240pt, dy: 60pt,
          rect(width: 14pt, height: 7pt, fill: rgb(255, 62, 0, 60), radius: 1pt))
        #place(left + top, dx: 225pt, dy: 90pt,
          rect(width: 16pt, height: 6pt, fill: rgb(255, 62, 0, 50), radius: 1pt))
        #place(left + top, dx: 245pt, dy: 120pt,
          rect(width: 12pt, height: 6pt, fill: rgb(255, 62, 0, 40), radius: 1pt))
        #place(left + top, dx: 230pt, dy: 148pt,
          rect(width: 10pt, height: 5pt, fill: rgb(255, 62, 0, 30), radius: 1pt))

        // Faint dots
        #place(left + top, dx: 290pt, dy: 35pt,
          rect(width: 8pt, height: 4pt, fill: rgb(255, 62, 0, 30), radius: 1pt))
        #place(left + top, dx: 300pt, dy: 70pt,
          rect(width: 6pt, height: 3pt, fill: rgb(255, 62, 0, 20), radius: 1pt))
        #place(left + top, dx: 295pt, dy: 105pt,
          rect(width: 7pt, height: 4pt, fill: rgb(255, 62, 0, 15), radius: 1pt))
        #place(left + top, dx: 305pt, dy: 140pt,
          rect(width: 5pt, height: 3pt, fill: rgb(255, 62, 0, 10), radius: 1pt))

        // Ghost traces
        #place(left + top, dx: 350pt, dy: 50pt,
          rect(width: 4pt, height: 3pt, fill: rgb(255, 62, 0, 8), radius: 1pt))
        #place(left + top, dx: 360pt, dy: 90pt,
          rect(width: 3pt, height: 2pt, fill: rgb(255, 62, 0, 5), radius: 1pt))
        #place(left + top, dx: 355pt, dy: 130pt,
          rect(width: 3pt, height: 2pt, fill: rgb(255, 62, 0, 4), radius: 1pt))

        // Phase labels
        #place(left + bottom, dx: 10pt, dy: -8pt,
          text(fill: rgb("#999999"), size: 7pt, weight: "bold", tracking: 0.5pt)[SOURCE])
        #place(left + bottom, dx: 155pt, dy: -8pt,
          text(fill: rgb("#bbbbbb"), size: 7pt, weight: "bold", tracking: 0.5pt)[COMPILE])
        #place(left + bottom, dx: 330pt, dy: -8pt,
          text(fill: rgb("#dddddd"), size: 7pt, weight: "bold", tracking: 0.5pt)[RUNTIME])

        // Direction arrows
        #place(left + bottom, dx: 60pt, dy: -10pt,
          text(fill: rgb("#cccccc"), size: 9pt)[→])
        #place(left + bottom, dx: 210pt, dy: -10pt,
          text(fill: rgb("#dddddd"), size: 9pt)[→])
      ]
    ]

    // Title
    #place(top + left, dx: 48pt, dy: 420pt)[
      #text(fill: cover-title, size: 52pt, weight: "bold")[#title]
    ]

    // Subtitle
    #place(top + left, dx: 48pt, dy: 488pt)[
      #text(fill: cover-sub, size: 22pt, style: "italic")[#subtitle]
    ]

    // Tagline / description
    #if tagline != none {
      place(top + left, dx: 48pt, dy: 530pt)[
        #box(width: 420pt)[
          #text(fill: cover-body, size: 12pt)[#tagline]
        ]
      ]
    }

    // Bottom separator band
    #place(bottom + left, dy: -120pt,
      rect(width: 100%, height: 4pt, fill: svelte-orange),
    )

    // Topics
    #place(bottom + left, dx: 48pt, dy: -70pt)[
      #text(fill: svelte-orange, size: 11pt, weight: "bold")[#topics]
    ]

    // Meta info
    #if meta != none {
      place(bottom + left, dx: 48pt, dy: -46pt)[
        #text(fill: cover-meta, size: 9pt)[#meta]
      ]
    }

    // Publisher badge
    #place(bottom + right, dx: -48pt, dy: -50pt)[
      #box(
        stroke: 1.5pt + svelte-orange,
        radius: 2pt,
        inset: (x: 10pt, y: 6pt),
      )[
        #text(fill: svelte-orange, size: 9pt, weight: "bold", tracking: 0.5pt)[#publisher]
      ]
    ]

    // Bottom band
    #place(bottom + left, rect(width: 100%, height: 12pt, fill: svelte-orange))
  ]
}

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

  // -- Raw/code block (fenced) --
  show raw.where(block: true): it => {
    block(
      width: 100%,
      fill: code-bg,
      stroke: 0.5pt + code-border,
      inset: 8pt,
      radius: 2pt,
      below: 8pt,
      text(font: "Fira Code", size: 9pt, it),
    )
  }

  // -- Document metadata --
  set document(title: title, author: author)

  body
}
