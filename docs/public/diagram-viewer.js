// Click a rendered Mermaid diagram to open it in a full-window dialog with
// zoom (buttons, wheel, pinch) and drag-to-pan. astro-mermaid renders the
// diagrams in the browser, so this listens on the document rather than
// binding to diagrams that may not exist yet.
(() => {
  const MIN = 0.25;
  const MAX = 8;
  let dialog, stage, canvas, label;
  let scale = 1, x = 0, y = 0;
  const pointers = new Map();
  let pinchStart = null;

  function apply() {
    canvas.style.transform = `translate(${x}px, ${y}px) scale(${scale})`;
    label.textContent = `${Math.round(scale * 100)}%`;
  }

  // Zoom by factor around a point in stage coordinates (defaults to center).
  function zoom(factor, cx, cy) {
    const rect = stage.getBoundingClientRect();
    cx ??= rect.width / 2;
    cy ??= rect.height / 2;
    const next = Math.min(MAX, Math.max(MIN, scale * factor));
    const k = next / scale;
    x = cx - (cx - x) * k;
    y = cy - (cy - y) * k;
    scale = next;
    apply();
  }

  // The diagram's natural size, from its viewBox.
  function size() {
    const svg = canvas.querySelector('svg');
    const box = svg.viewBox?.baseVal;
    const w = box?.width || svg.getBoundingClientRect().width / scale;
    const h = box?.height || svg.getBoundingClientRect().height / scale;
    svg.setAttribute('width', w);
    svg.setAttribute('height', h);
    return { w, h, rect: stage.getBoundingClientRect() };
  }

  // Fit the whole diagram inside the stage, centered.
  function fit() {
    const { w, h, rect } = size();
    scale = Math.min((rect.width - 48) / w, (rect.height - 48) / h, 2);
    x = (rect.width - w * scale) / 2;
    y = (rect.height - h * scale) / 2;
    apply();
  }

  // Open readable: fit the width (at most 150%), anchored at the top, so a
  // tall diagram is panned through instead of shrunk to fit the height.
  function fitWidth() {
    const { w, h, rect } = size();
    scale = Math.min((rect.width - 48) / w, 1.5);
    x = (rect.width - w * scale) / 2;
    y = h * scale <= rect.height - 48 ? (rect.height - h * scale) / 2 : 24;
    apply();
  }

  function button(text, title, onClick) {
    const b = document.createElement('button');
    b.type = 'button';
    b.textContent = text;
    b.title = title;
    b.setAttribute('aria-label', title);
    b.addEventListener('click', onClick);
    return b;
  }

  function build() {
    dialog = document.createElement('dialog');
    dialog.className = 'diagram-viewer';
    dialog.setAttribute('aria-label', 'Diagram viewer');

    const bar = document.createElement('div');
    bar.className = 'diagram-viewer-bar';
    label = document.createElement('span');
    label.className = 'diagram-viewer-zoom';
    bar.append(
      button('−', 'Zoom out', () => zoom(1 / 1.25)),
      label,
      button('+', 'Zoom in', () => zoom(1.25)),
      button('Fit', 'Fit to window', fit),
      button('✕', 'Close', () => dialog.close()),
    );

    stage = document.createElement('div');
    stage.className = 'diagram-viewer-stage';
    canvas = document.createElement('div');
    canvas.className = 'diagram-viewer-canvas';
    stage.append(canvas);

    stage.addEventListener('wheel', (e) => {
      e.preventDefault();
      const r = stage.getBoundingClientRect();
      zoom(Math.exp(-e.deltaY * 0.0015), e.clientX - r.left, e.clientY - r.top);
    }, { passive: false });

    stage.addEventListener('pointerdown', (e) => {
      stage.setPointerCapture(e.pointerId);
      pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pointers.size === 2) {
        const [a, b] = [...pointers.values()];
        pinchStart = { d: Math.hypot(a.x - b.x, a.y - b.y), scale };
      }
    });
    stage.addEventListener('pointermove', (e) => {
      const prev = pointers.get(e.pointerId);
      if (!prev) return;
      pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (pointers.size === 2 && pinchStart) {
        const [a, b] = [...pointers.values()];
        const r = stage.getBoundingClientRect();
        const target = pinchStart.scale * (Math.hypot(a.x - b.x, a.y - b.y) / pinchStart.d);
        zoom(target / scale, (a.x + b.x) / 2 - r.left, (a.y + b.y) / 2 - r.top);
      } else if (pointers.size === 1) {
        x += e.clientX - prev.x;
        y += e.clientY - prev.y;
        apply();
      }
    });
    const release = (e) => {
      pointers.delete(e.pointerId);
      if (pointers.size < 2) pinchStart = null;
    };
    stage.addEventListener('pointerup', release);
    stage.addEventListener('pointercancel', release);

    dialog.addEventListener('keydown', (e) => {
      if (e.key === '+' || e.key === '=') zoom(1.25);
      else if (e.key === '-') zoom(1 / 1.25);
      else if (e.key === '0') fit();
    });
    // A click on the backdrop (outside the panel) closes the dialog.
    dialog.addEventListener('click', (e) => {
      if (e.target === dialog) dialog.close();
    });
    dialog.addEventListener('close', () => {
      canvas.replaceChildren();
      document.documentElement.style.overflow = '';
    });

    dialog.append(bar, stage);
    document.body.append(dialog);
  }

  function open(svg) {
    if (!dialog) build();
    const copy = svg.cloneNode(true);
    copy.removeAttribute('style');
    copy.setAttribute('aria-hidden', 'false');
    canvas.replaceChildren(copy);
    document.documentElement.style.overflow = 'hidden';
    dialog.showModal();
    fitWidth();
  }

  function diagramFrom(target) {
    const pre = target.closest?.('pre.mermaid[data-processed]');
    return pre?.querySelector('svg') ?? null;
  }

  document.addEventListener('click', (e) => {
    const svg = diagramFrom(e.target);
    if (svg) open(svg);
  });
  // Rendered diagrams are focusable and open with Enter or Space.
  document.addEventListener('keydown', (e) => {
    if (e.key !== 'Enter' && e.key !== ' ') return;
    const svg = diagramFrom(e.target);
    if (!svg) return;
    e.preventDefault();
    open(svg);
  });
  new MutationObserver(() => {
    for (const pre of document.querySelectorAll('pre.mermaid[data-processed]:not([tabindex])')) {
      pre.tabIndex = 0;
      pre.setAttribute('role', 'button');
      pre.setAttribute('aria-label', `${pre.querySelector('svg title')?.textContent || 'Diagram'}: open enlarged`);
    }
  }).observe(document.documentElement, { subtree: true, attributes: true, attributeFilter: ['data-processed'] });
})();
