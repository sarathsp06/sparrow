#!/usr/bin/env python3
"""Generate the enriched Svelte 5 Tutorial PDF (16 lessons + reference card).

Each feature lesson teaches both the API and the internals together:
how to use $state, AND what signal object the compiler creates.
"""

from reportlab.lib.pagesizes import letter
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.colors import HexColor, black, white, Color
from reportlab.lib.units import inch
from reportlab.lib.enums import TA_CENTER, TA_LEFT, TA_JUSTIFY
from reportlab.platypus import (
    SimpleDocTemplate, Paragraph, Spacer, PageBreak, Table, TableStyle,
    KeepTogether, HRFlowable, Preformatted
)
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
import os
import math

# ---------------------------------------------------------------------------
# Font registration -- Inter (body) + Fira Code (code)
# ---------------------------------------------------------------------------
_font_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fonts")

# Inter: body text, headings, UI text
pdfmetrics.registerFont(TTFont("Inter", os.path.join(_font_dir, "Inter-Regular.ttf")))
pdfmetrics.registerFont(TTFont("Inter-Bold", os.path.join(_font_dir, "Inter-Bold.ttf")))
pdfmetrics.registerFont(TTFont("Inter-Italic", os.path.join(_font_dir, "Inter-Italic.ttf")))
pdfmetrics.registerFont(TTFont("Inter-BoldItalic", os.path.join(_font_dir, "Inter-BoldItalic.ttf")))
pdfmetrics.registerFontFamily(
    "Inter",
    normal="Inter", bold="Inter-Bold",
    italic="Inter-Italic", boldItalic="Inter-BoldItalic",
)

# Fira Code: code blocks, inline code, monospace
pdfmetrics.registerFont(TTFont("FiraCode", os.path.join(_font_dir, "FiraCode-Regular.ttf")))
pdfmetrics.registerFont(TTFont("FiraCode-Bold", os.path.join(_font_dir, "FiraCode-Bold.ttf")))
pdfmetrics.registerFontFamily(
    "FiraCode",
    normal="FiraCode", bold="FiraCode-Bold",
    italic="FiraCode", boldItalic="FiraCode-Bold",  # Fira Code has no italic; reuse regular/bold
)

# ---------------------------------------------------------------------------
# Document setup
# ---------------------------------------------------------------------------
SPARROW_VERSION = "v1.3.1"
output_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "svelte5-tutorial.pdf")

doc = SimpleDocTemplate(
    output_path,
    pagesize=letter,
    topMargin=0.6 * inch,
    bottomMargin=0.6 * inch,
    leftMargin=0.75 * inch,
    rightMargin=0.75 * inch,
    title="Svelte 5 Tutorial - Sparrow Codebase Examples",
    author="Sparrow Project",
)

# ---------------------------------------------------------------------------
# Styles
# ---------------------------------------------------------------------------
styles = getSampleStyleSheet()

# Colors
PRIMARY = HexColor("#1a1a2e")
SVELTE_ORANGE = HexColor("#ff3e00")  # Svelte's brand color
ACCENT = SVELTE_ORANGE
CODE_BG = HexColor("#f8f9fa")
CODE_BORDER = HexColor("#dee2e6")
BLUE = HexColor("#2563eb")
GRAY_600 = HexColor("#4b5563")
GRAY_400 = HexColor("#9ca3af")
GREEN = HexColor("#059669")
ORANGE = HexColor("#d97706")
DARK_BG = HexColor("#0f172a")
SLATE_700 = HexColor("#334155")

# Title page styles
styles.add(ParagraphStyle(
    "CoverTitle", parent=styles["Title"],
    fontSize=42, leading=48, textColor=PRIMARY,
    spaceAfter=6, alignment=TA_LEFT, fontName="Inter-Bold",
))
styles.add(ParagraphStyle(
    "CoverSubtitle", parent=styles["Normal"],
    fontSize=14, leading=20, textColor=GRAY_600,
    spaceAfter=4, fontName="Inter",
))
styles.add(ParagraphStyle(
    "CoverProject", parent=styles["Normal"],
    fontSize=12, leading=16, textColor=ACCENT,
    spaceAfter=2, fontName="Inter-Bold",
))
styles.add(ParagraphStyle(
    "CoverTagline", parent=styles["Normal"],
    fontSize=11, leading=16, textColor=SLATE_700,
    spaceAfter=8, fontName="Inter-Italic",
))
styles.add(ParagraphStyle(
    "CoverSmall", parent=styles["Normal"],
    fontSize=9.5, leading=14, textColor=GRAY_600,
    spaceAfter=3, fontName="Inter",
))

# Lesson title
styles.add(ParagraphStyle(
    "LessonTag", parent=styles["Normal"],
    fontSize=11, leading=14, textColor=ACCENT,
    spaceBefore=0, spaceAfter=2, fontName="Inter-Bold",
))
styles.add(ParagraphStyle(
    "LessonTitle", parent=styles["Heading1"],
    fontSize=22, leading=26, textColor=PRIMARY,
    spaceBefore=0, spaceAfter=12, fontName="Inter-Bold",
))

# Section heading
styles.add(ParagraphStyle(
    "SectionHead", parent=styles["Heading2"],
    fontSize=14, leading=18, textColor=PRIMARY,
    spaceBefore=16, spaceAfter=6, fontName="Inter-Bold",
))

# Body text
styles.add(ParagraphStyle(
    "Body", parent=styles["Normal"],
    fontSize=10.5, leading=15, textColor=black,
    spaceAfter=8, fontName="Inter", alignment=TA_JUSTIFY,
))

# Source reference
styles.add(ParagraphStyle(
    "Source", parent=styles["Normal"],
    fontSize=9, leading=12, textColor=BLUE,
    spaceBefore=8, spaceAfter=4, fontName="Inter-Italic",
))

# Code block
styles.add(ParagraphStyle(
    "CodeBlock", parent=styles["Code"],
    fontSize=9, leading=13, textColor=black,
    spaceBefore=4, spaceAfter=8, fontName="FiraCode",
    backColor=CODE_BG, borderColor=CODE_BORDER,
    borderWidth=0.5, borderPadding=8,
    leftIndent=0, rightIndent=0,
))

# Bullet explanation
styles.add(ParagraphStyle(
    "BulletItem", parent=styles["Normal"],
    fontSize=10, leading=14, textColor=black,
    spaceBefore=2, spaceAfter=4, fontName="Inter",
    leftIndent=18, bulletIndent=6,
))

# Gotcha heading
styles.add(ParagraphStyle(
    "GotchaHead", parent=styles["Normal"],
    fontSize=10.5, leading=14, textColor=ORANGE,
    spaceBefore=10, spaceAfter=2, fontName="Inter-Bold",
))

# Gotcha body
styles.add(ParagraphStyle(
    "Gotcha", parent=styles["Normal"],
    fontSize=10, leading=14, textColor=GRAY_600,
    spaceAfter=6, fontName="Inter",
    leftIndent=0,
))

# Page header/footer
styles.add(ParagraphStyle(
    "PageHeader", parent=styles["Normal"],
    fontSize=8, leading=10, textColor=GRAY_400,
    fontName="Inter",
))

# TOC styles
styles.add(ParagraphStyle(
    "TOCEntry", parent=styles["Normal"],
    fontSize=11, leading=18, textColor=PRIMARY,
    fontName="Inter", leftIndent=20,
))

# Reference table styles
styles.add(ParagraphStyle(
    "RefCell", parent=styles["Normal"],
    fontSize=9, leading=12, textColor=black,
    fontName="Inter",
))
styles.add(ParagraphStyle(
    "RefCellCode", parent=styles["Normal"],
    fontSize=8.5, leading=11, textColor=black,
    fontName="FiraCode",
))
styles.add(ParagraphStyle(
    "RefCellBold", parent=styles["Normal"],
    fontSize=9, leading=12, textColor=PRIMARY,
    fontName="Inter-Bold",
))

# ---------------------------------------------------------------------------
# Helper functions
# ---------------------------------------------------------------------------
def code_block(code_text):
    """Return a code block paragraph with monospace font.
    
    Preformatted renders text literally (no XML parsing), so we must NOT
    escape angle brackets — they should appear as-is in the PDF.
    """
    return Preformatted(code_text, styles["CodeBlock"])

def source_ref(text):
    return Paragraph(f"Source: {text}", styles["Source"])

def body(text):
    return Paragraph(text, styles["Body"])

def bullet(text):
    return Paragraph(f"<bullet>&bull;</bullet> {text}", styles["BulletItem"])

def gotcha(num, title, text):
    return [
        Paragraph(f"Gotcha {num}: {title}", styles["GotchaHead"]),
        Paragraph(text, styles["Gotcha"]),
    ]

def section(title):
    return Paragraph(title, styles["SectionHead"])

def lesson_header(num, title, anchor=None):
    tag = f"LESSON {num}" if num else ""
    title_text = title
    if anchor:
        title_text = f'<a name="{anchor}"/>{title}'
    return [
        Paragraph(tag, styles["LessonTag"]),
        Paragraph(title_text, styles["LessonTitle"]),
    ]

def hr():
    return HRFlowable(width="100%", thickness=0.5, color=CODE_BORDER, spaceBefore=8, spaceAfter=8)

# ---------------------------------------------------------------------------
# Page numbering and cover art
# ---------------------------------------------------------------------------
_PAGE_W, _PAGE_H = letter

