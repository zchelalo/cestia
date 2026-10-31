import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Select } from "./index";

const options = [
	{ value: "a", label: "Opción A" },
	{ value: "b", label: "Opción B", description: "Detalle de B" },
];

describe("Select component", () => {
	afterEach(() => {
		cleanup();
	});

	it("renders trigger with placeholder and opens options", () => {
		render(
			<Select
				aria-label="Ejemplo"
				options={options}
				placeholder="Elige algo"
			/>,
		);
		const trigger = screen.getByRole("button", { name: "Ejemplo" });
		expect(trigger).toHaveTextContent("Elige algo");
		fireEvent.click(trigger);
		expect(screen.getByRole("listbox")).toBeInTheDocument();
		expect(
			screen.getByRole("option", { name: /Opción A/ }),
		).toBeInTheDocument();
		expect(screen.getByText("Detalle de B")).toBeInTheDocument();
	});

	it("filters options with search input", () => {
		const onSearch = vi.fn();
		render(
			<Select
				aria-label="Ejemplo"
				options={options}
				searchable
				searchValue=""
				onSearchChange={onSearch}
			/>,
		);
		fireEvent.click(screen.getByRole("button", { name: "Ejemplo" }));
		const searchInput = screen.getByPlaceholderText("Buscar...");
		fireEvent.change(searchInput, { target: { value: "Opción B" } });
		expect(onSearch).toHaveBeenCalledWith("Opción B");
	});

	it("selects an option on click and closes menu", () => {
		const onChange = vi.fn();
		render(
			<Select aria-label="Ejemplo" options={options} onChange={onChange} />,
		);
		fireEvent.click(screen.getByRole("button", { name: "Ejemplo" }));
		fireEvent.click(screen.getByRole("option", { name: /Opción B/ }));
		expect(onChange).toHaveBeenCalledWith("b");
		expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
	});
});
