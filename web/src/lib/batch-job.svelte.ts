// A bulk re-push or bulk retry: the list page prepares a snapshot (the list
// call with prepare_repush / prepare_retry), the user confirms, then this
// starts the job and polls its progress until it ends. Job routes are
// id-scoped and ignore the path consumer, so "default" stands in when the
// page shows all consumers.
import { api, unwrap } from "./services";
import { formatAPIError } from "./utils";

export type BatchKind = "repush" | "retry";

export interface BatchProgressState {
  status: string;
  total: number;
  processed: number;
  failed: number;
}

const progressOf = (r: BatchProgressState): BatchProgressState => ({
  status: r.status,
  total: r.total,
  processed: r.processed,
  failed: r.failed,
});

const terminal = new Set(["completed", "failed", "cancelled"]);

export class BatchJob {
  /** Prepared snapshot id; empty when nothing is prepared. */
  id = $state("");
  /** Items in the prepared snapshot. */
  total = $state(0);
  /** The confirm dialog is open. */
  confirming = $state(false);
  /** Progress of the started job, for BatchProgress. */
  progress = $state<BatchProgressState | undefined>();
  #timer: ReturnType<typeof setInterval> | undefined;
  private kind: BatchKind;
  private consumer: () => string;
  private onerror: (msg: string) => void;

  constructor(kind: BatchKind, consumer: () => string, onerror: (msg: string) => void) {
    this.kind = kind;
    this.consumer = consumer;
    this.onerror = onerror;
  }

  get #label() {
    return this.kind === "repush" ? "re-push" : "retry";
  }

  /** Records a prepared snapshot and opens the confirm dialog; false when nothing matched. */
  prepared(id: string | undefined, total: number | undefined): boolean {
    if (!id) return false;
    this.id = id;
    this.total = total || 0;
    this.confirming = true;
    return true;
  }

  async start() {
    this.confirming = false;
    if (!this.id) return;
    const path = { consumer: this.consumer() || "default" };
    try {
      const res =
        this.kind === "repush"
          ? unwrap(await api.POST("/v1/consumers/{consumer}/events:rePush", { params: { path }, body: { repush_id: this.id } }))
          : unwrap(await api.POST("/v1/consumers/{consumer}/deliveries:retryBatch", { params: { path }, body: { retry_id: this.id } }));
      this.progress = progressOf(res);
      this.#poll();
    } catch (e) {
      this.onerror(formatAPIError(e, `Failed to start ${this.#label}`));
    }
  }

  #poll() {
    this.stop();
    this.#timer = setInterval(async () => {
      if (!this.id) return this.stop();
      const path = { consumer: this.consumer() || "default", job_id: this.id };
      try {
        const res =
          this.kind === "repush"
            ? unwrap(await api.GET("/v1/consumers/{consumer}/repush-jobs/{job_id}", { params: { path } }))
            : unwrap(await api.GET("/v1/consumers/{consumer}/retry-jobs/{job_id}", { params: { path } }));
        this.progress = progressOf(res);
        if (terminal.has(res.status)) this.stop();
      } catch {
        this.stop();
      }
    }, 2000);
  }

  /** Stops polling; call from onDestroy. */
  stop() {
    if (this.#timer) clearInterval(this.#timer);
    this.#timer = undefined;
  }

  #cancel(id: string) {
    const path = { consumer: this.consumer() || "default", job_id: id };
    return this.kind === "repush"
      ? api.POST("/v1/consumers/{consumer}/repush-jobs/{job_id}:cancel", { params: { path } })
      : api.POST("/v1/consumers/{consumer}/retry-jobs/{job_id}:cancel", { params: { path } });
  }

  /** Cancels the running job. */
  async cancel() {
    if (!this.id) return;
    try {
      unwrap(await this.#cancel(this.id));
    } catch (e) {
      this.onerror(formatAPIError(e, `Failed to cancel ${this.#label}`));
    }
  }

  /**
   * Closes the confirm dialog and cancels the prepared snapshot so it doesn't
   * linger as a pending batch job. Best effort: it expires anyway.
   */
  dismiss() {
    this.confirming = false;
    const id = this.id;
    this.id = "";
    if (id) this.#cancel(id).catch(() => {});
  }
}
