#!/usr/bin/env python3
"""Generate a multi-page Svelte 5 Tutorial PDF using ReportLab."""

import math
import os
from reportlab.lib.pagesizes import letter
from reportlab.lib.units import inch
from reportlab.lib.colors import HexColor, white, black
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.enums import TA_LEFT, TA_CENTER, TA_JUSTIFY
from reportlab.platypus import (
    SimpleDocTemplate, Paragraph, Spacer, PageBreak, Table, TableStyle,
    KeepTogether, HRFlowable, Flowable,
)
from reportlab.platypus.doctemplate import PageTemplate, BaseDocTemplate, Frame, NextPageTemplate
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont

# ─── Font Registration ──────────────────────────────────────────────────────
FONTS_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "fonts")

# FiraCode — used for all content pages
pdfmetrics.registerFont(TTFont("FiraCode", os.path.join(FONTS_DIR, "FiraCode-Regular.ttf")))
pdfmetrics.registerFont(TTFont("FiraCode-Bold", os.path.join(FONTS_DIR, "FiraCode-Bold.ttf")))
pdfmetrics.registerFont(TTFont("FiraCode-SemiBold", os.path.join(FONTS_DIR, "FiraCode-SemiBold.ttf")))

# Poppins — used for cover page
pdfmetrics.registerFont(TTFont("Poppins", os.path.join(FONTS_DIR, "Poppins-Regular.ttf")))
pdfmetrics.registerFont(TTFont("Poppins-Bold", os.path.join(FONTS_DIR, "Poppins-Bold.ttf")))
pdfmetrics.registerFont(TTFont("Poppins-SemiBold", os.path.join(FONTS_DIR, "Poppins-SemiBold.ttf")))
pdfmetrics.registerFont(TTFont("Poppins-Medium", os.path.join(FONTS_DIR, "Poppins-Medium.ttf")))
pdfmetrics.registerFont(TTFont("Poppins-Light", os.path.join(FONTS_DIR, "Poppins-Light.ttf")))

# Register font families for <b> and <i> tag support in Paragraphs
from reportlab.pdfbase.pdfmetrics import registerFontFamily
registerFontFamily("FiraCode", normal="FiraCode", bold="FiraCode-Bold")
registerFontFamily("Poppins", normal="Poppins", bold="Poppins-Bold")


# ─── Bookmark Flowable ──────────────────────────────────────────────────────
class BookmarkAnchor(Flowable):
    """Invisible flowable that creates a named destination (bookmark) in the PDF."""
    def __init__(self, name):
        Flowable.__init__(self)
        self.name = name
        self.width = 0
        self.height = 0

    def draw(self):
        self.canv.bookmarkHorizontal(self.name, 0, 0)


# Color Palette
DARK_BG       = HexColor("#111827")
ACCENT        = HexColor("#FF3E00")
GRAY_50       = HexColor("#F9FAFB")
GRAY_100      = HexColor("#F3F4F6")
GRAY_200      = HexColor("#E5E7EB")
GRAY_300      = HexColor("#D1D5DB")
GRAY_500      = HexColor("#6B7280")
GRAY_600      = HexColor("#4B5563")
GRAY_700      = HexColor("#374151")
GRAY_800      = HexColor("#1F2937")
GRAY_900      = HexColor("#111827")
CODE_BORDER   = HexColor("#E2E8F0")

W, H = letter

# Styles — all content pages use FiraCode
sTitle = ParagraphStyle("Title", fontName="FiraCode-Bold", fontSize=22,
                        leading=28, textColor=GRAY_900, spaceAfter=6)
sSubtitle = ParagraphStyle("Subtitle", fontName="FiraCode", fontSize=12,
                           leading=16, textColor=GRAY_600, spaceAfter=18)
sH1 = ParagraphStyle("H1", fontName="FiraCode-Bold", fontSize=18,
                      leading=24, textColor=GRAY_900, spaceAfter=10, spaceBefore=16)
sH2 = ParagraphStyle("H2", fontName="FiraCode-Bold", fontSize=14,
                      leading=18, textColor=GRAY_800, spaceAfter=8, spaceBefore=12)
sH3 = ParagraphStyle("H3", fontName="FiraCode-Bold", fontSize=12,
                      leading=15, textColor=GRAY_700, spaceAfter=6, spaceBefore=10)
sBody = ParagraphStyle("Body", fontName="FiraCode", fontSize=10,
                        leading=15, textColor=GRAY_800, spaceAfter=8,
                        alignment=TA_JUSTIFY)
sCode = ParagraphStyle("Code", fontName="FiraCode", fontSize=8.5,
                        leading=12.5, textColor=GRAY_900, spaceAfter=2,
                        leftIndent=4, rightIndent=4)
sExplain = ParagraphStyle("Explain", fontName="FiraCode", fontSize=9.5,
                           leading=14, textColor=GRAY_700, spaceAfter=6,
                           leftIndent=14, bulletIndent=4)
sGotcha = ParagraphStyle("Gotcha", fontName="FiraCode", fontSize=9.5,
                          leading=14, textColor=HexColor("#92400E"),
                          spaceAfter=8, leftIndent=14, bulletIndent=4)
sTOC = ParagraphStyle("TOC", fontName="FiraCode", fontSize=12,
                       leading=22, textColor=GRAY_800, spaceAfter=2)
sTOCsub = ParagraphStyle("TOCsub", fontName="FiraCode", fontSize=9,
                          leading=14, textColor=GRAY_600, leftIndent=24, spaceAfter=1)
sCaption = ParagraphStyle("Caption", fontName="FiraCode", fontSize=8.5,
                           leading=12, textColor=GRAY_500, spaceAfter=12,
                           alignment=TA_CENTER)
sLessonNum = ParagraphStyle("LessonNum", fontName="FiraCode-Bold", fontSize=11,
                             leading=14, textColor=ACCENT, spaceAfter=2)


