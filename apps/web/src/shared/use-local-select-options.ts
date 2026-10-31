import { useMemo, useState } from "react";
import type { SelectOption } from "@/components/molecules/select/types";

export function useLocalSelectOptions(options: SelectOption[]) {
	const [search, setSearch] = useState("");

	const filteredOptions = useMemo(() => {
		if (!search.trim()) return options;
		const query = search.trim().toLowerCase();
		return options.filter((opt) => {
			const labelText = opt.label?.toString().toLowerCase() ?? "";
			const descText = opt.description?.toString().toLowerCase() ?? "";
			return labelText.includes(query) || descText.includes(query);
		});
	}, [options, search]);

	return {
		options: filteredOptions,
		searchValue: search,
		onSearchChange: setSearch,
	};
}
