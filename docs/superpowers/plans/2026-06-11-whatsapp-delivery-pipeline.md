# WhatsApp Delivery Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make WhatsApp notification delivery durable, observable, and idempotent — so delivery status the host sees is real, sends survive process restarts and transient gateway failures, and duplicate/out-of-order webhooks can't corrupt status.

**Architecture:** Transactional outbox on `notification_logs` + a background worker that claims rows with `FOR UPDATE SKIP LOCKED`, sends via the WAGA gateway, and retries with exponential backoff; an idempotent webhook that advances status through a forward-only state machine via a compare-and-swap UPDATE. Reuses Postgres (already present) — no new broker.

**Tech Stack:** Go 1.25, Fiber v2, sqlx + PostgreSQL, golang-migrate, `github.com/glennprays/whatsapp-gateway-sdk-go` v0.2.0 (emits only `message.queued` / `message.sent` / `message.failed`).

**Test reality:** This repo has no DB test harness (1 test file total, no testify/sqlmock/testcontainers). Pure logic (state machine, backoff schedule) is TDD'd with stdlib `testing`. SQL/IO/wiring is verified with `go build ./...` + `go vet ./...` and documented manual staging checks. Work on a feature branch; **do not push** (CI auto-deploys `dev`).

---

## Status model (decided)

Gateway reality: only `queued` / `sent` / `failed` events exist — **no `delivered`/`read`**. So the status set is:

| status | meaning | set by |
|---|---|---|
| `queued` | legacy value (pre-existing rows); treated == `pending` | legacy |
| `pending` | enqueued in outbox, not yet picked up | write path |
| `sending` | claimed by a worker, in flight | worker |
| `sent` | gateway accepted (returned a message id) | worker / sync path / webhook `message.sent` |
| `failed` | send failed (will be retried while attempts remain) | worker / webhook `message.failed` |
| `dead` | retries exhausted; terminal | worker |

Forward-only rank for positive lifecycle: `queued/pending(0) < sending(1) < sent(2)`. `failed` applies from any non-terminal-failure state (including `sent`, since the gateway can report a post-accept failure); `dead` is worker-only and never set by the webhook.

---

## File Structure

**Phase 1A — correctness core (this plan, implemented first):**
- Modify: `migrations/000030_notification_outbox.up.sql` / `.down.sql` (new) — widen status CHECK, add outbox columns + indexes.
- Modify: `domain/entity/notification_log.go` — new status constants, new struct fields, `NextWebhookStatus` state machine + `rank`.
- Create: `domain/entity/notification_log_test.go` — TDD the state machine.
- Modify: `internal/repository/notification_log_postgres.go` — persist `whatsapp_message_id`+`error_message` in `Create`; add `UpdateStatusGuarded`.
- Modify: `domain/ports/notification_log_repository.go` — add `UpdateStatusGuarded` to interface.
- Modify: `internal/handler/webhook_handler.go` — idempotent state-machine webhook, return non-2xx on DB error.

**Phase 1B — durable outbox + worker (implemented next):**
- Create: `internal/service/notification_worker.go` — claim/send/retry/reaper loop.
- Create: `internal/service/backoff.go` (+ `_test.go`) — pure backoff schedule, TDD'd.
- Modify: `internal/repository/notification_log_postgres.go` — `ClaimPending`, `MarkSent`, `MarkFailed`, `RequeueStuck`.
- Modify: `internal/service/async_notifier.go` — `Dispatch` enqueues a `pending` row instead of spawning a goroutine.
- Modify: `internal/usecase/payment/reject_payment.go` — enqueue instead of synchronous send.
- Modify: `internal/usecase/notification/send_reminder.go`, `retry_notification.go` — consume reminder counter only on successful enqueue.
- Modify: `internal/infrastructure/wire.go` + `wire_gen.go`, `cmd/api/main.go` — construct + start/stop the worker on the app lifecycle.

---

## Phase 1A — Tasks

### Task 1: Migration — evolve `notification_logs` into an outbox

**Files:**
- Create: `migrations/000030_notification_outbox.up.sql`
- Create: `migrations/000030_notification_outbox.down.sql`

- [ ] **Step 1: Write the up migration** (additive + widened CHECK; backward compatible)

