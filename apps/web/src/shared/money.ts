import Decimal from "decimal.js";

export function formatMoney(value: string | null | undefined) {
	if (value == null || value === "") return "—";
	return new Intl.NumberFormat("es-MX", {
		style: "currency",
		currency: "MXN",
		minimumFractionDigits: 2,
	}).format(Number(value));
}

export function lineTotal(quantity: string, unitPrice: string) {
	if (
		!quantity ||
		!unitPrice ||
		!/^\d+(\.\d{1,4})?$/.test(quantity) ||
		!/^\d+(\.\d{1,4})?$/.test(unitPrice)
	) {
		return "";
	}
	try {
		return new Decimal(quantity)
			.times(unitPrice)
			.toDecimalPlaces(2, Decimal.ROUND_HALF_UP)
			.toFixed(2);
	} catch {
		return "";
	}
}

export function formatDate(value: string, withTime = false) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) return "—";
	return new Intl.DateTimeFormat("es-MX", {
		day: "2-digit",
		month: "short",
		year: "numeric",
		...(withTime ? { hour: "2-digit", minute: "2-digit" } : {}),
	}).format(date);
}

export const unitLabels: Record<string, string> = {
	unit: "pieza",
	kg: "kg",
	g: "g",
	l: "litro",
	ml: "ml",
	package: "paquete",
};

export function localDateTimeInput(date = new Date()) {
	const offset = date.getTimezoneOffset();
	return new Date(date.getTime() - offset * 60_000).toISOString().slice(0, 16);
}
