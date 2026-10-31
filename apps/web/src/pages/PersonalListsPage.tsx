import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Archive, ArrowRight, Plus } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { Button } from "@/components/Button";
import { Field, Input, Textarea } from "@/components/Field";
import { Modal } from "@/components/Modal";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatMoney } from "@/shared/money";
import type { ShoppingList, Summary } from "@/shared/types";

const schema = z.object({
	name: z
		.string()
		.trim()
		.min(1, "Escribe un nombre.")
		.max(120, "Máximo 120 caracteres."),
	description: z.string().max(2000, "Máximo 2,000 caracteres.").optional(),
});

type Form = z.infer<typeof schema>;

export function PersonalListsPage() {
	const { t } = useTranslation();
	const [showArchived, setShowArchived] = useState(false);
	const [creating, setCreating] = useState(false);
	const [error, setError] = useState("");
	const queryClient = useQueryClient();
	const lists = useQuery({
		queryKey: ["lists", "personal", showArchived],
		queryFn: () =>
			apiData<ShoppingList[]>(
				`/api/v1/shopping-lists?include_archived=${showArchived}`,
			),
	});
	const summary = useQuery({
		queryKey: ["summary", "personal"],
		queryFn: () =>
			apiData<Summary>("/api/v1/expenses/summary?include_archived=true"),
	});
	const form = useForm<Form>({
		resolver: zodResolver(schema),
		defaultValues: { name: "", description: "" },
	});
	const mutation = useMutation({
		mutationFn: (values: Form) =>
			apiData<ShoppingList>("/api/v1/shopping-lists", {
				method: "POST",
				body: { ...values, group_id: null },
			}),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ["lists"] });
			await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
			setCreating(false);
			form.reset();
		},
	});

	async function submit(values: Form) {
		setError("");
		try {
			await mutation.mutateAsync(values);
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	if (lists.isLoading) return <LoadingState />;
	if (lists.isError)
		return (
			<ErrorState
				message={lists.error.message}
				onRetry={() => lists.refetch()}
			/>
		);

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={t("lists.personal")}
				title={t("lists.title")}
				subtitle={t("lists.subtitle")}
				actions={
					<Button onClick={() => setCreating(true)}>
						<Plus size={16} />
						{t("lists.newList")}
					</Button>
				}
			/>

			<section className={styles.grid3}>
				<div className={styles.card}>
					<div className={styles.metricLabel}>{t("expenses.actual")}</div>
					<div className={styles.metricValue}>
						{formatMoney(summary.data?.actual_known)}
					</div>
				</div>
				<div className={styles.card}>
					<div className={styles.metricLabel}>{t("lists.pendingCount")}</div>
					<div className={styles.metricValue}>
						{summary.data?.pending_count ?? "—"}
					</div>
				</div>
				<div className={styles.card}>
					<div className={styles.metricLabel}>{t("lists.purchasedCount")}</div>
					<div className={styles.metricValue}>
						{summary.data?.purchased_count ?? "—"}
					</div>
				</div>
			</section>

			<div className={styles.toolbar}>
				<div className={styles.toggle}>
					<button
						type="button"
						className={`${styles.toggleButton} ${!showArchived ? styles.toggleActive : ""}`}
						onClick={() => setShowArchived(false)}
					>
						{t("common.active")}
					</button>
					<button
						type="button"
						className={`${styles.toggleButton} ${showArchived ? styles.toggleActive : ""}`}
						onClick={() => setShowArchived(true)}
					>
						{t("common.archived")}
					</button>
				</div>
			</div>

			{lists.data?.length === 0 ? (
				<div className={styles.empty}>{t("lists.empty")}</div>
			) : (
				<div className={styles.list}>
					{lists.data?.map((list) => (
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
									{list.pending_count}{" "}
									{list.pending_count === 1 ? "pendiente" : "pendientes"} ·{" "}
									{list.purchased_count}{" "}
									{list.purchased_count === 1 ? "comprado" : "comprados"}
									{list.description ? ` · ${list.description}` : ""}
								</span>
							</span>
							<span className={styles.rowActions}>
								{list.archived_at ? (
									<span className={styles.badge}>
										<Archive size={12} />
										{t("common.archived")}
									</span>
								) : null}
								<ArrowRight size={17} color="var(--color-muted)" />
							</span>
						</Link>
					))}
				</div>
			)}

			{creating ? (
				<Modal title={t("lists.newList")} onClose={() => setCreating(false)}>
					{error ? (
						<div className={styles.error} role="alert">
							{error}
						</div>
					) : null}
					<form className={styles.form} onSubmit={form.handleSubmit(submit)}>
						<Field
							label={t("common.name")}
							error={form.formState.errors.name?.message}
						>
							<Input
								autoFocus
								placeholder="Compras del mes"
								{...form.register("name")}
							/>
						</Field>
						<Field
							label={`${t("common.description")} (${t("common.optional")})`}
						>
							<Textarea
								placeholder="Qué quieres organizar en esta lista"
								{...form.register("description")}
							/>
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setCreating(false)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={mutation.isPending}>
								{mutation.isPending ? t("common.loading") : t("common.create")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}
		</div>
	);
}
