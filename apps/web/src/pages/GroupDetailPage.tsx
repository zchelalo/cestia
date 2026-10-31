import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "@tanstack/react-router";
import {
	Archive,
	ArrowRight,
	Pencil,
	Plus,
	RotateCcw,
	Settings2,
	Users,
} from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { ActivityTimeline } from "@/components/ActivityTimeline";
import { Button } from "@/components/Button";
import { Field, Input, Textarea } from "@/components/Field";
import { Modal } from "@/components/Modal";
import { PageHeader } from "@/components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import { SummaryCard } from "@/components/SummaryCard";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatMoney } from "@/shared/money";
import type { Activity, Group, ShoppingList, Summary } from "@/shared/types";

const groupSchema = z.object({
	name: z.string().trim().min(1, "Escribe un nombre.").max(120),
	description: z.string().max(2000).optional(),
});
type GroupForm = z.infer<typeof groupSchema>;

const listSchema = z.object({
	name: z.string().trim().min(1, "Escribe un nombre.").max(120),
	description: z.string().max(2000).optional(),
});
type ListForm = z.infer<typeof listSchema>;

export function GroupDetailPage() {
	const { t } = useTranslation();
	const { groupId } = useParams({ from: "/groups/$groupId" });
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const [editing, setEditing] = useState(false);
	const [creatingList, setCreatingList] = useState(false);
	const [message, setMessage] = useState("");
	const group = useQuery({
		queryKey: ["group", groupId],
		queryFn: () => apiData<Group>(`/api/v1/groups/${groupId}`),
	});
	const lists = useQuery({
		queryKey: ["lists", "group", groupId, true],
		queryFn: () =>
			apiData<ShoppingList[]>(
				`/api/v1/shopping-lists?group_id=${groupId}&include_archived=true`,
			),
		refetchInterval: 30_000,
	});
	const summary = useQuery({
		queryKey: ["summary", "group", groupId],
		queryFn: () =>
			apiData<Summary>(
				`/api/v1/expenses/summary?group_id=${groupId}&include_archived=true`,
			),
		refetchInterval: 30_000,
	});
	const activity = useQuery({
		queryKey: ["activity", "group", groupId],
		queryFn: () =>
			apiData<Activity[]>(`/api/v1/activity?group_id=${groupId}&limit=10`),
		refetchInterval: 30_000,
	});
	const groupForm = useForm<GroupForm>({
		resolver: zodResolver(groupSchema),
		values: {
			name: group.data?.name ?? "",
			description: group.data?.description ?? "",
		},
	});
	const listForm = useForm<ListForm>({
		resolver: zodResolver(listSchema),
		defaultValues: { name: "", description: "" },
	});
	const updateGroup = useMutation({
		mutationFn: (values: GroupForm) =>
			apiData<Group>(`/api/v1/groups/${groupId}`, {
				method: "PATCH",
				body: { ...values, expected_version: group.data?.version },
			}),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ["group", groupId] });
			await queryClient.invalidateQueries({ queryKey: ["groups"] });
			setEditing(false);
		},
	});
	const archiveGroup = useMutation({
		mutationFn: (archived: boolean) =>
			apiData<Group>(
				`/api/v1/groups/${groupId}/${archived ? "archive" : "restore"}`,
				{
					method: "POST",
					body: { expected_version: group.data?.version },
				},
			),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ["group", groupId] });
			await queryClient.invalidateQueries({ queryKey: ["groups"] });
			await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
		},
	});
	const createList = useMutation({
		mutationFn: (values: ListForm) =>
			apiData<ShoppingList>("/api/v1/shopping-lists", {
				method: "POST",
				body: { ...values, group_id: groupId },
			}),
		onSuccess: async () => {
			await queryClient.invalidateQueries({
				queryKey: ["lists", "group", groupId],
			});
			await queryClient.invalidateQueries({ queryKey: ["group", groupId] });
			setCreatingList(false);
			listForm.reset();
		},
	});

	if (group.isLoading) return <LoadingState />;
	if (group.isError)
		return (
			<ErrorState
				message={group.error.message}
				onRetry={() => group.refetch()}
			/>
		);
	const data = group.data;
	if (!data) return null;
	const canManage = data.role === "owner" || data.role === "admin";
	const canConfigure = data.role === "owner";

	async function submitGroup(values: GroupForm) {
		setMessage("");
		try {
			await updateGroup.mutateAsync(values);
		} catch (cause) {
			setMessage(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	async function submitList(values: ListForm) {
		setMessage("");
		try {
			await createList.mutateAsync(values);
		} catch (cause) {
			setMessage(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={`${t("groups.group")} · ${t(`groups.${data.role}`)}`}
				title={data.name}
				subtitle={data.description || t("groups.subtitle")}
				actions={
					<>
						<Link to="/groups/$groupId/members" params={{ groupId }}>
							<Button variant="secondary">
								<Users size={16} />
								{t("groups.members")}
							</Button>
						</Link>
						{canConfigure ? (
							<>
								<Button variant="quiet" onClick={() => setEditing(true)}>
									<Pencil size={16} />
									{t("common.edit")}
								</Button>
								<Button
									variant="quiet"
									onClick={() => archiveGroup.mutate(!data.archived_at)}
									disabled={archiveGroup.isPending}
								>
									{data.archived_at ? (
										<RotateCcw size={16} />
									) : (
										<Archive size={16} />
									)}
									{data.archived_at ? t("groups.restore") : t("groups.archive")}
								</Button>
							</>
						) : null}
					</>
				}
			/>

			{data.archived_at ? (
				<div className={styles.notice}>{t("lists.readOnlyArchived")}</div>
			) : null}
			{message ? (
				<div className={styles.error} role="alert">
					{message}
				</div>
			) : null}

			<section className={styles.grid4}>
				<SummaryCard
					label={t("expenses.actual")}
					value={formatMoney(summary.data?.actual_known)}
					hint={`${summary.data?.purchase_count ?? 0} ${(summary.data?.purchase_count ?? 0) === 1 ? "compra" : "compras"}`}
				/>
				<SummaryCard
					label={t("expenses.estimated")}
					value={formatMoney(summary.data?.estimated_known)}
					hint={`${summary.data?.estimated_unknown_items ?? 0} ${t("expenses.unknownEstimates")}`}
				/>
				<SummaryCard
					label={t("expenses.pendingItems")}
					value={summary.data?.pending_count ?? 0}
				/>
				<SummaryCard
					label={t("groups.members")}
					value={data.member_count}
					hint={`${data.list_count} ${data.list_count === 1 ? "lista activa" : "listas activas"}`}
				/>
			</section>

			<section className={styles.section}>
				<div className={styles.sectionHead}>
					<h2 className={styles.sectionTitle}>Listas del grupo</h2>
					{canManage && !data.archived_at ? (
						<Button small onClick={() => setCreatingList(true)}>
							<Plus size={15} />
							{t("lists.newList")}
						</Button>
					) : null}
				</div>
				{lists.data?.length === 0 ? (
					<EmptyState>{t("common.noResults")}</EmptyState>
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
									</span>
								</span>
								<span className={styles.rowActions}>
									{list.archived_at ? (
										<span className={styles.badge}>{t("common.archived")}</span>
									) : null}
									<ArrowRight size={17} color="var(--color-muted)" />
								</span>
							</Link>
						))}
					</div>
				)}
			</section>

			<section className={styles.section}>
				<div className={styles.sectionHead}>
					<h2 className={styles.sectionTitle}>{t("history.activity")}</h2>
				</div>
				<div className={styles.card}>
					<ActivityTimeline events={activity.data ?? []} />
				</div>
			</section>

			{editing ? (
				<Modal title={t("common.edit")} onClose={() => setEditing(false)}>
					<form
						className={styles.form}
						onSubmit={groupForm.handleSubmit(submitGroup)}
					>
						<Field
							label={t("common.name")}
							error={groupForm.formState.errors.name?.message}
						>
							<Input {...groupForm.register("name")} />
						</Field>
						<Field label={t("common.description")}>
							<Textarea {...groupForm.register("description")} />
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setEditing(false)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={updateGroup.isPending}>
								{t("common.save")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}

			{creatingList ? (
				<Modal
					title={t("lists.newList")}
					onClose={() => setCreatingList(false)}
				>
					<form
						className={styles.form}
						onSubmit={listForm.handleSubmit(submitList)}
					>
						<Field
							label={t("common.name")}
							error={listForm.formState.errors.name?.message}
						>
							<Input autoFocus {...listForm.register("name")} />
						</Field>
						<Field
							label={`${t("common.description")} (${t("common.optional")})`}
						>
							<Textarea {...listForm.register("description")} />
						</Field>
						<div className={styles.formActions}>
							<Button
								variant="secondary"
								onClick={() => setCreatingList(false)}
							>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={createList.isPending}>
								{t("common.create")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}

			<div className={styles.inline}>
				<Link to="/expenses" search={{ group_id: groupId } as never}>
					<Button variant="quiet">
						<Settings2 size={15} />
						{t("expenses.title")}
					</Button>
				</Link>
				<Button
					variant="quiet"
					onClick={() => void navigate({ to: "/groups" })}
				>
					{t("common.back")}
				</Button>
			</div>
		</div>
	);
}
