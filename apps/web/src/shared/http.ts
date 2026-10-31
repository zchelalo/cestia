import type { ApiEnvelope } from "@/shared/types";

const apiBase = (import.meta.env.VITE_API_URL as string | undefined) ?? "";

export class ApiError extends Error {
	code: string;
	status: number;
	requestId?: string;

	constructor(
		status: number,
		code: string,
		message: string,
		requestId?: string,
	) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
		this.requestId = requestId;
	}
}

type Options = {
	method?: "GET" | "POST" | "PATCH" | "DELETE";
	body?: unknown;
	idempotencyKey?: string;
	signal?: AbortSignal;
};

function csrfToken() {
	const match = document.cookie.match(/(?:^|; )cestia_csrf=([^;]*)/);
	return match ? decodeURIComponent(match[1]) : "";
}

export async function api<T>(
	path: string,
	options: Options = {},
): Promise<{ data: T; meta?: ApiEnvelope<T>["meta"] }> {
	const headers: Record<string, string> = { Accept: "application/json" };
	if (options.body !== undefined) headers["Content-Type"] = "application/json";
	if (options.method && options.method !== "GET")
		headers["X-CSRF-Token"] = csrfToken();
	if (options.idempotencyKey)
		headers["Idempotency-Key"] = options.idempotencyKey;
	const response = await fetch(`${apiBase}${path}`, {
		method: options.method ?? "GET",
		credentials: "include",
		headers,
		body: options.body === undefined ? undefined : JSON.stringify(options.body),
		signal: options.signal,
	});
	if (response.status === 204) return { data: undefined as T };
	let envelope: ApiEnvelope<T> | undefined;
	try {
		envelope = (await response.json()) as ApiEnvelope<T>;
	} catch {
		// The gateway may not return JSON.
	}
	if (!response.ok || envelope?.error) {
		throw new ApiError(
			response.status,
			envelope?.error?.code ?? "unknown_error",
			envelope?.error?.message ?? "No fue posible completar la solicitud.",
			envelope?.request_id,
		);
	}
	return { data: envelope?.data as T, meta: envelope?.meta };
}

export function apiData<T>(path: string, options?: Options) {
	return api<T>(path, options).then((result) => result.data);
}

export function apiPage<T>(path: string, options?: Options) {
	return api<T>(path, options);
}
