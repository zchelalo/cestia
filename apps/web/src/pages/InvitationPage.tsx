import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/Button";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatDate } from "@/shared/money";
import type { Invitation } from "@/shared/types";

export function InvitationPage() {
	const { t } = useTranslation();
	const search = useSearch({ strict: false }) as { token?: string };
	const token = search.token ?? "";
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const [message, setMessage] = useState("");
	const preview = useQuery({
		queryKey: ["invitation", token],
		queryFn: () =>
			apiData<Invitation>("/api/v1/invitations/preview", {
				method: "POST",
				body: { token },
			}),
		enabled: !!token,
		retry: false,
	});
	const resolve = useMutation({
		mutationFn: (accept: boolean) =>
			apiData<Invitation>(
				`/api/v1/invitations/${accept ? "accept" : "reject"}`,
				{
					method: "POST",
					body: { token },
				},
			),
		onSuccess: async (invitation, accept) => {
			await queryClient.invalidateQueries({ queryKey: ["groups"] });
			await queryClient.invalidateQueries({ queryKey: ["dashboard"] });
			if (accept) {
				await navigate({
					to: "/groups/$groupId",
					params: { groupId: invitation.group_id },
				});
			} else {
				setMessage("Invitación rechazada.");
				await navigate({ to: "/" });
			}
		},
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});

	if (!token)
		return <ErrorState message="El enlace no incluye una invitación válida." />;
	if (preview.isLoading) return <LoadingState />;
	if (preview.isError)
		return (
			<ErrorState
				message={preview.error.message}
				onRetry={() => preview.refetch()}
			/>
		);
	const invitation = preview.data;
	if (!invitation) return null;

	return (
		<div className={styles.page} style={{ maxWidth: "44rem" }}>
			<PageHeader
				eyebrow="Cestia"
				title={t("invitation.title")}
				subtitle={t("invitation.body")}
			/>
			{message ? (
				<div className={styles.error} role="alert">
					{message}
				</div>
			) : null}
			<section className={styles.card}>
				<span className={`${styles.badge} ${styles.badgePrimary}`}>
					Invitación pendiente
				</span>
				<h2
					className={styles.title}
					style={{ fontSize: "2rem", marginTop: "1rem" }}
				>
					{invitation.group_name}
				</h2>
				<p className={styles.cardMeta}>
					Para {invitation.email} · expira{" "}
					{formatDate(invitation.expires_at, true)}
				</p>
				<p className={styles.muted} style={{ marginTop: "1rem" }}>
					Al aceptar podrás consultar y participar en las listas compartidas de
					este grupo. No se comparten tus listas personales.
				</p>
				<div className={styles.actions} style={{ marginTop: "1.5rem" }}>
					<Button
						onClick={() => resolve.mutate(true)}
						disabled={resolve.isPending}
					>
						{t("invitation.accept")}
					</Button>
					<Button
						variant="secondary"
						onClick={() => resolve.mutate(false)}
						disabled={resolve.isPending}
					>
						{t("invitation.reject")}
					</Button>
				</div>
			</section>
		</div>
	);
}
