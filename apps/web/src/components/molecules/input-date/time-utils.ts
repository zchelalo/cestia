const localDateTimePattern = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2})$/;

export type LocalDateTimeParts = {
	date: string;
	time: string;
};

export function splitLocalDateTime(value?: string | null): LocalDateTimeParts {
	if (!value) return { date: "", time: "" };
	const match = localDateTimePattern.exec(value);
	if (match) return { date: match[1], time: `${match[2]}:${match[3]}` };
	return { date: value.slice(0, 10), time: "" };
}

export function applyTimeMask(rawValue: string): string {
	const [rawHours, rawMinutes] = rawValue.split(":");
	if (rawMinutes !== undefined) {
		const hours = rawHours.replace(/\D/g, "").slice(0, 2);
		const minutes = rawMinutes.replace(/\D/g, "").slice(0, 2);
		return `${hours}:${minutes}`;
	}
	const digits = rawHours.replace(/\D/g, "").slice(0, 4);
	if (digits.length <= 2) return digits;
	return `${digits.slice(0, 2)}:${digits.slice(2)}`;
}

export function normalizeTime(value: string): string | null {
	const match = /^(\d{1,2}):(\d{1,2})$/.exec(value.trim());
	if (!match) return null;
	const hours = Number(match[1]);
	const minutes = Number(match[2]);
	if (hours > 23 || minutes > 59) return null;
	return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}`;
}

export function joinLocalDateTime(date: string, time: string): string {
	const normalizedTime = normalizeTime(time);
	if (!date || !normalizedTime) return "";
	return `${date}T${normalizedTime}`;
}
