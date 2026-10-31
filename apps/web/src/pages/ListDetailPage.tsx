import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "@tanstack/react-router";
import {
	Archive,
	ArrowLeft,
	Check,
	Pencil,
	Plus,
	RotateCcw,
	ShoppingCart,
	Trash2,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { Button } from "@/components/Button";
import { Field, Input, Select, Textarea } from "@/components/Field";
import { Modal } from "@/components/Modal";
import { PageHeader } from "@/components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import { SummaryCard } from "@/components/SummaryCard";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import {
	formatDate,
	formatMoney,
	lineTotal,
	localDateTimeInput,
	unitLabels,
} from "@/shared/money";
import { useSession } from "@/shared/session";
import type {
	Member,
	Purchase,
	ShoppingItem,
	ShoppingList,
	Summary,
} from "@/shared/types";

const itemSchema = z.object({
	name: z.string().trim().min(1, "Escribe un nombre.").max(120),
	description: z.string().max(2000).optional(),
	quantity: z
		.string()
		.regex(/^\d+(\.\d{1,3})?$/, "Usa una cantidad válida.")
		.refine(
			(value) => Number(value) > 0,
			"La cantidad debe ser mayor que cero.",
		),
	unit: z.enum(["unit", "kg", "g", "l", "ml", "package"]),
	estimated_unit_price: z.string(),
	assigned_to: z.string(),
	notes: z.string().max(4000).optional(),
});
type ItemForm = z.infer<typeof itemSchema>;

const purchaseSchema = z.object({
	actual_unit_price: z.string(),
	purchased_by: z.string().min(1, "Selecciona quién compró."),
	paid_by: z.string(),
	purchased_at: z.string().min(1, "Indica la fecha."),
	notes: z.string().max(4000).optional(),
});
type PurchaseForm = z.infer<typeof purchaseSchema>;

const correctionSchema = purchaseSchema.extend({
	reason: z.string().trim().min(1, "Indica el motivo.").max(500),
});
type CorrectionForm = z.infer<typeof correctionSchema>;

const reverseSchema = z.object({
	reason: z.string().trim().min(1, "Indica el motivo.").max(500),
});
type ReverseForm = z.infer<typeof reverseSchema>;

const listSchema = z.object({
	name: z.string().trim().min(1, "Escribe un nombre.").max(120),
	description: z.string().max(2000).optional(),
});
type ListForm = z.infer<typeof listSchema>;

type Dialog =
	| { type: "item"; item?: ShoppingItem }
	| { type: "purchase"; item: ShoppingItem }
	| { type: "correction"; purchase: Purchase }
	| { type: "reverse"; purchase: Purchase }
	| { type: "list" }
	| null;

