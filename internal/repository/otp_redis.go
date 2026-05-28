package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Key prefixes. Two parallel keys are written for each OTP:
//   - `otp:user:<phone>` holds the JSON-encoded entity and drives
//     FindValidByPhone / InvalidatePreviousOTPs.
//   - `otpidx:<otp_id>`  is a reverse index used by MarkAsUsed (which
//     receives only the OTP ID). It carries the canonical phone so the
//     primary key can be located without a SCAN.
//
// Both keys share the same TTL so a missed delete on one will still
// expire automatically.
const (
	otpUserKeyFmt  = "otp:user:%s"
	otpIndexKeyFmt = "otpidx:%s"
)

// RedisOTPRepository implements OTPRepository using Redis as the storage
// backend instead of Postgres. OTPs are short-lived (≤ a few minutes),
// so we use the key's TTL instead of a periodic cleanup job.
type RedisOTPRepository struct {
	client *redis.Client
}

// NewRedisOTPRepository constructs a Redis-backed OTP repository.
func NewRedisOTPRepository(client *redis.Client) ports.OTPRepository {
	return &RedisOTPRepository{client: client}
}

// Create stores an OTP in Redis with a TTL matching its expiry.
//
// A stale entry for the same phone is overwritten — at most one valid
// OTP per number lives in the cache at a time. Callers who want to
// explicitly invalidate prior codes should call InvalidatePreviousOTPs
// (the auth use cases do this before each Create, matching the prior
// Postgres behaviour).
func (r *RedisOTPRepository) Create(ctx context.Context, otp *entity.OTPVerification) error {
	ttl := time.Until(otp.ExpiresAt)
	if ttl <= 0 {
		return domain.NewError(domain.ErrBadRequest, errors.New("otp expiry is in the past"))
	}

	payload, err := json.Marshal(otp)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	userKey := fmt.Sprintf(otpUserKeyFmt, otp.WhatsAppNumber)
	indexKey := fmt.Sprintf(otpIndexKeyFmt, otp.OTPID.String())

	pipe := r.client.TxPipeline()
	pipe.Set(ctx, userKey, payload, ttl)
	pipe.Set(ctx, indexKey, otp.WhatsAppNumber, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// FindValidByPhone returns the current OTP for a phone, or ErrNotFound
// when none is present. Entries that the cache returns as `is_used=true`
// are treated as not-found — same semantics as the prior SQL query.
func (r *RedisOTPRepository) FindValidByPhone(ctx context.Context, whatsappNumber string) (*entity.OTPVerification, error) {
	userKey := fmt.Sprintf(otpUserKeyFmt, whatsappNumber)
	raw, err := r.client.Get(ctx, userKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domain.NewError(domain.ErrNotFound, nil)
	}
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	var otp entity.OTPVerification
	if err := json.Unmarshal(raw, &otp); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	if otp.IsUsed {
		return nil, domain.NewError(domain.ErrNotFound, nil)
	}
	if otp.IsExpired() {
		// Defensive: TTL should have already evicted this; fall through
		// to NotFound so the caller surfaces a fresh "request a new code"
		// path.
		return nil, domain.NewError(domain.ErrNotFound, nil)
	}
	return &otp, nil
}

// FindByID is unused by the live OTP flow today but is part of the port
// interface — implement it via the reverse index so we don't have to
// scan Redis.
func (r *RedisOTPRepository) FindByID(ctx context.Context, otpID string) (*entity.OTPVerification, error) {
	if _, err := uuid.Parse(otpID); err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	phone, err := r.client.Get(ctx, fmt.Sprintf(otpIndexKeyFmt, otpID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, domain.NewError(domain.ErrNotFound, nil)
	}
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return r.FindValidByPhone(ctx, phone)
}

// MarkAsUsed invalidates the OTP by deleting both keys. Receivers may
// invoke this with an OTP ID that has since expired — that's not an
// error in Redis-land; we treat a missing key as a successful no-op.
func (r *RedisOTPRepository) MarkAsUsed(ctx context.Context, otpID string) error {
	indexKey := fmt.Sprintf(otpIndexKeyFmt, otpID)
	phone, err := r.client.Get(ctx, indexKey).Result()
	if errors.Is(err, redis.Nil) {
		// Already expired or already used — succeed silently to keep
		// callers idempotent.
		return nil
	}
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, fmt.Sprintf(otpUserKeyFmt, phone))
	pipe.Del(ctx, indexKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// InvalidatePreviousOTPs removes any cached OTP for the given phone. The
// reverse index entry is left to expire via its own TTL — at most a few
// minutes of stale data, harmless because the primary key is gone.
func (r *RedisOTPRepository) InvalidatePreviousOTPs(ctx context.Context, whatsappNumber string) error {
	if err := r.client.Del(ctx, fmt.Sprintf(otpUserKeyFmt, whatsappNumber)).Err(); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// CleanupExpired is a no-op against Redis — TTLs reap expired entries
// automatically. The method is retained so the OTPRepository contract
// is still satisfied; the return matches the old Postgres impl's
// "rows deleted" value (always 0 here).
func (r *RedisOTPRepository) CleanupExpired(ctx context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}