```sql
-- Evolve notification_logs into a durable outbox: widen the status set to
-- cover the worker lifecycle (pending/sending/dead) and add bookkeeping
-- columns for retry + claim. Existing rows keep status 'queued'/'sent'/
-- 'failed'; the app treats 'queued' as 'pending'. All adds are nullable or
-- defaulted so the current INSERT keeps working before code is updated.
ALTER TABLE notification_logs DROP CONSTRAINT IF EXISTS notification_logs_status_check;
ALTER TABLE notification_logs
    ADD CONSTRAINT notification_logs_status_check
    CHECK (status IN ('queued','pending','sending','sent','failed','dead'));

ALTER TABLE notification_logs
    ADD COLUMN IF NOT EXISTS attempts        INT          NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_attempts    INT          NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS locked_at       TIMESTAMP,
    ADD COLUMN IF NOT EXISTS locked_by       VARCHAR(64),
    ADD COLUMN IF NOT EXISTS phone           VARCHAR(32),
    ADD COLUMN IF NOT EXISTS updated_at      TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- One gateway message id ⇒ one log row, so the webhook compare-and-swap is
-- unambiguous. Partial unique to allow many NULLs (rows never sent).
CREATE UNIQUE INDEX IF NOT EXISTS uq_notification_logs_wamid
    ON notification_logs(whatsapp_message_id)
    WHERE whatsapp_message_id IS NOT NULL;

-- Claim query: WHERE status IN ('pending','queued','failed') AND next_attempt_at <= now()
CREATE INDEX IF NOT EXISTS idx_notification_logs_claim
    ON notification_logs(next_attempt_at)
    WHERE status IN ('pending','queued','failed');
```

- [ ] **Step 2: Write the down migration**

```sql
DROP INDEX IF EXISTS idx_notification_logs_claim;
DROP INDEX IF EXISTS uq_notification_logs_wamid;
ALTER TABLE notification_logs
    DROP COLUMN IF EXISTS attempts,
    DROP COLUMN IF EXISTS max_attempts,
    DROP COLUMN IF EXISTS next_attempt_at,
    DROP COLUMN IF EXISTS locked_at,
    DROP COLUMN IF EXISTS locked_by,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS updated_at;
ALTER TABLE notification_logs DROP CONSTRAINT IF EXISTS notification_logs_status_check;
ALTER TABLE notification_logs
    ADD CONSTRAINT notification_logs_status_check
    CHECK (status IN ('queued','sent','failed'));
```

- [ ] **Step 3: Commit** — `git add migrations/000030_* && git commit -m "feat(db): evolve notification_logs into a delivery outbox (000030)"`

### Task 2: Entity — statuses + forward-only state machine (TDD)

**Files:**
- Modify: `domain/entity/notification_log.go`
- Test: `domain/entity/notification_log_test.go`

- [ ] **Step 1: Write the failing test** for `NextWebhookStatus` and new constants.

```go
package entity

import "testing"

func TestNextWebhookStatus(t *testing.T) {
	cases := []struct {
		name    string
		current NotificationStatus
		event   string
		want    NotificationStatus
		wantOK  bool
	}{
		{"sent advances from sending", NotificationStatusSending, "message.sent", NotificationStatusSent, true},
		{"sent advances from queued", NotificationStatusQueued, "message.sent", NotificationStatusSent, true},
		{"duplicate sent is no-op", NotificationStatusSent, "message.sent", "", false},
		{"late queued never downgrades sent", NotificationStatusSent, "message.queued", "", false},
		{"failed applies after sent", NotificationStatusSent, "message.failed", NotificationStatusFailed, true},
		{"duplicate failed is no-op", NotificationStatusFailed, "message.failed", "", false},
		{"failed never resurrects dead", NotificationStatusDead, "message.failed", "", false},
		{"unknown event ignored", NotificationStatusSent, "message.read", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := NextWebhookStatus(c.current, c.event)
			if ok != c.wantOK || got != c.want {
				t.Fatalf("NextWebhookStatus(%q,%q) = (%q,%v), want (%q,%v)", c.current, c.event, got, ok, c.want, c.wantOK)
			}
		})
	}
}
```

