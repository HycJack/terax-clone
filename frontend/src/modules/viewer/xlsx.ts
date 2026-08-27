import * as XLSX from "xlsx";

const MAX_ROWS = 50_000;
const MAX_COLS = 1_000;

export type WorkbookLike = {
  buffer: ArrayBuffer;
  sheets: string[];
};

/** Read a workbook and keep the buffer so sheets can be re-extracted. */
export function readWorkbook(
  buffer: ArrayBuffer,
): WorkbookLike {
  const wb = XLSX.read(buffer, { type: "array" });
  return { buffer, sheets: wb.SheetNames };
}

export function extractSheet(
  buffer: ArrayBuffer,
  sheetName: string,
): { rows: string[][]; truncated: boolean; totalRows: number; totalCols: number } {
  const wb = XLSX.read(buffer, { type: "array" });
  const ws = wb.Sheets[sheetName];
  const raw = (ws
    ? XLSX.utils.sheet_to_json(ws, { header: 1, defval: "", raw: true })
    : []) as unknown[][];

  let rows = raw.map((r) =>
    r.map((c) => {
      if (c instanceof Date && !Number.isNaN(c.getTime())) {
        return c.toLocaleString();
      }
      return c == null ? "" : String(c);
    }),
  );
  let truncated = false;
  if (rows.length > MAX_ROWS) {
    rows = rows.slice(0, MAX_ROWS);
    truncated = true;
  }
  let totalCols = 0;
  for (const r of rows) totalCols = Math.max(totalCols, r.length);
  if (totalCols > MAX_COLS) {
    rows = rows.map((r) => r.slice(0, MAX_COLS));
    totalCols = MAX_COLS;
    truncated = true;
  }
  return {
    rows,
    truncated,
    totalRows: raw.length,
    totalCols,
  };
}