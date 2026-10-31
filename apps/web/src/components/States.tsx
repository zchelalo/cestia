import { AlertCircle, LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import styles from "@/components/States.module.css";

export function LoadingState() {
	const { t } = useTranslation();
	return (
		<div className={styles.state} role="status">
			<LoaderCircle className={styles.spin} size={20} />
			<span>{t("common.loading")}</span>
		</div>
	);
}

export function ErrorState({
	message,
	onRetry,
}: {
	message?: string;
	onRetry?: () => void;
}) {
	const { t } = useTranslation();
	return (
		<div className={`${styles.state} ${styles.error}`} role="alert">
			<AlertCircle size={20} />
			<span>{message ?? t("common.error")}</span>
			{onRetry ? (
				<button type="button" onClick={onRetry}>
					{t("common.retry")}
				</button>
			) : null}
		</div>
	);
}

export function EmptyState({ children }: { children: React.ReactNode }) {
	return <div className={styles.empty}>{children}</div>;
}