- [ ] **Step 2: Run it, verify it fails** — `go test ./domain/entity/ -run TestNextWebhookStatus -v` → FAIL (undefined: NotificationStatusSending/Dead/NextWebhookStatus).

- [ ] **Step 3: Implement** — add to `notification_log.go`:

```go
const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSending NotificationStatus = "sending"
	NotificationStatusDead    NotificationStatus = "dead"
)

// rank orders the positive delivery lifecycle. queued is the legacy alias
// for pending. Failure states are not on this ladder.
func (s NotificationStatus) rank() int {
	switch s {
	case NotificationStatusQueued, NotificationStatusPending:
		return 0
	case NotificationStatusSending:
		return 1
	case NotificationStatusSent:
		return 2
	default:
		return -1
	}
}

// NextWebhookStatus encodes the forward-only state machine for gateway
// status webhooks. It returns the status to transition to, or ("", false)
// when the event should be ignored — duplicate, out-of-order, or terminal.
// This is what makes webhook processing idempotent.
func NextWebhookStatus(current NotificationStatus, event string) (NotificationStatus, bool) {
	switch event {
	case "message.queued", "message.sent":
		target := NotificationStatusSent
		if event == "message.queued" {
			// queued is below sent on the ladder; only meaningful before send.
			target = NotificationStatusSent // gateway 'queued' still maps onto our accepted state's floor
		}
		if target.rank() > current.rank() {
			return target, true
		}
		return "", false
	case "message.failed":
		// Apply unless already in a terminal failure state.
		if current == NotificationStatusFailed || current == NotificationStatusDead {
			return "", false
		}
		return NotificationStatusFailed, true
	default:
		return "", false
	}
}
```

