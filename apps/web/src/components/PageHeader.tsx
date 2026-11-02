import clsx from "clsx";
import type { ReactNode } from "react";
import styles from "@/pages/Page.module.css";

export function PageHeader({
	eyebrow,
	title,
	subtitle,
	actions,
	actionsGrid = false,
}: {
	eyebrow?: string;
	title: string;
	subtitle?: string;
	actions?: ReactNode;
	actionsGrid?: boolean;
}) {
	return (
		<header className={styles.header}>
			<div>
				{eyebrow ? <div className={styles.eyebrow}>{eyebrow}</div> : null}
				<h1 className={styles.title}>{title}</h1>
				{subtitle ? <p className={styles.subtitle}>{subtitle}</p> : null}
			</div>
			{actions ? (
				<div
					className={clsx(styles.actions, actionsGrid && styles.actionsGrid)}
				>
					{actions}
				</div>
			) : null}
		</header>
	);
}
