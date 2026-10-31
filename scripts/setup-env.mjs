import { constants, copyFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(fileURLToPath(new URL("..", import.meta.url)));

export function ensureEnv(
	source = resolve(root, ".env.example"),
	target = resolve(root, "apps/api/.env"),
) {
	mkdirSync(dirname(target), { recursive: true });
	try {
		copyFileSync(source, target, constants.COPYFILE_EXCL);
		return "created";
	} catch (error) {
		if (error?.code === "EEXIST") {
			return "kept";
		}
		throw error;
	}
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	const result = ensureEnv(process.argv[2], process.argv[3]);
	console.log(
		result === "created"
			? "Created apps/api/.env from .env.example."
			: "Kept existing apps/api/.env.",
	);
}
