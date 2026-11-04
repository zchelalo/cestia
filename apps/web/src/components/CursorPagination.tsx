import { ChevronLeft, ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/Button";
import styles from "@/components/CursorPagination.module.css";
import { Select } from "@/components/molecules/select";
import { PAGE_SIZE_OPTIONS, type PageSize } from "@/shared/use-cursor-page";

type Props = {
	canGoNext: boolean;
	canGoPrevious: boolean;
	isFetching?: boolean;
	label?: string;
	onNext: () => void;
	onPageSizeChange: (size: PageSize) => void;
	onPrevious: () => void;
	pageSize: number;
};

const pageSizeOptions = PAGE_SIZE_OPTIONS.map((size) => ({
	label: String(size),
	value: String(size),
}));

export function CursorPagination({
	canGoNext,
	canGoPrevious,
	isFetching = false,
	label = "Paginación",
	onNext,
	onPageSizeChange,
	onPrevious,
	pageSize,
}: Props) {
	const { t } = useTranslation();

	const changePageSize = (value: string) => {
		const next = Number(value);
		if (PAGE_SIZE_OPTIONS.includes(next as PageSize)) {
			onPageSizeChange(next as PageSize);
		}
	};

	return (
		<nav className={styles.pagination} aria-label={label}>
			<div className={styles.pageSize}>
				<span className={styles.pageSizeLabel}>{t("common.perPage")}</span>
				<div className={styles.pageSizeSelect}>
					<Select
						aria-label={t("common.perPage")}
						disabled={isFetching}
						onChange={changePageSize}
						options={pageSizeOptions}
						searchable={false}
						value={String(pageSize)}
					/>
				</div>
			</div>
			<div className={styles.navigation}>
				<Button
					small
					variant="quiet"
					disabled={!canGoPrevious || isFetching}
					onClick={onPrevious}
				>
					<ChevronLeft size={16} aria-hidden="true" />
					{t("common.previous")}
				</Button>
				<Button
					small
					variant="quiet"
					disabled={!canGoNext || isFetching}
					onClick={onNext}
				>
					{t("common.next")}
					<ChevronRight size={16} aria-hidden="true" />
				</Button>
			</div>
		</nav>
	);
}
