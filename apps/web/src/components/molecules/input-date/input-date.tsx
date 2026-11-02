import clsx from "clsx";
import { CalendarDays } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { CalendarPanel } from "@/components/molecules/input-date/calendar-panel";
import {
	applyDateMask,
	formatDisplayDate,
	getInitialCalendarMonth,
	isDateWithinRange,
	parseDisplayDate,
	parseIsoDate,
} from "@/components/molecules/input-date/calendar-utils";
import styles from "@/components/molecules/input-date/input-date.module.css";
import { useDatePopover } from "@/components/molecules/input-date/use-date-popover";

export type InputDateProps = {
	"aria-label"?: string;
	className?: string;
	disabled?: boolean;
	id?: string;
	max?: string;
	min?: string;
	onBlur?: () => void;
	onChange: (value: string) => void;
	placeholder?: string;
	value: string;
};

export function InputDate({
	"aria-label": ariaLabel,
	className,
	disabled = false,
	id,
	max,
	min,
	onBlur,
	onChange,
	placeholder = "dd/mm/aaaa",
	value,
}: InputDateProps) {
	const [draft, setDraft] = useState(() => formatDisplayDate(value));
	const [open, setOpen] = useState(false);
	const [invalid, setInvalid] = useState(false);
	const [month, setMonth] = useState(() =>
		getInitialCalendarMonth(value, min, max),
	);
	const containerRef = useRef<HTMLDivElement>(null);
	const panelRef = useRef<HTMLDivElement>(null);
	const inputRef = useRef<HTMLInputElement>(null);
	const popoverStyle = useDatePopover({
		open,
		panelRef,
		triggerRef: containerRef,
	});

	useEffect(() => {
		setDraft(formatDisplayDate(value));
	}, [value]);

	useEffect(() => {
		if (!open) return;
		const onPointerDown = (event: PointerEvent) => {
			const target = event.target as Node;
			if (
				containerRef.current?.contains(target) ||
				panelRef.current?.contains(target)
			) {
				return;
			}
			setOpen(false);
		};
		document.addEventListener("pointerdown", onPointerDown);
		return () => document.removeEventListener("pointerdown", onPointerDown);
	}, [open]);

	useEffect(() => {
		if (disabled) setOpen(false);
	}, [disabled]);

	const openCalendar = () => {
		if (disabled) return;
		setInvalid(false);
		setMonth(getInitialCalendarMonth(value, min, max));
		setOpen(true);
	};

	const commit = (iso: string) => {
		setDraft(formatDisplayDate(iso));
		setInvalid(false);
		setOpen(false);
		onChange(iso);
		inputRef.current?.focus();
	};

	return (
		<div ref={containerRef} className={clsx(styles.root, className)}>
			<input
				ref={inputRef}
				id={id}
				type="text"
				className={clsx(
					styles.control,
					styles.dateInput,
					invalid && styles.invalid,
				)}
				value={draft}
				placeholder={placeholder}
				inputMode="numeric"
				autoComplete="off"
				aria-label={ariaLabel}
				aria-invalid={invalid || undefined}
				disabled={disabled}
				onChange={(event) => {
					const masked = applyDateMask(event.target.value);
					setDraft(masked);
					setInvalid(false);
					if (!masked) {
						onChange("");
						return;
					}
					const iso = parseDisplayDate(masked);
					if (iso && isDateWithinRange(iso, min, max)) onChange(iso);
				}}
				onBlur={() => {
					const iso = draft ? parseDisplayDate(draft) : "";
					const isValid =
						Boolean(iso) && isDateWithinRange(iso ?? "", min, max);
					if (draft && !isValid) {
						setDraft(formatDisplayDate(value));
						setInvalid(true);
					} else {
						setInvalid(false);
					}
					onBlur?.();
				}}
				onKeyDown={(event) => {
					if (event.key === "ArrowDown" && !open) {
						event.preventDefault();
						openCalendar();
					}
				}}
			/>
			<button
				type="button"
				className={styles.iconButton}
				aria-label="Abrir calendario"
				disabled={disabled}
				onClick={() => {
					if (open) {
						setOpen(false);
						return;
					}
					openCalendar();
				}}
			>
				<CalendarDays className={styles.icon} aria-hidden="true" />
			</button>
			{invalid ? (
				<span className="srOnly" role="alert">
					Fecha no válida. Se restauró el último valor válido.
				</span>
			) : null}
			{open
				? createPortal(
						<CalendarPanel
							panelRef={panelRef}
							style={popoverStyle}
							month={month}
							value={parseIsoDate(value) ? value : ""}
							min={min}
							max={max}
							onMonthChange={setMonth}
							onSelect={commit}
							onClose={() => {
								setOpen(false);
								inputRef.current?.focus();
							}}
						/>,
						document.body,
					)
				: null}
		</div>
	);
}
