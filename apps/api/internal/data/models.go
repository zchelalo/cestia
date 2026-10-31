package data

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	DisplayName     string     `json:"display_name"`
	PasswordHash    string     `json:"-"`
	EmailVerifiedAt *time.Time `json:"-"`
	Locale          string     `json:"locale"`
	Timezone        string     `json:"timezone"`
	DisabledAt      *time.Time `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (u User) Verified() bool { return u.EmailVerifiedAt != nil }

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte
	CSRFHash  []byte
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type Group struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	OwnerUserID uuid.UUID  `json:"owner_user_id"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	ArchivedAt  *time.Time `json:"archived_at"`
	Version     int64      `json:"version"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Role        string     `json:"role"`
	MemberCount int        `json:"member_count"`
	ListCount   int        `json:"list_count"`
}

type GroupMember struct {
	UserID      uuid.UUID  `json:"user_id"`
	DisplayName string     `json:"display_name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	JoinedAt    time.Time  `json:"joined_at"`
	RemovedAt   *time.Time `json:"removed_at,omitempty"`
}

type Invitation struct {
	ID         uuid.UUID  `json:"id"`
	GroupID    uuid.UUID  `json:"group_id"`
	GroupName  string     `json:"group_name,omitempty"`
	Email      string     `json:"email"`
	InvitedBy  uuid.UUID  `json:"invited_by"`
	Status     string     `json:"status"`
	Generation int        `json:"-"`
	TokenHash  []byte     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
	ResolvedBy *uuid.UUID `json:"resolved_by"`
	Version    int64      `json:"version"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	RawToken   string     `json:"token,omitempty"`
}

type List struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	PersonalOwnerID *uuid.UUID `json:"personal_owner_id"`
	GroupID         *uuid.UUID `json:"group_id"`
	CreatedBy       uuid.UUID  `json:"created_by"`
	ArchivedAt      *time.Time `json:"archived_at"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Role            string     `json:"role,omitempty"`
	PendingCount    int        `json:"pending_count"`
	PurchasedCount  int        `json:"purchased_count"`
}

type Item struct {
	ID                 uuid.UUID  `json:"id"`
	ListID             uuid.UUID  `json:"list_id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	Quantity           string     `json:"quantity"`
	Unit               string     `json:"unit"`
	EstimatedUnitPrice *string    `json:"estimated_unit_price"`
	CreatedBy          uuid.UUID  `json:"created_by"`
	AssignedTo         *uuid.UUID `json:"assigned_to"`
	Notes              string     `json:"notes"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
	Version            int64      `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	AssignedToName     *string    `json:"assigned_to_name,omitempty"`
	Purchase           *Purchase  `json:"purchase,omitempty"`
}

type SelectOption struct {
	Value       string  `json:"value"`
	Label       string  `json:"label"`
	Description *string `json:"description,omitempty"`
}

type Purchase struct {
	ID              uuid.UUID  `json:"id"`
	ItemID          uuid.UUID  `json:"item_id"`
	ListID          uuid.UUID  `json:"list_id"`
	GroupID         *uuid.UUID `json:"group_id"`
	ItemName        string     `json:"item_name"`
	Quantity        string     `json:"quantity"`
	Unit            string     `json:"unit"`
	ActualUnitPrice *string    `json:"actual_unit_price"`
	TotalAmount     *string    `json:"total_amount"`
	Currency        string     `json:"currency"`
	RecordedBy      uuid.UUID  `json:"recorded_by"`
	PurchasedBy     uuid.UUID  `json:"purchased_by"`
	PaidBy          *uuid.UUID `json:"paid_by"`
	PurchasedAt     time.Time  `json:"purchased_at"`
	Notes           string     `json:"notes"`
	ReversedBy      *uuid.UUID `json:"reversed_by"`
	ReversedAt      *time.Time `json:"reversed_at"`
	ReversalReason  *string    `json:"reversal_reason"`
	Version         int64      `json:"version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	PurchasedByName string     `json:"purchased_by_name,omitempty"`
	PaidByName      *string    `json:"paid_by_name,omitempty"`
	RecordedByName  string     `json:"recorded_by_name,omitempty"`
}

type ExpenseSummary struct {
	Scope                 string       `json:"scope"`
	PeriodFrom            *time.Time   `json:"period_from"`
	PeriodTo              *time.Time   `json:"period_to"`
	IncludeArchived       bool         `json:"include_archived"`
	EstimatedKnown        string       `json:"estimated_known"`
	EstimatedUnknownItems int          `json:"estimated_unknown_items"`
	PendingEstimated      string       `json:"pending_estimated"`
	ActualKnown           string       `json:"actual_known"`
	PurchasesWithoutPrice int          `json:"purchases_without_price"`
	PurchaseCount         int          `json:"purchase_count"`
	PendingCount          int          `json:"pending_count"`
	PurchasedCount        int          `json:"purchased_count"`
	UnidentifiedPaid      string       `json:"unidentified_paid"`
	ByPayer               []PayerTotal `json:"by_payer"`
}

type PayerTotal struct {
	PaidBy        *uuid.UUID `json:"paid_by"`
	DisplayName   string     `json:"display_name"`
	Amount        string     `json:"amount"`
	PurchaseCount int        `json:"purchase_count"`
}

type Activity struct {
	ID         uuid.UUID  `json:"id"`
	ScopeType  string     `json:"scope_type"`
	ActorID    *uuid.UUID `json:"actor_id"`
	ActorName  *string    `json:"actor_name"`
	GroupID    *uuid.UUID `json:"group_id"`
	ListID     *uuid.UUID `json:"list_id"`
	ItemID     *uuid.UUID `json:"item_id"`
	PurchaseID *uuid.UUID `json:"purchase_id"`
	Action     string     `json:"action"`
	Payload    any        `json:"payload"`
	CreatedAt  time.Time  `json:"created_at"`
}

type EmailDelivery struct {
	ID           uuid.UUID
	InvitationID *uuid.UUID
	ActionID     *uuid.UUID
	Generation   int
	Recipient    string
	Status       string
	AttemptCount int
	LeaseToken   *uuid.UUID
}

type ListScope struct {
	PersonalOwnerID *uuid.UUID
	GroupID         *uuid.UUID
	Role            string
}

type PageMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	Limit      int    `json:"limit"`
}
