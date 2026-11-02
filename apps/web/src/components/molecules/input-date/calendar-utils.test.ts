import { describe, expect, it } from "vitest";
import {
	applyDateMask,
	formatDisplayDate,
	formatIsoDate,
	getCalendarDays,
	getMonthOptions,
	getYearOptions,
	parseDisplayDate,
	parseIsoDate,
} from "@/components/molecules/input-date/calendar-utils";

describe("calendar utils", () => {
	it("rejects impossible dates instead of normalizing them", () => {
		expect(parseIsoDate("2026-02-31")).toBeNull();
		expect(parseDisplayDate("31/02/2026")).toBeNull();
		expect(parseDisplayDate("00/10/2026")).toBeNull();
	});

	it("accepts real leap days", () => {
		expect(parseDisplayDate("29/02/2028")).toBe("2028-02-29");
		expect(parseDisplayDate("29/02/2027")).toBeNull();
	});

	it("masks typed digits into dd/mm/yyyy", () => {
		expect(applyDateMask("3")).toBe("3");
		expect(applyDateMask("3110")).toBe("31/10");
		expect(applyDateMask("31102026")).toBe("31/10/2026");
		expect(applyDateMask("31/10/2026 extra")).toBe("31/10/2026");
	});

	it("round-trips between storage and display values", () => {
		expect(formatDisplayDate("2026-10-31")).toBe("31/10/2026");
		expect(formatIsoDate(new Date(2026, 9, 5))).toBe("2026-10-05");
		expect(parseIsoDate("2026-10-05")?.getDate()).toBe(5);
	});

	it("starts calendars on Monday and always renders six weeks", () => {
		const days = getCalendarDays(new Date(2026, 9, 1));
		expect(days).toHaveLength(42);
		expect(days[0]?.date.getDay()).toBe(1);
		expect(
			days.some((day) => day.iso === "2026-10-31" && !day.outsideMonth),
		).toBe(true);
	});

	it("exposes twelve localized months", () => {
		const months = getMonthOptions();
		expect(months).toHaveLength(12);
		expect(months[9]?.label).toBe("octubre");
	});

	it("always includes the selected year in the year options", () => {
		const options = getYearOptions({ selected: 1999 });
		expect(options.some((option) => option.value === 1999)).toBe(true);
		expect(options[0]?.value).toBeGreaterThan(options[1]?.value ?? 0);
	});
});
