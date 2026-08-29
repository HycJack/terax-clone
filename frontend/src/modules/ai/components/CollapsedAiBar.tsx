import { Button } from "@/components/ui/button";
import { SidebarRight01Icon } from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useChatStore } from "../store/chatStore";

export function CollapsedAiBar({ onExpand }: { onExpand: () => void }) {
	const isBusy = useChatStore(
		(s) => s.agentMeta.status === "streaming",
	);

	return (
		<div className="flex h-full w-12 shrink-0 flex-col items-center border-l border-border/60 bg-card py-2">
			<Button
				variant="ghost"
				size="icon-sm"
				onClick={onExpand}
				aria-label="Expand AI sidebar"
				className="relative text-muted-foreground hover:text-foreground"
			>
				<HugeiconsIcon icon={SidebarRight01Icon} size={16} strokeWidth={1.75} />
				{isBusy && (
					<span className="absolute -top-0.5 -right-0.5 h-2 w-2 rounded-full bg-amber-500" />
				)}
			</Button>
		</div>
	);
}
