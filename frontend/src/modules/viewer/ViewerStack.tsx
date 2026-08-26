import { cn } from "@/lib/utils";
import type { Tab, ViewerTab } from "@/modules/tabs";
import { SpreadsheetPane } from "./SpreadsheetPane";

type Props = {
  tabs: Tab[];
  activeId: number;
  onOpenInEditor: (path: string) => void;
};

export function ViewerStack({ tabs, activeId, onOpenInEditor }: Props) {
  const viewers = tabs.filter(
    (t): t is ViewerTab => t.kind === "viewer" && !t.cold,
  );
  if (viewers.length === 0) return null;
  return (
    <div className="relative h-full w-full">
      {viewers.map((t) => {
        const visible = t.id === activeId;
        return (
          <div
            key={t.id}
            className={cn(
              "absolute inset-0",
              !visible && "invisible pointer-events-none",
            )}
            aria-hidden={!visible}
          >
            <SpreadsheetPane
              path={t.path}
              onOpenInEditor={onOpenInEditor}
            />
          </div>
        );
      })}
    </div>
  );
}