package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/mailer"
	"github.com/zchelalo/cestia/apps/api/internal/secure"
)

type Worker struct {
	store       *data.Store
	mailer      *mailer.Mailer
	secret      string
	appOrigin   string
	logger      *slog.Logger
	now         func() time.Time
	maxAttempts int
}

func NewWorker(store *data.Store, sender *mailer.Mailer, secret, appOrigin string, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		store: store, mailer: sender, secret: secret, appOrigin: appOrigin, logger: logger,
		now: time.Now, maxAttempts: 5,
	}
}

func (w *Worker) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Process(ctx); err != nil {
				w.logger.Warn("email delivery pass failed", "stage", "process_deliveries", "error", err)
			}
		}
	}
}

func (w *Worker) Process(ctx context.Context) error {
	now := w.now().UTC()
	lease := uuid.Must(uuid.NewV7())
	var jobs []data.DeliveryJob
	err := w.store.Within(ctx, func(q data.Querier) error {
		var err error
		jobs, err = data.ClaimEmailDeliveries(ctx, q, lease, now, now.Add(2*time.Minute), 10)
		return err
	})
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err := w.processJob(ctx, job); err != nil {
			w.logger.Warn("email delivery failed", "stage", "deliver_email", "delivery_id", job.ID, "error", err)
		}
	}
	return nil
}

func (w *Worker) processJob(ctx context.Context, job data.DeliveryJob) error {
	now := w.now().UTC()
	var delivery data.DeliveryContext
	if err := w.store.Within(ctx, func(q data.Querier) error {
		var err error
		delivery, err = data.EmailDeliveryContext(ctx, q, job)
		return err
	}); err != nil {
		return w.retryOrFail(ctx, job, "resolve delivery context", now)
	}
	if delivery.Purpose == "invitation" {
		if delivery.Invitation.Status != "pending" || delivery.Invitation.Generation != job.Generation ||
			!delivery.Invitation.ExpiresAt.After(now) {
			return w.cancel(ctx, job, "invitation no longer active", now)
		}
	}
	if !delivery.Invitation.ExpiresAt.After(now) {
		return w.cancel(ctx, job, "email action expired", now)
	}
	token := secure.Derive(w.secret, delivery.Purpose, delivery.Invitation.ID.String(), job.Generation)
	var kind, subject, url string
	var templateData any
	switch delivery.Purpose {
	case "invitation":
		kind = "invitation"
		subject = "Te invitaron a " + delivery.Invitation.GroupName + " en Cestia"
		url = fmt.Sprintf("%s/invitations/accept?token=%s", w.appOrigin, token)
		templateData = map[string]any{
			"GroupName":   delivery.Invitation.GroupName,
			"InviterName": delivery.User.DisplayName,
			"URL":         url,
			"ExpiresAt":   delivery.Invitation.ExpiresAt.Format("2 Jan 2006, 15:04 MST"),
		}
	case "verify_email":
		kind = "verify"
		subject = "Verifica tu correo de Cestia"
		url = fmt.Sprintf("%s/verify-email?token=%s", w.appOrigin, token)
		templateData = map[string]any{"Name": delivery.User.DisplayName, "URL": url}
	case "reset_password":
		kind = "reset"
		subject = "Recupera tu acceso a Cestia"
		url = fmt.Sprintf("%s/reset-password?token=%s", w.appOrigin, token)
		templateData = map[string]any{"Name": delivery.User.DisplayName, "URL": url}
	default:
		return w.cancel(ctx, job, "unsupported email action", now)
	}
	if err := w.mailer.Send(ctx, mailer.Message{To: job.Recipient, Subject: subject, Kind: kind, Data: templateData}); err != nil {
		return w.retryOrFail(ctx, job, "smtp send failed: "+err.Error(), now)
	}
	return w.store.Within(ctx, func(q data.Querier) error {
		_, err := data.MarkDeliverySent(ctx, q, job.ID, job.LeaseToken, now)
		if err != nil {
			return err
		}
		if err := data.CleanupExpired(ctx, q, now); err != nil {
			w.logger.Warn("cleanup expired records failed", "stage", "cleanup", "error", err)
		}
		return nil
	})
}

func (w *Worker) retryOrFail(ctx context.Context, job data.DeliveryJob, reason string, now time.Time) error {
	return w.store.Within(ctx, func(q data.Querier) error {
		if job.AttemptCount >= w.maxAttempts {
			_, err := data.MarkDeliveryFailed(ctx, q, job.ID, job.LeaseToken, truncate(reason, 1000), now)
			return err
		}
		delay := backoff(job.AttemptCount)
		_, err := data.MarkDeliveryRetry(ctx, q, job.ID, job.LeaseToken, now.Add(delay), truncate(reason, 1000), now)
		return err
	})
}

func (w *Worker) cancel(ctx context.Context, job data.DeliveryJob, reason string, now time.Time) error {
	return w.store.Within(ctx, func(q data.Querier) error {
		_, err := data.MarkDeliveryCanceled(ctx, q, job.ID, job.LeaseToken, reason, now)
		return err
	})
}

func backoff(attempt int) time.Duration {
	switch {
	case attempt <= 1:
		return time.Minute
	case attempt == 2:
		return 5 * time.Minute
	case attempt == 3:
		return 30 * time.Minute
	default:
		return 2 * time.Hour
	}
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
