import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { CursorPagination } from "@/components/CursorPagination";
import { InputDate } from "@/components/molecules/input-date";
import { Select } from "@/components/molecules/select";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import { SummaryCard } from "@/components/SummaryCard";
import styles from "@/pages/Page.module.css";
import { useGroupSelectOptions } from "@/shared/catalogs";
import { apiData, apiPage } from "@/shared/http";
import { formatMoney } from "@/shared/money";
import type { Group, ListExpense, PayerTotal, Summary } from "@/shared/types";
import { useCursorPage } from "@/shared/use-cursor-page";

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
	const byList = useCursorPage<ListExpense>({
		queryKey: ["expenses-by-list", scope, from, to],
		fetchPage: (input) => {
			const search = new URLSearchParams({ include_archived: "true" });
			if (scope) search.set("group_id", scope);
			if (from) search.set("from", new Date(`${from}T00:00:00`).toISOString());
			if (to) search.set("to", new Date(`${to}T00:00:00`).toISOString());
			search.set("limit", String(input.limit));
			if (input.cursor) search.set("cursor", input.cursor);
			return apiPage<ListExpense[]>(
				`/api/v1/expenses/by-list?${search.toString()}`,
				{ signal: input.signal },
			);
		},
	});
	const payers = useCursorPage<PayerTotal>({
		queryKey: ["expenses-by-payer", scope, from, to],
		fetchPage: (input) => {
			const search = new URLSearchParams({ include_archived: "true" });
			if (scope) search.set("group_id", scope);
			if (from) search.set("from", new Date(`${from}T00:00:00`).toISOString());
			if (to) search.set("to", new Date(`${to}T00:00:00`).toISOString());
			search.set("limit", String(input.limit));
			if (input.cursor) search.set("cursor", input.cursor);
			return apiPage<PayerTotal[]>(
				`/api/v1/expenses/by-payer?${search.toString()}`,
				{ signal: input.signal },
			);
		},
	});

	const payerRows = payers.items;
	const maxPayer = Math.max(Number(summary.data?.max_payer_amount ?? 0), 1);

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
				{byList.items.length === 0 ? (
					<div className={styles.empty} style={{ marginTop: "1rem" }}>
						{t("common.noResults")}
					</div>
				) : (
					<div className={styles.list} style={{ marginTop: "1rem" }}>
						{byList.items.map((list) => (
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
				<CursorPagination
					label="Paginación de gasto por lista"
					canGoNext={byList.canGoNext}
					canGoPrevious={byList.canGoPrevious}
					isFetching={byList.isFetching}
					onNext={byList.goNext}
					onPrevious={byList.goPrevious}
				/>
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
									<div className={styles.payerHead}>
										<span className={styles.payerName}>
											{payer.paid_by
												? payer.display_name
												: t("expenses.noPayer")}
										</span>
										<span className={styles.payerAmount}>
											{formatMoney(payer.amount)}
										</span>
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
							))}
						</div>
					)}
					<CursorPagination
						label="Paginación de gasto por pagador"
						canGoNext={payers.canGoNext}
						canGoPrevious={payers.canGoPrevious}
						isFetching={payers.isFetching}
						onNext={payers.goNext}
						onPrevious={payers.goPrevious}
					/>
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
