import { describe, expect, it } from "vitest";
import { redactSensitive } from "./redact";

const SECRETS: Array<{ name: string; secret: string }> = [
  { name: "openai key", secret: "sk-proj-abcdefghijklmnopqrstuvwxyz012345" },
  { name: "anthropic key", secret: "sk-ant-abcdefghijklmnopqrstuvwxyz012345" },
  { name: "aws access key", secret: "AKIA1234567890ABCDEF" },
  { name: "github token", secret: `ghp_${"a".repeat(36)}` },
  { name: "github pat", secret: `github_pat_${"A".repeat(40)}` },
  { name: "google api key", secret: `AIza${"a".repeat(35)}` },
  { name: "slack token", secret: `xoxb-${"1".repeat(12)}` },
  { name: "stripe key", secret: `sk_live_${"a".repeat(24)}` },
  { name: "jwt", secret: "eyJabcdefgh.ZYXWVUTSR.qwertyuiop" },
];

describe("redactSensitive", () => {
  for (const { name, secret } of SECRETS) {
    it(`removes a ${name} from surrounding text`, () => {
      const out = redactSensitive(`prefix ${secret} suffix`);
      expect(out).not.toContain(secret);
      expect(out).toContain("<REDACTED");
      expect(out.startsWith("prefix ")).toBe(true);
      expect(out.endsWith(" suffix")).toBe(true);
    });
  }

  it("redacts a bearer token while keeping the header name", () => {
    const out = redactSensitive(
      "Authorization: Bearer abcdefghijklmnopqrstuvwx",
    );
    expect(out).not.toContain("abcdefghijklmnopqrstuvwx");
    expect(out).toContain("Authorization:");
  });

  it("redacts an assigned secret value but keeps the key name", () => {
    const out = redactSensitive('MY_SECRET_KEY="hunter2hunter2"');
    expect(out).toContain("MY_SECRET_KEY");
    expect(out).not.toContain("hunter2hunter2");
    expect(out).toContain("<REDACTED>");
  });

  it("redacts a password assignment", () => {
    const out = redactSensitive("DB_PASSWORD=p@ssw0rdlong");
    expect(out).not.toContain("p@ssw0rdlong");
    expect(out).toContain("DB_PASSWORD");
  });

  it("leaves non-sensitive text untouched", () => {
    const text = "just a normal log line with no secrets";
    expect(redactSensitive(text)).toBe(text);
  });

  it("redacts every secret when several appear together", () => {
    const input = `openai sk-proj-${"a".repeat(24)} and aws AKIA1234567890ABCDEF`;
    const out = redactSensitive(input);
    expect(out).not.toContain("AKIA1234567890ABCDEF");
    expect(out).not.toContain(`sk-proj-${"a".repeat(24)}`);
  });

  it("redacts arbitrary *_TOKEN / *_SECRET / *_PASSWORD assignments", () => {
    const input =
      "NPM_TOKEN=npm_abcdefghijklmnop github CI: GITHUB_TOKEN=ghp_xyz123";
    const out = redactSensitive(input);
    expect(out).not.toContain("npm_abcdefghijklmnop");
    expect(out).not.toContain("ghp_xyz123");
    expect(out).toContain("NPM_TOKEN");
    expect(out).toContain("GITHUB_TOKEN");
    expect(out).toContain("<REDACTED>");
  });

  it("redacts an npm token even without its key name", () => {
    const out = redactSensitive("token = npm_0123456789abcdef");
    expect(out).not.toContain("npm_0123456789abcdef");
  });

  it("redacts credentials embedded in a database URL but keeps scheme and host", () => {
    const out = redactSensitive(
      "DATABASE_URL=postgres://admin:hunter2@db.internal:5432/prod",
    );
    expect(out).not.toContain("hunter2");
    expect(out).not.toContain("admin:");
    expect(out).toContain("postgres://<REDACTED>@db.internal");
  });

  it("redacts PEM / OpenSSH private key blocks wholesale", () => {
    const pem = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
xxxxxx
-----END RSA PRIVATE KEY-----`;
    const out = redactSensitive(`found here:\n${pem}\nend`);
    expect(out).not.toContain("MIIEowIBAAKCAQEA");
    expect(out).toContain("<REDACTED:private-key-block>");
  });

  it("redacts OpenSSH private key blocks", () => {
    const block = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQ
-----END OPENSSH PRIVATE KEY-----`;
    const out = redactSensitive(block);
    expect(out).not.toContain("b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQ");
  });
});
