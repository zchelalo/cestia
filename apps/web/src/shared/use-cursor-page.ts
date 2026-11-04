import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

export const PAGE_SIZE_OPTIONS = [3, 5, 10, 15, 20, 50] as const;
export type PageSize = (typeof PAGE_SIZE_OPTIONS)[number];

const DEFAULT_PAGE_SIZE: PageSize = 3;

export type CursorPage<T> = {
	data: T[];
	meta?: { limit: number; next_cursor?: string };
};

type Params<T> = {
	enabled?: boolean;
	fetchPage: (input: {
		cursor: string;
		limit: number;
		signal?: AbortSignal;
	}) => Promise<CursorPage<T>>;
	limit?: number;
	queryKey: readonly unknown[];
	refetchInterval?: number | false;
};

/**
 * Page-by-page cursor navigation. Each page replaces the previous one and
 * previous pages are revisited with the cursor that produced them.
 */
export function useCursorPage<T>({
	enabled = true,
	fetchPage,
	limit = 3,
	queryKey,
	refetchInterval,
}: Params<T>) {
	const initialPageSize = PAGE_SIZE_OPTIONS.includes(limit as PageSize)
		? (limit as PageSize)
		: DEFAULT_PAGE_SIZE;
	const [cursors, setCursors] = useState<string[]>([""]);
	const [index, setIndex] = useState(0);
	const [pageSize, setPageSizeState] = useState<PageSize>(initialPageSize);
	const signature = JSON.stringify(queryKey);
	const previousSignature = useRef(signature);

	useEffect(() => {
		if (previousSignature.current === signature) return;
		previousSignature.current = signature;
		setCursors([""]);
		setIndex(0);
	}, [signature]);

	const cursor = cursors[index] ?? "";
	const query = useQuery({
		queryKey: [...queryKey, { cursor, pageSize }],
		queryFn: ({ signal }) => fetchPage({ cursor, limit: pageSize, signal }),
		enabled,
		refetchInterval,
		placeholderData: (previous) => previous,
	});

	// Never reuse the previous page's cursor while a new page size or cursor is
	// still loading.
	const next = query.isPlaceholderData
		? ""
		: (query.data?.meta?.next_cursor ?? "");

	const goNext = useCallback(() => {
		if (!next) return;
		setCursors((current) => [...current.slice(0, index + 1), next]);
		setIndex((current) => current + 1);
	}, [index, next]);

	const goPrevious = useCallback(() => {
		setIndex((current) => Math.max(0, current - 1));
	}, []);

	const reset = useCallback(() => {
		setCursors([""]);
		setIndex(0);
	}, []);

	const setPageSize = useCallback((size: number) => {
		if (!PAGE_SIZE_OPTIONS.includes(size as PageSize)) return;
		setPageSizeState(size as PageSize);
		// Changing the page size invalidates every cursor collected for the old
		// size, so return to the first page in the same state update.
		setCursors([""]);
		setIndex(0);
	}, []);

	useEffect(() => {
		if (!query.isFetching && index > 0 && query.data?.data.length === 0) {
			setIndex((current) => Math.max(0, current - 1));
		}
	}, [index, query.data, query.isFetching]);

	return {
		canGoNext: Boolean(next),
		canGoPrevious: index > 0,
		error: query.error,
		goNext,
		goPrevious,
		isError: query.isError,
		isFetching: query.isFetching,
		isLoading: query.isLoading,
		items: query.data?.data ?? [],
		pageSize,
		refetch: query.refetch,
		reset,
		setPageSize,
	};
}
