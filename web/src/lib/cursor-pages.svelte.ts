// Paging state for the cursor-paged lists (deliveries, event occurrences).
// The API returns no total, only has_more and next_cursor, so the UI pages
// Newer/Older: the cursors of the pages already seen are kept on a stack to
// step back. A filter change starts again from the newest page (reset).
export class CursorPages {
  /** Cursor of the page shown; empty for the newest page. */
  cursor = $state("");
  /** Cursor of the next, older page, from the last response; empty on the last page. */
  next = $state("");
  #history: string[] = $state([]);

  /** 1-based number of the page shown. */
  get page(): number {
    return this.#history.length + 1;
  }
  get hasNewer(): boolean {
    return this.#history.length > 0;
  }
  get hasOlder(): boolean {
    return this.next !== "";
  }

  /** Record the response's pagination. */
  update(p: { has_more: boolean; next_cursor?: string } | undefined) {
    this.next = p?.has_more ? (p.next_cursor ?? "") : "";
  }
  older() {
    if (!this.next) return;
    this.#history = [...this.#history, this.cursor];
    this.cursor = this.next;
  }
  newer() {
    if (!this.#history.length) return;
    this.cursor = this.#history[this.#history.length - 1];
    this.#history = this.#history.slice(0, -1);
  }
  reset() {
    this.cursor = "";
    this.next = "";
    this.#history = [];
  }
}
