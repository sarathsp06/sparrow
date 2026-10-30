// GitHub-style callouts for the field notes: a blockquote that starts with
// [!NOTE], [!TIP], [!IMPORTANT], [!WARNING] or [!CAUTION] renders as
//
//   <div class="markdown-alert markdown-alert-note">
//     <p class="markdown-alert-title">Note</p> …
//
// styled in blog.css. Docs pages are left alone: they use Starlight's :::note.
// A Sätteri mdast plugin (Astro's default Markdown processor).

const marker = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/i;

export const blogAlerts = {
  name: 'sparrow-blog-alerts',
  blockquote(node, ctx) {
    if (!ctx.fileURL?.pathname.includes('/src/blog/')) return;
    const first = node.children[0];
    const text = first?.type === 'paragraph' ? first.children[0] : undefined;
    const match = text?.type === 'text' && text.value.match(marker);
    if (!match) return;

    const type = match[1].toLowerCase();
    const rest = text.value.slice(match[0].length);
    if (rest) ctx.setProperty(text, 'value', rest);
    // A marker on a line of its own leaves the paragraph empty: drop it.
    else if (first.children.length === 1) ctx.removeNode(first);
    else ctx.removeNode(text);

    ctx.setProperty(node, 'data', { hName: 'div', hProperties: { className: ['markdown-alert', `markdown-alert-${type}`] } });
    ctx.prependChild(node, {
      type: 'paragraph',
      data: { hProperties: { className: ['markdown-alert-title'] } },
      children: [{ type: 'text', value: type[0].toUpperCase() + type.slice(1) }],
    });
  },
};
