import { useCallback, useEffect, useMemo, useState } from "react";
import { loadSpreadsheet, type SpreadsheetData } from "./spreadsheet";
import { extractSheet } from "./xlsx";

function basename(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

type Props = {
  path: string;
  /** Open the same file in the normal text editor (CSV/TSV). */
  onOpenInEditor?: (path: string) => void;
};

const MAX_CELLS_PER_ROW = 120;

export function SpreadsheetPane({ path, onOpenInEditor }: Props) {
  const [data, setData] = useState<SpreadsheetData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    loadSpreadsheet(path)
      .then((d) => {
        if (cancelled) return;
        setData(d);
        setLoading(false);
      })
      .catch((e) => {
        if (cancelled) return;
        setError(e instanceof Error ? e.message : String(e));
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [path]);

  const switchSheet = useCallback(
    (name: string) => {
      setData((prev) => {
        if (!prev?.buffer || !prev.sheets.includes(name)) return prev;
        const res = extractSheet(prev.buffer, name);
        return {
          ...prev,
          sheet: name,
          rows: res.rows,
          truncated: res.truncated,
          truncatedAt: { rows: res.totalRows, cols: res.totalCols },
        };
      });
    },
    [],
  );

  const colCount = data?.rows[0]?.length ?? 0;
  const rowCount = data?.rows.length ?? 0;
  const clippedCols = colCount > MAX_CELLS_PER_ROW;

  const cells = useMemo(
    () => data?.rows.slice(0, MAX_CELLS_PER_ROW) ?? [],
    [data],
  );

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-9 shrink-0 items-center gap-3 border-b border-border/60 bg-card px-3 text-xs">
        <span className="font-medium text-foreground">{basename(path)}</span>
        {data && data.sheets.length > 1 && (
          <select
            value={data.sheet}
            onChange={(e) => switchSheet(e.target.value)}
            className="rounded-md border border-border bg-background px-1.5 py-0.5 text-xs text-foreground outline-none focus:border-primary"
          >
            {data.sheets.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        )}
        {data && (
          <span className="text-muted-foreground">
            {rowCount.toLocaleString()} rows × {colCount.toLocaleString()} cols
            {data.truncated && " · truncated"}
          </span>
        )}
        <div className="flex-1" />
        {onOpenInEditor && (
          <button
            type="button"
            onClick={() => onOpenInEditor(path)}
            className="rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground"
            title="Open in text editor"
          >
            Open in editor
          </button>
        )}
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
        {loading && (
          <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
            Loading…
          </div>
        )}
        {error && (
          <div className="flex h-full items-center justify-center px-6 text-center text-xs text-destructive">
            {error}
          </div>
        )}
        {!loading && !error && data && (
          <table className="border-collapse text-xs">
            <thead>
              <tr>
                <th className="sticky left-0 z-20 border border-border bg-muted px-2 py-1 text-left font-medium text-muted-foreground">
                  #
                </th>
                {cells[0]?.map((_, ci) => (
                  <th
                    // biome-ignore lint/suspicious/noArrayIndexKey: static column snapshot
                    key={ci}
                    className="sticky top-0 z-10 border border-border bg-muted px-2 py-1 text-left font-medium text-muted-foreground"
                  >
                    {colLabel(ci)}
                  </th>
                ))}
                {clippedCols && (
                  <th className="border border-border bg-muted px-2 py-1 text-left font-medium text-muted-foreground">
                    …
                  </th>
                )}
              </tr>
            </thead>
            <tbody>
              {cells.map((row, ri) => (
                <tr
                  // biome-ignore lint/suspicious/noArrayIndexKey: static row snapshot
                  key={ri}
                  className={ri % 2 ? "bg-background" : "bg-muted/20"}
                >
                  <td className="sticky left-0 z-10 border border-border bg-muted/60 px-2 py-1 text-muted-foreground">
                    {ri + 1}
                  </td>
                  {row.map((cell, ci) => (
                    <td
                      // biome-ignore lint/suspicious/noArrayIndexKey: static cell snapshot
                      key={ci}
                      className="max-w-[320px] border border-border px-2 py-1 whitespace-pre-wrap break-words text-foreground"
                      title={cell}
                    >
                      {cell || "\u00A0"}
                    </td>
                  ))}
                  {clippedCols && (
                    <td className="border border-border px-2 py-1 text-muted-foreground">
                      …
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

function colLabel(i: number): string {
  let label = "";
  let n = i;
  do {
    label = String.fromCharCode(65 + (n % 26)) + label;
    n = Math.floor(n / 26) - 1;
  } while (n >= 0);
  return label;
}