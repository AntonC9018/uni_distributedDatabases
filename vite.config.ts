import { defineConfig } from "vite";
import { fileURLToPath, URL } from "url";
import * as path from "path";

export default (vite: any) => {
    const isProd = vite.mode == "production"

    return defineConfig({
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
        server: {
            watch: {
                ignored: (filePath) => {
                    const relativePath = path.relative(import.meta.url, filePath)
                    if (relativePath.startsWith("static")) {
                        return false;
                    }
                    return true;
                },
            },
        },
        resolve: {
            alias: {
                "@": fileURLToPath(new URL("./static/src", import.meta.url)),
                "htmx": (function()
                {
                    const path = isProd ? "htmx.org" : "htmx.org/dist/htmx";
                    return fileURLToPath(new URL(path, import.meta.url));
                })(),
            }
        },
    });
};
