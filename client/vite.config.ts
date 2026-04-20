import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import { readFileSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

// argon2-browser uses require('./argon2.wasm') which the CJS plugin converts
// to a namespace import. We provide a virtual module with the WASM as base64,
// then fix the namespace reference in the output so atob() gets the raw string.
const argon2WasmPlugin = {
  name: "argon2-wasm-inline",
  enforce: "pre" as const,
  resolveId(source: string, importer: string | undefined) {
    if (
      source.endsWith("argon2.wasm") &&
      importer &&
      importer.includes("argon2-browser")
    ) {
      return "\0virtual:argon2-wasm";
    }
    return null;
  },
  load(id: string) {
    if (id === "\0virtual:argon2-wasm") {
      const wasmPath = resolve(
        __dirname,
        "node_modules/argon2-browser/dist/argon2.wasm"
      );
      const wasmBuffer = readFileSync(wasmPath);
      const base64 = wasmBuffer.toString("base64");
      return `export default "${base64}";`;
    }
    return null;
  },
  // The CJS plugin wraps require('./argon2.wasm') in a namespace import,
  // so the argon2 code does atob({default: "base64..."}) instead of atob("base64...").
  // Fix by extracting .default from the namespace when it exists.
  renderChunk(code: string, chunk: { fileName: string }) {
    if (chunk.fileName.includes("argon2")) {
      // The CJS plugin converts require('./argon2.wasm') to a namespace import.
      // In the output, Promise.resolve(namespace) is called, but decodeWasmBinary
      // (atob) needs the raw base64 string, not the namespace object.
      // Fix: extract .default from the namespace when it exists.
      // Matches both un-minified (multi-line) and minified (single-line) patterns.
      return code.replace(
        /Promise\.resolve\(([\w$]+)\)\.then\s*\(\s*\n?\s*(?:\(\s*[\w$]+\s*\)\s*=>\s*\{\s*\n?\s*return\s+|function\s*\(\s*[\w$]+\s*\)\s*\{\s*\n?\s*return\s+)\s*decodeWasmBinary\(\s*[\w$]+\s*\)\s*;?\s*\n?\s*\}\s*\n?\s*\)/g,
        "Promise.resolve($1.default||$1).then((m)=>decodeWasmBinary(m))"
      );
    }
    return null;
  },
};

export default defineConfig({
  plugins: [TanStackRouterVite(), react(), tailwindcss(), argon2WasmPlugin],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/upload": "http://localhost:8080",
    },
  },
  optimizeDeps: {
    exclude: ["argon2-browser"],
  },
});