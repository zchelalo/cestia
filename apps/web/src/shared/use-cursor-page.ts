import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

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
	const [cursors, setCursors] = useState<string[]>([""]);
	const [index, setIndex] = useState(0);
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
		queryKey: [...queryKey, { cursor, limit }],
		queryFn: ({ signal }) => fetchPage({ cursor, limit, signal }),
		enabled,
		refetchInterval,
		placeholderData: (previous) => previous,
	});

	const next = query.data?.meta?.next_cursor ?? "";

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
		refetch: query.refetch,
		reset,
	};
}
