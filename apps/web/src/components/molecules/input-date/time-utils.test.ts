import { describe, expect, it } from "vitest";
import {
	applyTimeMask,
	joinLocalDateTime,
	normalizeTime,
	splitLocalDateTime,
} from "@/components/molecules/input-date/time-utils";

describe("time utils", () => {
	it("masks digits into hh:mm", () => {
		expect(applyTimeMask("14")).toBe("14");
		expect(applyTimeMask("1430")).toBe("14:30");
		expect(applyTimeMask("9:5")).toBe("9:5");
		expect(applyTimeMask("14:305")).toBe("14:30");
	});

	it("normalizes valid times and rejects invalid ones", () => {
		expect(normalizeTime("9:5")).toBe("09:05");
		expect(normalizeTime("14:30")).toBe("14:30");
		expect(normalizeTime("24:00")).toBeNull();
		expect(normalizeTime("12:60")).toBeNull();
		expect(normalizeTime("")).toBeNull();
	});

	it("splits and joins local date-time values", () => {
		expect(splitLocalDateTime("2026-10-31T14:05")).toEqual({
			date: "2026-10-31",
			time: "14:05",
		});
		expect(splitLocalDateTime("")).toEqual({ date: "", time: "" });
		expect(joinLocalDateTime("2026-10-31", "9:05")).toBe("2026-10-31T09:05");
		expect(joinLocalDateTime("2026-10-31", "")).toBe("");
		expect(joinLocalDateTime("", "10:00")).toBe("");
	});
});
