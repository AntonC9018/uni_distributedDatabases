import { defineConfig } from "vite";
import { fileURLToPath, URL } from "url";
import * as path from "path";

/** @type {import('vite')} */
export default (vite: { mode: string; }) => {
    const isProd = vite.mode == "production"

    return defineConfig({
        optimizeDeps: {
            noDiscovery: true,
        },
        root: "static",
        base: "/dist",
        build: {
            sourcemap: true,
            manifest: "manifest.json",
            rollupOptions: {
                input: [
                    "static/src/main.ts", 
                ],
                output: {
                    dir: "static/dist",
                },
            },
        },
        resolve: {
            alias: {
                "@": fileURLToPath(new URL("./static/src", import.meta.url)),
                "htmx": (function()
                {
                    if (isProd)
                    {
                        return "htmx.org"
                    }
                    return "htmx.org/dist/htmx";
                })(),
            }
        },
    });
};
