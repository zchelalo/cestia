import clsx from "clsx";
import { Check, ChevronsUpDown } from "lucide-react";
import {
	type CSSProperties,
	type KeyboardEvent,
	type UIEvent,
	useCallback,
	useEffect,
	useId,
	useLayoutEffect,
	useRef,
	useState,
} from "react";
import { createPortal } from "react-dom";
import styles from "@/components/molecules/select/select.module.css";
import type {
	SelectOption,
	SelectProps,
} from "@/components/molecules/select/types";

const viewportPadding = 8;
const menuGap = 4;
const minPreferredHeight = 160;
const maxMenuHeight = 240;
const loadMoreThreshold = 24;

export function Select({
	"aria-label": ariaLabel,
	className,
	disabled = false,
	emptyLabel = "No se encontraron opciones",
	errorLabel = "Error al cargar opciones",
	hasMore = false,
	id,
	isError = false,
	isLoading = false,
	isLoadingMore = false,
	loadingLabel = "Cargando opciones...",
	name,
	onBlur,
	onChange,
	onLoadMore,
	onRetry,
	onSearchChange,
	options,
	placeholder = "Selecciona una opción",
	remote = false,
	retryLabel = "Reintentar",
	searchPlaceholder = "Buscar...",
	searchValue,
	searchable = true,
	selectedLabel,
	value,
}: SelectProps) {
	const generatedId = useId();
	const selectId = id || generatedId;
	const [open, setOpen] = useState(false);
	const [style, setStyle] = useState<CSSProperties>({
		visibility: "hidden",
	});
	const root = useRef<HTMLDivElement>(null);
	const panel = useRef<HTMLDivElement>(null);
	const list = useRef<HTMLDivElement>(null);
	const search = useRef<HTMLDivElement>(null);
	const trigger = useRef<HTMLButtonElement>(null);
	const onSearchChangeRef = useRef(onSearchChange);
	const searchValueRef = useRef(searchValue);
	onSearchChangeRef.current = onSearchChange;
	searchValueRef.current = searchValue;

	const selected = options.find((option) => option.value === value);
	const label = selected?.label ?? selectedLabel ?? placeholder;
	const isPlaceholder = !selected && !selectedLabel;
	const hasRemoteResults = options.some((option) => !option.local);
	const showEmpty =
		!isLoading &&
		!isError &&
		(remote ? !hasRemoteResults : options.length === 0);

	const resetSearch = useCallback(() => {
		if (searchValueRef.current) onSearchChangeRef.current?.("");
	}, []);

	useLayoutEffect(() => {
		if (!open) return;

		const updatePosition = () => {
			const triggerRect = trigger.current?.getBoundingClientRect();
			if (!triggerRect) return;

			const below =
				window.innerHeight - triggerRect.bottom - menuGap - viewportPadding;
			const above = triggerRect.top - menuGap - viewportPadding;
			const positionBelow = below >= minPreferredHeight || below >= above;
			const availableHeight = Math.max(
				0,
				Math.min(maxMenuHeight, positionBelow ? below : above),
			);

			setStyle({
				bottom: positionBelow
					? undefined
					: window.innerHeight - triggerRect.top + menuGap,
				left: triggerRect.left,
				maxHeight: availableHeight,
				top: positionBelow ? triggerRect.bottom + menuGap : undefined,
				visibility: "visible",
				width: triggerRect.width,
			});
		};

		const closeOutside = (event: Event) => {
			const target = event.target as Node;
			if (!root.current?.contains(target) && !panel.current?.contains(target)) {
				resetSearch();
				setOpen(false);
				onBlur?.();
			}
		};

		const closeOnEscape = (event: globalThis.KeyboardEvent) => {
			if (event.key !== "Escape") return;
			// Stop immediate propagation so an outer modal doesn't close on this Escape
			event.stopPropagation();
			resetSearch();
			setOpen(false);
			trigger.current?.focus();
		};

		updatePosition();
		window.addEventListener("resize", updatePosition);
		document.addEventListener("scroll", updatePosition, true);
		document.addEventListener("pointerdown", closeOutside);
		document.addEventListener("keydown", closeOnEscape, true);

		return () => {
			window.removeEventListener("resize", updatePosition);
			document.removeEventListener("scroll", updatePosition, true);
			document.removeEventListener("pointerdown", closeOutside);
			document.removeEventListener("keydown", closeOnEscape, true);
		};
	}, [open, resetSearch, onBlur]);

	useEffect(() => {
		if (!open || !searchable) return;
		search.current?.querySelector("input")?.focus();
	}, [open, searchable]);

	const bounded = typeof style.maxHeight === "number";
	useEffect(() => {
		if (
			!open ||
			!bounded ||
			!hasMore ||
			isError ||
			isLoading ||
			isLoadingMore
		) {
			return;
		}
		const element = list.current;
		if (!element) return;
		if (element.scrollHeight <= element.clientHeight + loadMoreThreshold) {
			onLoadMore?.();
		}
	}, [bounded, hasMore, isError, isLoading, isLoadingMore, onLoadMore, open]);

	const selectOption = (option: SelectOption) => {
		onChange?.(option.value);
		resetSearch();
		setOpen(false);
		trigger.current?.focus();
	};

	const handleScroll = (event: UIEvent<HTMLDivElement>) => {
		if (!hasMore || isError || isLoadingMore) return;
		const element = event.currentTarget;
		const remaining =
			element.scrollHeight - element.scrollTop - element.clientHeight;
		if (remaining <= loadMoreThreshold) onLoadMore?.();
	};

	const handleTriggerKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
		if (
			event.key === "ArrowDown" ||
			event.key === "Enter" ||
			event.key === " "
		) {
			event.preventDefault();
			setStyle({ visibility: "hidden" });
			setOpen(true);
		}
	};

	const panelContent = open ? (
		<div
			ref={panel}
			className={styles.panel}
			data-select-layer="true"
			style={style}
		>
			{searchable ? (
				<div ref={search} className={styles.search}>
					<input
						className={styles.searchInput}
						aria-label={searchPlaceholder ?? ariaLabel}
						placeholder={searchPlaceholder}
						type="search"
						value={searchValue ?? ""}
						onChange={(event) => onSearchChange?.(event.target.value)}
					/>
				</div>
			) : null}
			<div
				ref={list}
				className={styles.options}
				role="listbox"
				aria-label={ariaLabel}
				onScroll={handleScroll}
			>
				{isLoading ? <div className={styles.status}>{loadingLabel}</div> : null}
				{!isLoading && isError ? (
					<div className={styles.status} role="alert">
						<span>{errorLabel}</span>
						{onRetry ? (
							<button type="button" className={styles.retry} onClick={onRetry}>
								{retryLabel}
							</button>
						) : null}
					</div>
				) : null}
				{showEmpty ? <div className={styles.status}>{emptyLabel}</div> : null}
				{options.map((option) => (
					<button
						key={option.value}
						type="button"
						role="option"
						aria-selected={option.value === value}
						className={clsx(
							styles.option,
							option.value === value && styles.optionSelected,
						)}
						onClick={() => selectOption(option)}
					>
						<span className={styles.optionContent}>
							<span className={styles.optionLabel}>{option.label}</span>
							{option.description ? (
								<span className={styles.optionDescription}>
									{option.description}
								</span>
							) : null}
						</span>
						{option.value === value ? (
							<Check className={styles.checkIcon} aria-hidden="true" />
						) : null}
					</button>
				))}
				{isLoadingMore ? (
					<div className={styles.status}>{loadingLabel}</div>
				) : null}
			</div>
		</div>
	) : null;

	return (
		<div ref={root} className={clsx(styles.root, className)}>
			<button
				ref={trigger}
				id={selectId}
				name={name}
				type="button"
				className={styles.trigger}
				aria-label={ariaLabel}
				aria-haspopup="listbox"
				aria-expanded={open}
				disabled={disabled}
				onClick={() => {
					if (open) {
						resetSearch();
						setOpen(false);
						onBlur?.();
						return;
					}
					setStyle({ visibility: "hidden" });
					setOpen(true);
				}}
				onKeyDown={handleTriggerKeyDown}
			>
				<span className={clsx(isPlaceholder && styles.placeholder)}>
					{label}
				</span>
				<ChevronsUpDown className={styles.icon} aria-hidden="true" />
			</button>
			{panelContent ? createPortal(panelContent, document.body) : null}
		</div>
	);
}
