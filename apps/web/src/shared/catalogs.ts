import type { SelectOption } from "@/components/molecules/select/types";
import { apiPage } from "@/shared/http";
import { useSelectOptions } from "@/shared/use-select-options";

type BackendOption = {
	value: string;
	label: string;
	description?: string;
};

export function useGroupSelectOptions(staticOptions?: SelectOption[]) {
	return useSelectOptions<BackendOption>({
		queryKey: ["groups", "select-options"],
		queryFn: async ({ cursor, limit, search, signal }) => {
			const query = new URLSearchParams({ limit: limit.toString() });
			if (cursor) query.set("cursor", cursor);
			if (search) query.set("search", search);
			const res = await apiPage<BackendOption[]>(
				`/api/v1/groups/options?${query.toString()}`,
				{ signal },
			);
			return {
				items: res.data ?? [],
				nextCursor: res.meta?.next_cursor,
			};
		},
		toOption: (item) => ({
			value: item.value,
			label: item.label,
			description: item.description,
		}),
		staticOptions,
	});
}

export function useGroupMemberSelectOptions(
	groupId?: string | null,
	staticOptions?: SelectOption[],
) {
	return useSelectOptions<BackendOption>({
		enabled: !!groupId,
		queryKey: ["groups", groupId, "member-select-options"],
		queryFn: async ({ cursor, limit, search, signal }) => {
			if (!groupId) return { items: [] };
			const query = new URLSearchParams({ limit: limit.toString() });
			if (cursor) query.set("cursor", cursor);
			if (search) query.set("search", search);
			const res = await apiPage<BackendOption[]>(
				`/api/v1/groups/${groupId}/members/options?${query.toString()}`,
				{ signal },
			);
			return {
				items: res.data ?? [],
				nextCursor: res.meta?.next_cursor,
			};
		},
		toOption: (item) => ({
			value: item.value,
			label: item.label,
			description: item.description,
		}),
		staticOptions,
	});
}
