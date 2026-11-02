import { addMonths } from "date-fns";
import { ChevronLeft, ChevronRight } from "lucide-react";
import {
	type CSSProperties,
	type RefObject,
	useEffect,
	useMemo,
	useState,
} from "react";
import { CalendarSelect } from "@/components/molecules/input-date/calendar-select";
import {
	areSameDay,
	changeMonthYear,
	getCalendarDays,
	getCalendarMonthState,
	getMonthLabel,
	getMonthOptions,
	getWeekdayLabels,
	getYearOptions,
	isDateWithinRange,
	parseIsoDate,
} from "@/components/molecules/input-date/calendar-utils";
import styles from "@/components/molecules/input-date/input-date.module.css";

const dayLabelFormatter = new Intl.DateTimeFormat("es-MX", {
	day: "numeric",
	month: "long",
	year: "numeric",
});

type Props = {
	max?: string;
	min?: string;
	month: Date;
	onClose: () => void;
	onMonthChange: (month: Date) => void;
	onSelect: (iso: string) => void;
	panelRef: RefObject<HTMLDivElement | null>;
	style: CSSProperties;
	value: string;
};

export function CalendarPanel({
	max,
	min,
	month,
	onClose,
	onMonthChange,
	onSelect,
	panelRef,
	style,
	value,
}: Props) {
	const [openSelect, setOpenSelect] = useState<"month" | "year" | null>(null);
	const selectedDate = useMemo(() => parseIsoDate(value), [value]);
	const today = useMemo(() => new Date(), []);
	const weekdayLabels = useMemo(getWeekdayLabels, []);
	const calendarDays = useMemo(() => getCalendarDays(month), [month]);
	const monthOptions = useMemo(getMonthOptions, []);
	const monthState = getCalendarMonthState(month);
	const yearOptions = useMemo(
		() => getYearOptions({ max, min, selected: monthState.year }),
		[max, min, monthState.year],
	);

	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key !== "Escape") return;
			// Own Escape so an outer modal stays open while the calendar closes.
			event.stopPropagation();
			if (openSelect) {
				setOpenSelect(null);
				return;
			}
			onClose();
		};
		document.addEventListener("keydown", onKeyDown, true);
		return () => document.removeEventListener("keydown", onKeyDown, true);
	}, [onClose, openSelect]);

	return (
		<div
			ref={panelRef}
			className={styles.popover}
			data-calendar-layer="true"
			role="dialog"
			aria-label="Calendario"
			aria-modal="false"
			style={style}
		>
			<div className={styles.calendarHeader}>
				<button
					type="button"
					className={styles.navButton}
					aria-label="Mes anterior"
					onClick={() => {
						setOpenSelect(null);
						onMonthChange(addMonths(month, -1));
					}}
				>
					<ChevronLeft size={16} aria-hidden="true" />
				</button>
				<div className={styles.monthYear}>
					<CalendarSelect
						ariaLabel="Mes"
						className={styles.monthSelect}
						isOpen={openSelect === "month"}
						options={monthOptions}
						value={monthState.month}
						onChange={(nextMonth) => {
							onMonthChange(changeMonthYear(month, { month: nextMonth }));
						}}
						onOpenChange={(isOpen) => setOpenSelect(isOpen ? "month" : null)}
					/>
					<CalendarSelect
						ariaLabel="Año"
						className={styles.yearSelect}
						isOpen={openSelect === "year"}
						options={yearOptions}
						value={monthState.year}
						onChange={(nextYear) => {
							onMonthChange(changeMonthYear(month, { year: nextYear }));
						}}
						onOpenChange={(isOpen) => setOpenSelect(isOpen ? "year" : null)}
					/>
				</div>
				<button
					type="button"
					className={styles.navButton}
					aria-label="Mes siguiente"
					onClick={() => {
						setOpenSelect(null);
						onMonthChange(addMonths(month, 1));
					}}
				>
					<ChevronRight size={16} aria-hidden="true" />
				</button>
			</div>

			<div className={styles.weekdayGrid}>
				{weekdayLabels.map((weekday) => (
					<span key={weekday} className={styles.weekday}>
						{weekday}
					</span>
				))}
			</div>

			<div className={styles.dayGrid}>
				{calendarDays.map((day) => {
					const isDisabled = !isDateWithinRange(day.iso, min, max);
					const isSelected = areSameDay(day.date, selectedDate);
					const isToday = areSameDay(day.date, today);
					return (
						<button
							key={day.iso}
							type="button"
							data-date={day.iso}
							aria-label={dayLabelFormatter.format(day.date)}
							aria-current={isToday ? "date" : undefined}
							className={[
								styles.dayButton,
								day.outsideMonth && styles.dayOutside,
								isToday && styles.dayToday,
								isSelected && styles.daySelected,
								isDisabled && styles.dayDisabled,
							]
								.filter(Boolean)
								.join(" ")}
							disabled={isDisabled}
							onClick={() => {
								if (isDisabled) return;
								setOpenSelect(null);
								onSelect(day.iso);
								onClose();
							}}
						>
							{day.day}
						</button>
					);
				})}
			</div>

			<span className="srOnly" aria-live="polite">
				{getMonthLabel(monthState.month)} {monthState.year}
			</span>
		</div>
	);
}
