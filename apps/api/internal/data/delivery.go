package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

type DeliveryJob struct {
	ID           uuid.UUID
	InvitationID *uuid.UUID
	ActionID     *uuid.UUID
	Generation   int
	Recipient    string
	AttemptCount int
	LeaseToken   uuid.UUID
}

func ClaimEmailDeliveries(ctx context.Context, q Querier, lease uuid.UUID, now, leaseUntil time.Time, limit int) ([]DeliveryJob, error) {
	rows, err := q.Queries().ClaimEmailDeliveries(ctx, cestia.ClaimEmailDeliveriesParams{
		LeaseToken: lease, LeaseUntil: leaseUntil, Now: now, LimitCount: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]DeliveryJob, 0, len(rows))
	for _, row := range rows {
		var leaseToken uuid.UUID
		if row.LeaseToken != nil {
			leaseToken = *row.LeaseToken
		}
		out = append(out, DeliveryJob{
			ID: row.ID, InvitationID: row.InvitationID, ActionID: row.ActionID,
			Generation: int(row.Generation), Recipient: row.Recipient,
			AttemptCount: int(row.AttemptCount), LeaseToken: leaseToken,
		})
	}
	return out, nil
}

func MarkDeliverySent(ctx context.Context, q Querier, id, lease uuid.UUID, at time.Time) (bool, error) {
	rows, err := q.Queries().MarkDeliverySent(ctx, cestia.MarkDeliverySentParams{
		ID: id, LeaseToken: lease, At: at,
	})
	return rows == 1, err
}

func MarkDeliveryRetry(ctx context.Context, q Querier, id, lease uuid.UUID, next time.Time, reason string, at time.Time) (bool, error) {
	rows, err := q.Queries().MarkDeliveryRetry(ctx, cestia.MarkDeliveryRetryParams{
		ID: id, LeaseToken: lease, NextAttemptAt: next, LastError: reason, At: at,
	})
	return rows == 1, err
}

func MarkDeliveryCanceled(ctx context.Context, q Querier, id, lease uuid.UUID, reason string, at time.Time) (bool, error) {
	rows, err := q.Queries().MarkDeliveryCanceled(ctx, cestia.MarkDeliveryCanceledParams{
		ID: id, LeaseToken: lease, LastError: reason, At: at,
	})
	return rows == 1, err
}

func MarkDeliveryFailed(ctx context.Context, q Querier, id, lease uuid.UUID, reason string, at time.Time) (bool, error) {
	rows, err := q.Queries().MarkDeliveryFailed(ctx, cestia.MarkDeliveryFailedParams{
		ID: id, LeaseToken: lease, LastError: reason, At: at,
	})
	return rows == 1, err
}

type DeliveryContext struct {
	Invitation Invitation
	User       User
	Purpose    string
}

func EmailDeliveryContext(ctx context.Context, q Querier, job DeliveryJob) (DeliveryContext, error) {
	var result DeliveryContext
	if job.InvitationID != nil {
		invitation, err := InvitationByID(ctx, q, *job.InvitationID, false)
		if err != nil {
			return result, err
		}
		result.Invitation = invitation
		result.Purpose = "invitation"
		user, err := UserByID(ctx, q, result.Invitation.InvitedBy)
		if err != nil {
			return result, err
		}
		result.User = user
		return result, nil
	}
	action, err := q.Queries().GetEmailActionByID(ctx, *job.ActionID)
	if err != nil {
		return result, err
	}
	result.Purpose = action.Purpose
	result.User, err = UserByID(ctx, q, action.UserID)
	if err != nil {
		return result, err
	}
	result.Invitation.ID = action.ID
	result.Invitation.Generation = int(action.Generation)
	result.Invitation.ExpiresAt = action.ExpiresAt
	result.Invitation.Email = result.User.Email
	return result, nil
}
