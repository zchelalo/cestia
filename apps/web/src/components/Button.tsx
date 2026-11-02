import clsx from "clsx";
import { type ButtonHTMLAttributes, Children, type ReactNode } from "react";
import styles from "@/components/Button.module.css";

type Variant =
	| "primary"
	| "secondary"
	| "surface"
	| "quiet"
	| "danger"
	| "success"
	| "primarySoft"
	| "dangerSoft";

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
	variant?: Variant;
	small?: boolean;
	fullWidth?: boolean;
};

function isTextChild(child: ReactNode) {
	return typeof child === "string" || typeof child === "number";
}

export function Button({
	variant = "primary",
	small,
	fullWidth,
	className,
	children,
	type = "button",
	...props
}: Props) {
	// Text is wrapped so the optical centering correction never moves icons.
	const content = Children.map(children, (child) =>
		isTextChild(child) ? <span className={styles.label}>{child}</span> : child,
	);

	return (
		<button
			type={type}
			className={clsx(
				styles.root,
				variant !== "primary" && styles[variant],
				small && styles.small,
				fullWidth && styles.fullWidth,
				className,
			)}
			{...props}
		>
			{content}
		</button>
	);
}
