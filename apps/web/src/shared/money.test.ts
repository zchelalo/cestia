import { describe, expect, it } from "vitest";
import { formatMoney, lineTotal } from "@/shared/money";

describe("lineTotal", () => {
	it("rounds half up to two decimals without binary floating point", () => {
		expect(lineTotal("0.125", "19.99")).toBe("2.50");
		expect(lineTotal("3", "0.10")).toBe("0.30");
	});

	it("returns an empty preview for malformed values", () => {
		expect(lineTotal("abc", "10")).toBe("");
		expect(lineTotal("-1", "10")).toBe("");
	});
});

describe("formatMoney", () => {
	it("keeps unknown amounts visually distinct from zero", () => {
		expect(formatMoney(null)).toBe("—");
		expect(formatMoney("0.00")).toContain("0.00");
	});
});
