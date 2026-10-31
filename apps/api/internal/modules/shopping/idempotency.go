package shopping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

func executeIdempotent[T any](
	ctx context.Context,
	store *data.Store,
	actorID uuid.UUID,
	resourceID uuid.UUID,
	operation, key string,
	input any,
	access func(data.Querier) error,
	mutate func(data.Querier) (T, error),
	resourceOf func(T) uuid.UUID,
) (T, error) {
	var zero T
	if key == "" {
		return zero, apperr.BadRequest("idempotency_key_required", "La cabecera Idempotency-Key es obligatoria.")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return zero, apperr.Internal(err)
	}
	hash := requestFingerprint(operation, resourceID, body)
	legacyHash := legacyRequestFingerprint(body)
	now := time.Now().UTC()

	var result T
	err = store.Within(ctx, func(q data.Querier) error {
		queries := q.Queries()
		if err := queries.AcquireIdempotencyLock(ctx, actorID.String()+":"+operation+":"+key); err != nil {
			return err
		}
		if err := access(q); err != nil {
			return err
		}
		stored, err := queries.GetIdempotencyRequest(ctx, cestia.GetIdempotencyRequestParams{
			ActorID: actorID, Operation: operation, Key: key, Now: now,
		})
		if err == nil {
			if stored.RequestHash == hash && stored.ResourceID != nil && *stored.ResourceID == resourceID {
				if err := json.Unmarshal(stored.ResponseBody, &result); err != nil {
					return apperr.Internal(fmt.Errorf("decode idempotent response: %w", err))
				}
				return nil
			}
			// Records created before resource-scoped fingerprints have no
			// resource_id and hash only the command body. Accept them only when
			// the stored response belongs to the requested resource, then
			// upgrade the row so later retries use the strict check.
			if stored.ResourceID == nil && stored.RequestHash == legacyHash {
				if err := json.Unmarshal(stored.ResponseBody, &result); err != nil {
					return apperr.Internal(fmt.Errorf("decode idempotent response: %w", err))
				}
				if resourceOf(result) == resourceID {
					if err := queries.UpgradeLegacyIdempotencyRequest(ctx, cestia.UpgradeLegacyIdempotencyRequestParams{
						ResourceID: resourceID, RequestHash: hash,
						ActorID: actorID, Operation: operation, Key: key,
					}); err != nil {
						return err
					}
					return nil
				}
			}
			return apperr.Conflict("idempotency_key_reused", "La clave de idempotencia ya fue usada con otros datos.")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		mutated, err := mutate(q)
		if err != nil {
			return err
		}
		response, err := json.Marshal(mutated)
		if err != nil {
			return apperr.Internal(err)
		}
		if err := queries.UpsertIdempotencyRequest(ctx, cestia.UpsertIdempotencyRequestParams{
			ActorID: actorID, Operation: operation, Key: key, RequestHash: hash,
			ResponseStatus: 200, ResponseBody: response, ResourceID: &resourceID,
			ExpiresAt: now.Add(30 * 24 * time.Hour), CreatedAt: now,
		}); err != nil {
			return err
		}
		result = mutated
		return nil
	})
	if err != nil {
		return zero, apperr.From(err)
	}
	return result, nil
}

func requestFingerprint(operation string, resourceID uuid.UUID, body []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(operation))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(resourceID.String()))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil))
}

// legacyRequestFingerprint reproduces the pre-resource fingerprint so records
// written by older versions can be recognised once and upgraded in place.
func legacyRequestFingerprint(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
