package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

const defaultPageSize = 20

type pageRequest struct {
	Cursor  string
	Limit   int
	Enabled bool
}

// parsePage enables cursor mode only when the client asks for it, keeping the
// previous unpaginated shape for older callers.
func parsePage(r *http.Request) (pageRequest, error) {
	query := r.URL.Query()
	cursor := query.Get("cursor")
	limit := 0
	enabled := false
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 100 {
			return pageRequest{}, apperr.BadRequest("invalid_limit", "El límite debe estar entre 1 y 100.")
		}
		limit, enabled = parsed, true
	}
	if cursor != "" {
		enabled = true
		if limit == 0 {
			limit = defaultPageSize
		}
	}
	return pageRequest{Cursor: cursor, Limit: limit, Enabled: enabled}, nil
}

func (p pageRequest) pageMeta(next string) *data.PageMeta {
	if !p.Enabled {
		return nil
	}
	return meta(p.Limit, next)
}

func pageError(err error) error {
	if errors.Is(err, data.ErrInvalidCursor) {
		return apperr.BadRequest("invalid_cursor", "El cursor no es válido.")
	}
	return err
}
