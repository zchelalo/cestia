import {
	cloneElement,
	type InputHTMLAttributes,
	isValidElement,
	type ReactNode,
	type SelectHTMLAttributes,
	type TextareaHTMLAttributes,
	useId,
} from "react";
import styles from "@/components/Field.module.css";

type Common = {
	label: string;
	htmlFor?: string;
	hint?: string;
	error?: string;
	children?: ReactNode;
};

export function Field({ label, htmlFor, hint, error, children }: Common) {
	const generatedId = useId();
	const childId =
		htmlFor ??
		(isValidElement<{ id?: string }>(children) && children.props.id
			? children.props.id
			: generatedId);
	const control =
		isValidElement<{ id?: string }>(children) && !children.props.id
			? cloneElement(children, { id: generatedId })
			: children;

	return (
		<div className={styles.field}>
			<label className={styles.label} htmlFor={childId}>
				{label}
			</label>
			{control}
			{hint && !error ? <span className={styles.hint}>{hint}</span> : null}
			{error ? <span className={styles.error}>{error}</span> : null}
		</div>
	);
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
	return <input {...props} className={styles.input} />;
}

export function Select(props: SelectHTMLAttributes<HTMLSelectElement>) {
	return <select {...props} className={styles.input} />;
}

export function Textarea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
	return <textarea {...props} className={styles.input} />;
}
