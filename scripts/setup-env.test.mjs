import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { ensureEnv } from "./setup-env.mjs";

function fixture() {
	const directory = mkdtempSync(join(tmpdir(), "cestia-env-"));
	const source = join(directory, ".env.example");
	const target = join(directory, "apps/api/.env");
	mkdirSync(join(directory, "apps/api"), { recursive: true });
	writeFileSync(source, "DATABASE_URL=example\n");
	return { source, target };
}

test("creates the env file from the example", () => {
	const { source, target } = fixture();
	assert.equal(ensureEnv(source, target), "created");
	assert.equal(readFileSync(target, "utf8"), "DATABASE_URL=example\n");
});

test("keeps an existing env file untouched", () => {
	const { source, target } = fixture();
	writeFileSync(target, "CUSTOM=value\n");
	assert.equal(ensureEnv(source, target), "kept");
	assert.equal(readFileSync(target, "utf8"), "CUSTOM=value\n");
});
