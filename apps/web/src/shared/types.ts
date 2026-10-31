export type User = {
	id: string;
	email: string;
	display_name: string;
	email_verified: boolean;
	locale: string;
	timezone: string;
	created_at: string;
};

export type Group = {
	id: string;
	name: string;
	description: string;
	owner_user_id: string;
	created_by: string;
	archived_at: string | null;
	version: number;
	created_at: string;
	updated_at: string;
	role: "owner" | "admin" | "member";
	member_count: number;
	list_count: number;
};

export type Member = {
	user_id: string;
	display_name: string;
	email: string;
	role: "owner" | "admin" | "member";
	joined_at: string;
};

export type Invitation = {
	id: string;
	group_id: string;
	group_name?: string;
	email: string;
	invited_by: string;
	status: "pending" | "accepted" | "rejected" | "canceled" | "expired";
	expires_at: string;
	version: number;
	created_at: string;
};

export type ShoppingList = {
	id: string;
	name: string;
	description: string;
	personal_owner_id: string | null;
	group_id: string | null;
	created_by: string;
	archived_at: string | null;
	version: number;
	created_at: string;
	updated_at: string;
	role?: "owner" | "admin" | "member";
	pending_count: number;
	purchased_count: number;
};

export type Purchase = {
	id: string;
	item_id: string;
	list_id: string;
	group_id: string | null;
	item_name: string;
	quantity: string;
	unit: string;
	actual_unit_price: string | null;
	total_amount: string | null;
	currency: "MXN";
	recorded_by: string;
	purchased_by: string;
	paid_by: string | null;
	purchased_at: string;
	notes: string;
	reversed_by: string | null;
	reversed_at: string | null;
	reversal_reason: string | null;
	version: number;
	created_at: string;
	updated_at: string;
	purchased_by_name?: string;
	paid_by_name?: string;
	recorded_by_name?: string;
};

export type ShoppingItem = {
	id: string;
	list_id: string;
	name: string;
	description: string;
	quantity: string;
	unit: string;
	estimated_unit_price: string | null;
	created_by: string;
	assigned_to: string | null;
	assigned_to_name?: string | null;
	notes: string;
	version: number;
	created_at: string;
	updated_at: string;
	purchase?: Purchase;
};

export type PayerTotal = {
	paid_by: string | null;
	display_name: string;
	amount: string;
	purchase_count: number;
};

export type Summary = {
	scope: string;
	period_from: string | null;
	period_to: string | null;
	include_archived: boolean;
	estimated_known: string;
	estimated_unknown_items: number;
	pending_estimated: string;
	actual_known: string;
	purchases_without_price: number;
	purchase_count: number;
	pending_count: number;
	purchased_count: number;
	unidentified_paid: string;
	by_payer: PayerTotal[];
};

export type ListExpense = {
	list_id: string;
	name: string;
	archived_at: string | null;
	actual_known: string;
	estimated_known: string;
	pending_estimated: string;
	purchase_count: number;
	pending_count: number;
};

export type Activity = {
	id: string;
	scope_type: "user" | "personal" | "group";
	actor_id: string | null;
	actor_name: string | null;
	group_id: string | null;
	list_id: string | null;
	item_id: string | null;
	purchase_id: string | null;
	action: string;
	payload: Record<string, unknown>;
	created_at: string;
};

export type Dashboard = {
	groups: Group[];
	personal_lists: ShoppingList[];
	summary: Summary;
	recent_purchases: Purchase[];
	recent_activity: Activity[];
};

export type Session = {
	id: string;
	created_at: string;
	expires_at: string;
	current: boolean;
};

export type ApiEnvelope<T> = {
	data: T;
	meta?: { next_cursor?: string; limit: number };
	error?: { code: string; message: string; details?: unknown };
	request_id?: string;
};