def _draw_cover(canvas_obj, doc_obj):
    """Draw a high-fidelity, creative Svelte-themed cover inspired by modern editorial design."""
    c = canvas_obj
    c.saveState()
    w, h = _PAGE_W, _PAGE_H

    # --- Background: Deep dark base ---
    bg_color = HexColor("#131313")
    c.setFillColor(bg_color)
    c.rect(0, 0, w, h, fill=1, stroke=0)

    # --- Subtle Technical Grid (Dots) ---
    c.setFillColor(Color(0.2, 0.2, 0.2, 0.15))
    grid_size = 40
    for x in range(0, int(w) + 1, grid_size):
        for y in range(0, int(h) + 1, grid_size):
            c.circle(x, y, 1, fill=1, stroke=0)

    # --- Radial Atmosphere Glows ---
    # Top-right orange glow
    for i in range(1, 11):
        r = 150 + i * 20
        alpha = 0.08 - i * 0.007
        if alpha > 0:
            c.setFillColor(Color(1, 0.24, 0, alpha))
            c.circle(w * 0.9, h * 0.85, r, fill=1, stroke=0)
    
    # Bottom-left blue/teal glow
    for i in range(1, 8):
        r = 120 + i * 15
        alpha = 0.05 - i * 0.006
        if alpha > 0:
            c.setFillColor(Color(0.2, 0.57, 1, alpha))
            c.circle(w * 0.1, h * 0.1, r, fill=1, stroke=0)

    # --- HERO SECTION (Editorial Left Column) ---
    left_x = 0.75 * inch
    
    # Version Pill Badge
    badge_w, badge_h = 130, 18
    badge_y = h * 0.75
    c.setFillColor(HexColor("#2a2a2a"))
    c.setStrokeColor(Color(1, 1, 1, 0.1))
    c.setLineWidth(0.5)
    c.roundRect(left_x, badge_y, badge_w, badge_h, 9, fill=1, stroke=1)
    
    # Pulsing dot inside pill
    c.setFillColor(SVELTE_ORANGE)
    c.circle(left_x + 10, badge_y + 9, 3, fill=1, stroke=0)
    
    c.setFillColor(Color(1, 1, 1, 0.6))
    c.setFont("Inter-Bold", 8)
    c.drawString(left_x + 20, badge_y + 6.5, f"RENDER {SPARROW_VERSION.upper()}-STABLE")

    # TITLE BLOCK
    title_y = badge_y - 80
    c.setFillColor(white)
    c.setFont("Inter-Bold", 86)
    c.drawString(left_x, title_y, "Svelte 5")
    
    c.setFillColor(SVELTE_ORANGE)
    c.setFont("Inter-Bold", 68)
    c.drawString(left_x, title_y - 62, "Tutorial")

    # SUBTITLE
    subtitle_y = title_y - 120
    c.setFillColor(white)
    c.setFont("Inter-Bold", 32)
    c.drawString(left_x, subtitle_y, "The Disappearing")
    c.drawString(left_x, subtitle_y - 36, "Framework")

    # Accent bar
    c.setFillColor(SVELTE_ORANGE)
    c.rect(left_x, subtitle_y - 62, 100, 4, fill=1, stroke=0)

    # TAGLINE (Wrapped)
    c.setFillColor(Color(1, 1, 1, 0.8))
    c.setFont("Inter", 17)
    tag_y = subtitle_y - 95
    c.drawString(left_x, tag_y, "A backend engineer\u2019s guide to")
    c.setFont("Inter-Bold", 17)
    c.setFillColor(SVELTE_ORANGE)
    c.drawString(left_x + c.stringWidth("A backend engineer\u2019s guide to ", "Inter", 17), tag_y, "high-performance")
    c.drawString(left_x, tag_y - 22, "reactivity")
    c.setFont("Inter", 17)
    c.setFillColor(Color(1, 1, 1, 0.8))
    c.drawString(left_x + c.stringWidth("reactivity", "Inter-Bold", 17) + 6, tag_y - 22, "and compiler-driven UI.")

    # --- VISUAL COMPONENT (The "Geometric Compilation Loom") ---
    loom_w, loom_h = w * 0.4, h * 0.45
    loom_x, loom_y = w - loom_w - 0.5 * inch, h * 0.35
    
    # Focal point (The Compiler Core)
    cx, cy = loom_x + loom_w * 0.45, loom_y + loom_h * 0.55
    
    # 1. INPUT SHARDS (Left side - Fragmented Declarative Source)
    # Shard positions (relative to loom_x/y)
    shards = [
        # (points, color, opacity)
        ([(0, 40), (40, 0), (60, 80)], HexColor("#fb7185"), 0.3),  # HTML-ish
        ([(10, 100), (50, 140), (20, 180)], HexColor("#38bdf8"), 0.25), # CSS-ish
        ([(5, 220), (45, 260), (15, 300)], SVELTE_ORANGE, 0.4), # JS-ish
    ]
    
    for pts, color, alpha in shards:
        p = c.beginPath()
        # Offset shards to the left of the core
        ox, oy = loom_x, loom_y + 40
        p.moveTo(ox + pts[0][0], oy + pts[0][1])
        for px, py in pts[1:]:
            p.lineTo(ox + px, oy + py)
        p.close()
        c.setFillColor(Color(color.red, color.green, color.blue, alpha))
        c.drawPath(p, fill=1, stroke=0)
        
        # Convergence lines from shard vertices to Core
        c.setStrokeColor(Color(1, 1, 1, 0.1))
        c.setLineWidth(0.5)
        for px, py in pts:
            c.line(ox + px, oy + py, cx, cy)

    # 2. THE COMPILER CORE (Geometric Prism)
    core_r = 45
    # Outer core glow
    for i in range(1, 6):
        c.setFillColor(Color(1, 0.24, 0, 0.1 / i))
        c.circle(cx, cy, core_r + i * 8, fill=1, stroke=0)
    
    # The Prism (Hexagon)
    c.setFillColor(HexColor("#1e293b"))
    c.setStrokeColor(SVELTE_ORANGE)
    c.setLineWidth(2)
    hex_pts = []
    for i in range(6):
        angle = math.radians(i * 60 - 30)
        hex_pts.append((cx + core_r * math.cos(angle), cy + core_r * math.sin(angle)))
    
    hp = c.beginPath()
    hp.moveTo(hex_pts[0][0], hex_pts[0][1])
    for hx, hy in hex_pts[1:]:
        hp.lineTo(hx, hy)
    hp.close()
    c.drawPath(hp, fill=1, stroke=1)
    
    # Internal "S" logic - Abstract geometric "S" inside core
    c.setStrokeColor(white)
    c.setLineWidth(3)
    c.setLineCap(1)
    sp = c.beginPath()
    sp.moveTo(cx - 15, cy + 20)
    sp.curveTo(cx + 25, cy + 30, cx + 25, cy, cx, cy)
    sp.curveTo(cx - 25, cy, cx - 25, cy - 30, cx + 15, cy - 20)
    c.drawPath(sp, fill=0, stroke=1)

    # 3. OUTPUT BEAMS (Right side - Precise Compiled Signals)
    # Beams exiting the core to the right
    beam_y_offsets = [-20, 0, 20]
    for i, oy in enumerate(beam_y_offsets):
        by = cy + oy
        # The main beam line
        c.setStrokeColor(SVELTE_ORANGE)
        c.setLineWidth(2.5 - i * 0.5)
        c.line(cx + core_r * 0.8, by, loom_x + loom_w, by)
        
        # Energy trail (faint wide stroke)
        c.setStrokeColor(Color(1, 0.24, 0, 0.1))
        c.setLineWidth(8)
        c.line(cx + core_r * 0.8, by, loom_x + loom_w, by)
        
        # DOM Node (Terminal point)
        c.setFillColor(SVELTE_ORANGE)
        c.circle(loom_x + loom_w, by, 4 - i, fill=1, stroke=0)
        # Node glow
        c.setFillColor(Color(1, 0.24, 0, 0.2))
        c.circle(loom_x + loom_w, by, 8 - i, fill=1, stroke=0)

    # Labels for the transformation
    c.setFont("Inter-Bold", 7)
    c.setFillColor(Color(1, 1, 1, 0.3))
    c.drawCentredString(loom_x + 30, loom_y + 20, "DECLARATIVE")
    c.drawCentredString(loom_x + loom_w - 30, loom_y + 20, "COMPILED")

    # Floating "RUNES READY" badge (positioned relative to Loom)
    badge_cx, badge_cy = loom_x + loom_w - 10, loom_y + loom_h - 10
    c.saveState()
    c.translate(badge_cx, badge_cy)
    c.rotate(12)
    c.setFillColor(SVELTE_ORANGE)
    c.circle(0, 0, 38, fill=1, stroke=0)
    c.setFillColor(HexColor("#611200"))
    c.setFont("Inter-Bold", 12)
    c.drawCentredString(0, 5, "RUNES")
    c.setFont("Inter-Bold", 6)
    c.drawCentredString(0, -8, "STABLE \u00b7 READY")
    c.restoreState()

    # Code Snippet Overlay (Balanced bottom-right)
    snip_w, snip_h = 180, 50
    snip_x, snip_y = w - snip_w - 0.75 * inch, h * 0.18
    c.setFillColor(HexColor("#0e0e0e"))
    c.setStrokeColor(Color(1, 1, 1, 0.1))
    c.setLineWidth(1)
    c.roundRect(snip_x, snip_y, snip_w, snip_h, 4, fill=1, stroke=1)
    
    c.setFont("FiraCode", 7.5)
    code_y = snip_y + 10
    code_lines = [
        ("$effect", "(() => console.log(double));"),
        ("const double = ", "$derived", "(count * 2);"),
        ("let count = ", "$state", "(0);"),
    ]
    for i, line in enumerate(code_lines):
        x = snip_x + 10
        y = code_y + i * 11
        if len(line) == 2:
            c.setFillColor(SVELTE_ORANGE)
            c.drawString(x, y, line[0])
            c.setFillColor(Color(1, 1, 1, 0.4))
            c.drawString(x + c.stringWidth(line[0], "FiraCode", 7.5), y, line[1])
        else:
            c.setFillColor(Color(1, 1, 1, 0.4))
            c.drawString(x, y, line[0])
            c.setFillColor(SVELTE_ORANGE)
            c.drawString(x + c.stringWidth(line[0], "FiraCode", 7.5), y, line[1])
            c.setFillColor(Color(1, 1, 1, 0.4))
            c.drawString(x + c.stringWidth(line[0]+line[1], "FiraCode", 7.5), y, line[2])

    # --- FOOTER ---
    footer_y = 0.5 * inch
    c.setFillColor(Color(1, 1, 1, 0.2))
    c.setFont("Inter-Bold", 10)
    c.drawString(0.75 * inch, footer_y, "SVELTE 5 REIFIED")
    
    c.drawRightString(w - 0.75 * inch, footer_y, "16 LESSONS \u00b7 BUILT BY & FOR BACKEND ENGINEERS")
    
    c.restoreState()


def _page_footer(canvas_obj, doc_obj):
    """Footer for content pages (not cover)."""
    canvas_obj.saveState()
    canvas_obj.setFont("Inter", 8)
    canvas_obj.setFillColor(GRAY_400)
    canvas_obj.drawString(0.75 * inch, 0.4 * inch, "Svelte 5 Tutorial")
    canvas_obj.drawRightString(_PAGE_W - 0.75 * inch, 0.4 * inch, "Sparrow Codebase Examples")
    canvas_obj.drawCentredString(_PAGE_W / 2, 0.4 * inch, str(doc_obj.page))
    canvas_obj.restoreState()

# ---------------------------------------------------------------------------
# Build story
# ---------------------------------------------------------------------------
story = []

# ===== COVER PAGE =====
# Cover art is drawn via canvas callback (_draw_cover).
# We just need an empty page that triggers the callback, then break to page 2.
story.append(Spacer(1, 1))  # minimal flowable to trigger page creation

# ===== TABLE OF CONTENTS =====
story.append(PageBreak())
story.append(Paragraph("Introduction", styles["LessonTitle"]))
story.append(body(
    "This tutorial is an attempt to learn and teach Svelte 5 through the lens of a real-world project. "
    "All examples are drawn from <b>Sparrow</b>, an open-source webhook delivery platform built using "
    "an AI-assisted development workflow."
))
story.append(body(
    "By exploring how Svelte 5 is used to build Sparrow's management UI, you'll see how runes like "
    "$state and $derived solve actual engineering problems—from handling complex delivery filters "
    "to visualizing real-time batch progress."
))
story.append(body(
    f"The Sparrow source code is available on GitHub: "
    f'<font color="{BLUE.hexval()}"><u>https://github.com/sarathsp06/sparrow</u></font>'
))
story.append(Spacer(1, 0.2 * inch))
story.append(hr())
story.append(Spacer(1, 0.2 * inch))

story.append(Paragraph("Table of Contents", styles["LessonTitle"]))

toc_full = [
    ("Lesson 1", "Reactive State with $state()", "Signals, proxies, and what the compiler actually emits"),
    ("Lesson 2", "Computed Values with $derived()", "Lazy evaluation, dirty propagation, derived signals"),
    ("Lesson 3", "Components &amp; Props with $props()", "Props as signal reads, compiled output"),
    ("Lesson 4", "Conditional Rendering with {#if}", "Showing and hiding content"),
    ("Lesson 5", "Rendering Lists with {#each}", "Repeating content for arrays"),
    ("Lesson 6", "Inline Constants with {@const}", "Local variables in templates"),
    ("Lesson 7", "Event Handling", "Responding to user interactions"),
    ("Lesson 8", "Side Effects with $effect()", "Dependency tracking, active_reaction, cleanup"),
    ("Lesson 9", "Snippets &amp; Render", "Reusable template blocks (replaces slots)"),
    ("Lesson 10", "Two-Way Binding", "Syncing inputs and parent-child state"),
    ("Lesson 11", "Lifecycle Hooks", "Setup on mount, cleanup on destroy"),
    ("Lesson 12", "Async Data Fetching", "Loading data with Connect-RPC"),
    ("Lesson 13", "Layout &amp; Navigation", "Shared UI shell and SvelteKit routing"),
    ("Lesson 14", "Component Composition", "Designing reusable component APIs"),
    ("Lesson 15", "The Build Pipeline", "Vite + SvelteKit + adapter-static + go:embed + Docker"),
    ("Lesson 16", "The Full Stack", "Proto to binary -- the complete typed architecture"),
    ("Reference", "Quick Reference Card &amp; TypeScript Cheat Sheet", ""),
]

for tag, title, desc in toc_full:
    anchor = tag.replace(" ", "").lower()
    line = f'<a href="#{anchor}"><b>{tag}</b> &nbsp; {title}</a>'
    if desc:
        line += f'<br/><font size="9" color="{GRAY_600.hexval()}">&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;{desc}</font>'
    story.append(Paragraph(line, styles["TOCEntry"]))

story.append(Spacer(1, 0.3 * inch))
story.append(body(
    '<b>Audience:</b> Backend engineers who know basic JavaScript/TypeScript and have '
    'the Sparrow repo checked out locally. No prior React, Vue, or Svelte experience required.'
))
story.append(body(
    '<b>Approach:</b> Each lesson teaches a Svelte feature with real Sparrow code, then '
    'shows what the compiler does with it under the hood. You learn the API and the '
    'internals together -- not in separate sections. Conceptual compiled-output '
    'illustrations are clearly labeled as such.'
))
story.append(body('<b>Source Code:</b> https://github.com/sarathsp06/sparrow'))


# ============================================================================
# LESSON 1: $state() -- with compiler + signal + proxy internals
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(1, "Reactive State with $state()", anchor="lesson1"))

story.append(section("The Problem"))
story.append(body(
    "In a web app, data changes constantly: a user clicks a button, an API returns "
    "results, a timer fires. The UI needs to update to reflect these changes. In "
    "vanilla JavaScript, you'd manually find DOM elements and update their text "
    "content. Svelte automates this: you declare a variable as reactive, change it, "
    "and the UI updates itself."
))

story.append(section("The Svelte 5 Solution: $state()"))
story.append(body(
    "The <font face='Courier'>$state()</font> rune tells the Svelte compiler to track a variable. When that "
    "variable's value changes (via plain assignment), every part of the UI that reads "
    "it re-renders automatically."
))
story.append(source_ref("health/+page.svelte, lines 15-21"))
story.append(code_block(
    'let loading = $state(true);\n'
    'let error = $state(\'\');\n'
    'let healthSummary = $state<HealthSummary | undefined>();\n'
    'let unhealthyWebhooks = $state<RegisteredWebhook[]>([]);\n'
    'let degradedWebhooks = $state<RegisteredWebhook[]>([]);\n'
    'let webhookMetrics = $state(new Map<string, HealthMetrics>());\n'
    'let namespaceStats = $state<NamespaceStats[]>([]);'
))

story.append(section("Line by Line"))
story.append(bullet(
    '<font face="Courier">let loading = $state(true)</font> -- <font face="Courier">$state(true)</font> creates a reactive variable initialized to '
    '<font face="Courier">true</font>. While data is loading, the UI shows a skeleton placeholder. When loading '
    'completes, <font face="Courier">loading = false</font> triggers the UI to swap to real content.'
))
story.append(bullet(
    '<font face="Courier">let error = $state(\'\')</font> -- An empty string means no error. If a fetch fails, <font face="Courier">error = '
    '\'Something went wrong\'</font> makes an error banner appear.'
))
story.append(bullet(
    '<font face="Courier">$state&lt;HealthSummary | undefined&gt;()</font> -- The angle brackets are a generic type parameter '
    '(TypeScript). <font face="Courier">HealthSummary | undefined</font> is a union type: the variable is either a '
    'HealthSummary object or undefined. Empty parentheses <font face="Courier">()</font> mean no initial value = undefined.'
))
story.append(bullet(
    '<font face="Courier">$state&lt;RegisteredWebhook[]&gt;([])</font> -- <font face="Courier">RegisteredWebhook[]</font> means "an array of '
    'RegisteredWebhook objects". The <font face="Courier">[]</font> inside parens is the initial value: empty array.'
))
story.append(bullet(
    '<font face="Courier">$state(new Map&lt;string, HealthMetrics&gt;())</font> -- A Map is like a dictionary. <font face="Courier">Map&lt;string, '
    'HealthMetrics&gt;</font> = keys are strings, values are HealthMetrics objects. Initialized empty.'
))

