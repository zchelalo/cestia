import clsx from "clsx";
import type { ButtonHTMLAttributes } from "react";
import styles from "@/components/Button.module.css";

type Variant = "primary" | "secondary" | "quiet" | "danger" | "success";

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
	variant?: Variant;
	small?: boolean;
	fullWidth?: boolean;
};

export function Button({
	variant = "primary",
	small,
	fullWidth,
	className,
	type = "button",
	...props
}: Props) {
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
		/>
	);
}
