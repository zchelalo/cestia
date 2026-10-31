import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ArrowRight, Plus, ShoppingBasket, Users } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ActivityTimeline } from "@/components/ActivityTimeline";
import { Button } from "@/components/Button";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import { SummaryCard } from "@/components/SummaryCard";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatDate, formatMoney } from "@/shared/money";
import type { Dashboard } from "@/shared/types";

export function DashboardPage() {
	const { t } = useTranslation();
	const dashboard = useQuery({
		queryKey: ["dashboard"],
		queryFn: () => apiData<Dashboard>("/api/v1/dashboard"),
	});

	if (dashboard.isLoading) return <LoadingState />;
	if (dashboard.isError)
		return (
			<ErrorState
				message={dashboard.error.message}
				onRetry={() => dashboard.refetch()}
			/>
		);
	const data = dashboard.data;
	if (!data) return null;
	const memberTotal = data.groups.reduce(
		(sum, group) => sum + group.member_count,
		0,
	);

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={formatDate(new Date().toISOString())}
				title={t("dashboard.title")}
				subtitle={t("dashboard.subtitle")}
				actions={
					<>
						<Link to="/personal">
							<Button variant="secondary">
								<Plus size={16} />
								{t("dashboard.createList")}
							</Button>
						</Link>
						<Link to="/groups">
							<Button>
								<Users size={16} />
								{t("dashboard.createGroup")}
							</Button>
						</Link>
					</>
				}
			/>

			<section className={styles.grid4} aria-label="Resumen">
				<SummaryCard
					label={t("dashboard.personalSpend")}
					value={formatMoney(data.summary.actual_known)}
					hint={`${data.summary.purchase_count} ${data.summary.purchase_count === 1 ? "compra" : "compras"} · ${data.summary.purchases_without_price} ${t("expenses.unknownPrices")}`}
				/>
				<SummaryCard
					label={t("dashboard.pending")}
					value={data.summary.pending_count}
					hint={`${formatMoney(data.summary.pending_estimated)} ${t("expenses.pendingEstimated").toLowerCase()}`}
				/>
				<SummaryCard
					label={t("dashboard.groups")}
					value={data.groups.length}
					hint={
						memberTotal === 1
							? "1 integrante en total"
							: `${memberTotal} integrantes en total`
					}
				/>
				<SummaryCard
					label={t("dashboard.lists")}
					value={data.personal_lists.length}
					hint="Espacio privado"
				/>
			</section>

			<section className={styles.section}>
				<div className={styles.sectionHead}>
					<h2 className={styles.sectionTitle}>{t("dashboard.groups")}</h2>
					<Link to="/groups" className={styles.small}>
						{t("dashboard.viewAll")} →
					</Link>
				</div>
				{data.groups.length === 0 ? (
					<div className={styles.empty}>{t("groups.empty")}</div>
				) : (
					<div className={styles.grid3}>
						{data.groups.slice(0, 3).map((group) => (
							<Link
								key={group.id}
								to="/groups/$groupId"
								params={{ groupId: group.id }}
								className={`${styles.card} ${styles.cardLink}`}
							>
								<div className={styles.inline}>
									<span className={`${styles.badge} ${styles.badgePrimary}`}>
										{t(`groups.${group.role}`)}
									</span>
									{group.archived_at ? (
										<span className={styles.badge}>{t("common.archived")}</span>
									) : null}
								</div>
								<h3
									className={styles.cardTitle}
									style={{ marginTop: "0.75rem" }}
								>
									{group.name}
								</h3>
								<p className={styles.cardMeta}>
									{group.member_count}{" "}
									{group.member_count === 1 ? "integrante" : "integrantes"} ·{" "}
									{group.list_count}{" "}
									{group.list_count === 1 ? "lista" : "listas"}
								</p>
							</Link>
						))}
					</div>
				)}
			</section>

			<div className={styles.grid2}>
				<section className={styles.section}>
					<div className={styles.sectionHead}>
						<h2 className={styles.sectionTitle}>{t("dashboard.lists")}</h2>
						<Link to="/personal" className={styles.small}>
							{t("dashboard.viewAll")} →
						</Link>
					</div>
					{data.personal_lists.length === 0 ? (
						<div className={styles.empty}>{t("lists.empty")}</div>
					) : (
						<div className={styles.list}>
							{data.personal_lists.slice(0, 5).map((list) => (
								<Link
									key={list.id}
									to="/lists/$listId"
									params={{ listId: list.id }}
									className={styles.row}
									style={{ textDecoration: "none", color: "inherit" }}
								>
									<span className={styles.rowMain}>
										<span className={styles.rowTitle}>{list.name}</span>
										<span className={styles.rowMeta}>
											{list.pending_count} pendientes · {list.purchased_count}{" "}
											comprados
										</span>
									</span>
									<ArrowRight size={17} color="var(--color-muted)" />
								</Link>
							))}
						</div>
					)}
				</section>

				<section className={styles.section}>
					<div className={styles.sectionHead}>
						<h2 className={styles.sectionTitle}>
							{t("dashboard.recentPurchases")}
						</h2>
						<Link to="/history" className={styles.small}>
							{t("dashboard.viewAll")} →
						</Link>
					</div>
					{data.recent_purchases.length === 0 ? (
						<div className={styles.empty}>
							<ShoppingBasket size={24} />
							<br />
							{t("common.noResults")}
						</div>
					) : (
						<div className={styles.list}>
							{data.recent_purchases.map((purchase) => (
								<div key={purchase.id} className={styles.row}>
									<span className={styles.rowMain}>
										<span className={styles.rowTitle}>
											{purchase.item_name}
										</span>
										<span className={styles.rowMeta}>
											{formatDate(purchase.purchased_at)} ·{" "}
											{purchase.purchased_by_name}
										</span>
									</span>
									<span className={styles.itemTotal}>
										{formatMoney(purchase.total_amount)}
									</span>
								</div>
							))}
						</div>
					)}
				</section>
			</div>

			<section className={styles.section}>
				<div className={styles.sectionHead}>
					<h2 className={styles.sectionTitle}>
						{t("dashboard.recentActivity")}
					</h2>
				</div>
				<div className={styles.card}>
					<ActivityTimeline events={data.recent_activity} />
				</div>
			</section>
		</div>
	);
}
