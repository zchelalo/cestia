import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "@tanstack/react-router";
import {
	ArrowLeft,
	Crown,
	MailPlus,
	UserMinus,
	UserRoundCog,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/Button";
import { CursorPagination } from "@/components/CursorPagination";
import { Modal } from "@/components/Modal";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatDate } from "@/shared/money";
import { pageFetcher } from "@/shared/pages";
import { useSession } from "@/shared/session";
import type { Group, Invitation, Member } from "@/shared/types";
import { useCursorPage } from "@/shared/use-cursor-page";

export function MembersPage() {
	const { t } = useTranslation();
	const { groupId } = useParams({ from: "/groups/$groupId/members" });
	const queryClient = useQueryClient();
	const { data: me } = useSession();
	const [email, setEmail] = useState("");
	const [error, setError] = useState("");
	const [message, setMessage] = useState("");
	const [transferring, setTransferring] = useState(false);
	const group = useQuery({
		queryKey: ["group", groupId],
		queryFn: () => apiData<Group>(`/api/v1/groups/${groupId}`),
	});
	const members = useCursorPage<Member>({
		queryKey: ["members", groupId],
		fetchPage: pageFetcher<Member>(`/api/v1/groups/${groupId}/members`),
	});
	const invitations = useCursorPage<Invitation>({
		queryKey: ["invitations", groupId, "pending"],
		fetchPage: pageFetcher<Invitation>(
			`/api/v1/groups/${groupId}/invitations`,
			{ status: "pending" },
		),
	});
	const transferCandidates = useCursorPage<Member>({
		queryKey: ["members", groupId, "transfer"],
		fetchPage: pageFetcher<Member>(`/api/v1/groups/${groupId}/members`, {
			include_owner: "false",
		}),
		enabled: transferring,
	});

	const refresh = async () => {
		await queryClient.invalidateQueries({ queryKey: ["members", groupId] });
		await queryClient.invalidateQueries({ queryKey: ["invitations", groupId] });
		await queryClient.invalidateQueries({ queryKey: ["group", groupId] });
		await queryClient.invalidateQueries({ queryKey: ["groups"] });
	};

	const invite = useMutation({
		mutationFn: () =>
			apiData<Invitation>(`/api/v1/groups/${groupId}/invitations`, {
				method: "POST",
				body: { email },
			}),
		onSuccess: async () => {
			setEmail("");
			setMessage(t("groups.inviteSent"));
			await refresh();
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const setRole = useMutation({
		mutationFn: ({
			userId,
			role,
		}: {
			userId: string;
			role: "admin" | "member";
		}) =>
			apiData(`/api/v1/groups/${groupId}/members/${userId}`, {
				method: "PATCH",
				body: { role },
			}),
		onSuccess: refresh,
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const remove = useMutation({
		mutationFn: (userId: string) =>
			apiData(`/api/v1/groups/${groupId}/members/${userId}`, {
				method: "DELETE",
			}),
		onSuccess: refresh,
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const resend = useMutation({
		mutationFn: (invitation: Invitation) =>
			apiData(`/api/v1/groups/${groupId}/invitations/${invitation.id}/resend`, {
				method: "POST",
				body: { expected_version: invitation.version },
			}),
		onSuccess: async () => {
			setMessage(t("groups.inviteSent"));
			await refresh();
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const cancel = useMutation({
		mutationFn: (invitationId: string) =>
			apiData(`/api/v1/groups/${groupId}/invitations/${invitationId}`, {
				method: "DELETE",
			}),
		onSuccess: refresh,
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const transfer = useMutation({
		mutationFn: (userId: string) =>
			apiData<Group>(`/api/v1/groups/${groupId}/transfer-ownership`, {
				method: "POST",
				body: { user_id: userId },
			}),
		onSuccess: async () => {
			setTransferring(false);
			await refresh();
			await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const leave = useMutation({
		mutationFn: () =>
			apiData(`/api/v1/groups/${groupId}/leave`, { method: "POST" }),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ["groups"] });
			window.location.assign("/groups");
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});

	if (group.isLoading || members.isLoading) return <LoadingState />;
	if (group.isError)
		return (
			<ErrorState
				message={group.error.message}
				onRetry={() => group.refetch()}
			/>
		);
	const current = group.data;
	if (!current) return null;
	const canInvite = current.role === "owner" || current.role === "admin";
	const isOwner = current.role === "owner";
	const pending = invitations.items;

	async function submitInvite(event: React.FormEvent) {
		event.preventDefault();
		setError("");
		setMessage("");
		await invite.mutateAsync().catch(() => undefined);
	}

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={current.name}
				title={t("groups.members")}
				subtitle="Invita personas, ajusta roles y transfiere la propiedad con cuidado."
				actions={
					<Link to="/groups/$groupId" params={{ groupId }}>
						<Button variant="secondary">
							<ArrowLeft size={16} />
							{t("common.back")}
						</Button>
					</Link>
				}
			/>

			{error ? (
				<div className={styles.error} role="alert">
					{error}
				</div>
			) : null}
			{message ? (
				<div className={styles.notice} role="status">
					{message}
				</div>
			) : null}

			<section className={styles.section}>
				<h2 className={styles.sectionTitle}>Integrantes activos</h2>
				<div className={styles.list}>
					{members.items.map((member) => {
						const isMe = member.user_id === me?.id;
						const canRemove =
							canInvite &&
							member.role !== "owner" &&
							!isMe &&
							(isOwner || member.role === "member");
						const canChangeRole = isOwner && member.role !== "owner";
						const hasActions = canChangeRole || canRemove;
						return (
							<div
								key={member.user_id}
								className={`${styles.row} ${styles.stackedRow}`}
							>
								<span className={styles.rowMain}>
									<span className={styles.memberHeader}>
										<span className={styles.rowTitle}>
											{member.display_name}
											{isMe ? " (tú)" : ""}
										</span>
										<span
											className={`${styles.badge} ${member.role === "owner" ? styles.badgeWarning : styles.badgePrimary}`}
										>
											{member.role === "owner" ? <Crown size={12} /> : null}
											{t(`groups.${member.role}`)}
										</span>
									</span>
									<span className={styles.rowMeta}>{member.email}</span>
									<span className={styles.rowMeta}>
										Desde {formatDate(member.joined_at)}
									</span>
								</span>
								{hasActions ? (
									<span
										className={`${styles.rowActions} ${styles.stackedRowActions}`}
									>
										{canChangeRole ? (
											<Button
												small
												variant="surface"
												disabled={setRole.isPending}
												onClick={() =>
													setRole.mutate({
														userId: member.user_id,
														role: member.role === "admin" ? "member" : "admin",
													})
												}
											>
												<UserRoundCog size={14} />
												{member.role === "admin"
													? t("groups.makeMember")
													: t("groups.makeAdmin")}
											</Button>
										) : null}
										{canRemove ? (
											<Button
												small
												variant="surface"
												disabled={remove.isPending}
												onClick={() => {
													if (
														window.confirm(
															`¿Eliminar a ${member.display_name} del grupo?`,
														)
													)
														remove.mutate(member.user_id);
												}}
											>
												<UserMinus size={14} />
												{t("groups.remove")}
											</Button>
										) : null}
									</span>
								) : null}
							</div>
						);
					})}
				</div>
				<CursorPagination
					label="Paginación de integrantes"
					canGoNext={members.canGoNext}
					canGoPrevious={members.canGoPrevious}
					isFetching={members.isFetching}
					pageSize={members.pageSize}
					onPageSizeChange={members.setPageSize}
					onNext={members.goNext}
					onPrevious={members.goPrevious}
				/>
			</section>

			{canInvite ? (
				<section className={styles.card}>
					<h2 className={styles.cardTitle}>{t("groups.invite")}</h2>
					<p className={styles.cardMeta}>
						La persona recibirá un enlace de un solo uso. Debe iniciar sesión
						con el mismo correo.
					</p>
					<form
						className={styles.form}
						onSubmit={submitInvite}
						style={{ marginTop: "1rem" }}
					>
						<div className={styles.inviteRow}>
							<div className={styles.inviteField}>
								<label className={styles.eyebrowInline} htmlFor="invite-email">
									{t("groups.inviteEmail")}
								</label>
								<input
									id="invite-email"
									className={styles.select}
									type="email"
									required
									value={email}
									onChange={(event) => setEmail(event.target.value)}
									placeholder="persona@correo.com"
								/>
							</div>
							<Button
								type="submit"
								className={styles.inviteButton}
								disabled={invite.isPending || !email}
							>
								<MailPlus size={16} />
								{t("groups.invite")}
							</Button>
						</div>
					</form>
				</section>
			) : null}

			<section className={styles.section}>
				<h2 className={styles.sectionTitle}>
					{t("groups.pendingInvitations")}
				</h2>
				{pending.length === 0 ? (
					<div className={styles.empty}>{t("groups.noPending")}</div>
				) : (
					<div className={styles.list}>
						{pending.map((invitation) => (
							<div
								key={invitation.id}
								className={`${styles.row} ${styles.stackedRow}`}
							>
								<span className={styles.rowMain}>
									<span className={styles.rowTitle}>{invitation.email}</span>
									<span className={styles.rowMeta}>
										Expira {formatDate(invitation.expires_at, true)}
									</span>
								</span>
								{canInvite ? (
									<span
										className={`${styles.rowActions} ${styles.stackedRowActions}`}
									>
										<Button
											small
											variant="surface"
											onClick={() => resend.mutate(invitation)}
											disabled={resend.isPending}
										>
											{t("groups.resend")}
										</Button>
										<Button
											small
											variant="surface"
											onClick={() => cancel.mutate(invitation.id)}
											disabled={cancel.isPending}
										>
											{t("groups.cancelInvite")}
										</Button>
									</span>
								) : null}
							</div>
						))}
					</div>
				)}
				<CursorPagination
					label="Paginación de invitaciones pendientes"
					canGoNext={invitations.canGoNext}
					canGoPrevious={invitations.canGoPrevious}
					isFetching={invitations.isFetching}
					pageSize={invitations.pageSize}
					onPageSizeChange={invitations.setPageSize}
					onNext={invitations.goNext}
					onPrevious={invitations.goPrevious}
				/>
			</section>

			<section className={`${styles.card} ${styles.dangerZone}`}>
				<h2 className={styles.cardTitle}>Acciones del grupo</h2>
				<div
					className={`${styles.actions} ${styles.groupDangerActions}`}
					style={{ marginTop: "1rem" }}
				>
					{isOwner ? (
						<Button variant="secondary" onClick={() => setTransferring(true)}>
							<Crown size={16} />
							{t("groups.transfer")}
						</Button>
					) : null}
					{current.role !== "owner" ? (
						<Button
							variant="danger"
							onClick={() => {
								if (
									window.confirm(
										"¿Abandonar este grupo? Perderás acceso a sus listas.",
									)
								)
									leave.mutate();
							}}
							disabled={leave.isPending}
						>
							{t("groups.leave")}
						</Button>
					) : (
						<span className={styles.muted}>{t("groups.ownerOnly")}</span>
					)}
				</div>
			</section>

			{transferring ? (
				<Modal
					title={t("groups.transfer")}
					onClose={() => setTransferring(false)}
				>
					<p className={styles.muted}>
						Elige a la persona que será propietaria. Tú quedarás como admin.
					</p>
					<div className={styles.list}>
						{transferCandidates.items.map((member) => (
							<button
								key={member.user_id}
								type="button"
								className={styles.row}
								style={{ cursor: "pointer", textAlign: "left" }}
								onClick={() => {
									if (
										window.confirm(
											`¿Transferir la propiedad a ${member.display_name}?`,
										)
									)
										transfer.mutate(member.user_id);
								}}
							>
								<span className={styles.rowMain}>
									<span className={styles.rowTitle}>{member.display_name}</span>
									<span className={styles.rowMeta}>{member.email}</span>
								</span>
								<Crown size={17} />
							</button>
						))}
					</div>
					<CursorPagination
						label="Paginación de candidatos a propietario"
						canGoNext={transferCandidates.canGoNext}
						canGoPrevious={transferCandidates.canGoPrevious}
						isFetching={transferCandidates.isFetching}
						onNext={transferCandidates.goNext}
						onPrevious={transferCandidates.goPrevious}
					/>
				</Modal>
			) : null}
		</div>
	);
}
