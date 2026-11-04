import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { useCursorPage } from "./use-cursor-page";

type Item = { id: string };

function wrapper() {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	return ({ children }: { children: ReactNode }) => (
		<QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
	);
}

function makeFetcher() {
	return vi.fn(async ({ cursor, limit }: { cursor: string; limit: number }) => {
		if (cursor) {
			return { data: [{ id: "next-page" }], meta: { limit } };
		}
		return {
			data: Array.from({ length: limit === 3 ? 3 : 4 }, (_, index) => ({
				id: `item-${index}`,
			})),
			meta: { limit, next_cursor: limit === 3 ? "next" : undefined },
		};
	});
}

describe("useCursorPage", () => {
	it("starts with three items and exposes the selected size", async () => {
		const fetchPage = makeFetcher();
		const { result } = renderHook(
			() => useCursorPage<Item>({ queryKey: ["items"], fetchPage }),
			{ wrapper: wrapper() },
		);

		await waitFor(() => expect(result.current.items).toHaveLength(3));
		expect(result.current.pageSize).toBe(3);
		expect(result.current.canGoNext).toBe(true);
		expect(fetchPage).toHaveBeenCalledWith(
			expect.objectContaining({ cursor: "", limit: 3 }),
		);
	});

	it("resets cursors when the page size changes from a later page", async () => {
		const fetchPage = makeFetcher();
		const { result } = renderHook(
			() => useCursorPage<Item>({ queryKey: ["items"], fetchPage }),
			{ wrapper: wrapper() },
		);

		await waitFor(() => expect(result.current.canGoNext).toBe(true));
		act(() => result.current.goNext());
		await waitFor(() =>
			expect(fetchPage).toHaveBeenLastCalledWith(
				expect.objectContaining({ cursor: "next", limit: 3 }),
			),
		);

		act(() => result.current.setPageSize(10));

		expect(result.current.pageSize).toBe(10);
		expect(result.current.canGoPrevious).toBe(false);
		await waitFor(() => expect(result.current.items).toHaveLength(4));
		expect(fetchPage).toHaveBeenLastCalledWith(
			expect.objectContaining({ cursor: "", limit: 10 }),
		);
	});

	it("ignores unsupported page sizes", async () => {
		const fetchPage = makeFetcher();
		const { result } = renderHook(
			() => useCursorPage<Item>({ queryKey: ["items"], fetchPage }),
			{ wrapper: wrapper() },
		);

		await waitFor(() => expect(result.current.items).toHaveLength(3));
		act(() => result.current.setPageSize(7));
		expect(result.current.pageSize).toBe(3);
	});
});
