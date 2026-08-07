import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Wails v2 期望 frontend/dist/ 是最终产物。
// base 必须为 "./" 以让打包后的相对路径在 file:// 协议下也能工作。
export default defineConfig({
  base: "./",
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    target: "es2020",
    sourcemap: false,
  },
  server: {
    port: 5173,
    strictPort: true,
  },
});
