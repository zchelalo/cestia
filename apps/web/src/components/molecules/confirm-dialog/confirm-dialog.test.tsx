import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ConfirmDialog } from "@/components/molecules/confirm-dialog";
import "@/shared/i18n";

afterEach(() => {
	cleanup();
});

function renderDialog(
	overrides: Partial<Parameters<typeof ConfirmDialog>[0]> = {},
) {
	const onCancel = vi.fn();
	const onConfirm = vi.fn();
	const view = render(
		<ConfirmDialog
			title="Eliminar artículo"
			description="Esta acción no se puede deshacer."
			confirmLabel="Eliminar"
			onCancel={onCancel}
			onConfirm={onConfirm}
			{...overrides}
		/>,
	);
	return { ...view, onCancel, onConfirm };
}

describe("ConfirmDialog", () => {
	it("confirms through the form and cancels through the secondary action", () => {
		const { onCancel, onConfirm } = renderDialog();

		fireEvent.click(screen.getByRole("button", { name: "Eliminar" }));
		fireEvent.click(screen.getByRole("button", { name: "Cancelar" }));

		expect(onConfirm).toHaveBeenCalledTimes(1);
		expect(onCancel).toHaveBeenCalledTimes(1);
	});

	it("shows server errors without closing", () => {
		renderDialog({ error: "No se pudo eliminar el artículo." });

		expect(screen.getByRole("alert")).toHaveTextContent(
			"No se pudo eliminar el artículo.",
		);
	});

	it("blocks confirmation, cancellation and Escape while pending", () => {
		const { onCancel, onConfirm } = renderDialog({ pending: true });

		expect(screen.getByRole("button", { name: "Procesando…" })).toBeDisabled();
		expect(screen.getByRole("button", { name: "Cancelar" })).toBeDisabled();

		fireEvent.keyDown(document, { key: "Escape" });

		expect(onConfirm).not.toHaveBeenCalled();
		expect(onCancel).not.toHaveBeenCalled();
	});

	it("keeps keyboard focus inside the dialog", () => {
		renderDialog();
		const close = screen.getByRole("button", { name: "Cerrar" });
		const cancel = screen.getByRole("button", { name: "Cancelar" });
		const confirm = screen.getByRole("button", { name: "Eliminar" });

		expect(cancel).toHaveFocus();
		close.focus();
		fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
		expect(confirm).toHaveFocus();
	});
});
