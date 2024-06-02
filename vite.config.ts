import { defineConfig } from "vite";
import { fileURLToPath, URL } from "url";

export default defineConfig({
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
        }
    },
})
