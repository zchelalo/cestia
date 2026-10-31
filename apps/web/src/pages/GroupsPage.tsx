import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Archive, Plus, Users } from "lucide-react";
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
import type { Group } from "@/shared/types";

const schema = z.object({
	name: z
		.string()
		.trim()
		.min(1, "Escribe un nombre.")
		.max(120, "Máximo 120 caracteres."),
	description: z.string().max(2000).optional(),
});
type Form = z.infer<typeof schema>;

export function GroupsPage() {
	const { t } = useTranslation();
	const [creating, setCreating] = useState(false);
	const [error, setError] = useState("");
	const queryClient = useQueryClient();
	const groups = useQuery({
		queryKey: ["groups"],
		queryFn: () => apiData<Group[]>("/api/v1/groups"),
	});
	const form = useForm<Form>({
		resolver: zodResolver(schema),
		defaultValues: { name: "", description: "" },
	});
	const mutation = useMutation({
		mutationFn: (values: Form) =>
			apiData<Group>("/api/v1/groups", { method: "POST", body: values }),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ["groups"] });
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

	if (groups.isLoading) return <LoadingState />;
	if (groups.isError)
		return (
			<ErrorState
				message={groups.error.message}
				onRetry={() => groups.refetch()}
			/>
		);

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={t("nav.groups")}
				title={t("groups.title")}
				subtitle={t("groups.subtitle")}
				actions={
					<Button onClick={() => setCreating(true)}>
						<Plus size={16} />
						{t("groups.newGroup")}
					</Button>
				}
			/>
			{groups.data?.length === 0 ? (
				<div className={styles.empty}>
					<Users size={28} />
					<br />
					{t("groups.empty")}
				</div>
			) : (
				<div className={styles.grid3}>
					{groups.data?.map((group) => (
						<Link
							key={group.id}
							to="/groups/$groupId"
							params={{ groupId: group.id }}
							className={`${styles.card} ${styles.cardLink}`}
						>
							<div className={styles.inline}>
								<span className={`${styles.badge} ${styles.badgePrimary}`}>
									{group.role}
								</span>
								{group.archived_at ? (
									<span className={styles.badge}>
										<Archive size={12} />
										{t("common.archived")}
									</span>
								) : null}
							</div>
							<h2 className={styles.cardTitle} style={{ marginTop: "0.8rem" }}>
								{group.name}
							</h2>
							<p className={styles.cardMeta}>
								{group.description || "Sin descripción"}
							</p>
							<p className={styles.metricHint}>
								{group.member_count} integrantes · {group.list_count} listas
							</p>
						</Link>
					))}
				</div>
			)}

			{creating ? (
				<Modal title={t("groups.newGroup")} onClose={() => setCreating(false)}>
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
								placeholder="Alacena, Casa, Oficina…"
								{...form.register("name")}
							/>
						</Field>
						<Field
							label={`${t("common.description")} (${t("common.optional")})`}
						>
							<Textarea
								placeholder="Para qué usan este grupo"
								{...form.register("description")}
							/>
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setCreating(false)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={mutation.isPending}>
								{t("common.create")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}
		</div>
	);
}