(Note: `message.queued` is collapsed onto the `sent` floor and gated by rank, so it is a no-op once we're at `sent` — which is the common case since the synchronous send marks `sent`. It only ever advances a `pending`/`sending` row, which is the correct behaviour.)

- [ ] **Step 4: Run tests, verify pass** — `go test ./domain/entity/ -run TestNextWebhookStatus -v` → PASS.

- [ ] **Step 5: Commit** — `git commit -am "feat(notif): forward-only webhook status state machine"`

### Task 3: Repository — persist message id + guarded CAS update

**Files:**
- Modify: `internal/repository/notification_log_postgres.go`
- Modify: `domain/ports/notification_log_repository.go`

- [ ] **Step 1: Fix `Create` to persist all columns** (the headline bug) — replace the INSERT:

```go
query := `
	INSERT INTO notification_logs
	    (log_id, participant_id, notification_type, whatsapp_message_id,
	     message_content, sent_at, status, error_message, phone)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`
_, err := r.db.ExecContext(ctx, query,
	log.LogID, log.ParticipantID, log.NotificationType, log.WhatsAppMessageID,
	log.MessageContent, log.SentAt, log.Status, log.ErrorMessage, log.Phone,
)
```

- [ ] **Step 2: Add `UpdateStatusGuarded`** (compare-and-swap; 0 rows ≠ error):

```go
// UpdateStatusGuarded advances status only if the row is still at
// expectedCurrent (optimistic CAS). Zero rows affected means another
// webhook delivery already advanced it — that is a successful no-op, not
// an error, which is what makes concurrent duplicate webhooks safe.
func (r *PostgresNotificationLogRepository) UpdateStatusGuarded(ctx context.Context, logID, newStatus, expectedCurrent string, errorMessage *string) error {
	query := `
		UPDATE notification_logs
		   SET status = $2, error_message = COALESCE($4, error_message), updated_at = CURRENT_TIMESTAMP
		 WHERE log_id = $1 AND status = $3
	`
	if _, err := r.db.ExecContext(ctx, query, logID, newStatus, expectedCurrent, errorMessage); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}
```

- [ ] **Step 3: Add `Phone *string` to the entity** struct (`db:"phone"`) and the relevant SELECT lists (optional now; required for worker). Add to `notification_log.go`:

```go
Phone *string `json:"phone,omitempty" db:"phone"`
```

- [ ] **Step 4: Add `UpdateStatusGuarded` to the port interface** in `notification_log_repository.go`:

```go
// UpdateStatusGuarded advances status via optimistic compare-and-swap.
UpdateStatusGuarded(ctx context.Context, logID, newStatus, expectedCurrent string, errorMessage *string) error
```

- [ ] **Step 5: Build + vet** — `go build ./... && go vet ./...` → no errors.

- [ ] **Step 6: Commit** — `git commit -am "fix(notif): persist whatsapp_message_id on insert; add guarded CAS update"`

### Task 4: Webhook — idempotent state-machine processing

**Files:**
- Modify: `internal/handler/webhook_handler.go`

- [ ] **Step 1: Replace the status-update switch** (lines ~58-99) with the state machine + CAS + correct error semantics:

```go
switch string(payload.Event) {
case "message.queued", "message.sent", "message.failed":
	outgoing, err := verifier.ParseOutgoingWebhook(body, signature)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": "Invalid webhook payload"})
	}
	log, err := h.notificationLogRepo.FindByWhatsAppMessageID(c.Context(), outgoing.MessageId)
	if err != nil {
		// Unknown message id: nothing to update. Ack so the gateway stops retrying.
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"success": true, "message": "no matching log"})
	}
	next, ok := entity.NextWebhookStatus(log.Status, string(payload.Event))
	if !ok {
		// Duplicate / out-of-order / terminal: idempotent no-op.
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"success": true, "message": "no-op"})
	}
	var errMsg *string
	if next == entity.NotificationStatusFailed {
		m := "Failed to deliver via WhatsApp"
		errMsg = &m
	}
	if err := h.notificationLogRepo.UpdateStatusGuarded(c.Context(), log.LogID.String(), string(next), string(log.Status), errMsg); err != nil {
		// Real DB error: return 5xx so the gateway retries later.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "error": "Failed to update status"})
	}
}
return c.Status(fiber.StatusOK).JSON(fiber.Map{"success": true, "message": "Webhook received"})
```

- [ ] **Step 2: Build + vet** — `go build ./... && go vet ./...` → no errors.

- [ ] **Step 3: Commit** — `git commit -am "fix(webhook): idempotent forward-only status updates; retryable on DB error"`

---

## Phase 1B — Durable outbox + worker (next; high-level acceptance)

1. **Backoff (TDD):** `service/backoff.go` — `NextBackoff(attempt int) time.Duration` returning 30s/2m/10m/1h/6h (capped) + ±20% jitter; pure, table-tested.
2. **Repo claim/mark:** `ClaimPending(ctx, workerID, limit)` (`SELECT ... FOR UPDATE SKIP LOCKED WHERE status IN ('pending','queued','failed') AND next_attempt_at<=now()`, set `sending`,`locked_at`,`locked_by`); `MarkSent`, `MarkFailed(attempts++, next_attempt_at, → failed|dead)`, `RequeueStuck(olderThan)`.
3. **Worker:** `service/notification_worker.go` — ticker loop bound to a shutdown `context.Context`, `defer recover()` per send, claims a batch, sends via `WhatsAppService`, marks result.
4. **Write path → enqueue:** `AsyncNotifier.Dispatch` inserts a `pending` row (no goroutine); empty phone → insert `dead` row with error so the host sees the skip.
5. **reject_payment:** enqueue instead of synchronous send.
6. **Reminder counters:** bump only after a successful enqueue.
7. **Lifecycle:** construct worker in Wire, start goroutine in `main.go` after routes, stop on SIGTERM before `fiberApp.Shutdown()`.

---

## Verification

- **Unit:** `go test ./domain/entity/... ./internal/service/... -v` (state machine, backoff).
- **Build:** `go build ./... && go vet ./...`.
- **Manual (staging, after migrate up):** trigger `POST /sessions/:id/send-notifications`; confirm a row persists `whatsapp_message_id`; replay an HMAC-signed `message.failed` webhook for that id and confirm status flips to `failed`; replay a duplicate `message.sent` and confirm no-op; send a `message.sent` for an unknown id and confirm 200 + no change.

## Rollout
Migration first (additive), then Phase 1A code, then Phase 1B. All backward compatible — old `Create` still works pre-deploy; new columns are defaulted.
