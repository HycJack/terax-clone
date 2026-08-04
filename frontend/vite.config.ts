import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "path";
import { defineConfig, type PluginOption, type UserConfig } from "vite";
import Inspect from "vite-plugin-inspect";

const host = process.env.WAILS_DEV_HOST;

// Bundle/treemap analysis is opt-in: `ANALYZE=true pnpm build` emits stats.html.
const analyze = process.env.ANALYZE === "true";

// Module-graph inspector is opt-in via `pnpm dev:inspect`; keeps plain
// `pnpm dev` from paying its transform-tracking overhead on every run.
const inspectGraph = process.env.INSPECT === "true";

// https://vite.dev/config/
export default defineConfig(async ({ mode }): Promise<UserConfig> => ({
  plugins: [
    react(),
    tailwindcss(),
    // Module-graph inspector at /__inspect (who-imports-what, per-plugin
    // transforms). Opt-in via `pnpm dev:inspect`, never in a production build.
    ...(mode === "development" && inspectGraph
      ? [Inspect() as PluginOption]
      : []),
    ...(analyze
      ? [
          // Visualizer is opt-in only when explicitly installed. To enable:
          //   pnpm add -D rollup-plugin-visualizer
          // and uncomment the block below.
          (null as unknown as PluginOption),
        ]
      : []),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      // Shim keeps the ~117 kB CJS protocol package out of the bundle.
      "vscode-languageserver-protocol": path.resolve(
        __dirname,
        "./src/modules/lsp/lib/protocolShim.ts",
      ),
      // Tauri → Wails shim layer (frontend code unchanged, backend swapped).
      "@tauri-apps/api/core": path.resolve(
        __dirname,
        "./src/lib/wails-shim/core.ts",
      ),
      "@tauri-apps/api/event": path.resolve(
        __dirname,
        "./src/lib/wails-shim/event.ts",
      ),
      "@tauri-apps/api/window": path.resolve(
        __dirname,
        "./src/lib/wails-shim/window.ts",
      ),
      "@tauri-apps/api/webview": path.resolve(
        __dirname,
        "./src/lib/wails-shim/webview.ts",
      ),
      "@tauri-apps/api/webviewWindow": path.resolve(
        __dirname,
        "./src/lib/wails-shim/webviewWindow.ts",
      ),
      "@tauri-apps/api/path": path.resolve(
        __dirname,
        "./src/lib/wails-shim/path.ts",
      ),
      "@tauri-apps/api/app": path.resolve(
        __dirname,
        "./src/lib/wails-shim/app.ts",
      ),
      "@tauri-apps/plugin-os": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-os.ts",
      ),
      "@tauri-apps/plugin-store": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-store.ts",
      ),
      "@tauri-apps/plugin-notification": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-notification.ts",
      ),
      "@tauri-apps/plugin-clipboard-manager": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-clipboard.ts",
      ),
      "@tauri-apps/plugin-opener": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-opener.ts",
      ),
      "@tauri-apps/plugin-autostart": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-autostart.ts",
      ),
      "@tauri-apps/plugin-process": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-process.ts",
      ),
      "@tauri-apps/plugin-updater": path.resolve(
        __dirname,
        "./src/lib/wails-shim/plugin-updater.ts",
      ),
      "#wails/runtime/runtime": path.resolve(
        __dirname,
        "./src/lib/wails-shim/wails-runtime-stub.ts",
      ),
      "#wails/runtime/window": path.resolve(
        __dirname,
        "./src/lib/wails-shim/wails-runtime-window-stub.ts",
      ),
    },
  },
  build: {
    target: "es2022",
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, "index.html"),
        settings: path.resolve(__dirname, "settings.html"),
      },
      // Oxc drops `debugger` by default. These calls return undefined, so
      // marking them pure lets DCE strip them from production builds.
      treeshake: {
        manualPureFunctions: [
          "console.debug",
          "console.info",
          "console.trace",
        ],
      },
      output: {
        manualChunks(id: string) {
          if (id.includes("vite/preload-helper") || id.includes("/vite/dist/"))
            return "react";

          if (!id.includes("node_modules")) return null;

          if (
            id.includes("/clsx/") ||
            id.includes("/tailwind-merge/") ||
            id.includes("/class-variance-authority/")
          )
            return "react";

          if (id.includes("@ai-sdk/anthropic")) return "ai-anthropic";
          if (id.includes("@ai-sdk/google")) return "ai-google";
          if (id.includes("@ai-sdk/openai-compatible"))
            return "ai-openai-compat";
          if (id.includes("@ai-sdk/openai")) return "ai-openai";
          if (id.includes("@ai-sdk/cerebras")) return "ai-cerebras";
          if (id.includes("@ai-sdk/groq")) return "ai-groq";
          if (id.includes("@ai-sdk/xai")) return "ai-xai";
          if (id.includes("@ai-sdk/")) return "ai-sdk-shared";

          if (id.includes("/xterm/") || id.includes("@xterm/")) return "xterm";
          {
            const m = id.match(/@codemirror\/lang-([\w-]+)/);
            if (m) return `cm-lang-${m[1]}`;
          }
          {
            const m = id.match(/@codemirror\/legacy-modes\/mode\/([\w-]+)/);
            if (m) return `cm-legacy-${m[1]}`;
          }
          if (id.includes("@replit/codemirror-lang-svelte"))
            return "cm-lang-svelte";
          if (
            id.includes("@codemirror/") ||
            id.includes("@uiw/codemirror") ||
            id.includes("@replit/codemirror")
          )
            return "codemirror";
          if (id.includes("/streamdown/") || id.includes("@streamdown/"))
            return "streamdown";
          if (
            id.includes("/react-dom/") ||
            id.includes("/react/") ||
            id.includes("/scheduler/")
          )
            return "react";
          if (id.includes("@radix-ui/") || id.includes("/radix-ui/"))
            return "radix";

          return null;
        },
      },
    },
  },
  clearScreen: false,
  server: {
    port: 34115, // Wails default
    strictPort: true,
    host: host || false,
    hmr: host
      ? {
          protocol: "ws",
          host,
          port: 34116,
        }
      : undefined,
  },
}));