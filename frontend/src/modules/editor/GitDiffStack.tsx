import type { GitCommitFileDiffTab, GitDiffTab, Tab } from "@/modules/tabs";
import { useMemo } from "react";
import { GitDiffPane } from "./GitDiffPane";

type Props = {
  tabs: Tab[];
  activeId: number;
};

export function GitDiffStack({ tabs, activeId }: Props) {
  const active = tabs.find(
    (t): t is GitDiffTab | GitCommitFileDiffTab =>
      (t.kind === "git-diff" || t.kind === "git-commit-file") &&
      t.id === activeId,
  );

  // The diff pane's load effect depends on `source`, so it must stay
  // referentially stable across re-renders. The parent surface re-renders on
  // a 2s source-control TTL (and other store updates); an inline object
  // literal would change identity every render and reset the fetch back to
  // its spinner before it ever completes. Memoize on the tab's primitive
  // identity fields instead.
  const source = useMemo((): Parameters<typeof GitDiffPane>[0]["source"] | null => {
    if (!active) return null;
    if (active.kind === "git-diff") {
      return {
        kind: "working",
        repoRoot: active.repoRoot,
        path: active.path,
        mode: active.mode,
        originalPath: active.originalPath,
      };
    }
    return {
      kind: "commit",
      repoRoot: active.repoRoot,
      sha: active.sha,
      path: active.path,
      originalPath: active.originalPath,
    };
  }, [
    active?.id,
    active?.kind,
    active?.repoRoot,
    active?.path,
    active && active.kind === "git-diff" ? active.mode : undefined,
    active && active.kind === "git-diff" ? active.originalPath : undefined,
    active && active.kind === "git-commit-file" ? active.sha : undefined,
    active && active.kind === "git-commit-file" ? active.originalPath : undefined,
  ]);

  if (!active) return null;

  return (
    <div className="h-full w-full">
      <GitDiffPane
        key={active.id}
        active
        source={source!}
        chipLabel={active.kind === "git-commit-file" ? active.shortSha : undefined}
      />
    </div>
  );
}
