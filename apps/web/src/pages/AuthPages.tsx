import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { z } from "zod";
import { AuthShell } from "@/components/AuthShell";
import { Button } from "@/components/Button";
import { Field, Input } from "@/components/Field";
import styles from "@/pages/AuthPages.module.css";
import { apiData } from "@/shared/http";
import { useSession } from "@/shared/session";
import type { User } from "@/shared/types";

const signInSchema = z.object({
	email: z.email("Ingresa un correo válido."),
	password: z.string().min(1, "Ingresa tu contraseña."),
});

const signUpSchema = z.object({
	display_name: z
		.string()
		.trim()
		.min(1, "Ingresa tu nombre.")
		.max(120, "Máximo 120 caracteres."),
	email: z.email("Ingresa un correo válido."),
	password: z
		.string()
		.min(12, "Usa al menos 12 caracteres.")
		.max(72, "Máximo 72 caracteres."),
});

const forgotSchema = z.object({ email: z.email("Ingresa un correo válido.") });

const resetSchema = z
	.object({
		password: z
			.string()
			.min(12, "Usa al menos 12 caracteres.")
			.max(72, "Máximo 72 caracteres."),
		confirm: z.string(),
	})
	.refine((value) => value.password === value.confirm, {
		message: "Las contraseñas no coinciden.",
		path: ["confirm"],
	});

type SignIn = z.infer<typeof signInSchema>;
type SignUp = z.infer<typeof signUpSchema>;
type Forgot = z.infer<typeof forgotSchema>;
type Reset = z.infer<typeof resetSchema>;

function searchValue(search: Record<string, unknown>, key: string) {
	const value = search[key];
	return typeof value === "string" ? value : "";
}

