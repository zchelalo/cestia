package data

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

var ErrInvalidOptionCursor = errors.New("invalid option cursor")

type optionCursor struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func encodeCursor(name string, id uuid.UUID) string {
	payload, err := json.Marshal(optionCursor{Name: name, ID: id.String()})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(cursor string) (*string, *uuid.UUID, error) {
	if cursor == "" {
		return nil, nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, nil, ErrInvalidOptionCursor
	}
	var decoded optionCursor
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, nil, ErrInvalidOptionCursor
	}
	id, err := uuid.Parse(decoded.ID)
	if err != nil {
		return nil, nil, ErrInvalidOptionCursor
	}
	name := decoded.Name
	return &name, &id, nil
}

func ListGroupOptions(ctx context.Context, q Querier, userID uuid.UUID, search, cursor string, limit int) ([]SelectOption, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	cursorName, cursorID, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Queries().ListGroupOptions(ctx, cestia.ListGroupOptionsParams{
		UserID:     userID,
		Search:     strings.TrimSpace(search),
		CursorName: cursorName,
		CursorID:   cursorID,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		return nil, "", err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	out := make([]SelectOption, 0, len(rows))
	for _, row := range rows {
		var desc *string
		if row.Description != "" {
			desc = &row.Description
		}
		out = append(out, SelectOption{
			Value:       row.ID.String(),
			Label:       row.Name,
			Description: desc,
		})
	}
	next := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		next = encodeCursor(last.Name, last.ID)
	}
	return out, next, nil
}

func ListGroupMemberOptions(ctx context.Context, q Querier, groupID uuid.UUID, search, cursor string, limit int) ([]SelectOption, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	cursorName, cursorID, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Queries().ListGroupMemberOptions(ctx, cestia.ListGroupMemberOptionsParams{
		GroupID:    groupID,
		Search:     strings.TrimSpace(search),
		CursorName: cursorName,
		CursorID:   cursorID,
		LimitCount: int32(limit + 1),
	})
	if err != nil {
		return nil, "", err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	out := make([]SelectOption, 0, len(rows))
	for _, row := range rows {
		desc := row.Email
		out = append(out, SelectOption{
			Value:       row.UserID.String(),
			Label:       row.DisplayName,
			Description: &desc,
		})
	}
	next := ""
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		next = encodeCursor(last.DisplayName, last.UserID)
	}
	return out, next, nil
}
