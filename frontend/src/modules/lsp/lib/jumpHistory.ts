/**
 * Editor jump history: a global back/forward stack over navigation positions
 * (currently driven by go-to-definition / find-references jumps).
 *
 * Each entry records a document URI + 0-based line/character. The store is
 * module-global so back/forward can span multiple open editors/files, mirroring
 * how desktop IDEs let you retrace where a definition jump came from
 * (Ctrl+Alt+← / Ctrl+Alt+→ in the editor).
 */

export type JumpPos = {
  /** file:// URI of the document. */
  uri: string;
  /** 0-based */
  line: number;
  /** 0-based */
  character: number;
};

const MAX_ENTRIES = 200;

/**
 * Pure, unit-testable navigation stack.
 *
 * `push(origin)` records where the cursor *was* before a jump and truncates
 * any forward tail. `back`/`forward` move an internal index and return the
 * position to restore without mutating the recorded entries.
 */
export class JumpHistory {
  private stack: JumpPos[] = [];
  private index = -1;

  push(origin: JumpPos): void {
    // Dropping the forward tail makes a new jump the new "now".
    this.stack = this.stack.slice(0, this.index + 1);
    this.stack.push(origin);
    if (this.stack.length > MAX_ENTRIES) {
      // Keep growth bounded; dropping the oldest entry shifts the index.
      this.stack.shift();
      this.index -= 1;
    } else {
      this.index = this.stack.length - 1;
    }
  }

  canBack(): boolean {
    return this.index > 0;
  }

  canForward(): boolean {
    return this.index >= 0 && this.index < this.stack.length - 1;
  }

  back(): JumpPos | null {
    if (!this.canBack()) return null;
    this.index -= 1;
    return this.stack[this.index];
  }

  forward(): JumpPos | null {
    if (!this.canForward()) return null;
    this.index += 1;
    return this.stack[this.index];
  }

  /** Number of recorded entries (for tests/debug). */
  size(): number {
    return this.stack.length;
  }
}

/** Shared instance used by the editor LSP extension. */
export const jumpHistory = new JumpHistory();