story.append(section("How Reactivity Triggers"))
story.append(body("To update a <font face='Courier'>$state</font> variable, use plain assignment:"))
story.append(code_block(
    '// This triggers a UI update:\n'
    'loading = false;\n\n'
    '// Replacing the whole array:\n'
    'unhealthyWebhooks = result.webhooks;\n\n'
    '// For Maps, reassign the whole Map:\n'
    'webhookMetrics = newMap;'
))
story.append(body(
    "The key insight: <font face='Courier'>$state()</font> is a compiler directive, not a regular function. At "
    "runtime, the variable holds the plain value (a boolean, a string, an array), not "
    "a wrapper object. Svelte's compiler rewrites your code during build to inject "
    "change tracking behind the scenes."
))

story.append(section("Under the Hood: Svelte Is a Compiler"))
story.append(body(
    "This is the most important thing to understand about Svelte: it is not a runtime "
    "library. React, Vue, and Angular ship a runtime that interprets your components "
    "in the browser. Svelte is a <b>compiler</b> that runs at build time and emits "
    "optimized JavaScript -- like the difference between CPython (interpreter) and Go "
    "(ahead-of-time compiler). The heavy lifting happens once, so the browser does less."
))
story.append(body(
    "The Svelte compiler processes each <font face='Courier'>.svelte</font> file through "
    "three phases, similar to <font face='Courier'>go build</font> "
    "(parse &rarr; type-check &rarr; codegen):"
))
story.append(code_block(
    "// sveltejs/svelte — packages/svelte/src/compiler/index.js\n"
    "export function compile(source, options) {\n"
    "  // Phase 1: PARSE — source text -> AST      (like go/parser)\n"
    "  const parsed = _parse(source);\n"
    "\n"
    "  // Phase 2: ANALYZE — resolve bindings       (like go/types)\n"
    "  const analysis = analyze_component(\n"
    "    parsed, source, combined_options\n"
    "  );\n"
    "\n"
    "  // Phase 3: TRANSFORM — AST -> optimized JS  (codegen)\n"
    "  return transform_component(\n"
    "    analysis, source, combined_options\n"
    "  );\n"
    "}"
))
story.append(source_ref("sveltejs/svelte -- packages/svelte/src/compiler/index.js"))

story.append(section("What $state Actually Compiles To"))
story.append(body(
    "When the compiler sees <font face='Courier'>$state()</font>, it replaces it with a "
    "<b>signal</b> -- a small reactive container object. The rune syntax is just "
    "syntactic sugar that disappears at build time:"
))
story.append(code_block(
    "// What you write:\n"
    "let count = $state(0);\n"
    "\n"
    "// What the compiler emits (conceptual illustration):\n"
    "import * as $ from 'svelte/internal/client';\n"
    "let count = $.state(0);   // creates a signal object\n"
    "\n"
    "// Template: <p>{count}</p>\n"
    "// Compiles to:\n"
    "var p = $.template('<p> </p>');\n"
    "var text = p.firstChild;\n"
    "$.render_effect(() => {\n"
    "  $.set_text(text, $.get(count));  // direct DOM update\n"
    "});\n"
    "\n"
    "// No virtual DOM tree. No diffing algorithm.\n"
    "// One render_effect per dynamic expression.\n"
    "// Cost = O(changed signals), not O(template size)."
))
story.append(body(
    "Contrast this with React, where the entire component function re-executes on "
    "every state change, returns a new virtual DOM tree, and React diffs the old and "
    "new trees to find what changed. Svelte skips all of that -- the compiler already "
    "knows which DOM node to update."
))

story.append(section("The Signal Object"))
story.append(body(
    "Under the hood, <font face='Courier'>$.state(0)</font> creates a signal -- a plain "
    "JavaScript object with a value, a version counter, and a list of who depends on it. "
    "If you've worked with the observer pattern (event emitters, pub/sub), this will "
    "feel familiar:"
))
story.append(code_block(
    "// sveltejs/svelte — internal/client/reactivity/sources.js\n"
    "export function source(v) {\n"
    "  return {\n"
    "    f: 0,              // flags (DIRTY, CLEAN, etc.)\n"
    "    v,                 // current value\n"
    "    reactions: null,   // effects/deriveds that read this\n"
    "    equals,            // equality check function\n"
    "    rv: 0,             // read version\n"
    "    wv: 0              // write version\n"
    "  };\n"
    "}"
))
story.append(source_ref("sveltejs/svelte -- packages/svelte/src/internal/client/reactivity/sources.js"))
story.append(body(
    "When you assign to a <font face='Courier'>$state</font> variable, the compiler emits "
    "<font face='Courier'>$.set(signal, newValue)</font>. This checks equality, updates the "
    "value, increments the write version, and walks the signal's reaction list marking "
    "each dependent as <b>DIRTY</b> -- like a database trigger notifying all subscribed "
    "queries that a column changed:"
))
story.append(code_block(
    "// sources.js -- set() (simplified)\n"
    "export function set(source, value) {\n"
    "  if (source.equals(value, source.v)) return;\n"
    "\n"
    "  source.v = value;              // update value\n"
    "  source.wv++;                   // increment write version\n"
    "  mark_reactions(source, DIRTY); // notify all dependents\n"
    "  schedule_effects();            // batch DOM updates\n"
    "}"
))

story.append(section("Deep Reactivity: The Proxy System"))
story.append(body(
    "When <font face='Courier'>$state</font> wraps an object or array, Svelte creates an "
    "ES6 Proxy with a <b>per-property signal Map</b>. Each property access creates (or "
    "reads) a dedicated signal. Changing <font face='Courier'>obj.name</font> only "
    "re-renders expressions that read <font face='Courier'>obj.name</font>, not those "
    "that read <font face='Courier'>obj.age</font>:"
))
story.append(code_block(
    "// proxy.js (simplified)\n"
    "export function proxy(value) {\n"
    "  const sources = new Map();  // property -> signal\n"
    "\n"
    "  return new Proxy(value, {\n"
    "    get(target, prop) {\n"
    "      let s = sources.get(prop);\n"
    "      if (!s) {\n"
    "        s = source(target[prop]);\n"
    "        sources.set(prop, s);\n"
    "      }\n"
    "      return get(s);  // registers dependency\n"
    "    },\n"
    "    set(target, prop, val) {\n"
    "      let s = sources.get(prop);\n"
    "      set(s, proxy(val));  // recursively proxy nested\n"
    "      return true;\n"
    "    }\n"
    "  });\n"
    "}"
))

story.append(section("$state.raw: Opting Out of Deep Tracking"))
story.append(body(
    "The proxy system has overhead: every property access goes through a trap, and each "
    "property gets its own signal. For large arrays or data you never mutate in place "
    "(like API responses you display read-only), use <font face='Courier'>$state.raw</font> "
    "to skip deep tracking. The variable itself is still reactive -- swapping the whole "
    "value triggers updates -- but individual properties are not tracked."
))
story.append(source_ref("web/src/routes/deliveries/+page.svelte, lines 5, 15, 72-87"))
story.append(code_block(
    "// Current code (deliveries/+page.svelte, line 15):\n"
    "let deliveries: WebhookDelivery[] = $state([]);\n"
    "\n"
    "// The fetch replaces the entire array (lines 77-79):\n"
    "const res = await deliveryClient\n"
    "  .listDeliveries(req);\n"
    "deliveries = res.deliveries || [];\n"
    "\n"
    "// $state([]) deep-proxies every WebhookDelivery object.\n"
    "// But we never mutate deliveries[0].status in place --\n"
    "// we always replace the whole array. Wasted overhead.\n"
    "\n"
    "// Optimized alternative:\n"
    "let deliveries = $state.raw<WebhookDelivery[]>([]);\n"
    "// No per-property signals. Reassigning the array still\n"
    "// triggers updates. Cheaper for read-only display data."
))

story.append(section("Performance Tiers"))
story.append(body("Think of reactive state in three tiers, cheapest to most expensive:"))
story.append(bullet('<b>Tier 1 -- Primitive $state</b>: '
    '<font face="Courier">$state(0)</font>, <font face="Courier">$state("")</font>. '
    'A single signal. Cheapest possible.'))
story.append(bullet('<b>Tier 2 -- $state.raw(obj)</b>: One signal for the reference. '
    'No per-property tracking. Use for API responses and large read-only data.'))
story.append(bullet('<b>Tier 3 -- $state(obj)</b>: Proxy + per-property signals. '
    'Full deep reactivity. Use when you mutate individual properties.'))

story.extend(gotcha(1, "Mutating arrays",
    "If you push to an array with <font face='Courier'>.push()</font>, Svelte 5 does detect the mutation (unlike "
    "Svelte 4 which required reassignment). However, reassignment is still the "
    "clearest pattern: <font face='Courier'>items = [...items, newItem]</font>."
))
story.extend(gotcha(2, "$state() is not a regular function",
    "You cannot do <font face='Courier'>const x = condition ? $state(1) : $state(2)</font>. The compiler must see "
    "<font face='Courier'>$state()</font> as a direct initializer in a <font face='Courier'>let</font> declaration at the top level of a "
    "component's script block."
))


# ============================================================================
# LESSON 2: $derived() -- with derived signal internals
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(2, "Computed Values with $derived()", anchor="lesson2"))

story.append(section("The Problem"))
story.append(body(
    "You have some state, and you need a value that's calculated from that state. For "
    "example: you have an array of webhooks and need a filtered list showing only the "
    "active ones. You want to declare the computation once and have it automatically stay in sync."
))

story.append(section("Simple Form: $derived(expression)"))
story.append(source_ref("CopyableId.svelte, lines 11-13"))
story.append(code_block(
    "const display = $derived(\n"
    "  truncate > 0 && id.length > truncate\n"
    "    ? id.substring(0, truncate) + '...'\n"
    "    : id\n"
    ");"
))
story.append(body(
    "This creates a <font face='Courier'>display</font> value that automatically recomputes whenever <font face='Courier'>truncate</font> or "
    "<font face='Courier'>id</font> changes. The ternary operator (<font face='Courier'>? :</font>) checks: if truncation is enabled and the "
    "ID is longer than the limit, show a shortened version; otherwise show the full ID."
))

story.append(section("Complex Form: $derived.by(() =&gt; { ... })"))
story.append(source_ref("webhooks/+page.svelte, lines 45-57"))
story.append(code_block(
    "let filteredWebhooks = $derived.by(() => {\n"
    "  let result = webhooks;\n"
    "  if (healthFilter) {\n"
    "    result = result.filter(\n"
    "      (wh) => wh.health === healthFilter\n"
    "    );\n"
    "  }\n"
    "  if (urlSearch.trim()) {\n"
    "    const q = urlSearch.toLowerCase();\n"
    "    result = result.filter((wh) =>\n"
    "      wh.url.toLowerCase().includes(q) ||\n"
    "      wh.description?.toLowerCase().includes(q)\n"
    "    );\n"
    "  }\n"
    "  return result;\n"
    "});"
))

story.append(section("Line by Line"))
story.append(bullet(
    '<font face="Courier">$derived.by(() =&gt; { ... })</font> -- When the computation needs multiple statements, use '
    '<font face="Courier">$derived.by()</font> with a function body.'
))
story.append(bullet(
    '<font face="Courier">.filter((wh) =&gt; wh.health === healthFilter)</font> -- <font face="Courier">.filter()</font> creates a new array keeping '
    'only elements where the callback returns true.'
))
story.append(bullet(
    '<font face="Courier">wh.description?.toLowerCase()</font> -- The <font face="Courier">?.</font> is optional chaining. <font face="Courier">description</font> might be '
    'undefined. Without <font face="Courier">?.</font>, calling <font face="Courier">.toLowerCase()</font> on undefined would crash.'
))

story.append(section("$derived vs $state: When to Use Which"))
story.append(body(
    "Use <font face='Courier'>$state</font> for data you set directly (API responses, user input). Use <font face='Courier'>$derived</font> "
    "for data computed from other state. If you find yourself writing code that sets "
    "a variable every time another changes, use <font face='Courier'>$derived</font>."
))

story.append(section("Under the Hood: Derived Signals"))
story.append(body(
    "The compiler transforms <font face='Courier'>$derived(expr)</font> into "
    "<font face='Courier'>$.derived(() =&gt; expr)</font>. A derived signal is <b>lazy</b> "
    "-- it doesn't recompute immediately when a dependency changes. Instead, the runtime "
    "marks it as <b>MAYBE_DIRTY</b> and only recalculates when something actually reads "
    "its value. Think of it as a lazy database view vs. a materialized one."
))
story.append(code_block(
    "// What happens when a source signal changes:\n"
    "//\n"
    "// 1. $.set(source, newValue)\n"
    "//    -> mark_reactions(source, DIRTY)\n"
    "//\n"
    "// 2. For each dependent:\n"
    "//    - If it's a render_effect -> mark DIRTY (will re-run)\n"
    "//    - If it's a derived -> mark MAYBE_DIRTY\n"
    "//      (might not need recalc if other deps cancel out)\n"
    "//\n"
    "// 3. Derived only recalculates when $.get() is called.\n"
    "//    If the source value didn't actually change its\n"
    "//    equality check, the derived stays clean.\n"
    "//\n"
    "// This is push-DIRTY, pull-VALUE:\n"
    "//   \"I know something changed\" (push)\n"
    "//   \"Let me check if I need to recompute\" (pull)"
))
story.append(body(
    "React's <font face='Courier'>useMemo</font> requires you to manually list dependencies "
    "in an array -- get it wrong and you have stale values or infinite loops. "
    "Vue's <font face='Courier'>computed()</font> auto-tracks at runtime (closer to Svelte). "
    "Svelte's compiler resolves the dependency graph at build time, so there is no "
    "dependency array to get wrong and no runtime cost for dependency tracking."
))

story.extend(gotcha(1, "Only reactive sources are tracked",
    "<font face='Courier'>$derived</font> only tracks variables from <font face='Courier'>$state</font>, <font face='Courier'>$derived</font>, or <font face='Courier'>$props</font>. A plain <font face='Courier'>let x = 5</font> is not reactive."
))
story.extend(gotcha(2, "Don't mutate inside $derived",
    "A <font face='Courier'>$derived</font> computation should be pure. Never modify other <font face='Courier'>$state</font> variables "
    "inside. That's what <font face='Courier'>$effect</font> is for."
))


