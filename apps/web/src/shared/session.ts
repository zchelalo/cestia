import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiData } from "@/shared/http";
import type { User } from "@/shared/types";

export const sessionQuery = {
	queryKey: ["session"] as const,
	queryFn: () => apiData<User>("/api/v1/me"),
	retry: false,
	staleTime: 30_000,
};

export function useSession() {
	return useQuery(sessionQuery);
}

export function useLogout() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => apiData<void>("/api/v1/auth/logout", { method: "POST" }),
		onSettled: () => queryClient.clear(),
	});
}

export function errorMessage(error: unknown) {
	return error instanceof Error ? error.message : undefined;
}
