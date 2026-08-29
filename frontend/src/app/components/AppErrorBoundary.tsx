import { Component, type ReactNode } from "react";

type Props = { children: ReactNode };
type State = { error: Error | null };

export class AppErrorBoundary extends Component<Props, State> {
	state: State = { error: null };

	static getDerivedStateFromError(error: Error): State {
		return { error };
	}

	render() {
		if (this.state.error) {
			return (
				<div className="flex h-screen flex-col items-center justify-center gap-4 bg-background p-8 text-foreground">
					<div className="text-center">
						<h1 className="text-lg font-semibold">Something went wrong</h1>
						<p className="mt-2 max-w-md text-sm text-muted-foreground">
							{this.state.error.message || "An unexpected error occurred."}
						</p>
					</div>
					<button
						type="button"
						onClick={() => {
							this.setState({ error: null });
							window.location.reload();
						}}
						className="rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90"
					>
						Reload app
					</button>
				</div>
			);
		}
		return this.props.children;
	}
}