# ============================================================================
# LESSON 3: $props() -- with compiled output internals
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(3, "Components & Props with $props()", anchor="lesson3"))

story.append(section("The Problem"))
story.append(body(
    "As your UI grows, you need to break it into reusable pieces. A health badge, a "
    "copyable ID display, a confirmation dialog. Each piece needs to accept data from "
    "its parent. In Svelte 5, components receive data through <font face='Courier'>$props()</font>."
))

story.append(section("Defining Props with TypeScript"))
story.append(source_ref("CopyableId.svelte, lines 2-8"))
story.append(code_block(
    "interface Props {\n"
    "  id: string;\n"
    "  href?: string;\n"
    "  truncate?: number;\n"
    "}\n\n"
    "let { id, href, truncate = 8 }: Props = $props();"
))

story.append(section("Line by Line"))
story.append(bullet(
    '<font face="Courier">interface Props { ... }</font> -- A TypeScript interface defines the shape of an object.'
))
story.append(bullet(
    '<font face="Courier">id: string</font> -- Required prop. Missing it = TypeScript compile error.'
))
story.append(bullet(
    '<font face="Courier">href?: string</font> -- The <font face="Courier">?</font> makes this optional. When omitted, the value is undefined.'
))
story.append(bullet(
    '<font face="Courier">let { id, href, truncate = 8 }: Props = $props()</font> -- Destructures props. <font face="Courier">truncate = 8</font> is a default value.'
))

story.append(section("Callback Props (Functions as Props)"))
story.append(source_ref("ConfirmDialog.svelte, lines 1-11"))
story.append(code_block(
    "interface Props {\n"
    "  open: boolean;\n"
    "  title: string;\n"
    "  message: string;\n"
    "  confirmLabel?: string;\n"
    "  cancelLabel?: string;\n"
    "  variant?: 'danger' | 'warning' | 'info';\n"
    "  onconfirm: () => void;\n"
    "  oncancel: () => void;\n"
    "}"
))
story.append(bullet(
    '<font face="Courier">onconfirm: () =&gt; void</font> -- This prop expects a function. The parent passes a callback. When the user clicks "Confirm", the dialog calls <font face="Courier">onconfirm()</font>.'
))
story.append(body(
    "This is the standard pattern for child-to-parent communication in Svelte 5: the "
    "parent passes a function down, the child calls it when something happens."
))

story.append(section("Under the Hood: What $props Compiles To"))
story.append(body(
    "The compiler transforms each destructured prop into a signal read via "
    "<font face='Courier'>$.prop()</font>. When a parent changes a prop value, only the "
    "DOM expressions that read <i>that specific prop</i> update -- not the entire child "
    "component. In React, every prop change re-runs the entire component function."
))
story.append(source_ref("web/src/lib/components/Pagination.svelte, lines 2-25"))
story.append(code_block(
    "// Real Sparrow code (Pagination.svelte, lines 2-25):\n"
    "interface Props {\n"
    "  currentPage: number;\n"
    "  totalPages: number;\n"
    "  totalCount: number;\n"
    "  pageSize: number;\n"
    "  onPageChange: (pageNum: number) => void;\n"
    "  itemLabel?: string;\n"
    "}\n"
    "\n"
    "let { currentPage, totalPages, totalCount,\n"
    "      pageSize, onPageChange,\n"
    "      itemLabel = 'items' }: Props = $props();\n"
    "\n"
    "function nextPage() {\n"
    "  if (currentPage < totalPages) {\n"
    "    onPageChange(currentPage + 1);\n"
    "  }\n"
    "}\n"
    "function previousPage() {\n"
    "  if (currentPage > 1) {\n"
    "    onPageChange(currentPage - 1);\n"
    "  }\n"
    "}"
))
story.append(body(
    "The compiler transforms this into signal reads. Below is a <b>conceptual "
    "illustration</b> of the compiled output (not literal Sparrow code):"
))
story.append(code_block(
    "// Conceptual compiled output (illustrative):\n"
    "// 1. Props become signal reads via $.prop()\n"
    "let currentPage = $.prop($$props, 'currentPage');\n"
    "let totalPages  = $.prop($$props, 'totalPages');\n"
    "let onPageChange = $.prop($$props, 'onPageChange');\n"
    "let itemLabel = $.prop($$props, 'itemLabel', 8, 'items');\n"
    "\n"
    "// 2. Template text {currentPage} becomes:\n"
    "$.render_effect(() => {\n"
    "  $.set_text(text_node, $.get(currentPage));\n"
    "});\n"
    "\n"
    "// 3. disabled={currentPage === 1} becomes:\n"
    "$.render_effect(() => {\n"
    "  $.set_attribute(btn, 'disabled',\n"
    "    $.get(currentPage) === 1);\n"
    "});\n"
    "\n"
    "// Each render_effect targets exactly one DOM node.\n"
    "// No tree diffing. No re-rendering sibling elements."
))

story.extend(gotcha(1, "Props are read-only",
    "You cannot reassign a prop inside the child. Props flow one direction: parent to child."
))
story.extend(gotcha(2, "Default values only apply when undefined",
    "If parent passes <font face='Courier'>truncate={0}</font>, default 8 does NOT apply. 0 is a valid value."
))


# ============================================================================
# LESSON 4: {#if}
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(4, "Conditional Rendering with {#if}", anchor="lesson4"))

story.append(section("The Problem"))
story.append(body(
    "Most UI isn't static. You need to show a loading spinner while data loads, an "
    "error message when something fails, and actual content when data arrives."
))

story.append(section("Three-State Pattern: Loading / Error / Content"))
story.append(source_ref("health/+page.svelte, lines 97-146"))
story.append(code_block(
    "{#if loading}\n"
    "  <!-- Skeleton placeholders -->\n"
    "  <div class=\"animate-pulse\">...</div>\n"
    "{:else if error}\n"
    "  <!-- Error message -->\n"
    "  <div class=\"text-red-600\">{error}</div>\n"
    "{:else}\n"
    "  <!-- Actual content -->\n"
    "  <div>{healthSummary.totalWebhooks}</div>\n"
    "{/if}"
))

story.append(section("How It Works"))
story.append(bullet('<font face="Courier">{#if loading}</font> -- If <font face="Courier">loading</font> is truthy, this branch renders.'))
story.append(bullet('<font face="Courier">{:else if error}</font> -- Check error. An empty string is falsy (no error).'))
story.append(bullet('<font face="Courier">{:else}</font> -- The fallback. Show the real content.'))
story.append(bullet('<font face="Courier">{/if}</font> -- Closes the block.'))

story.append(section("DOM Destruction vs CSS Hiding"))
story.append(body(
    "<font face='Courier'>{#if}</font> removes elements from the DOM when the condition is false. It doesn't hide "
    "them with CSS. Any internal state (scroll position, input values) is lost."
))

story.append(section("Inline Ternaries in Attributes"))
story.append(code_block(
    "<span class=\"{wh.active ? 'bg-green-500' : 'bg-gray-300'}\">\n"
    "  {wh.active ? 'Active' : 'Paused'}\n"
    "</span>"
))

story.extend(gotcha(1, "Equality checks",
    "JavaScript's <font face='Courier'>==</font> does type coercion. Always use <font face='Courier'>===</font> for strict equality."
))
story.extend(gotcha(2, "Empty arrays are truthy",
    "An empty array <font face='Courier'>[]</font> is truthy. Use <font face='Courier'>{#if myArray.length &gt; 0}</font> instead."
))


# ============================================================================
# LESSON 5: {#each}
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(5, "Rendering Lists with {#each}", anchor="lesson5"))

story.append(section("The Problem"))
story.append(body(
    "You have an array of data and need to render each item."
))

story.append(section("Basic Usage: Table Rows"))
story.append(source_ref("webhooks/+page.svelte, lines 484-512"))
story.append(code_block(
    "{#each filteredWebhooks as wh}\n"
    "  <tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>\n"
    "    <td>{wh.url}</td>\n"
    "    <td>\n"
    "      <CopyableId id={wh.webhookId}\n"
    "        href=\"/webhooks/{wh.webhookId}\" />\n"
    "    </td>\n"
    "    {#each wh.events.slice(0, 2) as event}\n"
    "      <span class=\"badge\">{event}</span>\n"
    "    {/each}\n"
    "    {#if wh.events.length > 2}\n"
    "      <span>+{wh.events.length - 2}</span>\n"
    "    {/if}\n"
    "  </tr>\n"
    "{/each}"
))

story.append(section("Key Points"))
story.append(bullet(
    '<font face="Courier">{#each filteredWebhooks as wh}</font> -- <font face="Courier">wh</font> is the loop variable holding one webhook per iteration.'
))
story.append(bullet(
    '<font face="Courier">wh.events.slice(0, 2)</font> -- Nested <font face="Courier">{#each}</font>. "Show first 2 + overflow count" pattern.'
))

story.append(section("Skeleton Placeholders with Array(n)"))
story.append(source_ref("health/+page.svelte, lines 104-109"))
story.append(code_block(
    "{#each Array(4) as _}\n"
    "  <div class=\"bg-white animate-pulse\">\n"
    "    <div class=\"h-8 bg-gray-200 rounded\"></div>\n"
    "  </div>\n"
    "{/each}"
))
story.append(bullet(
    '<font face="Courier">Array(4)</font> -- Standard idiom for "repeat N times". <font face="Courier">as _</font> means "I don\'t use this value."'
))

story.extend(gotcha(1, "Keyed vs Unkeyed Lists",
    "For reorderable lists, add a key: <font face='Courier'>{#each items as item (item.id)}</font>."
))
story.extend(gotcha(2, "Destructuring in the loop",
    "You can destructure: <font face='Courier'>{#each webhooks as { url, webhookId }}</font>."
))


# ============================================================================
# LESSON 6: {@const}
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(6, "Inline Constants with {@const}", anchor="lesson6"))

story.append(section("The Problem"))
story.append(body(
    "Inside an <font face='Courier'>{#each}</font> loop or <font face='Courier'>{#if}</font> block, you often need to compute a value. "
    "Without <font face='Courier'>{@const}</font>, you'd duplicate the expression everywhere."
))

story.append(section("Map Lookup"))
story.append(source_ref("health/+page.svelte, line 206"))
story.append(code_block(
    "{#snippet webhookCard(wh: RegisteredWebhook)}\n"
    "  {@const metrics = webhookMetrics.get(wh.webhookId)}\n"
    "  <a href=\"/webhooks/{wh.webhookId}\">\n"
    "    {#if metrics}\n"
    "      <span>{(metrics.successRate * 100).toFixed(1)}%</span>\n"
    "    {/if}\n"
    "  </a>\n"
    "{/snippet}"
))
story.append(bullet(
    '<font face="Courier">{@const metrics = webhookMetrics.get(wh.webhookId)}</font> -- Creates a block-scoped constant. Without it, you\'d repeat <font face="Courier">.get()</font> 5+ times.'
))

story.append(section("Computed Value for Rendering"))
story.append(source_ref("health/+page.svelte, line 228"))
story.append(code_block(
    "{@const totalErrors = (metrics.clientErrors || 0)\n"
    "  + (metrics.serverErrors || 0)\n"
    "  + (metrics.timeoutErrors || 0)\n"
    "  + (metrics.networkErrors || 0)\n"
    "  + (metrics.unexpectedStatusErrors || 0)}"
))

story.append(section("Function Call Result"))
story.append(source_ref("deliveries/[deliveryId]/+page.svelte, line 136"))
story.append(code_block(
    "{#if delivery.errorCategory !== 'success'}\n"
    "  {@const cat = getCategoryDisplay(delivery.errorCategory)}\n"
    "  <span class=\"{cat.bgColor} {cat.color}\">\n"
    "    {cat.label}\n"
    "  </span>\n"
    "{/if}"
))
story.append(body(
    "Calls the function once, stores the result, then uses <font face='Courier'>cat.bgColor</font>, <font face='Courier'>cat.color</font>, <font face='Courier'>cat.label</font>."
))

story.extend(gotcha(1, "Must be at the top of a block",
    "<font face='Courier'>{@const}</font> must be first in its block."
))
story.extend(gotcha(2, "Truly const",
    "Cannot reassign it. Mutable values belong in the script section as <font face='Courier'>$state</font>."
))
story.extend(gotcha(3, "No async",
    "<font face='Courier'>{@const}</font> is synchronous only. Cannot use <font face='Courier'>await</font>."
))


# ============================================================================
# LESSON 7: Event Handling
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(7, "Event Handling", anchor="lesson7"))

story.append(section("The Problem"))
story.append(body(
    "Users click buttons, type in inputs, press Enter, select dropdowns. In Svelte 5, "
    "event handlers are plain HTML attributes -- no special syntax."
))

story.append(section("Inline Handler"))
story.append(source_ref("webhooks/+page.svelte, line 487"))
story.append(code_block(
    "<tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>"
))
story.append(bullet(
    '<font face="Courier">onclick</font> -- Standard HTML attribute, lowercase. Different from Svelte 4\'s <font face="Courier">on:click</font> (deprecated).'
))

story.append(section("Handler with Event Object"))
story.append(source_ref("webhooks/+page.svelte, line 548"))
story.append(code_block(
    "<button onclick={(e) => toggleActive(wh, e)}>"
))

story.append(section("Named Function Reference"))
story.append(source_ref("CopyableId.svelte, lines 15-25 & 42"))
story.append(code_block(
    "async function copyId(e: Event) {\n"
    "  e.stopPropagation();\n"
    "  e.preventDefault();\n"
    "  try {\n"
    "    await navigator.clipboard.writeText(id);\n"
    "    copied = true;\n"
    "    setTimeout(() => { copied = false; }, 1500);\n"
    "  } catch { /* noop */ }\n"
    "}\n\n"
    "// In template:\n"
    "<button onclick={copyId}>"
))