def code_block(lines):
    content = []
    content.append(HRFlowable(width="100%", thickness=1, color=CODE_BORDER,
                               spaceAfter=0, spaceBefore=6))
    for line in lines:
        escaped = line.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
        if escaped.strip().startswith("//") or escaped.strip().startswith("#"):
            content.append(Paragraph(f'<font color="#6B7280">{escaped}</font>', sCode))
        else:
            highlighted = escaped
            for kw in ["let ", "const ", "function ", "async ", "await ", "return ",
                        "import ", "from ", "export ", "interface ", "if ", "else ",
                        "for ", "true", "false", "new ", "try ", "catch ", "type "]:
                k = kw.strip()
                highlighted = highlighted.replace(kw, f'<font color="#8B5CF6">{k}</font> ')
            for rune in ["$state", "$derived", "$props", "$effect"]:
                highlighted = highlighted.replace(rune, f'<font color="#FF3E00">{rune}</font>')
            content.append(Paragraph(highlighted, sCode))
    content.append(HRFlowable(width="100%", thickness=1, color=CODE_BORDER,
                               spaceBefore=0, spaceAfter=10))
    return content


def lesson_header(num, title):
    return [
        Spacer(1, 8),
        BookmarkAnchor(f"lesson{num}"),
        HRFlowable(width="100%", thickness=2, color=ACCENT, spaceAfter=10),
        Paragraph(f"LESSON {num}", sLessonNum),
        Paragraph(title, sH1),
        Spacer(1, 4),
    ]


def gotcha_box(title, text):
    return [
        Spacer(1, 6),
        Paragraph(f'<font color="#92400E"><b>{title}</b></font>', sGotcha),
        Paragraph(f'<font color="#78350F">{text}</font>', sExplain),
        Spacer(1, 4),
    ]


def explain_point(marker, text):
    return Paragraph(
        f'<bullet>&bull;</bullet><b><font face="FiraCode-Bold" size="8.5">{marker}</font></b> '
        f'&#8212; {text}', sExplain)


def draw_cover(c, doc):
    c.saveState()
    c.setFillColor(DARK_BG)
    c.rect(0, 0, W, H, fill=1, stroke=0)

    # Decorative circles (top right)
    c.setFillColor(HexColor("#FF3E0015"))
    c.circle(W - 80, H - 120, 180, fill=1, stroke=0)
    c.setFillColor(HexColor("#FF3E0020"))
    c.circle(W - 60, H - 100, 120, fill=1, stroke=0)
    c.setFillColor(HexColor("#FF3E0030"))
    c.circle(W - 40, H - 80, 60, fill=1, stroke=0)

    # Grid of dots (mid-left)
    c.setFillColor(HexColor("#374151"))
    for row in range(12):
        for col in range(8):
            x = 50 + col * 18
            y = H/2 - 80 + row * 18
            r = 2 if (row + col) % 3 != 0 else 3.5
            c.circle(x, y, r, fill=1, stroke=0)

    # Accent dots
    c.setFillColor(ACCENT)
    for (x, y) in [(50+2*18, H/2-80+3*18), (50+5*18, H/2-80+6*18),
                    (50+1*18, H/2-80+8*18), (50+6*18, H/2-80+2*18),
                    (50+4*18, H/2-80+10*18)]:
        c.circle(x, y, 4, fill=1, stroke=0)

    # Code-like lines (bottom)
    widths_list = [
        [120, 80, 60, 40], [60, 100, 80], [80, 60, 120, 40, 60],
        [40, 80, 100], [100, 60, 80, 40],
    ]
    line_colors = [HexColor("#374151"), HexColor("#4B5563"), HexColor("#374151")]
    for i, widths in enumerate(widths_list):
        x = 50
        for j, w in enumerate(widths):
            color = ACCENT if (i == 1 and j == 1) or (i == 3 and j == 2) else line_colors[j % len(line_colors)]
            c.setFillColor(color)
            c.roundRect(x, 130 + i * 20, w, 8, 3, fill=1, stroke=0)
            x += w + 12

    # Svelte-like shape (center-right)
    c.saveState()
    c.translate(W - 180, H/2 + 40)
    c.setFillColor(HexColor("#FF3E0040"))
    p = c.beginPath()
    p.moveTo(0, 60)
    p.curveTo(-20, 80, -30, 40, -15, 10)
    p.curveTo(0, -20, 30, -10, 40, 20)
    p.curveTo(50, 50, 30, 80, 0, 60)
    c.drawPath(p, fill=1, stroke=0)
    c.setFillColor(HexColor("#FF3E0060"))
    p2 = c.beginPath()
    p2.moveTo(5, 50)
    p2.curveTo(-10, 65, -15, 35, -5, 15)
    p2.curveTo(5, -5, 25, 0, 30, 20)
    p2.curveTo(35, 40, 20, 60, 5, 50)
    c.drawPath(p2, fill=1, stroke=0)
    c.restoreState()

    # Dashed connection lines
    c.setStrokeColor(HexColor("#374151"))
    c.setLineWidth(1)
    c.setDash(3, 3)
    c.line(200, H/2, W - 220, H/2 + 40)
    c.line(200, H/2 - 60, W - 120, H - 220)
    c.setDash()

    # Title
    c.setFillColor(white)
    c.setFont("Poppins-Bold", 44)
    c.drawString(50, H - 200, "Svelte 5")
    c.setFillColor(ACCENT)
    c.setFont("Poppins-Bold", 44)
    c.drawString(50, H - 255, "Tutorial")
    c.setFillColor(GRAY_500)
    c.setFont("Poppins-Light", 14)
    c.drawString(50, H - 290, "A practical guide using real-world code")

    # Project badge
    c.setFillColor(HexColor("#1F2937"))
    c.roundRect(50, H - 340, 240, 30, 5, fill=1, stroke=0)
    c.setFillColor(ACCENT)
    c.setFont("FiraCode-Bold", 11)
    c.drawString(60, H - 332, "PROJECT:")
    c.setFillColor(GRAY_300)
    c.setFont("FiraCode", 11)
    c.drawString(142, H - 332, "Sparrow Webhooks")

    # Topics
    topics = [
        "Reactive State  ($state)",
        "Computed Values  ($derived)",
        "Components & Props  ($props)",
        "Conditional Rendering  ({#if})",
        "Rendering Lists  ({#each})",
        "Inline Constants  ({@const})",
        "Event Handling  (onclick, onsubmit)",
    ]
    y = H/2 + 130
    for i, topic in enumerate(topics):
        num = f"0{i+1}"
        c.setFillColor(ACCENT)
        c.setFont("FiraCode-Bold", 10)
        c.drawString(250, y, num)
        c.setFillColor(GRAY_300)
        c.setFont("Poppins", 10)
        c.drawString(280, y, topic)
        y -= 22

    # Bottom
    c.setFillColor(GRAY_600)
    c.setFont("Poppins-Light", 9)
    c.drawString(50, 50, "Built with real examples from the Sparrow webhook platform codebase")
    c.setFillColor(HexColor("#60A5FA"))
    c.setFont("FiraCode", 9)
    link_text = "github.com/sarathsp06/sparrow"
    c.drawString(50, 36, link_text)
    link_width = c.stringWidth(link_text, "FiraCode", 9)
    c.linkURL("https://github.com/sarathsp06/sparrow", (50, 34, 50 + link_width, 46), relative=0)
    c.setFillColor(HexColor("#1F2937"))
    c.roundRect(W - 150, 40, 100, 24, 4, fill=1, stroke=0)
    c.setFillColor(GRAY_500)
    c.setFont("FiraCode", 9)
    c.drawString(W - 140, 48, "Svelte 5 / TS")
    c.restoreState()


