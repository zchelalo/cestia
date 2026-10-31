import { X } from "lucide-react";
import { type ReactNode, useEffect } from "react";
import styles from "@/components/Modal.module.css";

type Props = { title: string; onClose: () => void; children: ReactNode };

export function Modal({ title, onClose, children }: Props) {
	useEffect(() => {
		const onKey = (event: KeyboardEvent) => {
			if (event.key === "Escape") onClose();
		};
		document.addEventListener("keydown", onKey);
		const previous = document.body.style.overflow;
		document.body.style.overflow = "hidden";
		return () => {
			document.removeEventListener("keydown", onKey);
			document.body.style.overflow = previous;
		};
	}, [onClose]);

	return (
		<div className={styles.overlay}>
			<section
				className={styles.panel}
				role="dialog"
				aria-modal="true"
				aria-label={title}
			>
				<header className={styles.header}>
					<h2 className={styles.title}>{title}</h2>
					<button
						type="button"
						className={styles.close}
						onClick={onClose}
						aria-label="Cerrar"
					>
						<X size={18} />
					</button>
				</header>
				<div className={styles.body}>{children}</div>
			</section>
		</div>
	);
}
