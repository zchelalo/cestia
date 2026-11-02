import {
	addDays,
	getDate,
	getMonth,
	getYear,
	isSameDay,
	isSameMonth,
	setMonth,
	setYear,
	startOfMonth,
	startOfToday,
	startOfWeek,
} from "date-fns";

const isoDatePattern = /^(\d{4})-(\d{2})-(\d{2})$/;
const displayDatePattern = /^(\d{2})\/(\d{2})\/(\d{4})$/;
const calendarWeeks = 6;
const daysPerWeek = 7;

const monthFormatter = new Intl.DateTimeFormat("es-MX", { month: "long" });
const weekdayFormatter = new Intl.DateTimeFormat("es-MX", {
	weekday: "short",
});
// 2024-01-01 is a Monday, used only to derive stable Spanish weekday labels.
const referenceMonday = new Date(2024, 0, 1);

export type CalendarDay = {
	date: Date;
	day: number;
	iso: string;
	outsideMonth: boolean;
};

export type NumberOption = {
	label: string;
	value: number;
};

function pad(value: number) {
	return String(value).padStart(2, "0");
}

export function parseIsoDate(value?: string | null): Date | null {
	if (!value) return null;
	const match = isoDatePattern.exec(value);
	if (!match) return null;
	const [, rawYear, rawMonth, rawDay] = match;
	const year = Number(rawYear);
	const month = Number(rawMonth);
	const day = Number(rawDay);
	const date = new Date(year, month - 1, day);
	if (
		date.getFullYear() !== year ||
		date.getMonth() !== month - 1 ||
		date.getDate() !== day
	) {
		return null;
	}
	return date;
}

export function formatIsoDate(date: Date): string {
	return `${String(date.getFullYear()).padStart(4, "0")}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

export function formatDisplayDate(value?: string | null): string {
	const date = parseIsoDate(value);
	if (!date) return value ?? "";
	return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}/${date.getFullYear()}`;
}

export function applyDateMask(rawValue: string): string {
	const digits = rawValue.replace(/\D/g, "").slice(0, 8);
	if (digits.length <= 2) return digits;
	if (digits.length <= 4) return `${digits.slice(0, 2)}/${digits.slice(2)}`;
	return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

export function parseDisplayDate(value: string): string | null {
	const match = displayDatePattern.exec(value);
	if (!match) return null;
	const [, rawDay, rawMonth, rawYear] = match;
	const iso = `${rawYear}-${rawMonth}-${rawDay}`;
	return parseIsoDate(iso) ? iso : null;
}

export function isDateWithinRange(
	iso: string,
	min?: string,
	max?: string,
): boolean {
	if (min && iso < min) return false;
	if (max && iso > max) return false;
	return true;
}

export function areSameDay(a?: Date | null, b?: Date | null): boolean {
	return Boolean(a && b && isSameDay(a, b));
}

export function getMonthOptions(): NumberOption[] {
	return Array.from({ length: 12 }, (_, month) => ({
		label: monthFormatter.format(new Date(2000, month, 1)),
		value: month,
	}));
}

export function getMonthLabel(month: number): string {
	return monthFormatter.format(new Date(2000, month, 1));
}

export function getYearOptions({
	max,
	min,
	selected,
}: {
	max?: string;
	min?: string;
	selected?: number;
} = {}): NumberOption[] {
	const currentYear = new Date().getFullYear();
	const minYear = Math.min(
		parseIsoDate(min)?.getFullYear() ?? currentYear - 25,
		selected ?? currentYear,
	);
	const maxYear = Math.max(
		parseIsoDate(max)?.getFullYear() ?? currentYear + 5,
		selected ?? currentYear,
	);
	return Array.from({ length: maxYear - minYear + 1 }, (_, index) => {
		const year = maxYear - index;
		return { label: String(year), value: year };
	});
}

export function getWeekdayLabels(): string[] {
	return Array.from({ length: daysPerWeek }, (_, index) =>
		weekdayFormatter.format(addDays(referenceMonday, index)).replace(/\.$/, ""),
	);
}

export function getCalendarDays(month: Date): CalendarDay[] {
	const firstVisibleDate = startOfWeek(startOfMonth(month), {
		weekStartsOn: 1,
	});
	return Array.from({ length: calendarWeeks * daysPerWeek }, (_, index) => {
		const date = addDays(firstVisibleDate, index);
		return {
			date,
			day: getDate(date),
			iso: formatIsoDate(date),
			outsideMonth: !isSameMonth(date, month),
		};
	});
}

export function getInitialCalendarMonth(
	value?: string | null,
	min?: string,
	max?: string,
): Date {
	return startOfMonth(
		parseIsoDate(value) ??
			parseIsoDate(min) ??
			parseIsoDate(max) ??
			startOfToday(),
	);
}

export function changeMonthYear(
	month: Date,
	change: { month?: number; year?: number },
): Date {
	let next = month;
	if (change.year !== undefined) next = setYear(next, change.year);
	if (change.month !== undefined) next = setMonth(next, change.month);
	return startOfMonth(next);
}

export function getCalendarMonthState(month: Date) {
	return { month: getMonth(month), year: getYear(month) };
}
