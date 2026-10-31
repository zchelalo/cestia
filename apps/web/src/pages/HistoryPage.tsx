import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { ActivityTimeline } from "@/components/ActivityTimeline";
import { Button } from "@/components/Button";
import { Select } from "@/components/molecules/select";
import { PageHeader } from "@/components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import styles from "@/pages/Page.module.css";
import { useGroupSelectOptions } from "@/shared/catalogs";
import { apiData, apiPage } from "@/shared/http";
import { formatDate, formatMoney, unitLabels } from "@/shared/money";
import type { Activity, Group, Purchase } from "@/shared/types";

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

	const purchases = useInfiniteQuery({
		queryKey: ["history", "purchases", scope, from, to],
		initialPageParam: "",
		queryFn: ({ pageParam }) => {
			const query = new URLSearchParams({ limit: "30" });
			if (scope) query.set("group_id", scope);
			if (from) query.set("from", new Date(`${from}T00:00:00`).toISOString());
			if (to) query.set("to", new Date(`${to}T00:00:00`).toISOString());
			if (pageParam) query.set("cursor", pageParam);
			return apiPage<Purchase[]>(`/api/v1/purchases?${query.toString()}`);
		},
		getNextPageParam: (page) => page.meta?.next_cursor || undefined,
	});
	const activity = useQuery({
		queryKey: ["history", "activity", scope],
		queryFn: () =>
			apiData<Activity[]>(
				`/api/v1/activity?${scope ? `group_id=${scope}&` : ""}limit=50`,
			),
		enabled: tab === "activity",
	});

	const allPurchases = purchases.data?.pages.flatMap((page) => page.data) ?? [];

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
					<label className={styles.filterItem}>
						{t("expenses.from")}
						<input
							className={styles.select}
							type="date"
							value={from}
							onChange={(event) => setFrom(event.target.value)}
						/>
					</label>
					<label className={styles.filterItem}>
						{t("expenses.to")}
						<input
							className={styles.select}
							type="date"
							value={to}
							onChange={(event) => setTo(event.target.value)}
						/>
					</label>
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
						message={purchases.error.message}
						onRetry={() => purchases.refetch()}
					/>
				) : allPurchases.length === 0 ? (
					<EmptyState>{t("history.empty")}</EmptyState>
				) : (
					<>
						<div className={styles.list}>
							{allPurchases.map((purchase) => (
								<div key={purchase.id} className={styles.row}>
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
											Compró {purchase.purchased_by_name} · pagó{" "}
											{purchase.paid_by_name ?? t("lists.unidentified")} ·
											registró {purchase.recorded_by_name}
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
						{purchases.hasNextPage ? (
							<Button
								variant="secondary"
								onClick={() => purchases.fetchNextPage()}
								disabled={purchases.isFetchingNextPage}
							>
								Cargar más
							</Button>
						) : null}
					</>
				)
			) : activity.isLoading ? (
				<LoadingState />
			) : activity.isError ? (
				<ErrorState
					message={activity.error.message}
					onRetry={() => activity.refetch()}
				/>
			) : (
				<div className={styles.card}>
					<ActivityTimeline events={activity.data ?? []} />
				</div>
			)}
		</div>
	);
}
