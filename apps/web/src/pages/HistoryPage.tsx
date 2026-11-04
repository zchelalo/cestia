import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { ActivityTimeline } from "@/components/ActivityTimeline";
import { CursorPagination } from "@/components/CursorPagination";
import { InputDate } from "@/components/molecules/input-date";
import { Select } from "@/components/molecules/select";
import { PageHeader } from "@/components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import styles from "@/pages/Page.module.css";
import { useGroupSelectOptions } from "@/shared/catalogs";
import { apiData, apiPage } from "@/shared/http";
import { formatDate, formatMoney, unitLabels } from "@/shared/money";
import { pageFetcher } from "@/shared/pages";
import type { Activity, Group, Purchase } from "@/shared/types";
import { useCursorPage } from "@/shared/use-cursor-page";

export function HistoryPage() {
	const { t } = useTranslation();
	const search = useSearch({ strict: false }) as { group_id?: string };
	const [tab, setTab] = useState<"purchases" | "activity">("purchases");
	const [scope, setScope] = useState(search.group_id ?? "");
	const [from, setFrom] = useState("");
	const [to, setTo] = useState("");

	const groupCatalog = useGroupSelectOptions([
		{ value: "", label: t("expenses.personal"), local: true },
	]);
	const selectedGroup = useQuery({
		queryKey: ["group", scope],
		queryFn: () => apiData<Group>(`/api/v1/groups/${scope}`),
		enabled: !!scope && !groupCatalog.labelFor(scope),
	});
	const scopeLabel = groupCatalog.labelFor(scope) ?? selectedGroup.data?.name;

	const purchases = useCursorPage<Purchase>({
		queryKey: ["history", "purchases", scope, from, to],
		fetchPage: (input) => {
			const query = new URLSearchParams({ limit: String(input.limit) });
			if (scope) query.set("group_id", scope);
			if (from) query.set("from", new Date(`${from}T00:00:00`).toISOString());
			if (to) query.set("to", new Date(`${to}T00:00:00`).toISOString());
			if (input.cursor) query.set("cursor", input.cursor);
			return apiPage<Purchase[]>(`/api/v1/purchases?${query.toString()}`, {
				signal: input.signal,
			});
		},
	});
	const activity = useCursorPage<Activity>({
		queryKey: ["history", "activity", scope],
		fetchPage: pageFetcher<Activity>(
			"/api/v1/activity",
			scope ? { group_id: scope } : {},
		),
		enabled: tab === "activity",
	});

	const allPurchases = purchases.items;

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={t("nav.history")}
				title={t("history.title")}
				subtitle={t("history.subtitle")}
			/>

			<div className={styles.card}>
				<div className={styles.filters}>
					<div className={styles.filterItem}>
						<span>{t("expenses.scope")}</span>
						<Select
							aria-label={t("expenses.scope")}
							remote
							options={groupCatalog.options}
							value={scope}
							selectedLabel={scopeLabel}
							onChange={setScope}
							searchValue={groupCatalog.searchInput}
							onSearchChange={groupCatalog.setSearchInput}
							isLoading={groupCatalog.isLoading}
							isLoadingMore={groupCatalog.isFetchingNextPage}
							hasMore={groupCatalog.hasMore}
							onLoadMore={groupCatalog.loadMore}
							placeholder={t("expenses.personal")}
						/>
					</div>
					<div className={styles.filterItem}>
						<span>{t("expenses.from")}</span>
						<InputDate
							aria-label={t("expenses.from")}
							value={from}
							onChange={setFrom}
						/>
					</div>
					<div className={styles.filterItem}>
						<span>{t("expenses.to")}</span>
						<InputDate
							aria-label={t("expenses.to")}
							value={to}
							onChange={setTo}
						/>
					</div>
				</div>
			</div>

			<div className={styles.toggle}>
				<button
					type="button"
					className={`${styles.toggleButton} ${tab === "purchases" ? styles.toggleActive : ""}`}
					onClick={() => setTab("purchases")}
				>
					{t("history.purchases")}
				</button>
				<button
					type="button"
					className={`${styles.toggleButton} ${tab === "activity" ? styles.toggleActive : ""}`}
					onClick={() => setTab("activity")}
				>
					{t("history.activity")}
				</button>
			</div>

			{tab === "purchases" ? (
				purchases.isLoading ? (
					<LoadingState />
				) : purchases.isError ? (
					<ErrorState
						message={
							purchases.error?.message ?? "No se pudo cargar el historial."
						}
						onRetry={() => purchases.refetch()}
					/>
				) : allPurchases.length === 0 ? (
					<EmptyState>{t("history.empty")}</EmptyState>
				) : (
					<>
						<div className={styles.list}>
							{allPurchases.map((purchase) => (
								<div
									key={purchase.id}
									className={`${styles.row} ${styles.historyRow}`}
								>
									<span className={styles.rowMain}>
										<span className={styles.inline}>
											<span className={styles.rowTitle}>
												{purchase.item_name}
											</span>
											{purchase.reversed_at ? (
												<span
													className={`${styles.badge} ${styles.badgeDanger}`}
												>
													Revertida
												</span>
											) : (
												<span
													className={`${styles.badge} ${styles.badgeSuccess}`}
												>
													Activa
												</span>
											)}
										</span>
										<span className={styles.rowMeta}>
											{purchase.quantity}{" "}
											{unitLabels[purchase.unit] ?? purchase.unit} ·{" "}
											{formatDate(purchase.purchased_at, true)}
										</span>
										<span className={styles.rowMeta}>
											<span>Compró {purchase.purchased_by_name}</span>
											{" · "}
											<span>
												Pagó {purchase.paid_by_name ?? t("lists.unidentified")}
											</span>
											{" · "}
											<span>Registró {purchase.recorded_by_name}</span>
										</span>
										{purchase.reversal_reason ? (
											<span className={styles.rowMeta}>
												Motivo: {purchase.reversal_reason}
											</span>
										) : null}
									</span>
									<span className={styles.itemPrice}>
										<span className={styles.itemTotal}>
											{formatMoney(purchase.total_amount)}
										</span>
										<span className={styles.rowMeta}>
											{purchase.actual_unit_price
												? `${formatMoney(purchase.actual_unit_price)} / unidad`
												: t("lists.noPrice")}
										</span>
									</span>
								</div>
							))}
						</div>
						<CursorPagination
							label="Paginación del historial de compras"
							canGoNext={purchases.canGoNext}
							canGoPrevious={purchases.canGoPrevious}
							isFetching={purchases.isFetching}
							pageSize={purchases.pageSize}
							onPageSizeChange={purchases.setPageSize}
							onNext={purchases.goNext}
							onPrevious={purchases.goPrevious}
						/>
					</>
				)
			) : activity.isLoading ? (
				<LoadingState />
			) : activity.isError ? (
				<ErrorState
					message={activity.error?.message ?? "No se pudo cargar la actividad."}
					onRetry={() => activity.refetch()}
				/>
			) : (
				<>
					<div className={styles.card}>
						<ActivityTimeline events={activity.items} />
					</div>
					<CursorPagination
						label="Paginación de actividad"
						canGoNext={activity.canGoNext}
						canGoPrevious={activity.canGoPrevious}
						isFetching={activity.isFetching}
						pageSize={activity.pageSize}
						onPageSizeChange={activity.setPageSize}
						onNext={activity.goNext}
						onPrevious={activity.goPrevious}
					/>
				</>
			)}
		</div>
	);
}
