import type { ReactNode } from "react";
import styles from "@/pages/Page.module.css";

export function SummaryCard({
	label,
	value,
	hint,
	children,
}: {
	label: string;
	value: ReactNode;
	hint?: string;
	children?: ReactNode;
}) {
	return (
		<div className={styles.card}>
			<div className={styles.metricLabel}>{label}</div>
			<div className={styles.metricValue}>{value}</div>
			{hint ? <div className={styles.metricHint}>{hint}</div> : null}
			{children}
		</div>
	);
}