story.append(section("Keyboard Events"))
story.append(source_ref("deliveries/+page.svelte, line 267"))
story.append(code_block(
    "<input\n"
    "  bind:value={namespaceFilter}\n"
    '  onkeydown={(e) => e.key === \'Enter\' && applyFilters()}\n'
    "/>"
))

story.append(section("Form Submission"))
story.append(source_ref("events/[eventName]/update/+page.svelte, line 131"))
story.append(code_block(
    "<form onsubmit={updateEvent}>\n"
    "  <input bind:value={name} disabled />\n"
    "  <textarea bind:value={description}></textarea>\n"
    "  <button type=\"submit\">Update Event</button>\n"
    "</form>"
))

story.append(section("Change Events on Selects"))
story.append(source_ref("deliveries/+page.svelte, line 276"))
story.append(code_block(
    "<select bind:value={statusFilter} onchange={applyFilters}>\n"
    "  <option value=\"\">All</option>\n"
    "</select>"
))

story.extend(gotcha(1, "handler vs handler()",
    "<font face='Courier'>onclick={handler}</font> passes the function. <font face='Courier'>onclick={handler()}</font> CALLS it immediately. Almost always a bug."
))
story.extend(gotcha(2, "Svelte 5 vs Svelte 4 syntax",
    "Svelte 5 uses <font face='Courier'>onclick</font>. Svelte 4 used <font face='Courier'>on:click</font>. Both work, but <font face='Courier'>on:</font> is deprecated."
))


# ============================================================================
# LESSON 8: $effect() -- with dependency tracking internals
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(8, "Side Effects with $effect()", anchor="lesson8"))

story.append(section("The Problem"))
story.append(body(
    "Sometimes when state changes, you need to do something beyond just updating the UI: "
    "start a timer, set up a DOM observer, log analytics. These are side effects. "
    "<font face='Courier'>$effect()</font> runs a function whenever its reactive dependencies change, "
    "and optionally cleans up when dependencies change again or the component is destroyed."
))

story.append(section("Auto-Dismiss Timer"))
story.append(source_ref("BatchProgress.svelte, lines 64-74"))
story.append(code_block(
    "let autoDismissTimer: ReturnType<typeof setTimeout> | undefined;\n\n"
    "$effect(() => {\n"
    "  if (isTerminal && !dismissed) {\n"
    "    ondone?.();\n"
    "    autoDismissTimer = setTimeout(() => {\n"
    "      dismissed = true;\n"
    "    }, 5000);\n"
    "  }\n"
    "  return () => {\n"
    "    if (autoDismissTimer) clearTimeout(autoDismissTimer);\n"
    "  };\n"
    "});"
))

story.append(section("Line by Line"))
story.append(bullet(
    '<font face="Courier">$effect(() =&gt; { ... })</font> -- Runs after every render where its tracked dependencies '
    'change. Here it tracks <font face="Courier">isTerminal</font> (a <font face="Courier">$derived</font>) and <font face="Courier">dismissed</font> (a <font face="Courier">$state</font>).'
))
story.append(bullet(
    '<font face="Courier">ondone?.()</font> -- Optional chaining on a function call. If <font face="Courier">ondone</font> was passed as a prop, call it. If not, do nothing.'
))
story.append(bullet(
    '<font face="Courier">setTimeout(() =&gt; { dismissed = true }, 5000)</font> -- After 5 seconds, auto-dismiss the progress bar.'
))
story.append(bullet(
    '<font face="Courier">return () =&gt; { ... }</font> -- The cleanup function. Called when (1) the effect re-runs or (2) the component is destroyed. Clears the timer to prevent memory leaks.'
))

story.append(section("IntersectionObserver Pattern"))
story.append(source_ref("FloatingAction.svelte, lines 7-22"))
story.append(code_block(
    "let visible = $state(false);\n\n"
    "$effect(() => {\n"
    "  const target = document.querySelector(targetSelector);\n"
    "  if (!target) return;\n\n"
    "  const observer = new IntersectionObserver(\n"
    "    ([entry]) => {\n"
    "      visible = !entry.isIntersecting;\n"
    "    },\n"
    "    { threshold: 0 }\n"
    "  );\n\n"
    "  observer.observe(target);\n\n"
    "  return () => observer.disconnect();\n"
    "});"
))

story.append(section("How It Works"))
story.append(bullet(
    '<font face="Courier">IntersectionObserver</font> -- A browser API that fires a callback when an element enters or exits the viewport.'
))
story.append(bullet(
    '<font face="Courier">visible = !entry.isIntersecting</font> -- When the target scrolls out of view, show the floating action button.'
))
story.append(bullet(
    '<font face="Courier">return () =&gt; observer.disconnect()</font> -- Cleanup: stop observing when the component unmounts.'
))

story.append(section("$effect vs $derived"))
story.append(body(
    "Use <font face='Courier'>$derived</font> for pure computations (input -> output). "
    "Use <font face='Courier'>$effect</font> for anything that touches the outside world: DOM manipulation, "
    "timers, network requests, browser APIs."
))

story.append(section("Under the Hood: How $effect Tracks Dependencies"))
story.append(body(
    "The runtime maintains a global variable called <font face='Courier'>active_reaction</font>. "
    "When an effect runs, it sets itself as the active reaction. Any signal read via "
    "<font face='Courier'>$.get(signal)</font> during execution registers the active reaction "
    "as a dependent. This is how dependencies are tracked automatically -- no manual "
    "subscription, no dependency arrays like React's <font face='Courier'>useEffect([deps])</font>."
))
story.append(code_block(
    "// runtime.js (simplified)\n"
    "let active_reaction = null;  // global: who is running?\n"
    "\n"
    "export function get(signal) {\n"
    "  if (active_reaction !== null) {\n"
    "    // Record: active_reaction depends on this signal\n"
    "    signal.reactions.push(active_reaction);\n"
    "  }\n"
    "  return signal.v;  // return current value\n"
    "}\n"
    "\n"
    "// When an effect runs:\n"
    "function run_effect(effect) {\n"
    "  const prev = active_reaction;\n"
    "  active_reaction = effect;   // set ourselves as active\n"
    "  effect.fn();                // any $.get() calls register us\n"
    "  active_reaction = prev;     // restore previous\n"
    "}"
))
story.append(source_ref("sveltejs/svelte -- packages/svelte/src/internal/client/runtime.js"))
story.append(body(
    "So when the BatchProgress effect runs, it calls "
    "<font face='Courier'>$.get(isTerminal)</font> and <font face='Courier'>$.get(dismissed)</font>. "
    "Both signals now have this effect in their <font face='Courier'>reactions</font> list. "
    "When either signal changes, the effect is marked DIRTY and scheduled to re-run. "
    "The old cleanup function runs first, then the effect body executes again."
))
story.append(body(
    "The compiler transforms <font face='Courier'>$effect(() =&gt; {...})</font> into "
    "<font face='Courier'>$.user_effect(() =&gt; {...})</font>. The runtime distinguishes "
    "between <b>user effects</b> (your code, runs after DOM updates) and "
    "<b>render effects</b> (compiler-generated, updates the DOM). You never create "
    "render effects directly -- the compiler creates them from your template expressions."
))

story.extend(gotcha(1, "Don't use $effect for derived state",
    "Never write <font face='Courier'>$effect(() =&gt; { count = items.length })</font>. Use <font face='Courier'>$derived</font> instead. Effects that only set state are a code smell."
))
story.extend(gotcha(2, "Cleanup is critical",
    "If your effect creates timers, observers, or event listeners, always return a cleanup function. Forgetting cleanup causes memory leaks."
))
story.extend(gotcha(3, "$effect runs after render",
    "<font face='Courier'>$effect</font> runs asynchronously after the DOM updates. If you need to read DOM measurements, <font face='Courier'>$effect</font> is the right place."
))


# ============================================================================
# LESSON 9: Snippets & Render
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(9, "Snippets & Render", anchor="lesson9"))

story.append(section("The Problem"))
story.append(body(
    "You need to reuse a chunk of HTML template within a component, or pass template "
    "content from a parent to a child. In Svelte 4 this was done with slots. Svelte 5 replaces "
    "slots with snippets -- explicit, typed, and more powerful."
))

story.append(section("Declaring a Snippet in a Template"))
story.append(source_ref("health/+page.svelte, lines 205-259"))
story.append(code_block(
    "{#snippet webhookCard(wh: RegisteredWebhook)}\n"
    "  {@const metrics = webhookMetrics.get(wh.webhookId)}\n"
    "  <a href=\"/webhooks/{wh.webhookId}\">\n"
    "    <span>{wh.description || 'Webhook'}</span>\n"
    "    <HealthBadge health={wh.health} size=\"sm\" />\n"
    "    {#if metrics}\n"
    "      <span>{(metrics.successRate * 100).toFixed(1)}%</span>\n"
    "    {/if}\n"
    "  </a>\n"
    "{/snippet}\n\n"
    "<!-- Used twice: -->\n"
    "{#each unhealthyWebhooks as wh}\n"
    "  {@render webhookCard(wh)}\n"
    "{/each}\n"
    "{#each degradedWebhooks as wh}\n"
    "  {@render webhookCard(wh)}\n"
    "{/each}"
))

story.append(section("How It Works"))
story.append(bullet(
    '<font face="Courier">{#snippet webhookCard(wh: RegisteredWebhook)}</font> -- Declares a reusable template block. Think of it like a function that returns HTML.'
))
story.append(bullet(
    '<font face="Courier">{@render webhookCard(wh)}</font> -- Calls the snippet, passing the current webhook. Rendered inline at the call site.'
))
story.append(bullet(
    "Defined once, rendered in two loops (unhealthy + degraded). Without snippets, you'd copy the entire card HTML twice."
))

story.append(section("Snippets as Props (Replacing Slots)"))
story.append(source_ref("EmptyState.svelte, lines 1-25"))
story.append(code_block(
    "// EmptyState.svelte:\n"
    "import type { Snippet } from 'svelte';\n\n"
    "interface Props {\n"
    "  icon?: string;\n"
    "  title: string;\n"
    "  description?: string;\n"
    "  action?: Snippet;  // <-- Snippet type!\n"
    "}\n\n"
    "let { icon, title, description, action } = $props();\n\n"
    "// In template:\n"
    "{#if action}\n"
    "  {@render action()}\n"
    "{/if}"
))

story.append(section("Passing a Snippet from Parent"))
story.append(source_ref("SubscriptionManager.svelte, lines 425-433"))
story.append(code_block(
    "<EmptyState\n"
    "  icon=\"link\"\n"
    "  title=\"No subscriptions yet\"\n"
    "  description=\"Create subscriptions to define...\"\n"
    ">\n"
    "  {#snippet action()}\n"
    "    <button onclick={openCreateModal}>\n"
    "      Create First Subscription\n"
    "    </button>\n"
    "  {/snippet}\n"
    "</EmptyState>"
))
story.append(bullet(
    '<font face="Courier">action?: Snippet</font> -- The <font face="Courier">Snippet</font> type represents a renderable template block. Optional because not every empty state needs a button.'
))
story.append(bullet(
    '<font face="Courier">{#snippet action()}</font> inside parent -- Defines snippet content inline. The name must match the prop name.'
))

story.append(section("Layout Children Pattern"))
story.append(source_ref("+layout.svelte, lines 6 & 60"))
story.append(code_block(
    "let { children } = $props();\n\n"
    "<!-- At the bottom of the layout: -->\n"
    "{@render children?.()}"
))
story.append(body(
    "<font face='Courier'>children</font> is a built-in snippet prop in SvelteKit that contains the page content. "
    "The <font face='Courier'>?.</font> guards against undefined during SSR."
))

story.extend(gotcha(1, "Snippets replace slots",
    "Svelte 4's <font face='Courier'>&lt;slot /&gt;</font> is deprecated. Use <font face='Courier'>{#snippet}</font> + <font face='Courier'>{@render}</font> instead."
))
story.extend(gotcha(2, "Snippet scope",
    "Snippets defined inside a component access all variables in the component's scope (closures)."
))


# ============================================================================
# LESSON 10: Two-Way Binding
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(10, "Two-Way Binding", anchor="lesson10"))

story.append(section("The Problem"))
story.append(body(
    "Form inputs need to both display a value and update it when the user types. "
    "Svelte's <font face='Courier'>bind:</font> directive automates this two-way sync."
))

story.append(section("bind:value on Inputs"))
story.append(source_ref("SubscriptionManager.svelte, lines 674-680"))
story.append(code_block(
    "<input\n"
    "  type=\"text\"\n"
    "  bind:value={form.eventName}\n"
    "  oninput={() => handleEventChange(form.eventName)}\n"
    "  placeholder=\"Type event name\"\n"
    "/>"
))
story.append(bullet(
    '<font face="Courier">bind:value={form.eventName}</font> -- Two-way sync: the input displays the value, and typing updates it automatically.'
))
story.append(bullet(
    'The <font face="Courier">oninput</font> handler here is for additional logic (fetching event details), not for the binding itself.'
))

story.append(section("bind:value on Selects"))
story.append(source_ref("deliveries/+page.svelte, line 276"))
story.append(code_block(
    "<select bind:value={statusFilter} onchange={applyFilters}>\n"
    "  <option value=\"\">All</option>\n"
    "  <option value=\"delivered\">Delivered</option>\n"
    "  <option value=\"failed\">Failed</option>\n"
    "</select>"
))

story.append(section("$bindable: Two-Way Props"))
story.append(source_ref("SubscriptionManager.svelte, lines 19-29"))
story.append(code_block(
    "let {\n"
    "  webhookId,\n"
    "  namespace,\n"
    "  subscriptions = $bindable([]),\n"
    "  onRefresh,\n"
    "}: {\n"
    "  webhookId: string;\n"
    "  namespace: string;\n"
    "  subscriptions: EventSubscription[];\n"
    "  onRefresh?: () => void;\n"
    "} = $props();"
))

