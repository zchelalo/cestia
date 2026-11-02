import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Button } from "@/components/Button";

describe("Button", () => {
	it("wraps its text so it can be optically centered without moving icons", () => {
		render(
			<Button>
				<svg data-testid="icon" aria-hidden="true" />
				Guardar
			</Button>,
		);
		const button = screen.getByRole("button", { name: "Guardar" });
		const label = button.querySelector("span");
		expect(label).toHaveTextContent("Guardar");
		expect(button.querySelector('[data-testid="icon"]')).toBeInTheDocument();
	});

	it("keeps icon-only buttons free of text wrappers", () => {
		render(
			<Button aria-label="Editar">
				<svg data-testid="icon" aria-hidden="true" />
			</Button>,
		);
		const button = screen.getByRole("button", { name: "Editar" });
		expect(button.querySelector("span")).toBeNull();
	});
});
