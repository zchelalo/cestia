package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

type envelope struct {
	Data      any            `json:"data,omitempty"`
	Meta      *data.PageMeta `json:"meta,omitempty"`
	Error     *apiError      `json:"error,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (s *Server) writeJSON(w http.ResponseWriter, r *http.Request, status int, value any, meta *data.PageMeta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: value, Meta: meta, RequestID: requestID(r.Context())})
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	appError := apperr.From(err)
	if appError.Status >= 500 {
		s.logger.Error("request failed", "stage", "http_error", "code", appError.Code,
			"request_id", requestID(r.Context()), "error", appError.Cause)
	}
	w.Header().Set("Content-Type", "application/json")
	if appError.Status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "60")
	}
	w.WriteHeader(appError.Status)
	_ = json.NewEncoder(w).Encode(envelope{
		Error:     &apiError{Code: appError.Code, Message: appError.Message, Details: appError.Details},
		RequestID: requestID(r.Context()),
	})
}

func meta(limit int, cursor string) *data.PageMeta {
	return &data.PageMeta{Limit: limit, NextCursor: cursor}
}

func userView(user data.User) map[string]any {
	return map[string]any{
		"id": user.ID, "email": user.Email, "display_name": user.DisplayName,
		"email_verified": user.Verified(), "locale": user.Locale, "timezone": user.Timezone,
		"created_at": user.CreatedAt,
	}
}

func sessionView(session data.Session, current bool, now time.Time) map[string]any {
	return map[string]any{
		"id": session.ID, "created_at": session.CreatedAt, "expires_at": session.ExpiresAt,
		"current": current,
	}
}