export function SignInPage() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const search = useSearch({ strict: false });
	const [error, setError] = useState("");
	const form = useForm<SignIn>({
		resolver: zodResolver(signInSchema),
		defaultValues: { email: "", password: "" },
	});
	const mutation = useMutation({
		mutationFn: (values: SignIn) =>
			apiData<{ user: User; csrf_token: string }>("/api/v1/auth/login", {
				method: "POST",
				body: values,
			}),
	});

	async function submit(values: SignIn) {
		setError("");
		try {
			const result = await mutation.mutateAsync(values);
			queryClient.setQueryData(["session"], result.user);
			if (!result.user.email_verified) {
				window.location.assign("/verify-email");
				return;
			}
			const redirect = searchValue(
				search as Record<string, unknown>,
				"redirect",
			);
			if (redirect) {
				const target = new URL(redirect, window.location.origin);
				if (target.origin === window.location.origin) {
					window.location.assign(
						`${target.pathname}${target.search}${target.hash}`,
					);
					return;
				}
			}
			await navigate({ to: "/" });
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	return (
		<AuthShell title={t("auth.signIn")} subtitle={t("dashboard.subtitle")}>
			{error ? (
				<div className={styles.error} role="alert">
					{error}
				</div>
			) : null}
			<form
				className={styles.form}
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<Field
					label={t("auth.email")}
					error={form.formState.errors.email?.message}
				>
					<Input
						type="email"
						autoComplete="email"
						{...form.register("email")}
					/>
				</Field>
				<Field
					label={t("auth.password")}
					error={form.formState.errors.password?.message}
				>
					<Input
						type="password"
						autoComplete="current-password"
						{...form.register("password")}
					/>
				</Field>
				<Button type="submit" fullWidth disabled={mutation.isPending}>
					{mutation.isPending ? t("common.loading") : t("auth.signIn")}
				</Button>
			</form>
			<div className={styles.links}>
				<Link to="/forgot-password">{t("auth.forgot")}</Link>
				<span className={styles.muted}>
					{t("auth.noAccount")} <Link to="/sign-up">{t("auth.signUp")}</Link>
				</span>
			</div>
		</AuthShell>
	);
}

export function SignUpPage() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const [error, setError] = useState("");
	const form = useForm<SignUp>({
		resolver: zodResolver(signUpSchema),
		defaultValues: { display_name: "", email: "", password: "" },
	});
	const mutation = useMutation({
		mutationFn: (values: SignUp) =>
			apiData<User>("/api/v1/auth/register", { method: "POST", body: values }),
	});

	async function submit(values: SignUp) {
		setError("");
		try {
			await mutation.mutateAsync(values);
			await navigate({ to: "/verify-email" });
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	return (
		<AuthShell
			title={t("auth.signUp")}
			subtitle="Empieza con una lista personal y comparte solo lo que elijas."
		>
			{error ? (
				<div className={styles.error} role="alert">
					{error}
				</div>
			) : null}
			<form
				className={styles.form}
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<Field
					label={t("auth.displayName")}
					error={form.formState.errors.display_name?.message}
				>
					<Input autoComplete="name" {...form.register("display_name")} />
				</Field>
				<Field
					label={t("auth.email")}
					error={form.formState.errors.email?.message}
				>
					<Input
						type="email"
						autoComplete="email"
						{...form.register("email")}
					/>
				</Field>
				<Field
					label={t("auth.password")}
					error={form.formState.errors.password?.message}
					hint="12 caracteres o más."
				>
					<Input
						type="password"
						autoComplete="new-password"
						{...form.register("password")}
					/>
				</Field>
				<Button type="submit" fullWidth disabled={mutation.isPending}>
					{mutation.isPending ? t("common.loading") : t("auth.signUp")}
				</Button>
			</form>
			<div className={styles.links}>
				<span className={styles.muted}>
					{t("auth.hasAccount")} <Link to="/sign-in">{t("auth.signIn")}</Link>
				</span>
			</div>
		</AuthShell>
	);
}

export function ForgotPasswordPage() {
	const { t } = useTranslation();
	const [sent, setSent] = useState(false);
	const [error, setError] = useState("");
	const form = useForm<Forgot>({
		resolver: zodResolver(forgotSchema),
		defaultValues: { email: "" },
	});
	const mutation = useMutation({
		mutationFn: (values: Forgot) =>
			apiData("/api/v1/auth/forgot-password", { method: "POST", body: values }),
	});

	async function submit(values: Forgot) {
		setError("");
		try {
			await mutation.mutateAsync(values);
			setSent(true);
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	return (
		<AuthShell
			title={t("auth.reset")}
			subtitle="Te enviaremos un enlace de un solo uso."
		>
			{sent ? <div className={styles.notice}>{t("auth.resetSent")}</div> : null}
			{error ? (
				<div className={styles.error} role="alert">
					{error}
				</div>
			) : null}
			<form
				className={styles.form}
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<Field
					label={t("auth.email")}
					error={form.formState.errors.email?.message}
				>
					<Input
						type="email"
						autoComplete="email"
						{...form.register("email")}
					/>
				</Field>
				<Button type="submit" fullWidth disabled={mutation.isPending}>
					{t("auth.sendReset")}
				</Button>
			</form>
			<div className={styles.links}>
				<Link to="/sign-in">{t("auth.backToLogin")}</Link>
			</div>
		</AuthShell>
	);
}

export function ResetPasswordPage() {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const search = useSearch({ strict: false });
	const token = searchValue(search as Record<string, unknown>, "token");
	const [error, setError] = useState("");
	const [done, setDone] = useState(false);
	const form = useForm<Reset>({
		resolver: zodResolver(resetSchema),
		defaultValues: { password: "", confirm: "" },
	});
	const mutation = useMutation({
		mutationFn: (values: Reset) =>
			apiData("/api/v1/auth/reset-password", {
				method: "POST",
				body: { token, password: values.password },
			}),
	});

	async function submit(values: Reset) {
		setError("");
		try {
			await mutation.mutateAsync(values);
			setDone(true);
			window.setTimeout(() => void navigate({ to: "/sign-in" }), 1800);
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : t("common.error"));
		}
	}

	return (
		<AuthShell
			title="Nueva contraseña"
			subtitle="Elige una contraseña segura para recuperar el acceso."
		>
			{!token ? (
				<div className={styles.error}>
					El enlace no incluye un token válido.
				</div>
			) : null}
			{done ? (
				<div className={styles.notice}>
					Contraseña actualizada. Redirigiendo…
				</div>
			) : null}
			{error ? (
				<div className={styles.error} role="alert">
					{error}
				</div>
			) : null}
			<form
				className={styles.form}
				onSubmit={form.handleSubmit(submit)}
				noValidate
			>
				<Field
					label={t("auth.newPassword")}
					error={form.formState.errors.password?.message}
				>
					<Input
						type="password"
						autoComplete="new-password"
						{...form.register("password")}
					/>
				</Field>
				<Field
					label="Confirmar contraseña"
					error={form.formState.errors.confirm?.message}
				>
					<Input
						type="password"
						autoComplete="new-password"
						{...form.register("confirm")}
					/>
				</Field>
				<Button type="submit" fullWidth disabled={mutation.isPending || !token}>
					{t("common.save")}
				</Button>
			</form>
			<div className={styles.links}>
				<Link to="/sign-in">{t("auth.backToLogin")}</Link>
			</div>
		</AuthShell>
	);
}

export function VerifyEmailPage() {
	const { t } = useTranslation();
	const search = useSearch({ strict: false });
	const token = searchValue(search as Record<string, unknown>, "token");
	const { data: user } = useSession();
	const [status, setStatus] = useState<
		"idle" | "working" | "success" | "error"
	>("idle");
	const attempted = useRef(false);
	const queryClient = useQueryClient();
	const [message, setMessage] = useState("");
	const mutation = useMutation({
		mutationFn: () =>
			apiData("/api/v1/auth/verify-email", { method: "POST", body: { token } }),
		onSuccess: () => {
			setStatus("success");
			setMessage(t("verify.success"));
			void queryClient.invalidateQueries({ queryKey: ["session"] });
		},
		onError: (cause) => {
			setStatus("error");
			setMessage(cause instanceof Error ? cause.message : t("verify.failed"));
		},
	});
	const resend = useMutation({
		mutationFn: () =>
			apiData("/api/v1/auth/resend-verification", { method: "POST" }),
		onSuccess: () => setMessage(t("auth.verifySent")),
		onError: (cause) =>
			setMessage(cause instanceof Error ? cause.message : t("common.error")),
	});

	useEffect(() => {
		if (!token || attempted.current) return;
		attempted.current = true;
		setStatus("working");
		void mutation.mutateAsync();
	}, [token, mutation]);

	return (
		<AuthShell title={t("verify.title")} subtitle={t("auth.verifyPending")}>
			<div
				className={status === "error" ? styles.error : styles.notice}
				role="status"
			>
				{status === "working"
					? t("common.loading")
					: message || "Abre el enlace que enviamos a tu correo."}
			</div>
			{!token && user && !user.email_verified ? (
				<Button
					variant="secondary"
					onClick={() => resend.mutate()}
					disabled={resend.isPending}
					fullWidth
				>
					{t("auth.resend")}
				</Button>
			) : null}
			<div className={styles.links}>
				<Link to={status === "success" ? "/" : "/sign-in"}>
					{status === "success" ? t("nav.dashboard") : t("auth.backToLogin")}
				</Link>
			</div>
		</AuthShell>
	);
}
