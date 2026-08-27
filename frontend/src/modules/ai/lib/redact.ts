type Rule = {
  kind: string;
  re: RegExp;
  /** Replace callback — used by rules that need to keep part of the match
   *  (e.g. an env var's key name). Defaults to `<REDACTED:<kind>>`. */
  to?: (match: string, ...groups: string[]) => string;
};

const PATTERNS: Rule[] = [
  { kind: "openai-key", re: /\bsk-(?:proj-)?[A-Za-z0-9_-]{20,}\b/g },
  { kind: "anthropic-key", re: /\bsk-ant-[A-Za-z0-9_-]{20,}\b/g },
  { kind: "aws-access-key", re: /\b(?:AKIA|ASIA)[0-9A-Z]{16}\b/g },
  { kind: "github-token", re: /\bgh[opsur]_[A-Za-z0-9]{36,}\b/g },
  { kind: "github-pat", re: /\bgithub_pat_[A-Za-z0-9_]{40,}\b/g },
  { kind: "google-api-key", re: /\bAIza[0-9A-Za-z_-]{35}\b/g },
  { kind: "slack-token", re: /\bxox[bpsare]-[A-Za-z0-9-]{10,}\b/g },
  { kind: "stripe-key", re: /\b(?:sk|pk|rk)_(?:live|test)_[A-Za-z0-9]{24,}\b/g },
  { kind: "npm-token", re: /\bnpm_[A-Za-z0-9]{10,}\b/g },
  { kind: "jwt", re: /\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b/g },
  { kind: "bearer", re: /\bBearer\s+[A-Za-z0-9._-]{20,}/g },
  {
    // Key-value assignments whose key name is clearly secret-bearing:
    // MY_API_KEY=…, NPM_TOKEN=…, AWS_SECRET_ACCESS_TOKEN=…, DB_PASSWORD=…,
    // CLIENT_SECRET=…, etc. `TOKEN` / `SECRET` / `PASSWORD` are matched as
    // bare suffixes so arbitrary names (NPM_TOKEN, GITHUB_TOKEN, …) are
    // caught, not just the enumerated prefixes.
    kind: "env-assign",
    re: /\b((?:[A-Z][A-Z0-9_]*)?(?:API[_-]?KEY|SECRET(?:[_-]?KEY)?|ACCESS[_-]?TOKEN|AUTH[_-]?TOKEN|TOKEN|PASSWORD|PASSWD|PRIVATE[_-]?KEY|CLIENT[_-]?SECRET)[A-Z0-9_]*)\s*[:=]\s*(["']?)([^\s"';|&]+)\2/gi,
    to: (_m, name, q, _val) => `${name}=${q}<REDACTED>${q}`,
  },
  {
    // Credentials embedded in URLs: scheme://user:pass@host — keeps the
    // scheme + host (useful context), redacts the userinfo. Covers
    // DATABASE_URL=postgres://user:pass@… and bare occurrences.
    kind: "url-userinfo",
    re: /\b([a-z][a-z0-9+.-]*:\/\/)[^\s/'"`]*:[^\s@'"`]*@/gi,
    to: (_m, scheme) => `${scheme}<REDACTED>@`,
  },
  {
    // PEM / OpenSSH / PGP private key blocks (RSA, EC, DSA, OPENSSH, PGP).
    kind: "private-key-block",
    re: /-----BEGIN (?:[A-Z0-9 ]+ )?PRIVATE KEY(?: BLOCK)?-----[\s\S]*?-----END (?:[A-Z0-9 ]+ )?PRIVATE KEY(?: BLOCK)?-----/g,
  },
];

export function redactSensitive(text: string): string {
  let out = text;
  for (const { kind, re, to } of PATTERNS) {
    out = to
      ? out.replace(re, (m, ...groups: string[]) => to(m, ...groups))
      : out.replace(re, `<REDACTED:${kind}>`);
  }
  return out;
}