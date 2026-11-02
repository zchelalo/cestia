import clsx from "clsx";
import { type ChangeEvent, useEffect, useRef, useState } from "react";
import { InputDate } from "@/components/molecules/input-date/input-date";
import styles from "@/components/molecules/input-date/input-date.module.css";
import {
	applyTimeMask,
	joinLocalDateTime,
	normalizeTime,
	splitLocalDateTime,
} from "@/components/molecules/input-date/time-utils";

export type DateTimeInputProps = {
	"aria-label"?: string;
	disabled?: boolean;
	id?: string;
	onBlur?: () => void;
	onChange: (value: string) => void;
	timeAriaLabel?: string;
	value: string;
};

export function DateTimeInput({
	"aria-label": ariaLabel,
	disabled = false,
	id,
	onBlur,
	onChange,
	timeAriaLabel = "Hora",
	value,
}: DateTimeInputProps) {
	const initial = splitLocalDateTime(value);
	const [date, setDate] = useState(initial.date);
	const [time, setTime] = useState(initial.time);
	const [invalid, setInvalid] = useState(false);
	const lastEmitted = useRef(value);

	useEffect(() => {
		if (value === lastEmitted.current) return;
		const parts = splitLocalDateTime(value);
		setDate(parts.date);
		setTime(parts.time);
		lastEmitted.current = value;
	}, [value]);

	const emit = (nextDate: string, nextTime: string) => {
		const joined = joinLocalDateTime(nextDate, nextTime);
		lastEmitted.current = joined;
		onChange(joined);
	};

	const handleTimeChange = (event: ChangeEvent<HTMLInputElement>) => {
		const masked = applyTimeMask(event.target.value);
		setTime(masked);
		setInvalid(false);
		const normalized = masked ? normalizeTime(masked) : "";
		emit(date, normalized ?? "");
	};

	return (
		<div>
			<div className={styles.dateTime}>
				<InputDate
					id={id}
					value={date}
					aria-label={ariaLabel}
					disabled={disabled}
					onBlur={onBlur}
					onChange={(nextDate) => {
						setDate(nextDate);
						emit(nextDate, time);
					}}
				/>
				<input
					id={id ? `${id}-time` : undefined}
					type="text"
					className={clsx(styles.control, styles.timeInput)}
					value={time}
					placeholder="hh:mm"
					inputMode="numeric"
					autoComplete="off"
					aria-label={timeAriaLabel}
					aria-invalid={invalid || undefined}
					disabled={disabled}
					onChange={handleTimeChange}
					onBlur={() => {
						if (time && !normalizeTime(time)) {
							setTime(splitLocalDateTime(lastEmitted.current).time);
							setInvalid(true);
						}
						onBlur?.();
					}}
				/>
			</div>
			{invalid ? (
				<span className="srOnly" role="alert">
					Hora no válida. Usa el formato hh:mm.
				</span>
			) : null}
		</div>
	);
}
