package service

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/google/uuid"
)

// AsyncNotifier enqueues WhatsApp notifications into the notification_logs
// outbox. The actual gateway send is performed asynchronously by
// NotificationWorker, which drains the outbox with durable, retryable
// delivery. Enqueue is a single fast INSERT, so HTTP handlers return without
// blocking on the (potentially slow) WhatsApp gateway.
//
// This replaces the previous fire-and-forget goroutine: a process restart no
// longer loses in-flight sends, transient gateway failures are retried instead
// of permanently failing, and the host UI's delivery status reflects reality
// because every send is a tracked, persisted row.
type AsyncNotifier struct {
	logRepo ports.NotificationLogRepository
}

// NewAsyncNotifier creates an outbox enqueuer.
func NewAsyncNotifier(logRepo ports.NotificationLogRepository) *AsyncNotifier {
	return &AsyncNotifier{logRepo: logRepo}
}

// Dispatch enqueues a notification for delivery. A row with an empty phone or
// message is recorded as 'dead' (nothing to send) so the host UI can show the
// skip; otherwise a 'pending' row is created for the worker to pick up.
// Returns an error only if the enqueue (DB insert) itself fails, letting
// callers avoid committing side effects (e.g. burning a reminder quota) for a
// notification that was never queued.
func (n *AsyncNotifier) Dispatch(
	participantID uuid.UUID,
	notificationType valueobject.NotificationType,
	phone string,
	message string,
) error {
	// Decoupled from the request context on purpose: the enqueue should
	// complete even if the originating HTTP request is cancelling.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := entity.NotificationStatusPending
	var phonePtr *string
	var errMsgPtr *string
	if phone == "" || message == "" {
		status = entity.NotificationStatusDead
		m := "no whatsapp number or empty message"
		errMsgPtr = &m
	} else {
		p := phone
		phonePtr = &p
	}

	return n.logRepo.Create(ctx, &entity.NotificationLog{
		LogID:            uuid.New(),
		ParticipantID:    participantID,
		NotificationType: notificationType,
		MessageContent:   message,
		SentAt:           time.Now(),
		Status:           status,
		Phone:            phonePtr,
		ErrorMessage:     errMsgPtr,
	})
}
