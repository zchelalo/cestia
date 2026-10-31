package shopping

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
)

func TestRequestFingerprintBindsOperationAndResource(t *testing.T) {
	t.Parallel()

	body := []byte(`{"item_id":"same-body"}`)
	itemA := uuid.Must(uuid.NewV7())
	itemB := uuid.Must(uuid.NewV7())

	fingerprintA := requestFingerprint("purchase.record", itemA, body)
	if fingerprintA == requestFingerprint("purchase.record", itemB, body) {
		t.Fatal("different resources must not share an idempotency fingerprint")
	}
	if fingerprintA == requestFingerprint("purchase.correct", itemA, body) {
		t.Fatal("different operations must not share an idempotency fingerprint")
	}
	if fingerprintA != requestFingerprint("purchase.record", itemA, body) {
		t.Fatal("fingerprint must be deterministic")
	}
}

func TestLegacyRequestFingerprintMatchesPreviousFormat(t *testing.T) {
	t.Parallel()

	body := []byte(`{"actual_unit_price":"10.00"}`)
	sum := sha256.Sum256(body)
	want := hex.EncodeToString(sum[:])
	legacy := legacyRequestFingerprint(body)
	if legacy != want {
		t.Fatalf("legacy fingerprint = %s, want %s", legacy, want)
	}
	if legacy == requestFingerprint("purchase.record", uuid.Must(uuid.NewV7()), body) {
		t.Fatal("legacy and resource-scoped fingerprints must differ")
	}
}
