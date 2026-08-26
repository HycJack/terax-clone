/**
 * Robust delimiter-separated value parser (CSV/TSV). Handles quoted fields,
 * embedded delimiters, escaped quotes and CRLF / lone-CR line endings.
 */
export function parseDelimited(
  text: string,
  delimiter: string,
): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let field = "";
  let inQuotes = false;
  let i = 0;
  const n = text.length;

  const endField = () => {
    row.push(field);
    field = "";
  };
  const endRow = () => {
    endField();
    rows.push(row);
    row = [];
  };

  while (i < n) {
    const ch = text[i];
    if (inQuotes) {
      if (ch === '"') {
        if (text[i + 1] === '"') {
          field += '"';
          i += 1;
        } else {
          inQuotes = false;
        }
      } else {
        field += ch;
      }
    } else if (ch === '"') {
      inQuotes = true;
    } else if (ch === delimiter) {
      endField();
    } else if (ch === "\n") {
      endRow();
    } else if (ch === "\r") {
      if (text[i + 1] === "\n") i += 1;
      endRow();
    } else {
      field += ch;
    }
    i += 1;
  }
  if (field !== "" || row.length > 0) endRow();
  return rows;
}

/** Detect CSV vs TSV by counting delimiters in the first non-empty line. */
export function sniffDelimiter(text: string): string {
  const firstLine = text.split(/\r?\n/).find((l) => l.trim().length > 0) ?? "";
  const quoted = (s: string) =>
    s.replace(/"[^"]*"/g, (m) => m.replace(/[,;\t]/g, "x"));
  const tabs = (quoted(firstLine).match(/\t/g) ?? []).length;
  const commas = (quoted(firstLine).match(/,/g) ?? []).length;
  const semis = (quoted(firstLine).match(/;/g) ?? []).length;
  if (tabs > commas && tabs > semis) return "\t";
  if (semis > commas) return ";";
  return ",";
}

export function formatCell(value: unknown): string {
  if (value == null || value === "") return "";
  if (value instanceof Date) {
    const d = value as Date;
    if (Number.isNaN(d.getTime())) return String(value);
    return d.toLocaleString();
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? String(value) : String(value);
  }
  return String(value);
}