story.append(section("How $bindable Works"))
story.append(bullet(
    '<font face="Courier">subscriptions = $bindable([])</font> -- Declares that this prop can be two-way bound. The parent uses <font face="Courier">bind:subscriptions={myList}</font> and changes flow both directions.'
))
story.append(bullet(
    'Without <font face="Courier">$bindable</font>, props are read-only (Lesson 3). <font face="Courier">$bindable</font> explicitly opts into two-way communication.'
))
story.append(bullet(
    '<font face="Courier">$bindable([])</font> provides a default value if the parent doesn\'t bind.'
))

story.append(section("When to Use Each Pattern"))
story.append(body(
    "<b>bind:value</b> -- For native HTML elements (input, select, textarea).<br/>"
    "<b>$bindable</b> -- For component props needing two-way flow. Rare -- prefer callback props.<br/>"
    "<b>Callback props</b> -- For most child-to-parent communication (Lesson 3). More explicit."
))

story.extend(gotcha(1, "bind: is two-way",
    "Changes flow both directions. Setting the variable updates the input; typing updates the variable."
))
story.extend(gotcha(2, "$bindable is opt-in",
    "By default, props are read-only. You must explicitly mark a prop as <font face='Courier'>$bindable</font>."
))


# ============================================================================
# LESSON 11: Lifecycle
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(11, "Lifecycle Hooks", anchor="lesson11"))

story.append(section("The Problem"))
story.append(body(
    "Components need to do things at specific moments: fetch data when first rendered, "
    "clean up timers when removed."
))

story.append(section("onMount: Fetching Data on Load"))
story.append(source_ref("health/+page.svelte, line 70"))
story.append(code_block(
    "import { onMount } from 'svelte';\n\n"
    "let loading = $state(true);\n"
    "let error = $state('');\n\n"
    "async function fetchData() {\n"
    "  loading = true;\n"
    "  error = '';\n"
    "  try {\n"
    "    const res = await healthClient.getHealthSummary({});\n"
    "    healthSummary = res.summary;\n"
    "  } catch (e: any) {\n"
    "    error = formatAPIError(e, 'Failed to load');\n"
    "  } finally {\n"
    "    loading = false;\n"
    "  }\n"
    "}\n\n"
    "onMount(fetchData);"
))

story.append(section("How It Works"))
story.append(bullet(
    '<font face="Courier">onMount(fetchData)</font> -- Calls <font face="Courier">fetchData</font> once, after the component is first rendered into the DOM. NOT during SSR.'
))
story.append(bullet(
    'The <font face="Courier">try/catch/finally</font> pattern ensures <font face="Courier">loading</font> is always cleared, even on failure.'
))

story.append(section("onDestroy: Cleaning Up"))
story.append(source_ref("deliveries/+page.svelte, lines 4, 44-46"))
story.append(code_block(
    "import { onDestroy } from 'svelte';\n\n"
    "let pollingTimer: ReturnType<typeof setInterval> | undefined;\n\n"
    "onDestroy(() => {\n"
    "  if (pollingTimer) clearInterval(pollingTimer);\n"
    "});"
))
story.append(bullet(
    '<font face="Courier">onDestroy</font> -- Runs when the component is removed from the DOM. Use it to clean up intervals, event listeners, subscriptions.'
))

story.append(section("onMount vs $effect"))
story.append(body(
    "<b>onMount</b> -- Runs once after first render. Best for initial data fetching.<br/>"
    "<b>$effect</b> -- Runs after every render where dependencies change. Best for reactive side effects.<br/><br/>"
    "Use <font face='Courier'>onMount</font> for one-time setup. Use <font face='Courier'>$effect</font> when you need to react to state changes."
))

story.extend(gotcha(1, "onMount doesn't run on the server",
    "<font face='Courier'>onMount</font> is browser-only. During SSR, the component renders without calling it."
))
story.extend(gotcha(2, "Don't forget cleanup",
    "If <font face='Courier'>onMount</font> starts an interval, pair it with <font face='Courier'>onDestroy</font>. Or use <font face='Courier'>$effect</font> with return cleanup."
))


# ============================================================================
# LESSON 12: Async Data Fetching
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(12, "Async Data Fetching", anchor="lesson12"))

story.append(section("The Problem"))
story.append(body(
    "Every page loads data from a backend API. You need to handle loading state, "
    "display errors, and sometimes fetch multiple resources in parallel."
))

story.append(section("The API Client Layer"))
story.append(source_ref("lib/services.ts, lines 41-50"))
story.append(code_block(
    "import { createClient } from '@connectrpc/connect';\n"
    "import { createConnectTransport } from '@connectrpc/connect-web';\n"
    "import { WebhookService, EventService } from '../proto/webhook_pb.js';\n\n"
    "const transport = createConnectTransport({\n"
    "  baseUrl: PUBLIC_API_URL || '/',\n"
    "  interceptors,  // API key injection\n"
    "});\n\n"
    "export const webhookClient = createClient(WebhookService, transport);\n"
    "export const eventClient = createClient(EventService, transport);\n"
    "export const deliveryClient = createClient(DeliveryService, transport);\n"
    "export const healthClient = createClient(HealthService, transport);"
))

story.append(section("How It Works"))
story.append(bullet(
    '<font face="Courier">createConnectTransport</font> -- Creates an HTTP transport that speaks the Connect protocol.'
))
story.append(bullet(
    '<font face="Courier">createClient(WebhookService, transport)</font> -- Creates a typed client. You call <font face="Courier">webhookClient.listWebhooks({...})</font> and get back typed responses.'
))
story.append(bullet(
    '<font face="Courier">interceptors</font> -- Middleware that attaches the <font face="Courier">X-API-Key</font> header when configured.'
))

story.append(section("Parallel Fetching with Promise.all"))
story.append(source_ref("health/+page.svelte, lines 27-38"))
story.append(code_block(
    "const [summaryRes, statsRes, unhealthyRes, degradedRes] =\n"
    "  await Promise.all([\n"
    "    healthClient.getHealthSummary({}),\n"
    "    webhookClient.getNamespaceStats({ namespace: '' }),\n"
    "    healthClient.listWebhooksByHealth({\n"
    "      health: WebhookHealth.HEALTH_UNHEALTHY,\n"
    "      pagination: { limit: 20, offset: 0 },\n"
    "    }),\n"
    "    healthClient.listWebhooksByHealth({\n"
    "      health: WebhookHealth.HEALTH_DEGRADED,\n"
    "      pagination: { limit: 20, offset: 0 },\n"
    "    }),\n"
    "  ]);"
))
story.append(bullet(
    '<font face="Courier">Promise.all([...])</font> -- Runs all four API calls concurrently. Waits for all to complete.'
))
story.append(bullet(
    '<font face="Courier">const [a, b, c, d] = await ...</font> -- Array destructuring on the result.'
))

story.append(section("The Loading/Error/Content Pattern"))
story.append(code_block(
    "let loading = $state(true);\n"
    "let error = $state('');\n"
    "let data = $state<DataType | undefined>();\n\n"
    "async function fetchData() {\n"
    "  loading = true;\n"
    "  error = '';\n"
    "  try {\n"
    "    const res = await client.getData({...});\n"
    "    data = res.data;\n"
    "  } catch (e: any) {\n"
    "    error = formatAPIError(e, 'Failed to load');\n"
    "  } finally {\n"
    "    loading = false;\n"
    "  }\n"
    "}\n\n"
    "onMount(fetchData);"
))
story.append(body(
    "The template then uses the three-state <font face='Courier'>{#if}</font> pattern from Lesson 4."
))

story.append(section("Error Formatting"))
story.append(source_ref("lib/utils.ts, lines 217-241"))
story.append(code_block(
    "function formatAPIError(err: unknown, contextPrefix?: string): string {\n"
    "  let msg = (err as any)?.message ?? String(err);\n"
    "  // Strip gRPC code prefix: \"[internal] ...\"\n"
    "  msg = msg.replace(/^\\[\\w+\\]\\s*/, '');\n"
    "  if (!contextPrefix) return msg;\n"
    "  return `${contextPrefix}: ${msg}`;\n"
    "}"
))

story.extend(gotcha(1, "Always set loading = false",
    "Use <font face='Courier'>finally { loading = false }</font>. Without <font face='Courier'>finally</font>, an error leaves the page stuck on the skeleton."
))
story.extend(gotcha(2, "Promise.all is all-or-nothing",
    "If one call fails, you lose all results. Use <font face='Courier'>Promise.allSettled()</font> for partial results."
))


# ============================================================================
# LESSON 13: Layout & Navigation
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(13, "Layout & Navigation", anchor="lesson13"))

story.append(section("The Problem"))
story.append(body(
    "Every page shares a navigation header. You don't want to copy this HTML into every page. "
    "SvelteKit's layout system lets you define shared UI once."
))

story.append(section("The Layout Component"))
story.append(source_ref("+layout.svelte, all 60 lines"))
story.append(code_block(
    '<script lang="ts">\n'
    '  import { page } from "$app/state";\n'
    '  import "../app.css";\n\n'
    '  let { children } = $props();\n\n'
    '  const titles: Record<string, string> = {\n'
    '    "/webhooks": "Webhooks",\n'
    '    "/events": "Events",\n'
    '    "/health": "Health",\n'
    '  };\n\n'
    '  function getTitle(): string {\n'
    '    const path = page.route.id?.toString() || "/";\n'
    '    return titles[path] || "";\n'
    '  }\n'
    '</script>\n\n'
    '<header class="sticky top-0 ...">\n'
    '  <h2>{getTitle()}</h2>\n'
    '  <nav>\n'
    '    <a href="/webhooks">Webhooks</a>\n'
    '    <a href="/events">Events</a>\n'
    '    <a href="/deliveries">Deliveries</a>\n'
    '    <a href="/health">Health</a>\n'
    '  </nav>\n'
    '</header>\n\n'
    '{@render children?.()}'
))

story.append(section("How It Works"))
story.append(bullet(
    '<font face="Courier">let { children } = $props()</font> -- SvelteKit passes a <font face="Courier">children</font> snippet containing the current page\'s content.'
))
story.append(bullet(
    '<font face="Courier">{@render children?.()}</font> -- Renders the page content below the header.'
))
story.append(bullet(
    '<font face="Courier">page.route.id</font> -- SvelteKit\'s reactive page state gives the current route pattern for dynamic titles.'
))
story.append(bullet(
    '<font face="Courier">import "../app.css"</font> -- Global CSS (Tailwind) imported in layout, available everywhere.'
))

story.append(section("SvelteKit Navigation: goto()"))
story.append(source_ref("webhooks/+page.svelte, line 487"))
story.append(code_block(
    "import { goto } from '$app/navigation';\n\n"
    "<tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>"
))
story.append(bullet(
    '<font face="Courier">goto(url)</font> -- Programmatic navigation. No full page reload -- SvelteKit does client-side navigation.'
))

story.append(section("Dynamic Routes"))
story.append(code_block(
    "web/src/routes/\n"
    "  webhooks/\n"
    "    +page.svelte          -> /webhooks\n"
    "    register/+page.svelte -> /webhooks/register\n"
    "    [webhookId]/+page.svelte -> /webhooks/abc123\n"
    "  events/\n"
    "    [eventName]/\n"
    "      reports/+page.svelte -> /events/user.created/reports"
))
story.append(bullet(
    '<font face="Courier">[webhookId]</font> -- A dynamic segment. The value is available via <font face="Courier">page.params.webhookId</font>.'
))

story.append(section("svelte:head"))
story.append(source_ref("health/+page.svelte, lines 73-75"))
story.append(code_block(
    "<svelte:head>\n"
    "  <title>Health | Sparrow</title>\n"
    "</svelte:head>"
))
story.append(body(
    "<font face='Courier'>&lt;svelte:head&gt;</font> injects elements into the document head. Each page sets its own title."
))

story.extend(gotcha(1, "Layout wraps all pages",
    "Everything in <font face='Courier'>+layout.svelte</font> appears on every page in that directory (and subdirectories)."
))
story.extend(gotcha(2, "goto() vs &lt;a&gt;",
    "Prefer <font face='Courier'>&lt;a href=\"...\"&gt;</font> for regular links. Use <font face='Courier'>goto()</font> for programmatic navigation."
))


# ============================================================================
# LESSON 14: Component Composition
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(14, "Component Composition", anchor="lesson14"))

story.append(section("The Problem"))
story.append(body(
    "Real applications need reusable components that work together: a pagination control, "
    "a progress bar, a confirmation dialog. This lesson shows how to design them, combining Lessons 1-13."
))

story.append(section("Pattern 1: Callback-Driven Component"))
story.append(source_ref("Pagination.svelte, all 72 lines"))
story.append(code_block(
    "interface Props {\n"
    "  currentPage: number;\n"
    "  totalPages: number;\n"
    "  totalCount: number;\n"
    "  pageSize: number;\n"
    "  onPageChange: (pageNum: number) => void;\n"
    "  itemLabel?: string;\n"
    "}\n\n"
    "let { currentPage, totalPages, totalCount,\n"
    "  pageSize, onPageChange, itemLabel = 'items'\n"
    "}: Props = $props();\n\n"
    "// In template:\n"
    "<button onclick={() => onPageChange(pageNum)}>\n"
    "  {pageNum}\n"
    "</button>"
))
story.append(body(
    "Pagination is \"dumb\" -- it displays page numbers and calls <font face='Courier'>onPageChange</font> when "
    "clicked. The parent owns the state and data-fetching logic."
))

