import { X } from "lucide-react";
import { type ReactNode, useEffect, useRef } from "react";
import styles from "@/components/Modal.module.css";

type Props = { title: string; onClose: () => void; children: ReactNode };

export function Modal({ title, onClose, children }: Props) {
	const dialogRef = useRef<HTMLElement>(null);
	const onCloseRef = useRef(onClose);
	useEffect(() => {
		onCloseRef.current = onClose;
	}, [onClose]);

	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key === "Escape") {
				// If an open select dropdown exists in document body portal, let the select handle Escape
				if (document.querySelector("[data-select-layer='true']")) {
					return;
				}
				onCloseRef.current();
			}
		};
		document.addEventListener("keydown", onKeyDown);
		const previous = document.body.style.overflow;
		document.body.style.overflow = "hidden";
		return () => {
			document.removeEventListener("keydown", onKeyDown);
			document.body.style.overflow = previous;
		};
	}, []);

	return (
		<div className={styles.overlay}>
			<section
				ref={dialogRef}
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