export function ListDetailPage() {
	const { t } = useTranslation();
	const { listId } = useParams({ from: "/lists/$listId" });
	const { data: me } = useSession();
	const queryClient = useQueryClient();
	const [dialog, setDialog] = useState<Dialog>(null);
	const [filter, setFilter] = useState<"all" | "pending" | "purchased">("all");
	const [message, setMessage] = useState("");

	const list = useQuery({
		queryKey: ["list", listId],
		queryFn: () => apiData<ShoppingList>(`/api/v1/shopping-lists/${listId}`),
		refetchInterval: (query) => (query.state.data?.group_id ? 15_000 : false),
	});
	const items = useQuery({
		queryKey: ["items", listId],
		queryFn: () =>
			apiData<ShoppingItem[]>(`/api/v1/shopping-lists/${listId}/items`),
		refetchInterval: list.data?.group_id ? 15_000 : false,
	});
	const summary = useQuery({
		queryKey: ["summary", "list", listId],
		queryFn: async () => {
			const scope = list.data?.group_id ? `group_id=${list.data.group_id}` : "";
			return apiData<Summary>(
				`/api/v1/expenses/summary?${scope}&include_archived=true`,
			);
		},
		enabled: !!list.data,
	});
	const members = useQuery({
		queryKey: ["members", list.data?.group_id],
		queryFn: () =>
			apiData<Member[]>(`/api/v1/groups/${list.data?.group_id}/members`),
		enabled: !!list.data?.group_id,
	});

	const itemForm = useForm<ItemForm>({ resolver: zodResolver(itemSchema) });
	const purchaseForm = useForm<PurchaseForm>({
		resolver: zodResolver(purchaseSchema),
	});
	const correctionForm = useForm<CorrectionForm>({
		resolver: zodResolver(correctionSchema),
	});
	const reverseForm = useForm<ReverseForm>({
		resolver: zodResolver(reverseSchema),
	});
	const listForm = useForm<ListForm>({ resolver: zodResolver(listSchema) });

	const refresh = async () => {
		await queryClient.invalidateQueries({ queryKey: ["list", listId] });
		await queryClient.invalidateQueries({ queryKey: ["items", listId] });
		await queryClient.invalidateQueries({ queryKey: ["summary"] });
		await queryClient.invalidateQueries({ queryKey: ["lists"] });
		await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
		await queryClient.invalidateQueries({ queryKey: ["history"] });
		await queryClient.invalidateQueries({ queryKey: ["activity"] });
	};

	const saveItem = useMutation({
		mutationFn: (values: ItemForm) => {
			const current = dialog?.type === "item" ? dialog.item : undefined;
			const body = {
				name: current?.purchase ? current.name : values.name,
				description: current?.purchase
					? current.description
					: (values.description ?? ""),
				quantity: current?.purchase ? current.quantity : values.quantity,
				unit: current?.purchase ? current.unit : values.unit,
				estimated_unit_price: current?.purchase
					? (current.estimated_unit_price ?? "")
					: values.estimated_unit_price,
				assigned_to: current?.purchase
					? current.assigned_to
					: values.assigned_to || null,
				notes: values.notes ?? "",
				expected_version: current?.version ?? 0,
			};
			return current
				? apiData<ShoppingItem>(`/api/v1/items/${current.id}`, {
						method: "PATCH",
						body,
					})
				: apiData<ShoppingItem>(`/api/v1/shopping-lists/${listId}/items`, {
						method: "POST",
						body,
					});
		},
		onSuccess: async () => {
			setDialog(null);
			itemForm.reset();
			await refresh();
		},
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});
	const deleteItem = useMutation({
		mutationFn: (item: ShoppingItem) =>
			apiData(`/api/v1/items/${item.id}?expected_version=${item.version}`, {
				method: "DELETE",
			}),
		onSuccess: refresh,
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});
	const recordPurchase = useMutation({
		mutationFn: ({
			item,
			values,
		}: {
			item: ShoppingItem;
			values: PurchaseForm;
		}) =>
			apiData<Purchase>(`/api/v1/items/${item.id}/purchases`, {
				method: "POST",
				idempotencyKey: crypto.randomUUID(),
				body: {
					expected_version: item.version,
					actual_unit_price: values.actual_unit_price,
					purchased_by: values.purchased_by,
					paid_by: values.paid_by || null,
					purchased_at: new Date(values.purchased_at).toISOString(),
					notes: values.notes ?? "",
				},
			}),
		onSuccess: async () => {
			setDialog(null);
			purchaseForm.reset();
			await refresh();
		},
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});
	const correctPurchase = useMutation({
		mutationFn: ({
			purchase,
			values,
		}: {
			purchase: Purchase;
			values: CorrectionForm;
		}) =>
			apiData<Purchase>(`/api/v1/purchases/${purchase.id}`, {
				method: "PATCH",
				idempotencyKey: crypto.randomUUID(),
				body: {
					expected_version: purchase.version,
					actual_unit_price: values.actual_unit_price,
					purchased_by: values.purchased_by,
					paid_by: values.paid_by || null,
					purchased_at: new Date(values.purchased_at).toISOString(),
					notes: values.notes ?? "",
					reason: values.reason,
				},
			}),
		onSuccess: async () => {
			setDialog(null);
			correctionForm.reset();
			await refresh();
		},
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});
	const reversePurchase = useMutation({
		mutationFn: ({
			purchase,
			values,
		}: {
			purchase: Purchase;
			values: ReverseForm;
		}) =>
			apiData<Purchase>(`/api/v1/purchases/${purchase.id}/reverse`, {
				method: "POST",
				idempotencyKey: crypto.randomUUID(),
				body: { expected_version: purchase.version, reason: values.reason },
			}),
		onSuccess: async () => {
			setDialog(null);
			reverseForm.reset();
			await refresh();
		},
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});
	const updateList = useMutation({
		mutationFn: (values: ListForm) =>
			apiData<ShoppingList>(`/api/v1/shopping-lists/${listId}`, {
				method: "PATCH",
				body: { ...values, expected_version: list.data?.version },
			}),
		onSuccess: async () => {
			setDialog(null);
			await refresh();
		},
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});
	const archiveList = useMutation({
		mutationFn: (archived: boolean) =>
			apiData<ShoppingList>(
				`/api/v1/shopping-lists/${listId}/${archived ? "archive" : "restore"}`,
				{
					method: "POST",
					body: { expected_version: list.data?.version },
				},
			),
		onSuccess: refresh,
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});

	const people = useMemo<Member[]>(() => {
		if (list.data?.group_id) return members.data ?? [];
		if (!me) return [];
		return [
			{
				user_id: me.id,
				display_name: me.display_name,
				email: me.email,
				role: "owner",
				joined_at: me.created_at,
			},
		];
	}, [list.data?.group_id, members.data, me]);

	if (list.isLoading || items.isLoading) return <LoadingState />;
	if (list.isError)
		return (
			<ErrorState message={list.error.message} onRetry={() => list.refetch()} />
		);
	const current = list.data;
	if (!current) return null;
	const canManage = current.role === "owner" || current.role === "admin";
	const writable = !current.archived_at;
	const visibleItems = (items.data ?? []).filter((item) =>
		filter === "all"
			? true
			: filter === "pending"
				? !item.purchase
				: !!item.purchase,
	);

	function openItem(item?: ShoppingItem) {
		setMessage("");
		if (item) {
			itemForm.reset({
				name: item.name,
				description: item.description,
				quantity: item.quantity,
				unit: item.unit as ItemForm["unit"],
				estimated_unit_price: item.estimated_unit_price ?? "",
				assigned_to: item.assigned_to ?? "",
				notes: item.notes,
			});
		} else {
			itemForm.reset({
				name: "",
				description: "",
				quantity: "1",
				unit: "unit",
				estimated_unit_price: "",
				assigned_to: "",
				notes: "",
			});
		}
		setDialog({ type: "item", item });
	}

	function openPurchase(item: ShoppingItem) {
		setMessage("");
		purchaseForm.reset({
			actual_unit_price: "",
			purchased_by: me?.id ?? "",
			paid_by: me?.id ?? "",
			purchased_at: localDateTimeInput(),
			notes: "",
		});
		setDialog({ type: "purchase", item });
	}

	function openCorrection(purchase: Purchase) {
		setMessage("");
		correctionForm.reset({
			actual_unit_price: purchase.actual_unit_price ?? "",
			purchased_by: purchase.purchased_by,
			paid_by: purchase.paid_by ?? "",
			purchased_at: localDateTimeInput(new Date(purchase.purchased_at)),
			notes: purchase.notes,
			reason: "",
		});
		setDialog({ type: "correction", purchase });
	}

	function openReverse(purchase: Purchase) {
		setMessage("");
		reverseForm.reset({ reason: "" });
		setDialog({ type: "reverse", purchase });
	}

	function openListEdit(listItem: ShoppingList) {
		setMessage("");
		listForm.reset({ name: listItem.name, description: listItem.description });
		setDialog({ type: "list" });
	}

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={current.group_id ? t("lists.group") : t("lists.personal")}
				title={current.name}
				subtitle={current.description || t("lists.subtitle")}
				actions={
					<>
						<Link
							to={current.group_id ? "/groups/$groupId" : "/personal"}
							params={current.group_id ? { groupId: current.group_id } : {}}
						>
							<Button variant="secondary">
								<ArrowLeft size={16} />
								{t("common.back")}
							</Button>
						</Link>
						{canManage ? (
							<>
								<Button variant="quiet" onClick={() => openListEdit(current)}>
									<Pencil size={16} />
									{t("common.edit")}
								</Button>
								<Button
									variant="quiet"
									onClick={() => archiveList.mutate(!current.archived_at)}
									disabled={archiveList.isPending}
								>
									{current.archived_at ? (
										<RotateCcw size={16} />
									) : (
										<Archive size={16} />
									)}
									{current.archived_at
										? t("lists.restore")
										: t("lists.archive")}
								</Button>
							</>
						) : null}
						{writable ? (
							<Button onClick={() => openItem()}>
								<Plus size={16} />
								{t("lists.addItem")}
							</Button>
						) : null}
					</>
				}
			/>

			{current.archived_at ? (
				<div className={styles.notice}>{t("lists.readOnlyArchived")}</div>
			) : null}
			{message ? (
				<div className={styles.error} role="alert">
					{message}
				</div>
			) : null}

			<section className={styles.grid4}>
				<SummaryCard
					label={t("lists.pendingCount")}
					value={current.pending_count}
				/>
				<SummaryCard
					label={t("lists.purchasedCount")}
					value={current.purchased_count}
				/>
				<SummaryCard
					label={t("lists.estimated")}
					value={formatMoney(summary.data?.estimated_known)}
				/>
				<SummaryCard
					label={t("lists.actual")}
					value={formatMoney(summary.data?.actual_known)}
				/>
			</section>

			<div className={styles.toolbar}>
				<div className={styles.toggle}>
					{(["all", "pending", "purchased"] as const).map((value) => (
						<button
							key={value}
							type="button"
							className={`${styles.toggleButton} ${filter === value ? styles.toggleActive : ""}`}
							onClick={() => setFilter(value)}
						>
							{value === "all"
								? t("common.all")
								: value === "pending"
									? t("common.pending")
									: t("common.purchased")}
						</button>
					))}
				</div>
			</div>

			{visibleItems.length === 0 ? (
				<EmptyState>{t("lists.emptyItems")}</EmptyState>
			) : (
				<div className={styles.list}>
					{visibleItems.map((item) => {
						const purchase = item.purchase;
						const canCorrect =
							purchase && (canManage || purchase.recorded_by === me?.id);
						return (
							<div key={item.id} className={styles.row}>
								<span className={styles.rowMain}>
									<span className={styles.inline}>
										<span className={styles.rowTitle}>{item.name}</span>
										{purchase ? (
											<span
												className={`${styles.badge} ${styles.badgeSuccess}`}
											>
												<Check size={12} />
												{t("common.purchased")}
											</span>
										) : (
											<span
												className={`${styles.badge} ${styles.badgePrimary}`}
											>
												{t("common.pending")}
											</span>
										)}
									</span>
									<span className={styles.rowMeta}>
										{item.quantity} {unitLabels[item.unit] ?? item.unit}
										{item.estimated_unit_price
											? ` · ${formatMoney(item.estimated_unit_price)} estimado`
											: ""}
										{item.assigned_to
											? ` · ${people.find((person) => person.user_id === item.assigned_to)?.display_name ?? "Asignado"}`
											: ""}
									</span>
									{purchase ? (
										<span className={styles.rowMeta}>
											{formatDate(purchase.purchased_at)} · compró{" "}
											{purchase.purchased_by_name} · pagó{" "}
											{purchase.paid_by_name ?? t("lists.unidentified")} ·{" "}
											{formatMoney(purchase.total_amount)}
										</span>
									) : null}
									{item.notes ? (
										<span className={styles.rowMeta}>{item.notes}</span>
									) : null}
								</span>
								<span className={styles.rowActions}>
									{!purchase && writable ? (
										<>
											<Button
												small
												variant="secondary"
												onClick={() => openPurchase(item)}
											>
												<ShoppingCart size={14} />
												{t("lists.registerPurchase")}
											</Button>
											<Button
												small
												variant="quiet"
												onClick={() => openItem(item)}
												aria-label={t("common.edit")}
											>
												<Pencil size={14} />
											</Button>
											{canManage ? (
												<Button
													small
													variant="quiet"
													onClick={() => {
														if (
															window.confirm(
																`¿Eliminar ${item.name}? Las compras históricas se conservarán.`,
															)
														)
															deleteItem.mutate(item);
													}}
													aria-label={t("common.delete")}
												>
													<Trash2 size={14} />
												</Button>
											) : null}
										</>
									) : null}
									{purchase && writable && canCorrect ? (
										<>
											<Button
												small
												variant="quiet"
												onClick={() => openItem(item)}
											>
												{t("common.edit")}
											</Button>
											<Button
												small
												variant="quiet"
												onClick={() => openCorrection(purchase)}
											>
												{t("lists.correctPurchase")}
											</Button>
											<Button
												small
												variant="quiet"
												onClick={() => openReverse(purchase)}
											>
												{t("lists.reversePurchase")}
											</Button>
										</>
									) : null}
									{purchase && writable && !canCorrect ? (
										<Button
											small
											variant="quiet"
											onClick={() => openItem(item)}
										>
											{t("common.edit")}
										</Button>
									) : null}
								</span>
							</div>
						);
					})}
				</div>
			)}

			{dialog?.type === "item" ? (
				<Modal
					title={dialog.item ? t("lists.editItem") : t("lists.addItem")}
					onClose={() => setDialog(null)}
				>
					<form
						className={styles.form}
						onSubmit={itemForm.handleSubmit((values) =>
							saveItem.mutate(values),
						)}
					>
						{dialog.item?.purchase ? (
							<>
								<div className={styles.notice}>
									{t("lists.purchaseKnown")}. Solo puedes editar las notas del
									artículo; los datos de compra se corrigen desde la compra.
								</div>
								<Field label={t("common.notes")}>
									<Textarea {...itemForm.register("notes")} />
								</Field>
							</>
						) : (
							<>
								<Field
									label={t("common.name")}
									error={itemForm.formState.errors.name?.message}
								>
									<Input autoFocus {...itemForm.register("name")} />
								</Field>
								<div className={styles.formGrid}>
									<Field
										label={t("lists.quantity")}
										error={itemForm.formState.errors.quantity?.message}
									>
										<Input
											inputMode="decimal"
											{...itemForm.register("quantity")}
										/>
									</Field>
									<Field label={t("lists.unit")}>
										<Select {...itemForm.register("unit")}>
											{Object.entries(unitLabels).map(([value, label]) => (
												<option key={value} value={value}>
													{label}
												</option>
											))}
										</Select>
									</Field>
									<Field
										label={t("lists.estimatedUnitPrice")}
										hint="MXN · opcional"
									>
										<Input
											inputMode="decimal"
											{...itemForm.register("estimated_unit_price")}
										/>
									</Field>
									<Field label={t("lists.assignedTo")}>
										<Select {...itemForm.register("assigned_to")}>
											<option value="">{t("lists.unassigned")}</option>
											{people.map((person) => (
												<option key={person.user_id} value={person.user_id}>
													{person.display_name}
												</option>
											))}
										</Select>
									</Field>
								</div>
								<Field
									label={`${t("common.description")} (${t("common.optional")})`}
								>
									<Textarea {...itemForm.register("description")} />
								</Field>
								<Field label={t("common.notes")}>
									<Textarea {...itemForm.register("notes")} />
								</Field>
							</>
						)}
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setDialog(null)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={saveItem.isPending}>
								{t("common.save")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}

			{dialog?.type === "purchase" ? (
				<Modal title={t("lists.purchaseTitle")} onClose={() => setDialog(null)}>
					<div className={styles.notice}>
						{dialog.item.name} · {dialog.item.quantity}{" "}
						{unitLabels[dialog.item.unit]}
					</div>
					<form
						className={styles.form}
						onSubmit={purchaseForm.handleSubmit((values) =>
							recordPurchase.mutate({ item: dialog.item, values }),
						)}
					>
						<div className={styles.formGrid}>
							<Field
								label={t("lists.actualUnitPrice")}
								hint="MXN · déjalo vacío si no lo conoces"
							>
								<Input
									inputMode="decimal"
									{...purchaseForm.register("actual_unit_price")}
								/>
							</Field>
							<Field
								label={t("lists.purchaseDate")}
								error={purchaseForm.formState.errors.purchased_at?.message}
							>
								<Input
									type="datetime-local"
									{...purchaseForm.register("purchased_at")}
								/>
							</Field>
							<Field
								label={t("lists.purchasedBy")}
								error={purchaseForm.formState.errors.purchased_by?.message}
							>
								<Select {...purchaseForm.register("purchased_by")}>
									{people.map((person) => (
										<option key={person.user_id} value={person.user_id}>
											{person.display_name}
										</option>
									))}
								</Select>
							</Field>
							<Field label={t("lists.paidBy")}>
								<Select {...purchaseForm.register("paid_by")}>
									<option value="">{t("lists.unidentified")}</option>
									{people.map((person) => (
										<option key={person.user_id} value={person.user_id}>
											{person.display_name}
										</option>
									))}
								</Select>
							</Field>
						</div>
						<TotalPreview
							quantity={dialog.item.quantity}
							register={purchaseForm.watch("actual_unit_price")}
						/>
						<Field label={t("common.notes")}>
							<Textarea {...purchaseForm.register("notes")} />
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setDialog(null)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={recordPurchase.isPending}>
								{t("lists.registerPurchase")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}

			{dialog?.type === "correction" ? (
				<Modal
					title={t("lists.correctPurchase")}
					onClose={() => setDialog(null)}
				>
					<form
						className={styles.form}
						onSubmit={correctionForm.handleSubmit((values) =>
							correctPurchase.mutate({ purchase: dialog.purchase, values }),
						)}
					>
						<div className={styles.formGrid}>
							<Field label={t("lists.actualUnitPrice")} hint="MXN">
								<Input
									inputMode="decimal"
									{...correctionForm.register("actual_unit_price")}
								/>
							</Field>
							<Field label={t("lists.purchaseDate")}>
								<Input
									type="datetime-local"
									{...correctionForm.register("purchased_at")}
								/>
							</Field>
							<Field label={t("lists.purchasedBy")}>
								<Select {...correctionForm.register("purchased_by")}>
									{people.map((person) => (
										<option key={person.user_id} value={person.user_id}>
											{person.display_name}
										</option>
									))}
									{!people.some(
										(person) => person.user_id === dialog.purchase.purchased_by,
									) ? (
										<option value={dialog.purchase.purchased_by}>
											{dialog.purchase.purchased_by_name} (histórico)
										</option>
									) : null}
								</Select>
							</Field>
							<Field label={t("lists.paidBy")}>
								<Select {...correctionForm.register("paid_by")}>
									<option value="">{t("lists.unidentified")}</option>
									{people.map((person) => (
										<option key={person.user_id} value={person.user_id}>
											{person.display_name}
										</option>
									))}
									{dialog.purchase.paid_by &&
									!people.some(
										(person) => person.user_id === dialog.purchase.paid_by,
									) ? (
										<option value={dialog.purchase.paid_by}>
											{dialog.purchase.paid_by_name} (histórico)
										</option>
									) : null}
								</Select>
							</Field>
						</div>
						<Field label={t("common.notes")}>
							<Textarea {...correctionForm.register("notes")} />
						</Field>
						<Field
							label={t("lists.correctionReason")}
							error={correctionForm.formState.errors.reason?.message}
						>
							<Input {...correctionForm.register("reason")} />
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setDialog(null)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={correctPurchase.isPending}>
								{t("common.save")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}

			{dialog?.type === "reverse" ? (
				<Modal
					title={t("lists.reversePurchase")}
					onClose={() => setDialog(null)}
				>
					<p className={styles.muted}>
						La compra se conservará en el historial, pero dejará de contar en
						los gastos.
					</p>
					<form
						className={styles.form}
						onSubmit={reverseForm.handleSubmit((values) =>
							reversePurchase.mutate({ purchase: dialog.purchase, values }),
						)}
					>
						<Field
							label={t("lists.reverseReason")}
							error={reverseForm.formState.errors.reason?.message}
						>
							<Textarea autoFocus {...reverseForm.register("reason")} />
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setDialog(null)}>
								{t("common.cancel")}
							</Button>
							<Button
								variant="danger"
								type="submit"
								disabled={reversePurchase.isPending}
							>
								{t("lists.reversePurchase")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}

			{dialog?.type === "list" ? (
				<Modal title={t("common.edit")} onClose={() => setDialog(null)}>
					<form
						className={styles.form}
						onSubmit={listForm.handleSubmit((values) =>
							updateList.mutate(values),
						)}
					>
						<Field
							label={t("common.name")}
							error={listForm.formState.errors.name?.message}
						>
							<Input {...listForm.register("name")} />
						</Field>
						<Field label={t("common.description")}>
							<Textarea {...listForm.register("description")} />
						</Field>
						<div className={styles.formActions}>
							<Button variant="secondary" onClick={() => setDialog(null)}>
								{t("common.cancel")}
							</Button>
							<Button type="submit" disabled={updateList.isPending}>
								{t("common.save")}
							</Button>
						</div>
					</form>
				</Modal>
			) : null}
		</div>
	);
}

function TotalPreview({
	quantity,
	register,
}: {
	quantity: string;
	register: string;
}) {
	const total = lineTotal(quantity, register);
	return (
		<div className={styles.notice}>
			Total calculado: <strong>{total ? formatMoney(total) : "—"}</strong>. El
			servidor confirma el importe final.
		</div>
	);
}
