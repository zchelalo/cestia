import clsx from "clsx";
import { ChevronDown } from "lucide-react";
import { useEffect, useLayoutEffect, useRef } from "react";
import type { NumberOption } from "@/components/molecules/input-date/calendar-utils";
import styles from "@/components/molecules/input-date/input-date.module.css";

type Props = {
	ariaLabel: string;
	className?: string;
	isOpen: boolean;
	onChange: (value: number) => void;
	onOpenChange: (isOpen: boolean) => void;
	options: NumberOption[];
	value: number;
};

export function CalendarSelect({
	ariaLabel,
	className,
	isOpen,
	onChange,
	onOpenChange,
	options,
	value,
}: Props) {
	const rootRef = useRef<HTMLDivElement>(null);
	const listRef = useRef<HTMLDivElement>(null);
	const selectedLabel =
		options.find((option) => option.value === value)?.label ?? String(value);

	useEffect(() => {
		if (!isOpen) return;
		const onPointerDown = (event: PointerEvent) => {
			if (!rootRef.current?.contains(event.target as Node)) {
				onOpenChange(false);
			}
		};
		document.addEventListener("pointerdown", onPointerDown);
		return () => document.removeEventListener("pointerdown", onPointerDown);
	}, [isOpen, onOpenChange]);

	useLayoutEffect(() => {
		if (!isOpen) return;
		const selectedOption = listRef.current?.querySelector<HTMLElement>(
			'[data-selected="true"]',
		);
		selectedOption?.scrollIntoView?.({ block: "nearest" });
	}, [isOpen]);

	return (
		<div ref={rootRef} className={clsx(styles.calendarSelect, className)}>
			<button
				type="button"
				aria-expanded={isOpen}
				aria-haspopup="listbox"
				aria-label={ariaLabel}
				className={styles.calendarSelectTrigger}
				onClick={() => onOpenChange(!isOpen)}
			>
				<span className={styles.calendarSelectLabel}>{selectedLabel}</span>
				<ChevronDown className={styles.calendarSelectIcon} aria-hidden="true" />
			</button>
			{isOpen ? (
				<div
					ref={listRef}
					role="listbox"
					aria-label={ariaLabel}
					className={styles.calendarSelectList}
				>
					{options.map((option) => {
						const isSelected = option.value === value;
						return (
							<button
								key={option.value}
								type="button"
								role="option"
								aria-selected={isSelected}
								data-selected={isSelected ? "true" : undefined}
								className={clsx(
									styles.calendarSelectOption,
									isSelected && styles.calendarSelectOptionSelected,
								)}
								onClick={() => {
									onChange(option.value);
									onOpenChange(false);
								}}
							>
								{option.label}
							</button>
						);
					})}
				</div>
			) : null}
		</div>
	);
}
