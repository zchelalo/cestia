import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DateTimeInput, InputDate } from "@/components/molecules/input-date";

afterEach(() => {
	cleanup();
});

describe("InputDate", () => {
	it("opens the custom calendar instead of a native date picker", () => {
		render(
			<InputDate aria-label="Desde" value="2026-10-15" onChange={() => {}} />,
		);
		const input = screen.getByLabelText("Desde");
		expect(input).toHaveAttribute("type", "text");
		fireEvent.click(screen.getByRole("button", { name: "Abrir calendario" }));
		expect(
			screen.getByRole("dialog", { name: "Calendario" }),
		).toBeInTheDocument();
	});

	it("selects a day and closes the calendar", () => {
		const onChange = vi.fn();
		render(
			<InputDate aria-label="Desde" value="2026-10-15" onChange={onChange} />,
		);
		fireEvent.click(screen.getByRole("button", { name: "Abrir calendario" }));
		fireEvent.click(
			screen.getByRole("button", { name: "20 de octubre de 2026" }),
		);
		expect(onChange).toHaveBeenCalledWith("2026-10-20");
		expect(
			screen.queryByRole("dialog", { name: "Calendario" }),
		).not.toBeInTheDocument();
	});

	it("changes month and year from the custom selectors", () => {
		render(
			<InputDate aria-label="Desde" value="2026-10-15" onChange={() => {}} />,
		);
		fireEvent.click(screen.getByRole("button", { name: "Abrir calendario" }));

		fireEvent.click(screen.getByRole("button", { name: "Mes" }));
		fireEvent.click(screen.getByRole("option", { name: "diciembre" }));
		expect(screen.getByRole("button", { name: "Mes" })).toHaveTextContent(
			"diciembre",
		);
		expect(
			screen.getByRole("button", { name: "25 de diciembre de 2026" }),
		).toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: "Año" }));
		fireEvent.click(screen.getByRole("option", { name: "2028" }));
		expect(screen.getByRole("button", { name: "Año" })).toHaveTextContent(
			"2028",
		);
	});

	it("masks typed dates and emits the ISO value", () => {
		const onChange = vi.fn();
		render(<InputDate aria-label="Desde" value="" onChange={onChange} />);
		fireEvent.change(screen.getByLabelText("Desde"), {
			target: { value: "31102026" },
		});
		expect(onChange).toHaveBeenCalledWith("2026-10-31");
	});

	it("does not emit impossible dates and restores the last valid value", () => {
		const onChange = vi.fn();
		render(
			<InputDate aria-label="Desde" value="2026-10-15" onChange={onChange} />,
		);
		const input = screen.getByLabelText("Desde");
		fireEvent.change(input, { target: { value: "31022026" } });
		expect(onChange).not.toHaveBeenCalled();
		fireEvent.blur(input);
		expect(input).toHaveValue("15/10/2026");
	});

	it("closes with Escape without closing an outer dialog", () => {
		const outer = vi.fn();
		document.addEventListener("keydown", outer);
		render(
			<InputDate aria-label="Desde" value="2026-10-15" onChange={() => {}} />,
		);
		fireEvent.click(screen.getByRole("button", { name: "Abrir calendario" }));
		fireEvent.keyDown(screen.getByLabelText("Desde"), { key: "Escape" });
		document.removeEventListener("keydown", outer);
		expect(
			screen.queryByRole("dialog", { name: "Calendario" }),
		).not.toBeInTheDocument();
		expect(outer).not.toHaveBeenCalled();
	});
});

describe("DateTimeInput", () => {
	it("keeps date and time in sync", () => {
		const onChange = vi.fn();
		render(
			<DateTimeInput
				aria-label="Fecha de compra"
				value="2026-10-31T14:05"
				onChange={onChange}
			/>,
		);
		expect(screen.getByLabelText("Fecha de compra")).toHaveValue("31/10/2026");
		expect(screen.getByLabelText("Hora")).toHaveValue("14:05");

		fireEvent.change(screen.getByLabelText("Hora"), {
			target: { value: "1530" },
		});
		expect(onChange).toHaveBeenLastCalledWith("2026-10-31T15:30");
	});

	it("combines a calendar selection with the current time", () => {
		const onChange = vi.fn();
		render(
			<DateTimeInput
				aria-label="Fecha de compra"
				value="2026-10-31T14:05"
				onChange={onChange}
			/>,
		);
		fireEvent.click(screen.getByRole("button", { name: "Abrir calendario" }));
		fireEvent.click(
			screen.getByRole("button", { name: "20 de octubre de 2026" }),
		);
		expect(onChange).toHaveBeenLastCalledWith("2026-10-20T14:05");
	});
});
