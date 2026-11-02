import {
	type CSSProperties,
	type RefObject,
	useLayoutEffect,
	useState,
} from "react";

const viewportPadding = 8;
const popoverGap = 4;
const preferredWidth = 304;
const fallbackHeight = 340;

type Params = {
	open: boolean;
	panelRef: RefObject<HTMLDivElement | null>;
	triggerRef: RefObject<HTMLElement | null>;
};

export function useDatePopover({ open, panelRef, triggerRef }: Params) {
	const [style, setStyle] = useState<CSSProperties>({ visibility: "hidden" });

	useLayoutEffect(() => {
		if (!open) return;

		const update = () => {
			const triggerRect = triggerRef.current?.getBoundingClientRect();
			if (!triggerRect) return;
			const width = Math.min(
				preferredWidth,
				window.innerWidth - viewportPadding * 2,
			);
			const height = panelRef.current?.offsetHeight || fallbackHeight;
			const below =
				window.innerHeight - triggerRect.bottom - popoverGap - viewportPadding;
			const above = triggerRect.top - popoverGap - viewportPadding;
			const openBelow = below >= height || below >= above;
			const left = Math.min(
				Math.max(viewportPadding, triggerRect.left),
				window.innerWidth - viewportPadding - width,
			);

			setStyle({
				bottom: openBelow
					? undefined
					: window.innerHeight - triggerRect.top + popoverGap,
				left,
				maxHeight: Math.max(0, Math.min(height, openBelow ? below : above)),
				top: openBelow ? triggerRect.bottom + popoverGap : undefined,
				visibility: "visible",
				width,
			});
		};

		update();
		window.addEventListener("resize", update);
		document.addEventListener("scroll", update, true);
		window.visualViewport?.addEventListener("resize", update);
		window.visualViewport?.addEventListener("scroll", update);

		return () => {
			window.removeEventListener("resize", update);
			document.removeEventListener("scroll", update, true);
			window.visualViewport?.removeEventListener("resize", update);
			window.visualViewport?.removeEventListener("scroll", update);
		};
	}, [open, panelRef, triggerRef]);

	return style;
}
