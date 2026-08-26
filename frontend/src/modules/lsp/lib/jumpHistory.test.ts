import { describe, expect, it } from "vitest";
import { JumpHistory, type JumpPos } from "./jumpHistory";

const pos = (uri: string, line: number, character = 0): JumpPos => ({
  uri,
  line,
  character,
});

describe("JumpHistory", () => {
  it("starts empty with no back/forward movement", () => {
    const h = new JumpHistory();
    expect(h.size()).toBe(0);
    expect(h.canBack()).toBe(false);
    expect(h.canForward()).toBe(false);
    expect(h.back()).toBeNull();
    expect(h.forward()).toBeNull();
  });

  it("walks back then forward across recorded origins", () => {
    const h = new JumpHistory();
    h.push(pos("a", 1));
    h.push(pos("a", 5));
    h.push(pos("b", 2));

    expect(h.canForward()).toBe(false);
    // back to the origin of the latest jump
    expect(h.back()).toEqual(pos("a", 5));
    expect(h.canBack()).toBe(true);
    expect(h.back()).toEqual(pos("a", 1));
    expect(h.canBack()).toBe(false);
    expect(h.back()).toBeNull();

    // forward back out again
    expect(h.canForward()).toBe(true);
    expect(h.forward()).toEqual(pos("a", 5));
    expect(h.forward()).toEqual(pos("b", 2));
    expect(h.canForward()).toBe(false);
    expect(h.forward()).toBeNull();
  });

  it("truncates the forward tail when a new jump is pushed", () => {
    const h = new JumpHistory();
    h.push(pos("a", 1));
    h.push(pos("a", 5));
    expect(h.back()).toEqual(pos("a", 1));

    // A new jump after going back must discard the (now-forward) entry.
    h.push(pos("c", 9));
    expect(h.size()).toBe(2);
    expect(h.canForward()).toBe(false);
    expect(h.back()).toEqual(pos("a", 1));
  });

  it("binds size to MAX_ENTRIES and stays consistent", () => {
    const h = new JumpHistory();
    for (let i = 0; i < 210; i++) h.push(pos("a", i));
    expect(h.size()).toBe(200);
    // Oldest entries were dropped, so back walks only the newest 199.
    let count = 0;
    while (h.back()) count++;
    expect(count).toBe(199);
  });
});
