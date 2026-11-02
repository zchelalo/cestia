import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Modal } from "@/components/Modal";

afterEach(() => {
	cleanup();
});

describe("Modal", () => {
	it("closes when the overlay is pressed", () => {
		const onClose = vi.fn();
		const { container } = render(
			<Modal title="Editar artículo" onClose={onClose}>
				<button type="button">Guardar</button>
			</Modal>,
		);
		const overlay = container.firstElementChild as HTMLElement;
		fireEvent.pointerDown(overlay);
		fireEvent.pointerUp(overlay);
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it("stays open when the panel is pressed", () => {
		const onClose = vi.fn();
		render(
			<Modal title="Editar artículo" onClose={onClose}>
				<button type="button">Guardar</button>
			</Modal>,
		);
		const panel = screen.getByRole("dialog");
		fireEvent.pointerDown(panel);
		fireEvent.pointerUp(panel);
		expect(onClose).not.toHaveBeenCalled();
	});

	it("closes with Escape when no popover layer is open", () => {
		const onClose = vi.fn();
		render(
			<Modal title="Editar artículo" onClose={onClose}>
				<button type="button">Guardar</button>
			</Modal>,
		);
		fireEvent.keyDown(document, { key: "Escape" });
		expect(onClose).toHaveBeenCalledTimes(1);
	});
});
