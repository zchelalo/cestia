import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react-swc";
import { defineConfig } from "vitest/config";

export default defineConfig({
	appType: "spa",
	plugins: [react()],
	resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
	server: {
		port: 5173,
		strictPort: true,
		proxy: {
			"/api": {
				target: "http://127.0.0.1:8000",
				changeOrigin: true,
				cookieDomainRewrite: "",
			},
		},
	},
	test: {
		environment: "jsdom",
		exclude: ["**/node_modules/**", "**/dist/**", "**/e2e/**"],
		setupFiles: ["./src/test/setup.ts"],
	},
});
