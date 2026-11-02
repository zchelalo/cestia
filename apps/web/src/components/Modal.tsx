import { X } from "lucide-react";
import { type ReactNode, useEffect, useRef } from "react";
import styles from "@/components/Modal.module.css";

type Props = { title: string; onClose: () => void; children: ReactNode };

export function Modal({ title, onClose, children }: Props) {
	const dialogRef = useRef<HTMLElement>(null);
	const overlayRef = useRef<HTMLDivElement>(null);
	const onCloseRef = useRef(onClose);
	const pressedOnOverlay = useRef(false);

	useEffect(() => {
		onCloseRef.current = onClose;
	}, [onClose]);

	useEffect(() => {
		const previouslyFocused = document.activeElement;
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key !== "Escape") return;
			// An open select or calendar layer owns Escape until it closes itself.
			if (
				document.querySelector(
					"[data-select-layer='true'], [data-calendar-layer='true']",
				)
			) {
				return;
			}
			onCloseRef.current();
		};
		const onPointerDown = (event: PointerEvent) => {
			pressedOnOverlay.current = event.target === overlayRef.current;
		};
		const onPointerUp = (event: PointerEvent) => {
			const shouldClose =
				pressedOnOverlay.current && event.target === overlayRef.current;
			pressedOnOverlay.current = false;
			if (shouldClose) onCloseRef.current();
		};

		document.addEventListener("keydown", onKeyDown);
		document.addEventListener("pointerdown", onPointerDown);
		document.addEventListener("pointerup", onPointerUp);
		const previousOverflow = document.body.style.overflow;
		document.body.style.overflow = "hidden";
		return () => {
			document.removeEventListener("keydown", onKeyDown);
			document.removeEventListener("pointerdown", onPointerDown);
			document.removeEventListener("pointerup", onPointerUp);
			document.body.style.overflow = previousOverflow;
			if (previouslyFocused instanceof HTMLElement) previouslyFocused.focus();
		};
	}, []);

	return (
		<div ref={overlayRef} className={styles.overlay}>
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
