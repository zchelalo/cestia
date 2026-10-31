import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { LogOut, Shield, Trash2, UserRound } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { Button } from "@/components/Button";
import { Field, Input, Select } from "@/components/Field";
import { PageHeader } from "@/components/PageHeader";
import { ErrorState, LoadingState } from "@/components/States";
import styles from "@/pages/Page.module.css";
import { apiData } from "@/shared/http";
import { formatDate } from "@/shared/money";
import { useSession } from "@/shared/session";
import type { Session, User } from "@/shared/types";

const profileSchema = z.object({
	display_name: z.string().trim().min(1, "Escribe tu nombre.").max(120),
	locale: z.string(),
	timezone: z.string().min(1),
});
const passwordSchema = z.object({
	current_password: z.string().min(1, "Ingresa tu contraseña actual."),
	new_password: z.string().min(12, "Usa al menos 12 caracteres.").max(72),
});
type Profile = z.infer<typeof profileSchema>;
type Password = z.infer<typeof passwordSchema>;

export function SettingsPage() {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const navigate = useNavigate();
	const { data: me } = useSession();
	const [message, setMessage] = useState("");
	const [error, setError] = useState("");
	const sessions = useQuery({
		queryKey: ["sessions"],
		queryFn: () => apiData<Session[]>("/api/v1/me/sessions"),
	});
	const profileForm = useForm<Profile>({
		resolver: zodResolver(profileSchema),
		values: {
			display_name: me?.display_name ?? "",
			locale: me?.locale ?? "es",
			timezone: me?.timezone ?? "America/Mexico_City",
		},
	});
	const passwordForm = useForm<Password>({
		resolver: zodResolver(passwordSchema),
		defaultValues: { current_password: "", new_password: "" },
	});
	const updateProfile = useMutation({
		mutationFn: (values: Profile) =>
			apiData<User>("/api/v1/me", { method: "PATCH", body: values }),
		onSuccess: async (user) => {
			queryClient.setQueryData(["session"], user);
			setMessage(t("settings.saved"));
			setError("");
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const changePassword = useMutation({
		mutationFn: (values: Password) =>
			apiData("/api/v1/auth/change-password", { method: "POST", body: values }),
		onSuccess: () => {
			queryClient.clear();
			void navigate({ to: "/sign-in" });
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const revoke = useMutation({
		mutationFn: (id: string) =>
			apiData(`/api/v1/me/sessions/${id}`, { method: "DELETE" }),
		onSuccess: async (_data, id) => {
			const current = sessions.data?.find(
				(session) => session.id === id,
			)?.current;
			if (current) {
				queryClient.clear();
				await navigate({ to: "/sign-in" });
			} else {
				await sessions.refetch();
			}
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});
	const deactivate = useMutation({
		mutationFn: (password: string) =>
			apiData("/api/v1/me/deactivate", { method: "POST", body: { password } }),
		onSuccess: async () => {
			queryClient.clear();
			await navigate({ to: "/sign-in" });
		},
		onError: (cause) =>
			setError(cause instanceof Error ? cause.message : t("common.error")),
	});

	if (sessions.isLoading) return <LoadingState />;
	if (sessions.isError)
		return (
			<ErrorState
				message={sessions.error.message}
				onRetry={() => sessions.refetch()}
			/>
		);

	async function deactivateAccount() {
		const password = window.prompt(
			"Escribe tu contraseña para desactivar la cuenta. Esta acción cerrará todas tus sesiones.",
		);
		if (password) deactivate.mutate(password);
	}

	return (
		<div className={styles.page}>
			<PageHeader
				eyebrow={t("nav.settings")}
				title={t("settings.title")}
				subtitle="Administra tu perfil, acceso y sesiones."
			/>
			{message ? (
				<div className={styles.notice} role="status">
					{message}
				</div>
			) : null}
			{error ? (
				<div className={styles.error} role="alert">
					{error}
				</div>
			) : null}

			<section className={styles.grid2}>
				<div className={styles.card}>
					<div className={styles.inline}>
						<UserRound size={18} />
						<h2 className={styles.cardTitle}>{t("settings.profile")}</h2>
					</div>
					<p className={styles.cardMeta}>
						Tu correo no se puede cambiar en esta versión.
					</p>
					<form
						className={styles.form}
						style={{ marginTop: "1.25rem" }}
						onSubmit={profileForm.handleSubmit((values) =>
							updateProfile.mutate(values),
						)}
					>
						<Field
							label={t("auth.displayName")}
							error={profileForm.formState.errors.display_name?.message}
						>
							<Input {...profileForm.register("display_name")} />
						</Field>
						<Field label="Correo">
							<Input value={me?.email ?? ""} disabled readOnly />
						</Field>
						<Field label="Idioma">
							<Select {...profileForm.register("locale")}>
								<option value="es">Español</option>
							</Select>
						</Field>
						<Field label="Zona horaria">
							<Select {...profileForm.register("timezone")}>
								<option value="America/Mexico_City">Ciudad de México</option>
								<option value="America/Tijuana">Tijuana</option>
								<option value="America/Cancun">Cancún</option>
								<option value="America/Monterrey">Monterrey</option>
							</Select>
						</Field>
						<div className={styles.formActions}>
							<Button type="submit" disabled={updateProfile.isPending}>
								{t("common.save")}
							</Button>
						</div>
					</form>
				</div>

				<div className={styles.card}>
					<div className={styles.inline}>
						<Shield size={18} />
						<h2 className={styles.cardTitle}>{t("settings.security")}</h2>
					</div>
					<p className={styles.cardMeta}>
						Cambiar la contraseña revoca todas las sesiones activas.
					</p>
					<form
						className={styles.form}
						style={{ marginTop: "1.25rem" }}
						onSubmit={passwordForm.handleSubmit((values) =>
							changePassword.mutate(values),
						)}
					>
						<Field
							label={t("auth.currentPassword")}
							error={passwordForm.formState.errors.current_password?.message}
						>
							<Input
								type="password"
								autoComplete="current-password"
								{...passwordForm.register("current_password")}
							/>
						</Field>
						<Field
							label={t("auth.newPassword")}
							error={passwordForm.formState.errors.new_password?.message}
						>
							<Input
								type="password"
								autoComplete="new-password"
								{...passwordForm.register("new_password")}
							/>
						</Field>
						<div className={styles.formActions}>
							<Button type="submit" disabled={changePassword.isPending}>
								{t("settings.changePassword")}
							</Button>
						</div>
					</form>
				</div>
			</section>

			<section className={styles.card}>
				<h2 className={styles.cardTitle}>{t("settings.sessions")}</h2>
				<p className={styles.cardMeta}>
					Revoca cualquier sesión que no reconozcas.
				</p>
				<div className={styles.list} style={{ marginTop: "1rem" }}>
					{sessions.data?.map((session) => (
						<div key={session.id} className={styles.row}>
							<span className={styles.rowMain}>
								<span className={styles.rowTitle}>
									{session.current ? "Sesión actual" : "Sesión"}
								</span>
								<span className={styles.rowMeta}>
									Inició {formatDate(session.created_at, true)} · expira{" "}
									{formatDate(session.expires_at, true)}
								</span>
							</span>
							<Button
								small
								variant={session.current ? "quiet" : "secondary"}
								onClick={() => revoke.mutate(session.id)}
								disabled={revoke.isPending}
							>
								<LogOut size={14} />
								Revocar
							</Button>
						</div>
					))}
				</div>
			</section>

			<section className={`${styles.card} ${styles.dangerZone}`}>
				<h2 className={styles.cardTitle}>{t("settings.deactivate")}</h2>
				<p className={styles.cardMeta}>{t("settings.deactivateWarning")}</p>
				<Button
					variant="danger"
					style={{ marginTop: "1rem" }}
					onClick={deactivateAccount}
					disabled={deactivate.isPending}
				>
					<Trash2 size={16} />
					{t("settings.deactivate")}
				</Button>
			</section>
		</div>
	);
}
