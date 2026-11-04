import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CursorPagination } from "@/components/CursorPagination";
import "@/shared/i18n";

afterEach(() => {
	cleanup();
});

const baseProps = {
	canGoNext: true,
	canGoPrevious: true,
	onNext: vi.fn(),
	onPageSizeChange: vi.fn(),
	onPrevious: vi.fn(),
	pageSize: 3,
};

describe("CursorPagination", () => {
	it("keeps the page-size selector visible when there is a single page", () => {
		render(
			<CursorPagination
				{...baseProps}
				canGoNext={false}
				canGoPrevious={false}
			/>,
		);

		expect(
			screen.getByRole("button", { name: "Por página" }),
		).toBeInTheDocument();
	});

	it("changes the page size through the shared select", () => {
		const onPageSizeChange = vi.fn();
		render(
			<CursorPagination {...baseProps} onPageSizeChange={onPageSizeChange} />,
		);

		fireEvent.click(screen.getByRole("button", { name: "Por página" }));
		fireEvent.click(screen.getByRole("option", { name: "10" }));

		expect(onPageSizeChange).toHaveBeenCalledWith(10);
	});

	it("navigates between pages", () => {
		const onNext = vi.fn();
		const onPrevious = vi.fn();
		render(
			<CursorPagination
				{...baseProps}
				onNext={onNext}
				onPrevious={onPrevious}
			/>,
		);

		fireEvent.click(screen.getByRole("button", { name: "Siguiente" }));
		fireEvent.click(screen.getByRole("button", { name: "Anterior" }));

		expect(onNext).toHaveBeenCalledTimes(1);
		expect(onPrevious).toHaveBeenCalledTimes(1);
	});
});
