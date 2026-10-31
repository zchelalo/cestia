import { useInfiniteQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { SelectOption } from "@/components/molecules/select/types";

export type PaginatedSelectPage<T> = {
	items: T[];
	nextCursor?: string;
};

export type PaginatedSelectQuery<T> = (params: {
	cursor?: string;
	limit: number;
	search?: string;
	signal?: AbortSignal;
}) => Promise<PaginatedSelectPage<T>>;

export function usePaginatedSelect<T>({
	enabled = true,
	limit = 20,
	queryFn,
	queryKey,
}: {
	enabled?: boolean;
	limit?: number;
	queryFn: PaginatedSelectQuery<T>;
	queryKey: readonly unknown[];
}) {
	const [searchInput, setSearchInput] = useState("");
	const [search, setSearch] = useState("");

	useEffect(() => {
		if (searchInput === "") {
			setSearch("");
			return;
		}
		const timer = window.setTimeout(() => setSearch(searchInput.trim()), 300);
		return () => window.clearTimeout(timer);
	}, [searchInput]);

	const query = useInfiniteQuery<PaginatedSelectPage<T>>({
		enabled,
		getNextPageParam: (lastPage, _pages, lastPageParam) => {
			const next = lastPage.nextCursor;
			if (!next || next === lastPageParam) return undefined;
			return next;
		},
		initialPageParam: undefined as string | undefined,
		queryFn: ({ pageParam, signal }) =>
			queryFn({
				cursor: pageParam as string | undefined,
				limit,
				search: search || undefined,
				signal,
			}),
		queryKey: [...queryKey, { search }],
	});

	const items = useMemo(
		() => query.data?.pages.flatMap((page) => page.items) ?? [],
		[query.data],
	);

	const loadMore = useCallback(() => {
		if (query.hasNextPage && !query.isFetchingNextPage) {
			void query.fetchNextPage();
		}
	}, [query]);

	return {
		error: query.error,
		hasMore: query.hasNextPage,
		isDebouncing: searchInput.trim() !== search,
		isError: query.isError,
		isFetchingNextPage: query.isFetchingNextPage,
		isLoading: query.isLoading,
		items,
		loadMore,
		refetch: query.refetch,
		searchInput,
		setSearchInput,
	};
}

export function useSelectOptions<T>({
	enabled = true,
	limit,
	queryFn,
	queryKey,
	toOption,
	staticOptions = [],
}: {
	enabled?: boolean;
	limit?: number;
	queryFn: PaginatedSelectQuery<T>;
	queryKey: readonly unknown[];
	toOption: (item: T) => SelectOption;
	staticOptions?: SelectOption[];
}) {
	const pagination = usePaginatedSelect({ enabled, limit, queryFn, queryKey });
	const [labels, setLabels] = useState<Record<string, string>>({});

	const remoteOptions = useMemo(
		() =>
			pagination.isDebouncing
				? []
				: pagination.items.map((item) => toOption(item)),
		[pagination.isDebouncing, pagination.items, toOption],
	);

	const filteredStatic = useMemo(() => {
		if (!pagination.searchInput) return staticOptions;
		const query = pagination.searchInput.toLowerCase();
		return staticOptions.filter((opt) =>
			opt.label?.toString().toLowerCase().includes(query),
		);
	}, [staticOptions, pagination.searchInput]);

	const options = useMemo(() => {
		return [...filteredStatic, ...remoteOptions];
	}, [filteredStatic, remoteOptions]);

	useEffect(() => {
		if (options.length === 0) return;
		setLabels((current) => {
			let changed = false;
			const next = { ...current };
			for (const option of options) {
				if (
					typeof option.label === "string" &&
					next[option.value] !== option.label
				) {
					next[option.value] = option.label;
					changed = true;
				}
			}
			return changed ? next : current;
		});
	}, [options]);

	const labelFor = useCallback(
		(value?: string) => (value ? labels[value] : undefined),
		[labels],
	);

	return {
		...pagination,
		isLoading: pagination.isLoading || pagination.isDebouncing,
		labelFor,
		options,
	};
}