def page_footer(canvas_obj, doc):
    canvas_obj.saveState()
    canvas_obj.setFillColor(GRAY_300)
    canvas_obj.setFont("FiraCode", 8)
    canvas_obj.drawString(W/2 - 10, 30, f"{doc.page}")
    canvas_obj.setStrokeColor(GRAY_200)
    canvas_obj.setLineWidth(0.5)
    canvas_obj.line(50, H - 45, W - 50, H - 45)
    canvas_obj.setFillColor(GRAY_500)
    canvas_obj.setFont("FiraCode", 7)
    canvas_obj.drawString(50, H - 40, "Svelte 5 Tutorial")
    canvas_obj.drawRightString(W - 50, H - 40, "Sparrow Codebase Examples")
    canvas_obj.restoreState()


def build_pdf():
    output_file = "svelte5-tutorial.pdf"

    doc = BaseDocTemplate(
        output_file, pagesize=letter,
        leftMargin=55, rightMargin=55, topMargin=60, bottomMargin=55,
        title="Svelte 5 Tutorial - Real-World Examples from Sparrow",
        author="Sparrow Project",
    )

    frame = Frame(doc.leftMargin, doc.bottomMargin, doc.width, doc.height, id="main")
    cover_template = PageTemplate(id="cover", frames=[frame], onPage=draw_cover)
    content_template = PageTemplate(id="content", frames=[frame], onPage=page_footer)
    doc.addPageTemplates([cover_template, content_template])

    story = []

    # COVER
    story.append(Spacer(1, 1))
    story.append(NextPageTemplate("content"))
    story.append(PageBreak())

    # TABLE OF CONTENTS
    story.append(Paragraph("Table of Contents", sTitle))
    story.append(Spacer(1, 16))
    toc_items = [
        ("Lesson 1", "Reactive State with $state()", "Understanding how Svelte tracks changes", "lesson1"),
        ("Lesson 2", "Computed Values with $derived()", "Automatic recalculation from state", "lesson2"),
        ("Lesson 3", "Components & Props with $props()", "Building reusable pieces", "lesson3"),
        ("Lesson 4", "Conditional Rendering with {#if}", "Showing and hiding content", "lesson4"),
        ("Lesson 5", "Rendering Lists with {#each}", "Repeating content for arrays", "lesson5"),
        ("Lesson 6", "Inline Constants with {@const}", "Local variables in templates", "lesson6"),
        ("Lesson 7", "Event Handling", "Responding to user interactions", "lesson7"),
    ]
    for num, title, desc, anchor in toc_items:
        story.append(Paragraph(
            f'<a href="#{anchor}" color="#FF3E00"><b>{num}</b></a>'
            f'&nbsp;&nbsp;&nbsp;'
            f'<a href="#{anchor}" color="#111827">{title}</a>', sTOC))
        story.append(Paragraph(desc, sTOCsub))
        story.append(Spacer(1, 4))
    # Quick reference link
    story.append(Paragraph(
        f'<a href="#quickref" color="#FF3E00"><b>Reference</b></a>'
        f'&nbsp;&nbsp;'
        f'<a href="#quickref" color="#111827">Quick Reference Card &amp; TypeScript Cheat Sheet</a>', sTOC))
    story.append(Spacer(1, 4))
    story.append(Spacer(1, 24))
    story.append(HRFlowable(width="100%", thickness=1, color=GRAY_200, spaceAfter=12))
    story.append(Paragraph(
        "<b>Prerequisites:</b> Basic JavaScript (ES6 level: var/let/const, arrow functions, "
        "promises, .map/.forEach). TypeScript syntax is explained inline as encountered.", sBody))
    story.append(Paragraph(
        "<b>Approach:</b> Every example comes from the Sparrow webhook platform codebase with "
        "exact file paths and line numbers. Modern JS/TS syntax is explained as it appears.", sBody))
    story.append(Paragraph(
        '<b>Source Code:</b> <font color="#3B82F6">'
        '<a href="https://github.com/sarathsp06/sparrow" color="#3B82F6">'
        'https://github.com/sarathsp06/sparrow</a></font>', sBody))
    story.append(PageBreak())

    # LESSON 1
    story.extend(lesson_header(1, "Reactive State with $state()"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "In a web app, data changes constantly: a user clicks a button, an API returns results, "
        "a timer fires. The UI needs to update to reflect these changes. In vanilla JavaScript, "
        "you'd manually find DOM elements and update their text content. Svelte automates this: "
        "you declare a variable as reactive, change it, and the UI updates itself.", sBody))
    story.append(Paragraph(
        "<b>The Svelte 5 Solution: <font face='FiraCode' color='#FF3E00'>$state()</font></b>", sH2))
    story.append(Paragraph(
        "The <font face='FiraCode'>$state()</font> rune tells the Svelte compiler to track a "
        "variable. When that variable's value changes (via plain assignment), every part of the "
        "UI that reads it re-renders automatically.", sBody))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: health/+page.svelte, lines 15-21</i></font>', sCaption))
    story.extend(code_block([
        "let loading = $state(true);",
        "let error = $state('');",
        "let healthSummary = $state<HealthSummary | undefined>();",
        "let unhealthyWebhooks = $state<RegisteredWebhook[]>([]);",
        "let degradedWebhooks = $state<RegisteredWebhook[]>([]);",
        "let webhookMetrics = $state(new Map<string, HealthMetrics>());",
        "let namespaceStats = $state<NamespaceStats[]>([]);",
    ]))
    story.append(Paragraph("<b>Line by Line</b>", sH3))
    story.append(explain_point("let loading = $state(true)",
        "<font face='FiraCode'>$state(true)</font> creates a reactive variable initialized to "
        "<font face='FiraCode'>true</font>. While data is loading, the UI shows a skeleton "
        "placeholder. When loading completes, <font face='FiraCode'>loading = false</font> "
        "triggers the UI to swap to real content."))
    story.append(explain_point("let error = $state('')",
        "An empty string means no error. If a fetch fails, "
        "<font face='FiraCode'>error = 'Something went wrong'</font> makes an error banner appear."))
    story.append(explain_point("$state&lt;HealthSummary | undefined&gt;()",
        "The angle brackets are a <b>generic type parameter</b> (TypeScript). "
        "<font face='FiraCode'>HealthSummary | undefined</font> is a <b>union type</b>: "
        "the variable is either a HealthSummary object or undefined. "
        "Empty parentheses <font face='FiraCode'>()</font> mean no initial value = undefined."))
    story.append(explain_point("$state&lt;RegisteredWebhook[]&gt;([])",
        "<font face='FiraCode'>RegisteredWebhook[]</font> means \"an array of RegisteredWebhook "
        "objects\". The <font face='FiraCode'>[]</font> inside parens is the initial value: empty array."))
    story.append(explain_point("$state(new Map&lt;string, HealthMetrics&gt;())",
        "A <font face='FiraCode'>Map</font> is like a dictionary. "
        "<font face='FiraCode'>Map&lt;string, HealthMetrics&gt;</font> = keys are strings, "
        "values are HealthMetrics objects. Initialized empty."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>How Reactivity Triggers</b>", sH3))
    story.append(Paragraph("To update a <font face='FiraCode'>$state</font> variable, use plain assignment:", sBody))
    story.extend(code_block([
        "// This triggers a UI update:",
        "loading = false;",
        "",
        "// Replacing the whole array:",
        "unhealthyWebhooks = result.webhooks;",
        "",
        "// For Maps, reassign the whole Map:",
        "webhookMetrics = newMap;",
    ]))
    story.append(Paragraph(
        "The key insight: <font face='FiraCode'>$state()</font> is a <b>compiler directive</b>, not "
        "a regular function. At runtime, the variable holds the plain value (a boolean, a string, an "
        "array), not a wrapper object. Svelte's compiler rewrites your code during build to inject "
        "change tracking behind the scenes.", sBody))
    story.extend(gotcha_box("Gotcha 1: Mutating arrays",
        "If you push to an array with <font face='FiraCode'>.push()</font>, Svelte 5 <b>does</b> detect "
        "the mutation (unlike Svelte 4 which required reassignment). However, reassignment is still "
        "the clearest pattern: <font face='FiraCode'>items = [...items, newItem]</font>."))
    story.extend(gotcha_box("Gotcha 2: $state() is not a regular function",
        "You cannot do <font face='FiraCode'>const x = condition ? $state(1) : $state(2)</font>. "
        "The compiler must see <font face='FiraCode'>$state()</font> as a direct initializer in a "
        "<font face='FiraCode'>let</font> declaration at the top level of a component's script block."))
    story.append(PageBreak())

    # LESSON 2
    story.extend(lesson_header(2, "Computed Values with $derived()"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "You have some state, and you need a value that's <i>calculated from</i> that state. "
        "For example: you have an array of webhooks and need a filtered list showing only the "
        "active ones. You want to declare the computation once and have it automatically stay in sync.", sBody))
    story.append(Paragraph(
        "<b>Simple Form: <font face='FiraCode' color='#FF3E00'>$derived(expression)</font></b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: CopyableId.svelte, lines 11-13</i></font>', sCaption))
    story.extend(code_block([
        "const display = $derived(",
        "  truncate > 0 && id.length > truncate",
        "    ? id.substring(0, truncate) + '...'",
        "    : id",
        ");",
    ]))
    story.append(Paragraph(
        "This creates a <font face='FiraCode'>display</font> value that automatically recomputes "
        "whenever <font face='FiraCode'>truncate</font> or <font face='FiraCode'>id</font> changes. "
        "The ternary operator (<font face='FiraCode'>? :</font>) checks: if truncation is enabled "
        "and the ID is longer than the limit, show a shortened version; otherwise show the full ID.", sBody))
    story.append(Paragraph(
        "<b>Complex Form: <font face='FiraCode' color='#FF3E00'>$derived.by(() =&gt; { ... })</font></b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: webhooks/+page.svelte, lines 45-57</i></font>', sCaption))
    story.extend(code_block([
        "let filteredWebhooks = $derived.by(() => {",
        "  let result = webhooks;",
        "",
        "  if (healthFilter) {",
        "    result = result.filter(",
        "      (wh) => wh.health === healthFilter",
        "    );",
        "  }",
        "",
        "  if (urlSearch.trim()) {",
        "    const q = urlSearch.toLowerCase();",
        "    result = result.filter((wh) =>",
        "      wh.url.toLowerCase().includes(q) ||",
        "      wh.description?.toLowerCase().includes(q)",
        "    );",
        "  }",
        "",
        "  return result;",
        "});",
    ]))
    story.append(Paragraph("<b>Line by Line</b>", sH3))
    story.append(explain_point("$derived.by(() =&gt; { ... })",
        "When the computation needs multiple statements, use <font face='FiraCode'>$derived.by()</font> "
        "with a function body. The <font face='FiraCode'>() =&gt; { ... }</font> is an arrow function "
        "with a body that must explicitly <font face='FiraCode'>return</font>."))
    story.append(explain_point(".filter((wh) =&gt; wh.health === healthFilter)",
        "<font face='FiraCode'>.filter()</font> creates a new array keeping only elements where "
        "the callback returns <font face='FiraCode'>true</font>."))
    story.append(explain_point(".includes(q)",
        "A string method returning <font face='FiraCode'>true</font> if the string contains the "
        "substring. Case-insensitive search done by calling <font face='FiraCode'>.toLowerCase()</font> on both sides."))
    story.append(explain_point("wh.description?.toLowerCase()",
        "The <font face='FiraCode'>?.</font> is <b>optional chaining</b>. "
        "<font face='FiraCode'>description</font> might be undefined. Without <font face='FiraCode'>?.</font>, "
        "calling <font face='FiraCode'>.toLowerCase()</font> on undefined would crash. With it, "
        "it safely returns undefined instead."))
    story.append(Spacer(1, 8))
    story.append(Paragraph("<b>$derived vs $state: When to Use Which</b>", sH3))
    story.append(Paragraph(
        "Use <font face='FiraCode'>$state</font> for data you set directly (API responses, user input). "
        "Use <font face='FiraCode'>$derived</font> for data computed from other state. If you find yourself "
        "writing code that sets a variable every time another changes, use <font face='FiraCode'>$derived</font>.", sBody))
    story.extend(gotcha_box("Gotcha 1: Only reactive sources are tracked",
        "$derived only tracks variables from $state, $derived, or $props. "
        "A plain <font face='FiraCode'>let x = 5</font> is not reactive."))
    story.extend(gotcha_box("Gotcha 2: Don't mutate inside $derived",
        "A $derived computation should be pure. Never modify other $state variables inside. "
        "That's what $effect is for."))
    story.append(PageBreak())

    # LESSON 3
    story.extend(lesson_header(3, "Components & Props with $props()"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "As your UI grows, you need to break it into reusable pieces. A health badge, a copyable "
        "ID display, a confirmation dialog. Each piece needs to accept data from its parent. "
        "In Svelte 5, components receive data through <font face='FiraCode'>$props()</font>.", sBody))
    story.append(Paragraph("<b>Defining Props with TypeScript</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: CopyableId.svelte, lines 2-8</i></font>', sCaption))
    story.extend(code_block([
        "interface Props {",
        "  id: string;",
        "  href?: string;",
        "  truncate?: number;",
        "}",
        "",
        "let { id, href, truncate = 8 }: Props = $props();",
    ]))
    story.append(Paragraph("<b>Line by Line</b>", sH3))
    story.append(explain_point("interface Props { ... }",
        "A TypeScript <b>interface</b> defines the shape of an object. \"Any object matching this "
        "type must have these properties with these types.\" The name Props is convention, not special to Svelte."))
    story.append(explain_point("id: string",
        "Required prop. The component must receive a string id. Missing it = TypeScript compile error."))
    story.append(explain_point("href?: string",
        "The <font face='FiraCode'>?</font> makes this optional. Parent can pass it or omit it. "
        "When omitted, the value is <font face='FiraCode'>undefined</font>."))
    story.append(explain_point("let { id, href, truncate = 8 }: Props = $props()",
        "Destructures props. <font face='FiraCode'>truncate = 8</font> is a default value. "
        "The <font face='FiraCode'>: Props</font> tells TypeScript to type-check against the interface."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>Callback Props (Functions as Props)</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: ConfirmDialog.svelte, lines 1-8</i></font>', sCaption))
    story.extend(code_block([
        "interface Props {",
        "  show: boolean;",
        "  title: string;",
        "  message: string;",
        "  confirmLabel?: string;",
        "  cancelLabel?: string;",
        "  onconfirm: () => void;",
        "  oncancel: () => void;",
        "}",
    ]))
    story.append(explain_point("onconfirm: () =&gt; void",
        "This prop expects a function that takes no arguments and returns nothing "
        "(<font face='FiraCode'>void</font>). The parent passes a callback. "
        "When the user clicks \"Confirm\", the dialog calls <font face='FiraCode'>onconfirm()</font>."))
    story.append(Paragraph(
        "This is the standard pattern for child-to-parent communication in Svelte 5: the parent "
        "passes a function down, the child calls it when something happens.", sBody))
    story.extend(gotcha_box("Gotcha 1: Props are read-only",
        "You cannot reassign a prop inside the child. <font face='FiraCode'>id = 'new'</font> "
        "would be a compile error. Props flow one direction: parent to child."))
    story.extend(gotcha_box("Gotcha 2: Default values only apply when undefined",
        "If parent passes <font face='FiraCode'>truncate={0}</font>, default 8 does NOT apply. "
        "0 is a valid value. Defaults only kick in when omitted or explicitly undefined."))
    story.append(PageBreak())

    # LESSON 4
    story.extend(lesson_header(4, "Conditional Rendering with {#if}"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "Most UI isn't static. You need to show a loading spinner while data loads, an error "
        "message when something fails, and actual content when data arrives. "
        "Svelte's <font face='FiraCode'>{#if}</font> block controls what HTML is in the DOM.", sBody))
    story.append(Paragraph("<b>Three-State Pattern: Loading / Error / Content</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: health/+page.svelte, lines 97-146</i></font>', sCaption))
    story.extend(code_block([
        "{#if loading}",
        "  <!-- Skeleton placeholders -->",
        "  <div class=\"animate-pulse\">...</div>",
        "",
        "{:else if error}",
        "  <!-- Error message -->",
        "  <div class=\"text-red-600\">{error}</div>",
        "",
        "{:else}",
        "  <!-- Actual content -->",
        "  <div>{healthSummary.totalWebhooks}</div>",
        "{/if}",
    ]))
    story.append(Paragraph("<b>How It Works</b>", sH3))
    story.append(explain_point("{#if loading}",
        "Opens a conditional block. If <font face='FiraCode'>loading</font> is truthy "
        "(not false, 0, '', null, undefined, or NaN), this branch renders."))
    story.append(explain_point("{:else if error}",
        "If loading was falsy, check error. An empty string '' is falsy (no error), "
        "a non-empty string is truthy (there is an error)."))
    story.append(explain_point("{:else}",
        "The fallback. If neither loading nor error is truthy, show the real content."))
    story.append(explain_point("{/if}",
        "Closes the block. Every <font face='FiraCode'>{#if}</font> must have a matching "
        "<font face='FiraCode'>{/if}</font>."))
    story.append(Spacer(1, 8))
    story.append(Paragraph("<b>DOM Destruction vs CSS Hiding</b>", sH3))
    story.append(Paragraph(
        "<font face='FiraCode'>{#if}</font> <b>removes elements from the DOM</b> when "
        "the condition is false. It doesn't hide them with CSS. When true again, Svelte creates "
        "fresh DOM elements. Any internal state (scroll position, input values) is lost.", sBody))
    story.append(Paragraph("<b>Inline Ternaries in Attributes</b>", sH2))
    story.extend(code_block([
        "<span class=\"{wh.active ? 'bg-green-500' : 'bg-gray-300'}\">",
        "  {wh.active ? 'Active' : 'Paused'}",
        "</span>",
    ]))
    story.append(Paragraph(
        "Use ternaries for attribute variations (colors, text). Use <font face='FiraCode'>{#if}</font> "
        "when entire HTML sections need to appear/disappear.", sBody))
    story.extend(gotcha_box("Gotcha 1: Equality checks",
        "JavaScript's == does type coercion (\"5\" == 5 is true). Always use === for strict equality."))
    story.extend(gotcha_box("Gotcha 2: Empty arrays are truthy",
        "An empty array [] is truthy. <font face='FiraCode'>{#if myArray}</font> is always true. "
        "Use <font face='FiraCode'>{#if myArray.length &gt; 0}</font> instead."))
    story.append(PageBreak())

    # LESSON 5
    story.extend(lesson_header(5, "Rendering Lists with {#each}"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "You have an array of data and need to render each item. Svelte's "
        "<font face='FiraCode'>{#each}</font> block iterates over an array and renders a template for each.", sBody))
    story.append(Paragraph("<b>Basic Usage: Table Rows</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: webhooks/+page.svelte, lines 484-512</i></font>', sCaption))
    story.extend(code_block([
        "{#each filteredWebhooks as wh}",
        "  <tr onclick={() => goto(`/webhooks/${wh.webhookId}`)}>",
        "    <td>{wh.url}</td>",
        "    <td>",
        "      <CopyableId id={wh.webhookId}",
        "        href=\"/webhooks/{wh.webhookId}\" />",
        "    </td>",
        "    {#each wh.events.slice(0, 2) as event}",
        "      <span class=\"badge\">{event}</span>",
        "    {/each}",
        "    {#if wh.events.length > 2}",
        "      <span>+{wh.events.length - 2}</span>",
        "    {/if}",
        "  </tr>",
        "{/each}",
    ]))
    story.append(Paragraph("<b>Key Points</b>", sH3))
    story.append(explain_point("{#each filteredWebhooks as wh}",
        "<font face='FiraCode'>filteredWebhooks</font> is the array. <font face='FiraCode'>wh</font> "
        "is the loop variable holding one webhook object per iteration."))
    story.append(explain_point("wh.events.slice(0, 2)",
        "Nested {#each} inside the outer loop. <font face='FiraCode'>.slice(0, 2)</font> returns the "
        "first 2 elements, creating a \"show first 2 + overflow count\" pattern."))
    story.append(explain_point("{#if wh.events.length &gt; 2}",
        "After the inner loop, show a '+N' badge if there are more events. "
        "Mixing {#each} and {#if} is common and natural."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>Skeleton Placeholders with Array(n)</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: health/+page.svelte, lines 104-109</i></font>', sCaption))
    story.extend(code_block([
        "{#each Array(4) as _}",
        "  <div class=\"bg-white animate-pulse\">",
        "    <div class=\"h-8 bg-gray-200 rounded\"></div>",
        "    <div class=\"h-4 bg-gray-100 rounded\"></div>",
        "  </div>",
        "{/each}",
    ]))
    story.append(explain_point("Array(4)",
        "Creates an array with 4 empty slots. We just need the loop to repeat 4 times. "
        "This is the standard idiom for \"repeat N times\" since Svelte has no for loop."))
    story.append(explain_point("as _",
        "The underscore _ means \"I don't use this value.\" Each element is undefined."))
    story.extend(gotcha_box("Gotcha 1: Keyed vs Unkeyed Lists",
        "Default {#each} is unkeyed (reuses DOM by position). For reorderable lists with inputs, "
        "add a key: <font face='FiraCode'>{#each items as item (item.id)}</font>."))
    story.extend(gotcha_box("Gotcha 2: Destructuring in the loop",
        "You can destructure: <font face='FiraCode'>{#each webhooks as { url, webhookId }}</font>. "
        "Sparrow prefers <font face='FiraCode'>wh.url</font> for clarity with many-property objects."))
    story.append(PageBreak())

    # LESSON 6
    story.extend(lesson_header(6, "Inline Constants with {@const}"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "Inside an {#each} loop or {#if} block, you often need to compute a value. "
        "Without <font face='FiraCode'>{@const}</font>, you'd duplicate the expression everywhere "
        "or pre-compute it in the script section (overkill for template-only values).", sBody))
    story.append(Paragraph("<b>Map Lookup</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: health/+page.svelte, line 206</i></font>', sCaption))
    story.extend(code_block([
        "{#snippet webhookCard(wh: RegisteredWebhook)}",
        "  {@const metrics = webhookMetrics.get(wh.webhookId)}",
        "  <a href=\"/webhooks/{wh.webhookId}\">",
        "    {#if metrics}",
        "      <span>{(metrics.successRate * 100).toFixed(1)}%</span>",
        "      <span>{metrics.failedDeliveries}</span>",
        "    {/if}",
        "  </a>",
        "{/snippet}",
    ]))
    story.append(explain_point("{@const metrics = webhookMetrics.get(wh.webhookId)}",
        "Creates a block-scoped constant. <font face='FiraCode'>.get()</font> is the Map method "
        "returning the value or undefined. Without {@const}, you'd repeat the .get() call 5+ times."))
    story.append(Spacer(1, 8))
    story.append(Paragraph("<b>Computed Value for Rendering</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: health/+page.svelte, line 228</i></font>', sCaption))
    story.extend(code_block([
        "{@const totalErrors = (metrics.clientErrors || 0)",
        "  + (metrics.serverErrors || 0)",
        "  + (metrics.timeoutErrors || 0)",
        "  + (metrics.networkErrors || 0)",
        "  + (metrics.unexpectedStatusErrors || 0)}",
        "{#if totalErrors > 0}",
        "  <div style=\"width: {(metrics.clientErrors / totalErrors) * 100}%\">",
        "  </div>",
        "{/if}",
    ]))
    story.append(explain_point("(metrics.clientErrors || 0)",
        "The <font face='FiraCode'>|| 0</font> fallback prevents NaN from undefined + undefined."))
    story.append(Spacer(1, 8))
    story.append(Paragraph("<b>Function Call Result</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: deliveries/[deliveryId]/+page.svelte, line 136</i></font>', sCaption))
    story.extend(code_block([
        "{#if delivery.errorCategory !== 'success'}",
        "  {@const cat = getCategoryDisplay(delivery.errorCategory)}",
        "  <span class=\"{cat.bgColor} {cat.color}\">",
        "    {cat.label}",
        "  </span>",
        "{/if}",
    ]))
    story.append(Paragraph(
        "Calls the function once, stores the result, then uses cat.bgColor, cat.color, "
        "cat.label &#8212; four references to one function call.", sBody))
    story.append(Paragraph("<b>Reactivity Behavior</b>", sH3))
    story.append(Paragraph(
        "{@const} is <b>not</b> reactive on its own. It re-evaluates when its surrounding block "
        "re-renders. If webhookMetrics ($state Map) changes, Svelte re-renders the snippet, "
        "and the {@const} line runs again with new Map contents.", sBody))
    story.extend(gotcha_box("Gotcha 1: Must be at the top of a block",
        "{@const} must be first in its block (or after another {@const}). "
        "Cannot be placed between two div elements mid-block."))
    story.extend(gotcha_box("Gotcha 2: Truly const",
        "Cannot reassign it. Mutable values belong in the script section as $state."))
    story.extend(gotcha_box("Gotcha 3: No async",
        "{@const} is synchronous only. Cannot use await. All async data fetching happens in "
        "the script section, stored in $state. Templates only do synchronous reads."))
    story.append(PageBreak())

    # LESSON 7
    story.extend(lesson_header(7, "Event Handling"))
    story.append(Paragraph("<b>The Problem</b>", sH2))
    story.append(Paragraph(
        "Users click buttons, type in inputs, press Enter, select dropdowns. In Svelte 5, "
        "event handlers are plain HTML attributes &#8212; no special syntax. If you know "
        "DOM events, you know 80% of this.", sBody))
    story.append(Paragraph("<b>Inline Handler</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: webhooks/+page.svelte, line 487</i></font>', sCaption))
    story.extend(code_block([
        "<tr",
        "  onclick={() => goto(`/webhooks/${wh.webhookId}`)}",
        ">",
    ]))
    story.append(explain_point("onclick",
        "Standard HTML attribute, lowercase. In Svelte 5, pass a function. Different from "
        "Svelte 4's on:click (colon syntax is deprecated)."))
    story.append(explain_point("() =&gt; goto(...)",
        "Arrow function calling goto(), SvelteKit's navigation function. "
        "No event object needed &#8212; we just care that the click happened."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>Handler with Event Object</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: webhooks/+page.svelte, line 548</i></font>', sCaption))
    story.extend(code_block([
        "<button",
        "  onclick={(e) => toggleActive(wh, e)}",
        ">",
    ]))
    story.append(explain_point("(e) =&gt; toggleActive(wh, e)",
        "Receives the native DOM event object. The handler likely calls "
        "e.stopPropagation() to prevent bubbling to the parent tr's onclick."))
    story.append(explain_point("Closure over wh",
        "wh comes from the {#each} loop. Each row's handler captures its own wh. "
        "Standard JavaScript closure behavior."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>Named Function Reference</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: CopyableId.svelte, lines 15-25 &amp; 42</i></font>', sCaption))
    story.extend(code_block([
        "// In <script>:",
        "async function copyId(e: Event) {",
        "  e.stopPropagation();",
        "  e.preventDefault();",
        "  try {",
        "    await navigator.clipboard.writeText(id);",
        "    copied = true;",
        "    setTimeout(() => { copied = false; }, 1500);",
        "  } catch {",
        "    // Fallback: noop",
        "  }",
        "}",
        "",
        "// In template:",
        "<button onclick={copyId}>",
    ]))
    story.append(explain_point("onclick={copyId}",
        "Passes the function <i>reference</i>. No parentheses &#8212; copyId not copyId(). "
        "Svelte passes the native event as the first argument."))
    story.append(explain_point("e.stopPropagation()",
        "Prevents the click from bubbling to parent elements."))
    story.append(explain_point("e.preventDefault()",
        "Prevents default browser behavior (e.g., navigation if inside an &lt;a&gt; tag)."))
    story.append(explain_point("await navigator.clipboard.writeText(id)",
        "Browser clipboard API. Async because the browser may need permission. "
        "await pauses until the copy completes."))
    story.append(explain_point("copied = true",
        "A $state variable. Setting it triggers a re-render, swapping the copy icon for a green checkmark."))
    story.append(explain_point("setTimeout(() =&gt; { copied = false }, 1500)",
        "After 1.5 seconds, reset the icon. setTimeout runs the callback after the delay (milliseconds)."))
    story.append(PageBreak())

    # Lesson 7 continued
    story.append(Paragraph("<b>Keyboard Events</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: deliveries/+page.svelte, line 267</i></font>', sCaption))
    story.extend(code_block([
        "<input",
        "  bind:value={namespaceFilter}",
        "  onkeydown={(e) => e.key === 'Enter' && applyFilters()}",
        "/>",
    ]))
    story.append(explain_point("onkeydown",
        "Fires on every key press while the input is focused."))
    story.append(explain_point("e.key === 'Enter' &amp;&amp; applyFilters()",
        "The &amp;&amp; short-circuit trick. A &amp;&amp; B evaluates B only if A is truthy. "
        "If key is Enter, call applyFilters(). Otherwise, do nothing."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>Form Submission</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: events/[eventName]/update/+page.svelte, line 131</i></font>', sCaption))
    story.extend(code_block([
        "<form onsubmit={updateEvent}>",
        "  <input bind:value={name} disabled />",
        "  <textarea bind:value={description}></textarea>",
        "  <button type=\"submit\">Update Event</button>",
        "</form>",
    ]))
    story.append(explain_point("onsubmit={updateEvent}",
        "Fires on form submission. The handler calls e.preventDefault() to stop the browser's "
        "default form POST, handling it with JavaScript instead."))
    story.append(Spacer(1, 10))
    story.append(Paragraph("<b>Change Events on Selects</b>", sH2))
    story.append(Paragraph(
        '<font color="#6B7280"><i>Source: deliveries/+page.svelte, line 276</i></font>', sCaption))
    story.extend(code_block([
        "<select",
        "  bind:value={statusFilter}",
        "  onchange={applyFilters}",
        ">",
        "  <option value=\"\">All</option>",
        "</select>",
    ]))
    story.append(explain_point("onchange={applyFilters}",
        "Fires when the user picks a different option. bind:value updates statusFilter, then "
        "onchange triggers applyFilters() to re-fetch data."))
    story.extend(gotcha_box("Gotcha 1: handler vs handler()",
        "onclick={handler} passes the function (runs on click). "
        "onclick={handler()} CALLS it immediately during render. Almost always a bug."))
    story.extend(gotcha_box("Gotcha 2: Svelte 5 vs Svelte 4 syntax",
        "Old tutorials: on:click={handler}. Svelte 5: onclick={handler}. "
        "The colon syntax still works but is deprecated. Our codebase uses the new style."))
    story.append(PageBreak())

    # QUICK REFERENCE
    story.append(BookmarkAnchor("quickref"))
    story.append(Paragraph("Quick Reference Card", sTitle))
    story.append(Spacer(1, 12))

    ref_data = [
        ["Concept", "Syntax", "Use When"],
        ["Reactive state", "$state(initialValue)", "Data you set directly\n(API results, user input)"],
        ["Computed value", "$derived(expression)\n$derived.by(() => { })", "Value calculated from\nother state"],
        ["Component props", "let { a, b } = $props()", "Receiving data from\nparent component"],
        ["Conditional", "{#if cond}\n{:else if}\n{:else}\n{/if}", "Showing/hiding\nentire DOM sections"],
        ["List rendering", "{#each array as item}\n{#each arr as x (key)}", "Rendering arrays.\nAdd (key) for reorder"],
        ["Inline constant", "{@const x = expr}", "Avoiding repeated\nexpressions in templates"],
        ["Event handler", "onclick={handler}\nonclick={(e) => { }}", "Responding to\nuser interactions"],
    ]

    t = Table(ref_data, colWidths=[90, 180, 160])
    t.setStyle(TableStyle([
        ("FONTNAME", (0, 0), (-1, 0), "FiraCode-Bold"),
        ("FONTSIZE", (0, 0), (-1, 0), 9),
        ("FONTNAME", (0, 1), (-1, -1), "FiraCode"),
        ("FONTSIZE", (0, 1), (-1, -1), 8),
        ("FONTNAME", (0, 1), (0, -1), "FiraCode-Bold"),
        ("FONTSIZE", (0, 1), (0, -1), 9),
        ("FONTNAME", (2, 1), (2, -1), "FiraCode"),
        ("FONTSIZE", (2, 1), (2, -1), 8),
        ("BACKGROUND", (0, 0), (-1, 0), GRAY_800),
        ("TEXTCOLOR", (0, 0), (-1, 0), white),
        ("ROWBACKGROUNDS", (0, 1), (-1, -1), [GRAY_50, white]),
        ("ALIGN", (0, 0), (-1, -1), "LEFT"),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("GRID", (0, 0), (-1, -1), 0.5, GRAY_200),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
        ("LEFTPADDING", (0, 0), (-1, -1), 8),
        ("RIGHTPADDING", (0, 0), (-1, -1), 8),
    ]))
    story.append(t)

    story.append(Spacer(1, 24))
    story.append(Paragraph("<b>Key TypeScript Syntax Explained in This Tutorial</b>", sH2))

    ts_data = [
        ["Syntax", "Meaning", "Example"],
        ["x: string", "Type annotation", "let name: string = 'hi'"],
        ["x?: string", "Optional property", "interface { href?: string }"],
        ["A | B", "Union type (A or B)", "string | undefined"],
        ["T[]", "Array of type T", "RegisteredWebhook[]"],
        ["Map<K, V>", "Map with key/value types", "Map<string, Metrics>"],
        ["() => void", "Fn, no args, no return", "onconfirm: () => void"],
        ["x?.y", "Optional chaining", "desc?.toLowerCase()"],
        ["<T>()", "Generic type parameter", "$state<Webhook[]>([])"],
        ["interface", "Object shape definition", "interface Props { ... }"],
    ]

    t2 = Table(ts_data, colWidths=[95, 150, 175])
    t2.setStyle(TableStyle([
        ("FONTNAME", (0, 0), (-1, 0), "FiraCode-Bold"),
        ("FONTSIZE", (0, 0), (-1, 0), 9),
        ("FONTNAME", (0, 1), (0, -1), "FiraCode"),
        ("FONTSIZE", (0, 1), (-1, -1), 8),
        ("FONTNAME", (2, 1), (2, -1), "FiraCode"),
        ("FONTNAME", (1, 1), (1, -1), "FiraCode"),
        ("BACKGROUND", (0, 0), (-1, 0), GRAY_800),
        ("TEXTCOLOR", (0, 0), (-1, 0), white),
        ("ROWBACKGROUNDS", (0, 1), (-1, -1), [GRAY_50, white]),
        ("ALIGN", (0, 0), (-1, -1), "LEFT"),
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("GRID", (0, 0), (-1, -1), 0.5, GRAY_200),
        ("TOPPADDING", (0, 0), (-1, -1), 6),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 6),
        ("LEFTPADDING", (0, 0), (-1, -1), 8),
        ("RIGHTPADDING", (0, 0), (-1, -1), 8),
    ]))
    story.append(t2)

    story.append(Spacer(1, 30))
    story.append(HRFlowable(width="100%", thickness=1, color=GRAY_200, spaceAfter=12))
    story.append(Paragraph(
        '<font color="#6B7280">All code examples from the Sparrow webhook platform.</font>', sCaption))
    story.append(Paragraph(
        '<font color="#3B82F6"><a href="https://github.com/sarathsp06/sparrow" color="#3B82F6">'
        'https://github.com/sarathsp06/sparrow</a></font>', sCaption))

    # BUILD
    doc.build(story)
    print(f"PDF generated: {output_file}")


if __name__ == "__main__":
    build_pdf()
