import { apiPage } from "@/shared/http";

/**
 * Builds a page fetcher for an endpoint that accepts `limit` and `cursor`.
 */
export function pageFetcher<T>(
	basePath: string,
	params: Record<string, string> = {},
) {
	return (input: { cursor: string; limit: number; signal?: AbortSignal }) => {
		const search = new URLSearchParams(params);
		search.set("limit", String(input.limit));
		if (input.cursor) search.set("cursor", input.cursor);
		const separator = basePath.includes("?") ? "&" : "?";
		return apiPage<T[]>(`${basePath}${separator}${search.toString()}`, {
			signal: input.signal,
		});
	};
}
