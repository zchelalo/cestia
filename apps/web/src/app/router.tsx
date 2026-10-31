import {
	createRootRouteWithContext,
	createRoute,
	createRouter,
	Navigate,
	Outlet,
	redirect,
} from "@tanstack/react-router";
import { AppShell } from "@/components/AppShell";
import {
	ForgotPasswordPage,
	ResetPasswordPage,
	SignInPage,
	SignUpPage,
	VerifyEmailPage,
} from "@/pages/AuthPages";
import { DashboardPage } from "@/pages/DashboardPage";
import { ExpensesPage } from "@/pages/ExpensesPage";
import { GroupDetailPage } from "@/pages/GroupDetailPage";
import { GroupsPage } from "@/pages/GroupsPage";
import { HistoryPage } from "@/pages/HistoryPage";
import { InvitationPage } from "@/pages/InvitationPage";
import { ListDetailPage } from "@/pages/ListDetailPage";
import { MembersPage } from "@/pages/MembersPage";
import { PersonalListsPage } from "@/pages/PersonalListsPage";
import { SettingsPage } from "@/pages/SettingsPage";
import { queryClient } from "@/shared/query";
import { sessionQuery } from "@/shared/session";
import type { User } from "@/shared/types";

type RouterContext = { queryClient: typeof queryClient };

const rootRoute = createRootRouteWithContext<RouterContext>()({
	component: Outlet,
	notFoundComponent: () => <Navigate to="/" />,
});

function publicOnly() {
	return async ({
		context,
		location,
	}: {
		context: RouterContext;
		location: { href: string };
	}) => {
		try {
			const user = await context.queryClient.ensureQueryData(sessionQuery);
			if (user.email_verified) throw redirect({ to: "/" });
			if (!location.href.startsWith("/verify-email"))
				throw redirect({ to: "/verify-email" });
		} catch (error) {
			if (error && typeof error === "object" && "to" in error) throw error;
		}
	};
}

async function requireSession({
	context,
	location,
}: {
	context: RouterContext;
	location: { href: string };
}) {
	let user: User;
	try {
		user = await context.queryClient.ensureQueryData(sessionQuery);
	} catch {
		throw redirect({ to: "/sign-in", search: { redirect: location.href } });
	}
	if (!user.email_verified) throw redirect({ to: "/verify-email" });
}

function protectedPage(component: React.ComponentType) {
	const Component = component;
	return () => (
		<AppShell>
			<Component />
		</AppShell>
	);
}

const signInRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/sign-in",
	beforeLoad: publicOnly(),
	component: SignInPage,
});

const signUpRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/sign-up",
	beforeLoad: publicOnly(),
	component: SignUpPage,
});

const forgotRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/forgot-password",
	component: ForgotPasswordPage,
});

const resetRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/reset-password",
	component: ResetPasswordPage,
});

const verifyRoute = createRoute({
	getParentRoute: () => rootRoute,
	path: "/verify-email",
	component: VerifyEmailPage,
});

const dashboardRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/",
	component: protectedPage(DashboardPage),
});

const personalRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/personal",
	component: protectedPage(PersonalListsPage),
});

const groupsRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/groups",
	component: protectedPage(GroupsPage),
});

const groupRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/groups/$groupId",
	component: protectedPage(GroupDetailPage),
});

const membersRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/groups/$groupId/members",
	component: protectedPage(MembersPage),
});

const listRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/lists/$listId",
	component: protectedPage(ListDetailPage),
});

const expensesRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/expenses",
	component: protectedPage(ExpensesPage),
});

const historyRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/history",
	component: protectedPage(HistoryPage),
});

const settingsRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/settings",
	component: protectedPage(SettingsPage),
});

const invitationRoute = createRoute({
	getParentRoute: () => rootRoute,
	beforeLoad: requireSession,
	path: "/invitations/accept",
	component: protectedPage(InvitationPage),
});

const routeTree = rootRoute.addChildren([
	signInRoute,
	signUpRoute,
	forgotRoute,
	resetRoute,
	verifyRoute,
	dashboardRoute,
	personalRoute,
	groupsRoute,
	groupRoute,
	membersRoute,
	listRoute,
	expensesRoute,
	historyRoute,
	settingsRoute,
	invitationRoute,
]);

export const router = createRouter({
	routeTree,
	context: { queryClient },
	defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
	interface Register {
		router: typeof router;
	}
}
