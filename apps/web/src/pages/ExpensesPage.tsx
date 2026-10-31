import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Select } from "@/components/molecules/select";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import { SummaryCard } from "@/components/SummaryCard";
import styles from "@/pages/Page.module.css";
import { useGroupSelectOptions } from "@/shared/catalogs";
import { apiData } from "@/shared/http";
import { formatMoney } from "@/shared/money";
import type { Group, ListExpense, Summary } from "@/shared/types";

export function ExpensesPage() {
	const { t } = useTranslation();
	const search = useSearch({ strict: false }) as { group_id?: string };
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

	const query = new URLSearchParams({ include_archived: "true" });
	if (scope) query.set("group_id", scope);
	if (from) query.set("from", new Date(`${from}T00:00:00`).toISOString());
	if (to) query.set("to", new Date(`${to}T00:00:00`).toISOString());
	const summary = useQuery({
		queryKey: ["summary", "expenses", scope, from, to],
		queryFn: () =>
			apiData<Summary>(`/api/v1/expenses/summary?${query.toString()}`),
	});
	const byList = useQuery({
		queryKey: ["expenses-by-list", scope, from, to],
		queryFn: () =>
			apiData<ListExpense[]>(`/api/v1/expenses/by-list?${query.toString()}`),
	});

	const payerRows = summary.data?.by_payer ?? [];
	const maxPayer = useMemo(() => {
		return Math.max(...payerRows.map((payer) => Number(payer.amount)), 1);
	}, [payerRows]);

	if (summary.isLoading) return <LoadingState />;
	if (summary.isError)
		return (
			<ErrorState
				message={summary.error.message}
				onRetry={() => summary.refetch()}
			/>
		);
	const data = summary.data;
	if (!data) return null;

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={t("nav.expenses")}
				title={t("expenses.title")}
				subtitle={t("expenses.subtitle")}
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
					{from || to ? (
						<button
							type="button"
							className={styles.toggleButton}
							onClick={() => {
								setFrom("");
								setTo("");
							}}
						>
							Limpiar fechas
						</button>
					) : null}
				</div>
			</div>

			<section className={styles.grid4}>
				<SummaryCard
					label={t("expenses.estimated")}
					value={formatMoney(data.estimated_known)}
					hint={`${data.estimated_unknown_items} ${t("expenses.unknownEstimates")}`}
				/>
				<SummaryCard
					label={t("expenses.pendingEstimated")}
					value={formatMoney(data.pending_estimated)}
					hint={`${data.pending_count} pendientes`}
				/>
				<SummaryCard
					label={t("expenses.actual")}
					value={formatMoney(data.actual_known)}
					hint={`${data.purchase_count} ${data.purchase_count === 1 ? "compra" : "compras"} · ${data.purchases_without_price} ${t("expenses.unknownPrices")}`}
				/>
				<SummaryCard
					label={t("expenses.purchases")}
					value={data.purchased_count}
					hint={`${data.pending_count} por comprar`}
				/>
			</section>

			<div className={styles.notice}>{t("expenses.noDebts")}</div>
			<div className={styles.notice}>{t("expenses.knownNotice")}</div>

			<section className={styles.card}>
				<h2 className={styles.cardTitle}>Gasto por lista</h2>
				<p className={styles.cardMeta}>
					Los totales reales incluyen únicamente compras activas. Las listas
					archivadas conservan su historial.
				</p>
				{byList.data?.length === 0 ? (
					<div className={styles.empty} style={{ marginTop: "1rem" }}>
						{t("common.noResults")}
					</div>
				) : (
					<div className={styles.list} style={{ marginTop: "1rem" }}>
						{byList.data?.map((list) => (
							<div key={list.list_id} className={styles.row}>
								<span className={styles.rowMain}>
									<span className={styles.rowTitle}>
										{list.name}
										{list.archived_at ? " · archivada" : ""}
									</span>
									<span className={styles.rowMeta}>
										{list.purchase_count}{" "}
										{list.purchase_count === 1 ? "compra" : "compras"} ·{" "}
										{list.pending_count}{" "}
										{list.pending_count === 1 ? "pendiente" : "pendientes"} ·{" "}
										{formatMoney(list.pending_estimated)} estimado pendiente
									</span>
								</span>
								<span className={styles.itemTotal}>
									{formatMoney(list.actual_known)}
								</span>
							</div>
						))}
					</div>
				)}
			</section>

			<section className={styles.grid2}>
				<div className={styles.card}>
					<h2 className={styles.cardTitle}>{t("expenses.byPayer")}</h2>
					<p className={styles.cardMeta}>
						Agrupado por quien pagó, no por quien registró o compró.
					</p>
					{payerRows.length === 0 ? (
						<div className={styles.empty} style={{ marginTop: "1rem" }}>
							{t("common.noResults")}
						</div>
					) : (
						<div
							className={`${styles.payerList} ${styles.section}`}
							style={{ marginTop: "1rem" }}
						>
							{payerRows.map((payer) => (
								<div
									key={payer.paid_by ?? "unknown"}
									className={styles.payerRow}
								>
									<div>
										<div className={styles.payerName}>
											{payer.paid_by
												? payer.display_name
												: t("expenses.noPayer")}
										</div>
										<div className={styles.bar}>
											<div
												className={styles.barFill}
												style={{
													width: `${Math.max(3, (Number(payer.amount) / maxPayer) * 100)}%`,
												}}
											/>
										</div>
									</div>
									<span className={styles.payerAmount}>
										{formatMoney(payer.amount)}
									</span>
								</div>
							))}
						</div>
					)}
				</div>
				<div className={styles.card}>
					<h2 className={styles.cardTitle}>Lectura del resumen</h2>
					<div className={styles.timeline} style={{ marginTop: "1rem" }}>
						<div className={styles.timelineItem}>
							<span className={styles.dot} />
							<div>
								<strong>Planificado</strong>
								<p className={styles.rowMeta}>
									Precios estimados de los artículos. No es dinero gastado.
								</p>
							</div>
						</div>
						<div className={styles.timelineItem}>
							<span className={styles.dot} />
							<div>
								<strong>Pendiente estimado</strong>
								<p className={styles.rowMeta}>
									Artículos que aún no tienen una compra activa.
								</p>
							</div>
						</div>
						<div className={styles.timelineItem}>
							<span className={styles.dot} />
							<div>
								<strong>Gastado real</strong>
								<p className={styles.rowMeta}>
									Suma de compras con precio registrado. Los precios
									desconocidos no se convierten en cero.
								</p>
							</div>
						</div>
					</div>
				</div>
			</section>
		</div>
	);
}
