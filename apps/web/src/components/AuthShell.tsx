import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import styles from "@/components/AuthShell.module.css";

export function AuthShell({
	title,
	subtitle,
	children,
}: {
	title: string;
	subtitle: string;
	children: ReactNode;
}) {
	const { t } = useTranslation();
	return (
		<div className={styles.page}>
			<section className={styles.formSide}>
				<Link to="/" className={styles.brand} aria-label={t("appName")}>
					<span className={styles.mark}>C</span>
					<span className={styles.brandName}>{t("appName")}</span>
				</Link>
				<div className={styles.content}>
					<header>
						<h1 className={styles.title}>{title}</h1>
						<p className={styles.subtitle}>{subtitle}</p>
					</header>
					{children}
				</div>
			</section>
			<aside className={styles.storySide}>
				<div className={styles.story}>
					<span className={styles.eyebrow}>{t("tagline")}</span>
					<h2 className={styles.storyTitle}>
						Compras compartidas.
						<br />
						Aportes reales.
					</h2>
					<p className={styles.storyText}>
						Planifica lo que falta, registra lo que costó y consulta quién pagó.
						Sin balances que nadie pidió.
					</p>
					<div className={styles.receipt} aria-hidden="true">
						<div className={styles.receiptRow}>
							<span>Leche</span>
							<strong>$120.00</strong>
						</div>
						<div className={styles.receiptRow}>
							<span>Huevos</span>
							<strong>$85.00</strong>
						</div>
						<div className={styles.receiptRow}>
							<span>Arroz</span>
							<strong>$60.00</strong>
						</div>
						<div className={styles.receiptDivider} />
						<div className={styles.receiptRow}>
							<span>Pagado por Ana</span>
							<strong>$120.00</strong>
						</div>
						<div className={styles.receiptRow}>
							<span>Pagado por Pedro</span>
							<strong>$145.00</strong>
						</div>
						<span className={styles.receiptNote}>
							Aportaciones reales, no deudas.
						</span>
					</div>
				</div>
			</aside>
		</div>
	);
}
