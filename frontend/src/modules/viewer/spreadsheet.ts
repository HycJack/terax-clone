import { convertFileSrc } from "@/lib/wails/core";
import { parseDelimited, sniffDelimiter } from "./dsv";
import { extractSheet, readWorkbook } from "./xlsx";

export type SpreadsheetData = {
  kind: "csv" | "workbook";
  /** Sheet names (empty for CSV). */
  sheets: string[];
  /** Active sheet ("" for CSV). */
  sheet: string;
  rows: string[][];
  truncated: boolean;
  truncatedAt: { rows: number; cols: number };
  /** Retained for sheet switching in workbooks. */
  buffer?: ArrayBuffer;
};

const MAX_ROWS = 50_000;
const MAX_COLS = 1_000;

export function isSpreadsheetPath(path: string): boolean {
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  return ["csv", "tsv", "xlsx", "xls", "ods"].includes(ext);
}

async function fetchBuffer(path: string): Promise<ArrayBuffer> {
  const url = convertFileSrc(path);
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Failed to load (HTTP ${res.status})`);
  return await res.arrayBuffer();
}

function trimRows(rows: string[][]): {
  rows: string[][];
  truncated: boolean;
  totalCols: number;
} {
  let totalCols = 0;
  for (const r of rows) totalCols = Math.max(totalCols, r.length);
  let truncated = false;
  if (rows.length > MAX_ROWS) {
    rows = rows.slice(0, MAX_ROWS);
    truncated = true;
  }
  if (totalCols > MAX_COLS) {
    rows = rows.map((r) => r.slice(0, MAX_COLS));
    totalCols = MAX_COLS;
    truncated = true;
  }
  return { rows, truncated, totalCols };
}

export async function loadSpreadsheet(
  path: string,
): Promise<SpreadsheetData> {
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  const buffer = await fetchBuffer(path);

  if (ext === "xlsx" || ext === "xls" || ext === "ods") {
    const wb = readWorkbook(buffer);
    const sheet = wb.sheets[0] ?? "";
    const res = extractSheet(buffer, sheet);
    return {
      kind: "workbook",
      sheets: wb.sheets,
      sheet,
      rows: res.rows,
      truncated: res.truncated,
      truncatedAt: { rows: res.totalRows, cols: res.totalCols },
      buffer,
    };
  }

  const text = new TextDecoder("utf-8").decode(buffer);
  const delimiter = sniffDelimiter(text);
  const all = parseDelimited(text, delimiter);
  const { rows, truncated, totalCols } = trimRows(all);
  return {
    kind: "csv",
    sheets: [],
    sheet: "",
    rows,
    truncated,
    truncatedAt: { rows: all.length, cols: totalCols },
  };
}