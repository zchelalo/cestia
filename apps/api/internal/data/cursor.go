package data

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCursor is returned when a page cursor cannot be decoded or does
// not belong to the requested collection.
var ErrInvalidCursor = errors.New("invalid cursor")

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	legacyPageLimit  = 500
)

type cursorEnvelope struct {
	Kind string          `json:"k"`
	Data json.RawMessage `json:"d"`
}

type nameCursor struct {
	Archived int       `json:"a"`
	Name     string    `json:"n"`
	ID       uuid.UUID `json:"id"`
}

type memberCursor struct {
	Priority int       `json:"p"`
	Name     string    `json:"n"`
	ID       uuid.UUID `json:"id"`
}

type createdCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"id"`
}

type updatedCursor struct {
	Archived  int       `json:"a"`
	UpdatedAt time.Time `json:"u"`
	ID        uuid.UUID `json:"id"`
}

type payerCursor struct {
	Amount  string    `json:"a"`
	PayerID uuid.UUID `json:"p"`
}

func encodePageCursor(kind string, payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	raw, err := json.Marshal(cursorEnvelope{Kind: kind, Data: data})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodePageCursor(cursor, kind string, dest any) error {
	if cursor == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return ErrInvalidCursor
	}
	var envelope cursorEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ErrInvalidCursor
	}
	if envelope.Kind != kind {
		return ErrInvalidCursor
	}
	if err := json.Unmarshal(envelope.Data, dest); err != nil {
		return ErrInvalidCursor
	}
	return nil
}

// pageLimit resolves request limits: zero keeps the historical "return
// everything" behavior for clients that do not request pagination.
func pageLimit(limit int) (page int, fetch int32) {
	if limit <= 0 {
		return 0, legacyPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	return limit, int32(limit + 1)
}

func normalizePageLimit(limit int) int {
	if limit <= 0 {
		return defaultPageLimit
	}
	if limit > maxPageLimit {
		return maxPageLimit
	}
	return limit
}

// pageOf trims the extra row used to detect a next page and builds the cursor
// for the last returned row.
func pageOf[T any](rows []T, limit int, next func(T) string) ([]T, string) {
	if limit <= 0 || len(rows) <= limit {
		return rows, ""
	}
	rows = rows[:limit]
	if len(rows) == 0 {
		return rows, ""
	}
	return rows, next(rows[len(rows)-1])
}
