import { X } from "lucide-react";
import { type ReactNode, useEffect, useRef } from "react";
import styles from "@/components/Modal.module.css";

type Props = {
	children: ReactNode;
	closeDisabled?: boolean;
	onClose: () => void;
	title: string;
	trapFocus?: boolean;
};

const focusableSelector = [
	"button:not([disabled])",
	"[href]",
	"input:not([disabled])",
	"select:not([disabled])",
	"textarea:not([disabled])",
	"[tabindex]:not([tabindex='-1'])",
].join(",");

export function Modal({
	children,
	closeDisabled = false,
	onClose,
	title,
	trapFocus = false,
}: Props) {
	const dialogRef = useRef<HTMLElement>(null);
	const overlayRef = useRef<HTMLDivElement>(null);
	const onCloseRef = useRef(onClose);
	const closeDisabledRef = useRef(closeDisabled);
	const trapFocusRef = useRef(trapFocus);
	const pressedOnOverlay = useRef(false);
	// Captured during the first render, before ConfirmDialog's autoFocus moves
	// focus into the panel, so closing returns focus to the trigger.
	const previouslyFocused = useRef<Element | null>(document.activeElement);

	useEffect(() => {
		onCloseRef.current = onClose;
	}, [onClose]);

	useEffect(() => {
		closeDisabledRef.current = closeDisabled;
	}, [closeDisabled]);

	useEffect(() => {
		trapFocusRef.current = trapFocus;
	}, [trapFocus]);

	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key === "Escape") {
				if (closeDisabledRef.current) return;
				// An open select or calendar layer owns Escape until it closes itself.
				if (
					document.querySelector(
						"[data-select-layer='true'], [data-calendar-layer='true']",
					)
				) {
					return;
				}
				onCloseRef.current();
				return;
			}
			if (event.key !== "Tab" || !trapFocusRef.current) return;
			const dialog = dialogRef.current;
			if (!dialog) return;
			const focusable = Array.from(
				dialog.querySelectorAll<HTMLElement>(focusableSelector),
			);
			if (focusable.length === 0) return;
			const first = focusable[0];
			const last = focusable[focusable.length - 1];
			const active = document.activeElement;
			if (!dialog.contains(active)) return;
			if (event.shiftKey && active === first) {
				event.preventDefault();
				last.focus();
			} else if (!event.shiftKey && active === last) {
				event.preventDefault();
				first.focus();
			}
		};
		const onPointerDown = (event: PointerEvent) => {
			if (closeDisabledRef.current) return;
			pressedOnOverlay.current = event.target === overlayRef.current;
		};
		const onPointerUp = (event: PointerEvent) => {
			if (closeDisabledRef.current) {
				pressedOnOverlay.current = false;
				return;
			}
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
			if (previouslyFocused.current instanceof HTMLElement) {
				previouslyFocused.current.focus();
			}
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
						disabled={closeDisabled}
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