story.append(section("Pattern 2: Self-Contained with $effect"))
story.append(source_ref("BatchProgress.svelte, lines 24-74"))
story.append(code_block(
    "let { batch, label = 'Batch', oncancel, ondone } = $props();\n\n"
    "let dismissed = $state(false);\n\n"
    "let progressPercent = $derived(\n"
    "  batch && batch.total > 0\n"
    "    ? Math.round(((batch.processed + batch.failed)\n"
    "        / batch.total) * 100)\n"
    "    : 0\n"
    ");\n\n"
    "let isTerminal = $derived(\n"
    "  batch?.status === 'completed' ||\n"
    "  batch?.status === 'failed'\n"
    ");\n\n"
    "let statusColor = $derived.by(() => {\n"
    "  switch (batch.status) {\n"
    "    case 'completed': return 'bg-green-500';\n"
    "    case 'failed':    return 'bg-red-500';\n"
    "    case 'processing': return 'bg-blue-500';\n"
    "    default:           return 'bg-yellow-500';\n"
    "  }\n"
    "});\n\n"
    "$effect(() => {\n"
    "  if (isTerminal && !dismissed) {\n"
    "    ondone?.();\n"
    "    autoDismissTimer = setTimeout(\n"
    "      () => { dismissed = true; }, 5000);\n"
    "  }\n"
    "  return () => clearTimeout(autoDismissTimer);\n"
    "});"
))

story.append(section("How It Combines Lessons"))
story.append(bullet('<b>$props()</b> (Lesson 3) -- Receives batch status and callbacks.'))
story.append(bullet('<b>$state</b> (Lesson 1) -- Local <font face="Courier">dismissed</font> state.'))
story.append(bullet('<b>$derived</b> (Lesson 2) -- <font face="Courier">progressPercent</font>, <font face="Courier">isTerminal</font>, <font face="Courier">statusColor</font>.'))
story.append(bullet('<b>$derived.by()</b> (Lesson 2) -- Switch statement needs block form.'))
story.append(bullet('<b>$effect</b> (Lesson 8) -- Auto-dismiss timer with cleanup.'))
story.append(bullet('<b>Callback props</b> (Lesson 3) -- <font face="Courier">oncancel</font>, <font face="Courier">ondone</font>.'))

story.append(section("Pattern 3: Snippet Slot Component"))
story.append(source_ref("EmptyState.svelte"))
story.append(code_block(
    "import type { Snippet } from 'svelte';\n\n"
    "interface Props {\n"
    "  icon?: string;\n"
    "  title: string;\n"
    "  description?: string;\n"
    "  action?: Snippet;  // Slot replacement\n"
    "}\n\n"
    "// Template:\n"
    "{#if action}\n"
    "  {@render action()}\n"
    "{/if}"
))
story.append(body(
    "EmptyState provides structure (icon, title) and a snippet slot for custom content. "
    "The parent decides what action to show; the child decides where."
))

story.append(section("Design Guidelines"))
story.append(bullet('<b>Keep components focused</b> -- Each component has one job.'))
story.append(bullet('<b>Parent owns state, child reports events</b> -- Use callback props.'))
story.append(bullet('<b>Use snippets for flexible content</b> -- Accept <font face="Courier">Snippet</font> props for custom HTML.'))
story.append(bullet('<b>Type everything</b> -- <font face="Courier">interface Props</font> for every component.'))


# ============================================================================
# LESSON 15: The Build Pipeline (was Lesson 17)
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(15, "The Build Pipeline", anchor="lesson15"))

story.append(body(
    "This lesson traces the complete build pipeline for Sparrow's frontend -- from "
    "<font face='Courier'>.svelte</font> source files to a single Go binary with the UI "
    "embedded. Every config file shown is from the actual Sparrow codebase."
))

story.append(section("Pipeline Overview"))
story.append(code_block(
    "+-------------+    +----------+    +--------------+\n"
    "| .svelte     |    | Vite +   |    | adapter-     |\n"
    "| .ts files   |--->| SvelteKit|--->| static       |\n"
    "| (web/src/)  |    | compiler |    | (SPA output) |\n"
    "+-------------+    +----------+    +------+-------+\n"
    "                                          |\n"
    "                                          v\n"
    "                                   +--------------+\n"
    "                                   | internal/ui/ |\n"
    "                                   | dist/        |\n"
    "                                   +------+-------+\n"
    "                                          |\n"
    "                   +----------+            | go:embed\n"
    "                   | go build |<-----------+\n"
    "                   | (single  |\n"
    "                   |  binary) |\n"
    "                   +----------+"
))

story.append(section("Step 1: Vite + SvelteKit Compilation"))
story.append(body(
    "Vite is the build tool that orchestrates everything. It calls the Svelte compiler "
    "as a plugin for each <font face='Courier'>.svelte</font> file, then bundles, "
    "tree-shakes, and minifies the output."
))
story.append(source_ref("web/vite.config.ts"))
story.append(code_block(
    "import { sveltekit } from '@sveltejs/kit/vite';\n"
    "import tailwindcss from '@tailwindcss/vite';\n"
    "import { defineConfig } from 'vite';\n"
    "\n"
    "export default defineConfig({\n"
    "  plugins: [\n"
    "    tailwindcss(),   // Tailwind v4 -- CSS at build time\n"
    "    sveltekit(),     // Svelte compiler + SvelteKit routing\n"
    "  ],\n"
    "  resolve: {\n"
    "    alias: {\n"
    "      // Force ESM for protobuf (Vite needs ESM)\n"
    "      '@bufbuild/protobuf': resolve(\n"
    "        __dirname,\n"
    "        './node_modules/@bufbuild/protobuf/dist/esm'\n"
    "      ),\n"
    "    },\n"
    "  },\n"
    "});"
))
story.append(body(
    "The plugin order matters: Tailwind runs first (processes "
    "<font face='Courier'>@import 'tailwindcss'</font> in CSS), "
    "then SvelteKit compiles <font face='Courier'>.svelte</font> files, "
    "and Vite handles bundling and tree-shaking."
))

story.append(section("Step 2: adapter-static (SPA Mode)"))
story.append(body(
    "SvelteKit supports multiple deployment targets via adapters. Sparrow uses "
    "<font face='Courier'>adapter-static</font> which pre-renders to static HTML/JS/CSS "
    "files. The <font face='Courier'>fallback: 'index.html'</font> option enables SPA "
    "mode -- all routes are handled client-side."
))
story.append(source_ref("web/svelte.config.js"))
story.append(code_block(
    "import staticAdapter from '@sveltejs/adapter-static';\n"
    "\n"
    "const config = {\n"
    "  kit: {\n"
    "    adapter: staticAdapter({\n"
    "      pages: '../internal/ui/dist',  // output dir\n"
    "      assets: '../internal/ui/dist',\n"
    "      fallback: 'index.html',        // SPA mode\n"
    "      strict: false  // allow non-prerendered routes\n"
    "    }),\n"
    "  }\n"
    "};"
))
story.append(body(
    "The output directory is <font face='Courier'>../internal/ui/dist</font> -- this "
    "places the built frontend exactly where the Go embed directive expects it."
))

story.append(section("Step 3: go:embed (Compile into Binary)"))
story.append(body(
    "Go's <font face='Courier'>embed</font> package bakes files into the binary at compile "
    "time. Sparrow uses <font face='Courier'>//go:embed all:dist</font> to include the "
    "entire frontend build output. The <font face='Courier'>all:</font> prefix includes "
    "dotfiles and hidden files."
))
story.append(source_ref("internal/ui/embed.go, lines 24-25, 40-88"))
story.append(code_block(
    "// Real Sparrow code (internal/ui/embed.go):\n"
    "//go:embed all:dist\n"
    "var embeddedFS embed.FS\n"
    "\n"
    "func Handler(logger *slog.Logger,\n"
    "             config *Config) http.Handler {\n"
    "  staticFS, _ := fs.Sub(embeddedFS, \"dist\")\n"
    "  fileServer := http.FileServer(http.FS(staticFS))\n"
    "  configScript := buildConfigScript(config)\n"
    "\n"
    "  return http.HandlerFunc(func(w http.ResponseWriter,\n"
    "                               r *http.Request) {\n"
    "    path := strings.TrimPrefix(r.URL.Path, \"/\")\n"
    "\n"
    "    // Try to serve static file directly\n"
    "    if path != \"\" {\n"
    "      if f, err := staticFS.Open(path); err == nil {\n"
    "        _ = f.Close()\n"
    "        // Immutable assets get 1-year cache\n"
    "        if strings.HasPrefix(path,\n"
    "            \"_app/immutable/\") {\n"
    "          w.Header().Set(\"Cache-Control\",\n"
    "            \"public, max-age=31536000, immutable\")\n"
    "        }\n"
    "        fileServer.ServeHTTP(w, r)\n"
    "        return\n"
    "      }\n"
    "    }\n"
    "\n"
    "    // SPA fallback: serve index.html\n"
    "    indexBytes, _ := fs.ReadFile(staticFS,\n"
    "      \"index.html\")\n"
    "    html := string(indexBytes)\n"
    "    // Inject runtime config into </head>\n"
    "    if configScript != \"\" {\n"
    "      html = strings.Replace(html,\n"
    "        \"</head>\", configScript+\"</head>\", 1)\n"
    "    }\n"
    "    w.Header().Set(\"Content-Type\",\n"
    "      \"text/html; charset=utf-8\")\n"
    "    w.Write([]byte(html))\n"
    "  })\n"
    "}"
))

story.append(section("Step 4: Docker Multi-Stage Build"))
story.append(body(
    "The Dockerfile chains all stages together: Node builds the frontend, Go compiles "
    "the binary with the embedded UI, and the final image is distroless (no shell, no "
    "package manager, minimal attack surface)."
))
story.append(source_ref("Dockerfile, lines 2-63"))
story.append(code_block(
    "# Stage 1: Build frontend (Dockerfile, lines 2-23)\n"
    "FROM node:22-alpine AS frontend\n"
    "WORKDIR /build/web\n"
    "COPY web/package.json web/package-lock.json* ./\n"
    "RUN npm ci --ignore-scripts\n"
    "COPY web/ .\n"
    "# Proto files needed by SvelteKit imports:\n"
    "COPY proto/webhook_pb.js proto/webhook_pb.d.ts \\\n"
    "     /build/proto/\n"
    "RUN PUBLIC_API_URL=/ npm run build\n"
    "# Output: /build/internal/ui/dist/\n"
    "\n"
    "# Stage 2: Build Go binary (lines 25-49)\n"
    "FROM golang:1.26.1-alpine AS builder\n"
    "WORKDIR /build\n"
    "COPY go.mod go.sum ./\n"
    "RUN go mod download\n"
    "COPY . .\n"
    "COPY --from=frontend /build/internal/ui/dist/ \\\n"
    "     /build/internal/ui/dist/\n"
    "RUN CGO_ENABLED=0 go build \\\n"
    "    -ldflags=\"-w -s\" -trimpath \\\n"
    "    -o server ./cmd/server\n"
    "\n"
    "# Stage 3: Runtime (lines 51-63)\n"
    "FROM gcr.io/distroless/static-debian12:nonroot\n"
    "COPY --from=builder /build/server /app/server\n"
    "EXPOSE 50051 8080\n"
    "ENTRYPOINT [\"/app/server\"]"
))

story.append(section("Runtime Config Injection"))
story.append(body(
    "The Go server injects runtime configuration (like the API key) into "
    "<font face='Courier'>index.html</font> on every request -- no frontend rebuild "
    "needed when configuration changes. The SPA reads "
    "<font face='Courier'>window.__SPARROW_CONFIG__</font> at startup."
))
story.append(code_block(
    "// Go server injects into </head>:\n"
    "<script>window.__SPARROW_CONFIG__={\"apiKey\":\"...\"};</script>\n"
    "\n"
    "// SvelteKit reads it (web/src/lib/services.ts):\n"
    "const runtimeConfig =\n"
    "  (typeof window !== 'undefined'\n"
    "    && window.__SPARROW_CONFIG__) || {};\n"
    "const apiKey = runtimeConfig.apiKey || '';"
))

story.append(section("What Gets Shipped: Bundle Size"))
story.append(body(
    "Because Svelte compiles away the framework, your bundle contains only the runtime "
    "primitives you actually use (signals, effects, template helpers) plus your compiled "
    "component code. There is no large framework runtime to download and parse. "
    "Approximate industry figures (minified + gzipped):"
))
story.append(code_block(
    "Framework runtime sizes (approx, min+gzip):\n"
    "\n"
    "Framework           Runtime shipped    Notes\n"
    "---                 ---                ---\n"
    "React 18 + DOM      ~42-44 KB          Always included\n"
    "Vue 3               ~33 KB             Always included\n"
    "Angular 17+         ~90-130 KB         Varies by features\n"
    "Svelte 5            ~5-8 KB            Tree-shaken; only\n"
    "                                       used primitives\n"
    "\n"
    "The Svelte runtime is smaller because the compiler\n"
    "inlines only the helpers each component actually uses.\n"
    "React and Vue ship their entire reconciler/VDOM engine\n"
    "regardless of app complexity."
))


# ============================================================================
# LESSON 16: The Full Stack (was Lesson 19)
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header(16, "The Full Stack", anchor="lesson16"))

story.append(body(
    "This final lesson traces a complete path through Sparrow's stack -- from protobuf "
    "definition to compiled Svelte component, embedded in a Go binary, served to the "
    "browser. Every technology choice at each layer exists for a concrete reason."
))

story.append(section("The Journey of a Type"))
story.append(code_block(
    "+--------------+   buf generate   +------------------+\n"
    "| webhook.proto |---------------> | Go structs       |\n"
    "| (source of   |                  | (proto/webhook.  |\n"
    "|  truth)      |                  |  pb.go)          |\n"
    "|              |   protoc-gen-es  +------------------+\n"
    "| message      |---------------> | JS classes + .dts|\n"
    "| Webhook {    |                  | (proto/webhook_  |\n"
    "|   id, url,   |                  |  pb.js/d.ts)     |\n"
    "|   secret ... |                  +--------+---------+\n"
    "| }            |                           |\n"
    "+--------------+                           | import\n"
    "                                           v\n"
    "                                    +--------------+\n"
    "                                    | services.ts  |\n"
    "                                    | (typed       |\n"
    "                                    |  RPC client) |\n"
    "                                    +------+-------+\n"
    "                                           |\n"
    "                                           v\n"
    "                                    +--------------+\n"
    "                                    | Svelte       |\n"
    "                                    | components   |\n"
    "                                    | (type-safe)  |\n"
    "                                    +--------------+"
))

