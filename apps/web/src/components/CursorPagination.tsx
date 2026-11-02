import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/Button";
import styles from "@/components/CursorPagination.module.css";

type Props = {
	canGoNext: boolean;
	canGoPrevious: boolean;
	isFetching?: boolean;
	label?: string;
	onNext: () => void;
	onPrevious: () => void;
};

export function CursorPagination({
	canGoNext,
	canGoPrevious,
	isFetching = false,
	label = "Paginación",
	onNext,
	onPrevious,
}: Props) {
	if (!canGoNext && !canGoPrevious) return null;
	return (
		<nav className={styles.pagination} aria-label={label}>
			<Button
				small
				variant="quiet"
				disabled={!canGoPrevious || isFetching}
				onClick={onPrevious}
			>
				<ChevronLeft size={16} aria-hidden="true" />
				Anterior
			</Button>
			<Button
				small
				variant="quiet"
				disabled={!canGoNext || isFetching}
				onClick={onNext}
			>
				Siguiente
				<ChevronRight size={16} aria-hidden="true" />
			</Button>
		</nav>
	);
}
