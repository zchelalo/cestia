import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ArrowRight, Plus, ShoppingBasket, Users } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ActivityTimeline } from "@/components/ActivityTimeline";
import { Button } from "@/components/Button";
import { CursorPagination } from "@/components/CursorPagination";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import { SummaryCard } from "@/components/SummaryCard";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatDate, formatMoney } from "@/shared/money";
import { pageFetcher } from "@/shared/pages";
import type {
	Activity,
	DashboardSummary,
	Group,
	Purchase,
	ShoppingList,
} from "@/shared/types";
import { useCursorPage } from "@/shared/use-cursor-page";

function SectionError({ onRetry }: { onRetry: () => void }) {
	return (
		<div className={styles.empty}>
			No se pudieron cargar los datos.{" "}
			<button type="button" className={styles.small} onClick={onRetry}>
				Reintentar
			</button>
		</div>
	);
}

export function DashboardPage() {
	const { t } = useTranslation();
	const dashboard = useQuery({
		queryKey: ["dashboard", "summary"],
		queryFn: () => apiData<DashboardSummary>("/api/v1/dashboard/summary"),
	});
	const groups = useCursorPage<Group>({
		queryKey: ["groups", "dashboard"],
		fetchPage: pageFetcher<Group>("/api/v1/groups"),
	});
	const lists = useCursorPage<ShoppingList>({
		queryKey: ["lists", "dashboard"],
		fetchPage: pageFetcher<ShoppingList>("/api/v1/shopping-lists", {
			include_archived: "false",
		}),
	});
	const purchases = useCursorPage<Purchase>({
		queryKey: ["purchases", "dashboard"],
		fetchPage: pageFetcher<Purchase>("/api/v1/purchases"),
	});
	const activity = useCursorPage<Activity>({
		queryKey: ["activity", "dashboard"],
		fetchPage: pageFetcher<Activity>("/api/v1/activity"),
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

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={formatDate(new Date().toISOString())}
				title={t("dashboard.title")}
				subtitle={t("dashboard.subtitle")}
				actionsGrid
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
					value={data.groups_count}
					hint={
						data.members_total === 1
							? "1 integrante en total"
							: `${data.members_total} integrantes en total`
					}
				/>
				<SummaryCard
					label={t("dashboard.lists")}
					value={data.personal_lists_count}
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
				{groups.isError ? (
					<SectionError onRetry={() => groups.refetch()} />
				) : groups.items.length === 0 ? (
					<div className={styles.empty}>{t("groups.empty")}</div>
				) : (
					<div className={styles.grid3}>
						{groups.items.map((group) => (
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
				<CursorPagination
					label="Paginación de grupos"
					canGoNext={groups.canGoNext}
					canGoPrevious={groups.canGoPrevious}
					isFetching={groups.isFetching}
					pageSize={groups.pageSize}
					onPageSizeChange={groups.setPageSize}
					onNext={groups.goNext}
					onPrevious={groups.goPrevious}
				/>
			</section>

			<div className={styles.grid2}>
				<section className={`${styles.section} ${styles.dashboardSection}`}>
					<div className={styles.sectionHead}>
						<h2 className={styles.sectionTitle}>{t("dashboard.lists")}</h2>
						<Link to="/personal" className={styles.small}>
							{t("dashboard.viewAll")} →
						</Link>
					</div>
					{lists.isError ? (
						<SectionError onRetry={() => lists.refetch()} />
					) : lists.items.length === 0 ? (
						<div className={styles.empty}>{t("lists.empty")}</div>
					) : (
						<div className={`${styles.list} ${styles.dashboardBody}`}>
							{lists.items.map((list) => (
								<Link
									key={list.id}
									to="/lists/$listId"
									params={{ listId: list.id }}
									className={`${styles.row} ${styles.compactRow}`}
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
					<CursorPagination
						label="Paginación de mis listas"
						canGoNext={lists.canGoNext}
						canGoPrevious={lists.canGoPrevious}
						isFetching={lists.isFetching}
						pageSize={lists.pageSize}
						onPageSizeChange={lists.setPageSize}
						onNext={lists.goNext}
						onPrevious={lists.goPrevious}
					/>
				</section>

				<section className={`${styles.section} ${styles.dashboardSection}`}>
					<div className={styles.sectionHead}>
						<h2 className={styles.sectionTitle}>
							{t("dashboard.recentPurchases")}
						</h2>
						<Link to="/history" className={styles.small}>
							{t("dashboard.viewAll")} →
						</Link>
					</div>
					{purchases.isError ? (
						<SectionError onRetry={() => purchases.refetch()} />
					) : purchases.items.length === 0 ? (
						<div className={styles.empty}>
							<ShoppingBasket size={24} />
							<br />
							{t("common.noResults")}
						</div>
					) : (
						<div className={`${styles.list} ${styles.dashboardBody}`}>
							{purchases.items.map((purchase) => (
								<div
									key={purchase.id}
									className={`${styles.row} ${styles.compactRow}`}
								>
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
					<CursorPagination
						label="Paginación de compras recientes"
						canGoNext={purchases.canGoNext}
						canGoPrevious={purchases.canGoPrevious}
						isFetching={purchases.isFetching}
						pageSize={purchases.pageSize}
						onPageSizeChange={purchases.setPageSize}
						onNext={purchases.goNext}
						onPrevious={purchases.goPrevious}
					/>
				</section>
			</div>

			<section className={styles.section}>
				<div className={styles.sectionHead}>
					<h2 className={styles.sectionTitle}>
						{t("dashboard.recentActivity")}
					</h2>
				</div>
				<div className={styles.card}>
					{activity.isError ? (
						<SectionError onRetry={() => activity.refetch()} />
					) : (
						<ActivityTimeline events={activity.items} />
					)}
				</div>
				<CursorPagination
					label="Paginación de actividad reciente"
					canGoNext={activity.canGoNext}
					canGoPrevious={activity.canGoPrevious}
					isFetching={activity.isFetching}
					pageSize={activity.pageSize}
					onPageSizeChange={activity.setPageSize}
					onNext={activity.goNext}
					onPrevious={activity.goPrevious}
				/>
			</section>
		</div>
	);
}
