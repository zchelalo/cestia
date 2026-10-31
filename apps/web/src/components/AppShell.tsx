import {
	Link,
	Outlet,
	useNavigate,
	useRouterState,
} from "@tanstack/react-router";
import {
	Clock3,
	Home,
	LogOut,
	PieChart,
	Settings,
	ShoppingBasket,
	Users,
} from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import styles from "@/components/AppShell.module.css";
import { Button } from "@/components/Button";
import { useLogout, useSession } from "@/shared/session";

const links = [
	{ to: "/", label: "nav.dashboard", icon: Home },
	{ to: "/personal", label: "nav.personal", icon: ShoppingBasket },
	{ to: "/groups", label: "nav.groups", icon: Users },
	{ to: "/expenses", label: "nav.expenses", icon: PieChart },
	{ to: "/history", label: "nav.history", icon: Clock3 },
] as const;

export function AppShell({ children }: { children?: ReactNode }) {
	const { t } = useTranslation();
	const { data: user } = useSession();
	const logout = useLogout();
	const navigate = useNavigate();
	const pathname = useRouterState({
		select: (state) => state.location.pathname,
	});

	const isActive = (to: string) =>
		to === "/" ? pathname === "/" : pathname.startsWith(to);

	async function signOut() {
		await logout.mutateAsync().catch(() => undefined);
		await navigate({ to: "/sign-in" });
	}

	return (
		<div className={styles.shell}>
			<aside className={styles.sidebar}>
				<Link to="/" className={styles.brand}>
					<span className={styles.brandMark}>C</span>
					<span className={styles.brandText}>
						<span className={styles.brandName}>{t("appName")}</span>
						<span className={styles.brandTag}>{t("tagline")}</span>
					</span>
				</Link>
				<nav className={styles.nav} aria-label="Principal">
					{links.map(({ to, label, icon: Icon }) => (
						<Link
							key={to}
							to={to}
							className={`${styles.navLink} ${isActive(to) ? styles.navLinkActive : ""}`}
						>
							<Icon size={17} />
							{t(label)}
						</Link>
					))}
					<Link
						to="/settings"
						className={`${styles.navLink} ${isActive("/settings") ? styles.navLinkActive : ""}`}
					>
						<Settings size={17} />
						{t("nav.settings")}
					</Link>
				</nav>
				<div className={styles.sidebarFooter}>
					<div className={styles.userCard}>
						<span className={styles.avatar}>
							{user?.display_name?.charAt(0).toUpperCase()}
						</span>
						<span className={styles.userMeta}>
							<span className={styles.userName}>{user?.display_name}</span>
							<span className={styles.userEmail}>{user?.email}</span>
						</span>
					</div>
					<Button
						variant="quiet"
						small
						onClick={signOut}
						disabled={logout.isPending}
					>
						<LogOut size={15} />
						{t("auth.signOut")}
					</Button>
				</div>
			</aside>

			<header className={styles.mobileHeader}>
				<Link to="/" className={styles.brand}>
					<span className={styles.brandMark}>C</span>
					<span className={styles.brandName}>{t("appName")}</span>
				</Link>
				<div className={styles.mobileHeaderActions}>
					<Link
						to="/settings"
						className={styles.mobileIconLink}
						aria-label={t("nav.settings")}
					>
						<Settings size={18} />
					</Link>
					<Button
						variant="quiet"
						small
						onClick={signOut}
						aria-label={t("auth.signOut")}
					>
						<LogOut size={18} />
					</Button>
				</div>
			</header>

			<main className={styles.main}>{children ?? <Outlet />}</main>

			<nav className={styles.mobileNav} aria-label="Principal">
				{links.map(({ to, label, icon: Icon }) => (
					<Link
						key={to}
						to={to}
						className={`${styles.mobileNavLink} ${isActive(to) ? styles.mobileNavLinkActive : ""}`}
					>
						<Icon size={20} />
						<span>{t(label)}</span>
					</Link>
				))}
			</nav>
		</div>
	);
}
