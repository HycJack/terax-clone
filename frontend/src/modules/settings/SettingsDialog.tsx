import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { IS_MAC } from "@/lib/platform";
import type { SettingsTab } from "@/modules/settings/openSettingsWindow";
import { usePreferencesStore } from "@/modules/settings/preferences";
import {
  AiScanIcon,
  ArrowLeft01Icon,
  InformationCircleIcon,
  KeyboardIcon,
  PaintBoardIcon,
  Settings01Icon,
  SourceCodeIcon,
  UserMultiple02Icon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { type JSX, useEffect, useState } from "react";
import { AboutSection } from "@/settings/sections/AboutSection";
import { AgentsSection } from "@/settings/sections/AgentsSection";
import { EditorSection } from "@/settings/sections/EditorSection";
import { GeneralSection } from "@/settings/sections/GeneralSection";
import { ModelsSection } from "@/settings/sections/ModelsSection";
import { ShortcutsSection } from "@/settings/sections/ShortcutsSection";
import { ThemesSection } from "@/settings/sections/ThemesSection";
import { SectionErrorBoundary } from "@/settings/components/SectionErrorBoundary";

const TABS: {
  id: SettingsTab;
  label: string;
  icon: typeof Settings01Icon;
  component: () => JSX.Element;
}[] = [
  {
    id: "general",
    label: "General",
    icon: Settings01Icon,
    component: GeneralSection,
  },
  {
    id: "editor",
    label: "Editor",
    icon: SourceCodeIcon,
    component: EditorSection,
  },
  {
    id: "themes",
    label: "Themes",
    icon: PaintBoardIcon,
    component: ThemesSection,
  },
  {
    id: "shortcuts",
    label: "Shortcuts",
    icon: KeyboardIcon,
    component: ShortcutsSection,
  },
  { id: "models", label: "Models", icon: AiScanIcon, component: ModelsSection },
  {
    id: "agents",
    label: "Agents",
    icon: UserMultiple02Icon,
    component: AgentsSection,
  },
  {
    id: "about",
    label: "About",
    icon: InformationCircleIcon,
    component: AboutSection,
  },
];

type Props = {
  open: boolean;
  initialTab?: SettingsTab;
  onOpenChange: (open: boolean) => void;
};

export function SettingsDialog({ open, initialTab, onOpenChange }: Props) {
  const [active, setActive] = useState<SettingsTab>(initialTab ?? "general");
  const init = usePreferencesStore((s) => s.init);
  const ActiveSection = TABS.find((t) => t.id === active)?.component;

  useEffect(() => {
    void init();
  }, [init]);

  // Sync initialTab prop changes
  useEffect(() => {
    if (initialTab && open) {
      setActive(initialTab);
    }
  }, [initialTab, open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="!left-0 !top-0 !translate-x-0 !translate-y-0 flex h-screen w-screen max-w-none flex-col gap-0 overflow-hidden rounded-none p-0 sm:max-w-none"
        aria-describedby={undefined}
        showCloseButton={false}
      >
        <header className="relative flex h-11 shrink-0 items-center border-b border-border/60 bg-card/60 px-3">
          {/* Back button in the top-left corner; on macOS keep it clear of the
              native traffic lights that overlay this header. */}
          <Button
            variant="ghost"
            size="icon-sm"
            className={cn(
              "absolute z-10 shrink-0",
              IS_MAC ? "left-20" : "left-3",
            )}
            onClick={() => onOpenChange(false)}
          >
            <HugeiconsIcon icon={ArrowLeft01Icon} size={16} strokeWidth={2} />
          </Button>

          <Tabs
            value={active}
            onValueChange={(v) => setActive(v as SettingsTab)}
            orientation="horizontal"
            className="flex-1 items-center"
          >
            <TabsList className="mx-auto h-7 bg-muted/40 px-2">
              {TABS.map((t) => (
                <TabsTrigger
                  key={t.id}
                  value={t.id}
                  className="h-6 gap-1.5 px-2.5 text-[11.5px]"
                >
                  <HugeiconsIcon icon={t.icon} size={12} strokeWidth={1.75} />
                  <span>{t.label}</span>
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </header>

        <main className="min-h-0 flex-1 overflow-y-auto px-8 pt-6 pb-7 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <div className="mx-auto w-full max-w-160">
            <SectionErrorBoundary>
              {ActiveSection && <ActiveSection />}
            </SectionErrorBoundary>
          </div>
        </main>
      </DialogContent>
    </Dialog>
  );
}
