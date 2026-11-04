import type { FormEvent, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/Button";
import { Modal } from "@/components/Modal";
import styles from "@/components/molecules/confirm-dialog/confirm-dialog.module.css";

type Props = {
	cancelLabel?: string;
	children?: ReactNode;
	confirmDisabled?: boolean;
	confirmLabel: string;
	description?: ReactNode;
	error?: string | null;
	onCancel: () => void;
	onConfirm: () => void;
	pending?: boolean;
	title: string;
};

export function ConfirmDialog({
	cancelLabel,
	children,
	confirmDisabled = false,
	confirmLabel,
	description,
	error,
	onCancel,
	onConfirm,
	pending = false,
	title,
}: Props) {
	const { t } = useTranslation();

	const submit = (event: FormEvent<HTMLFormElement>) => {
		event.preventDefault();
		if (pending || confirmDisabled) return;
		onConfirm();
	};

	return (
		<Modal title={title} onClose={onCancel} closeDisabled={pending} trapFocus>
			<form className={styles.form} onSubmit={submit}>
				{description ? (
					<p className={styles.description}>{description}</p>
				) : null}
				{children}
				{error ? (
					<div className={styles.error} role="alert">
						{error}
					</div>
				) : null}
				<div className={styles.actions}>
					<Button
						variant="secondary"
						onClick={onCancel}
						disabled={pending}
						autoFocus
					>
						{cancelLabel ?? t("common.cancel")}
					</Button>
					<Button
						type="submit"
						variant="danger"
						disabled={pending || confirmDisabled}
					>
						{pending ? t("common.processing") : confirmLabel}
					</Button>
				</div>
			</form>
		</Modal>
	);
}