story.append(section("Layer 1: Protobuf -- Single Source of Truth"))
story.append(body(
    "The <font face='Courier'>webhook.proto</font> file defines all types and RPC services "
    "once. The <font face='Courier'>buf generate</font> command produces Go server code, "
    "Go clients, JS/TS clients, and Python clients from this single definition. "
    "No type duplication across languages."
))
story.append(source_ref("buf.gen.yaml"))
story.append(code_block(
    "# buf.gen.yaml -- generates code for all languages\n"
    "plugins:\n"
    "  # Go server (protobuf + gRPC + Connect-RPC)\n"
    "  - plugin: buf.build/protocolbuffers/go\n"
    "    out: .\n"
    "  - plugin: buf.build/connectrpc/go\n"
    "    out: .\n"
    "\n"
    "  # Web UI (Connect-RPC / @bufbuild/protobuf v2)\n"
    "  - name: es\n"
    "    path: web/node_modules/.bin/protoc-gen-es\n"
    "    out: .\n"
    "    opt:\n"
    "      - target=js+dts  # JS classes + TypeScript types"
))

story.append(section("Layer 2: Connect-RPC Transport"))
story.append(body(
    "Connect-RPC is a protocol-compatible alternative to gRPC-Web that works over "
    "standard HTTP. The Svelte frontend creates a typed transport with an interceptor "
    "that attaches the API key header to every request."
))
story.append(source_ref("web/src/lib/services.ts, lines 1-53"))
story.append(code_block(
    "// Real Sparrow code (services.ts):\n"
    "import { PUBLIC_API_URL }\n"
    "  from '$env/static/public';\n"
    "import { createClient }\n"
    "  from '@connectrpc/connect';\n"
    "import { createConnectTransport }\n"
    "  from '@connectrpc/connect-web';\n"
    "import type { Interceptor }\n"
    "  from '@connectrpc/connect';\n"
    "import {\n"
    "  WebhookService, EventService,\n"
    "  SubscriptionService, DeliveryService,\n"
    "  HealthService\n"
    "} from '../../../proto/webhook_pb.js';\n"
    "\n"
    "// Runtime config injected by Go server\n"
    "const runtimeConfig: SparrowConfig =\n"
    "  (typeof window !== 'undefined'\n"
    "    && window.__SPARROW_CONFIG__) || {};\n"
    "const apiKey = runtimeConfig.apiKey || '';\n"
    "\n"
    "// Interceptor attaches API key header\n"
    "const apiKeyInterceptor: Interceptor =\n"
    "  (next) => async (req) => {\n"
    "    req.header.set('X-API-Key', apiKey);\n"
    "    return next(req);\n"
    "  };\n"
    "\n"
    "const transport = createConnectTransport({\n"
    "  baseUrl: PUBLIC_API_URL || '/',\n"
    "  interceptors: apiKey\n"
    "    ? [apiKeyInterceptor] : [],\n"
    "});\n"
    "\n"
    "// Typed clients (full autocomplete on RPCs)\n"
    "export const webhookClient =\n"
    "  createClient(WebhookService, transport);\n"
    "export const eventClient =\n"
    "  createClient(EventService, transport);\n"
    "export const deliveryClient =\n"
    "  createClient(DeliveryService, transport);\n"
    "export const healthClient =\n"
    "  createClient(HealthService, transport);"
))

story.append(section("Layer 3: Svelte Components (Type-Safe)"))
story.append(body(
    "Components call the typed RPC clients and display results. TypeScript ensures "
    "the response types match the proto definitions. The Svelte compiler then turns "
    "these components into optimized DOM update code -- no virtual DOM, just direct "
    "mutations as described in Lessons 1 and 3."
))
story.append(source_ref("web/src/routes/webhooks/+page.svelte, lines 2-17, 62-98"))
story.append(code_block(
    "// Real Sparrow code (webhooks/+page.svelte):\n"
    "import { webhookClient as client,\n"
    "         healthClient } from '$lib/services';\n"
    "import { onMount } from 'svelte';\n"
    "import type { RegisteredWebhook }\n"
    "  from '../../../../proto/webhook_pb.js';\n"
    "\n"
    "let webhooks: RegisteredWebhook[] = $state([]);\n"
    "let loading = $state(true);\n"
    "let error = $state('');\n"
    "\n"
    "// Pagination state\n"
    "let limit = $state(25);\n"
    "let offset = $state(0);\n"
    "let totalCount = $state(0);\n"
    "\n"
    "async function fetchWebhooks() {\n"
    "  loading = true;\n"
    "  error = '';\n"
    "  try {\n"
    "    const res = await client.listWebhooks({\n"
    "      namespace: '',\n"
    "      pagination: { limit, offset },\n"
    "    });\n"
    "    // res.webhooks is RegisteredWebhook[]\n"
    "    // TypeScript enforces field access:\n"
    "    //   res.webhooks[0].url       OK\n"
    "    //   res.webhooks[0].bogus     compile error\n"
    "    webhooks = res.webhooks || [];\n"
    "    totalCount =\n"
    "      res.pagination?.totalCount || 0;\n"
    "  } catch (e: any) {\n"
    "    error = formatAPIError(e,\n"
    "      'Failed to load webhooks');\n"
    "  } finally {\n"
    "    loading = false;\n"
    "  }\n"
    "}\n"
    "\n"
    "onMount(fetchWebhooks);  // fetch on page load"
))

story.append(section("Layer 4: Build + Embed"))
story.append(body(
    "The SvelteKit build produces static HTML/JS/CSS in "
    "<font face='Courier'>internal/ui/dist/</font>. The Go compiler's "
    "<font face='Courier'>embed</font> directive bakes these files into the binary. "
    "The result is a single executable with no external dependencies -- no Node runtime, "
    "no file server, no CDN required."
))
story.append(code_block(
    "// The complete build chain:\n"
    "$ cd web && npm run build\n"
    "  # Vite -> Svelte compiler -> Tailwind -> tree-shake\n"
    "  # -> adapter-static -> ../internal/ui/dist/\n"
    "\n"
    "$ go build ./cmd/server\n"
    "  # go:embed all:dist -> embed FS in binary\n"
    "  # Output: single static binary (~15MB)\n"
    "\n"
    "$ ./server\n"
    "  # :8080 -- Connect-RPC API + embedded SPA\n"
    "  # :50051 -- gRPC (direct)\n"
    "  # No Node. No nginx. No CDN. One process."
))

story.append(section("Why This Architecture"))
story.append(body(
    "Each technology at each layer was chosen for a specific reason:"
))
story.append(bullet('<b>Protobuf</b> -- Single source of truth for types across Go, '
    'TypeScript, and Python. Schema evolution without breaking changes.'))
story.append(bullet('<b>Connect-RPC</b> -- Works over standard HTTP (no gRPC-Web proxy '
    'needed). Browser-compatible. Same proto definitions as gRPC.'))
story.append(bullet('<b>Svelte 5</b> -- Compiled to vanilla JS. No runtime framework to '
    'ship. ~5-8 KB tree-shaken runtime vs ~44 KB for React. Runes provide type-safe reactivity.'))
story.append(bullet('<b>adapter-static</b> -- Pure static files that can be embedded. '
    'No SSR server needed. SPA mode for client-side routing.'))
story.append(bullet('<b>go:embed</b> -- Single binary deployment. No filesystem '
    'dependencies. Copy one file, run it. Works in distroless containers.'))
story.append(bullet('<b>Distroless</b> -- No shell, no package manager. ~5MB base image. '
    'Minimal attack surface for a self-hosted webhook platform.'))

story.append(section("The End-to-End Type Safety Chain"))
story.append(body(
    "The most important property of this architecture is the unbroken type chain. "
    "A field added to <font face='Courier'>webhook.proto</font> automatically appears "
    "in the Go server struct, the TypeScript client, and the Svelte component -- with "
    "compile-time errors if any layer gets it wrong. There is no manually-maintained "
    "API contract, no JSON schema to keep in sync, no runtime type assertion. "
    "The proto file IS the contract, and code generation enforces it everywhere."
))


# ============================================================================
# EXPANDED REFERENCE CARD
# ============================================================================
story.append(PageBreak())
story.extend(lesson_header("", "Quick Reference Card", anchor="reference"))

ref_data = [
    ["Concept", "Syntax", "Use When"],
    ["Reactive state", "$state(initialValue)", "Data you set directly"],
    ["Computed value", "$derived(expr)\n$derived.by(() => { })", "Value calculated from state"],
    ["Side effects", "$effect(() => {\n  ...\n  return () => cleanup;\n})", "Timers, DOM, browser APIs"],
    ["Component props", "let { a, b } = $props()", "Receiving data from parent"],
    ["Two-way prop", "x = $bindable([])", "Parent reads/writes child state"],
    ["Conditional", "{#if cond}\n{:else if}\n{:else}\n{/if}", "Show/hide DOM sections"],
    ["List rendering", "{#each arr as item}\n{#each arr as x (key)}", "Rendering arrays"],
    ["Inline constant", "{@const x = expr}", "Avoid repeated expressions"],
    ["Snippet (define)", "{#snippet name(params)}\n  HTML...\n{/snippet}", "Reusable template blocks"],
    ["Snippet (render)", "{@render name(args)}", "Calling a snippet"],
    ["Event handler", "onclick={handler}\nonclick={(e) => {}}", "User interactions"],
    ["Two-way binding", "bind:value={x}", "Form inputs"],
    ["Lifecycle", "onMount(fn)\nonDestroy(fn)", "Setup/cleanup"],
    ["Head elements", "<svelte:head>\n  <title>...</title>\n</svelte:head>", "Per-page title/meta"],
]

ref_table = []
for row in ref_data:
    is_header = row == ref_data[0]
    ref_table.append([
        Paragraph(row[0], styles["RefCellBold"]),
        Preformatted(row[1], styles["RefCellCode"]) if not is_header else Paragraph(row[1], styles["RefCellBold"]),
        Paragraph(row[2].replace("\n", "<br/>"), styles["RefCell"]),
    ])

t = Table(ref_table, colWidths=[1.3 * inch, 2.7 * inch, 2.5 * inch])
t.setStyle(TableStyle([
    ("BACKGROUND", (0, 0), (-1, 0), HexColor("#f3f4f6")),
    ("TEXTCOLOR", (0, 0), (-1, 0), PRIMARY),
    ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
    ("FONTSIZE", (0, 0), (-1, -1), 8),
    ("ALIGN", (0, 0), (-1, -1), "LEFT"),
    ("VALIGN", (0, 0), (-1, -1), "TOP"),
    ("GRID", (0, 0), (-1, -1), 0.5, CODE_BORDER),
    ("TOPPADDING", (0, 0), (-1, -1), 4),
    ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ("LEFTPADDING", (0, 0), (-1, -1), 6),
    ("RIGHTPADDING", (0, 0), (-1, -1), 6),
]))
story.append(t)

# TypeScript cheat sheet
story.append(Spacer(1, 0.3 * inch))
story.append(Paragraph("Key TypeScript Syntax Explained in This Tutorial", styles["SectionHead"]))

ts_data = [
    ["Syntax", "Meaning", "Example"],
    ["x: string", "Type annotation", 'let name: string = \'hi\''],
    ["x?: string", "Optional property", 'interface { href?: string }'],
    ["A | B", "Union type (A or B)", 'string | undefined'],
    ["T[]", "Array of type T", 'RegisteredWebhook[]'],
    ["Map<K, V>", "Map with key/value types", 'Map<string, Metrics>'],
    ["() => void", "Fn, no args, no return", 'onconfirm: () => void'],
    ["x?.y", "Optional chaining", 'desc?.toLowerCase()'],
    ["<T>()", "Generic type parameter", '$state<Webhook[]>([])'],
    ["interface", "Object shape definition", 'interface Props { ... }'],
    ["Snippet", "Renderable template type", 'action?: Snippet'],
    ["ReturnType<T>", "Extract return type", 'ReturnType<typeof setTimeout>'],
    ["Record<K,V>", "Object with typed keys", 'Record<string, string>'],
    ["as any", "Type assertion", '(err as any)?.message'],
]

ts_table = []
for row in ts_data:
    is_header = row == ts_data[0]
    ts_table.append([
        Paragraph(row[0], styles["RefCellBold"] if is_header else styles["RefCellCode"]),
        Paragraph(row[1], styles["RefCell"]),
        Preformatted(row[2], styles["RefCellCode"]) if not is_header else Paragraph(row[2], styles["RefCellBold"]),
    ])

t2 = Table(ts_table, colWidths=[1.5 * inch, 2.2 * inch, 2.8 * inch])
t2.setStyle(TableStyle([
    ("BACKGROUND", (0, 0), (-1, 0), HexColor("#f3f4f6")),
    ("TEXTCOLOR", (0, 0), (-1, 0), PRIMARY),
    ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
    ("FONTSIZE", (0, 0), (-1, -1), 8),
    ("ALIGN", (0, 0), (-1, -1), "LEFT"),
    ("VALIGN", (0, 0), (-1, -1), "TOP"),
    ("GRID", (0, 0), (-1, -1), 0.5, CODE_BORDER),
    ("TOPPADDING", (0, 0), (-1, -1), 4),
    ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
    ("LEFTPADDING", (0, 0), (-1, -1), 6),
    ("RIGHTPADDING", (0, 0), (-1, -1), 6),
]))
story.append(t2)

# ===== FINAL PAGE =====
story.append(Spacer(1, 0.5 * inch))
story.append(hr())
story.append(Spacer(1, 0.1 * inch))
story.append(body(
    "All code examples from the Sparrow webhook platform.<br/>"
    "https://github.com/sarathsp06/sparrow"
))

# ---------------------------------------------------------------------------
# Build PDF
# ---------------------------------------------------------------------------
doc.build(story, onFirstPage=_draw_cover, onLaterPages=_page_footer)
print(f"Generated: {output_path}")
print(f"Pages: {doc.page}")
