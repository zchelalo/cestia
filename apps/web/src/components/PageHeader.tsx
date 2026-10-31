import type { ReactNode } from "react";
import styles from "@/pages/Page.module.css";

export function PageHeader({
	eyebrow,
	title,
	subtitle,
	actions,
}: {
	eyebrow?: string;
	title: string;
	subtitle?: string;
	actions?: ReactNode;
}) {
	return (
		<header className={styles.header}>
			<div>
				{eyebrow ? <div className={styles.eyebrow}>{eyebrow}</div> : null}
				<h1 className={styles.title}>{title}</h1>
				{subtitle ? <p className={styles.subtitle}>{subtitle}</p> : null}
			</div>
			{actions ? <div className={styles.actions}>{actions}</div> : null}
		</header>
	);
}
