import type { ReactNode } from "react";

export type SelectOption = {
	local?: boolean;
	label: ReactNode;
	value: string;
	description?: ReactNode;
};

export type SelectProps = {
	"aria-label"?: string;
	className?: string;
	disabled?: boolean;
	emptyLabel?: string;
	errorLabel?: string;
	hasMore?: boolean;
	id?: string;
	isError?: boolean;
	isLoading?: boolean;
	isLoadingMore?: boolean;
	loadingLabel?: string;
	name?: string;
	onBlur?: () => void;
	onChange?: (value: string) => void;
	onLoadMore?: () => void;
	onRetry?: () => void;
	onSearchChange?: (value: string) => void;
	options: SelectOption[];
	placeholder?: string;
	remote?: boolean;
	retryLabel?: string;
	searchPlaceholder?: string;
	searchValue?: string;
	searchable?: boolean;
	selectedLabel?: string;
	value?: string;
};
