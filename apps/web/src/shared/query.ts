import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			retry: (failures, error) => {
				const status = (error as { status?: number }).status;
				if (status && status < 500) return false;
				return failures < 1;
			},
			refetchOnWindowFocus: true,
			staleTime: 10_000,
		},
	},
});
