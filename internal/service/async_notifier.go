package service

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/google/uuid"
)

// AsyncNotifier dispatches WhatsApp messages on background goroutines
// so the HTTP handler can return immediately. The WAGA gateway can
// take seconds-to-minutes per send when the upstream WhatsApp API is
// slow; without this wrapper, bulk-notify on a 50-person session
// would block the request well past the FE's fetch timeout.
//
// Each Dispatch:
//   - spawns one goroutine,
//   - uses a fresh context.Background() with a per-send timeout so
//     the request context cancelling on response doesn't kill the
//     send mid-flight,
//   - writes a notification_logs row whether the send succeeds or fails.
//
// We deliberately don't wait for the goroutines: the caller treats
// the return as "queued". For a real production system this would be
// upgraded to a durable queue (NATS/Redis/SQS) but for a single-node
// dev/staging deployment goroutines + the per-row log are sufficient.
type AsyncNotifier struct {
	whatsapp *WhatsAppService
	logRepo  ports.NotificationLogRepository
	timeout  time.Duration
}

func NewAsyncNotifier(whatsapp *WhatsAppService, logRepo ports.NotificationLogRepository) *AsyncNotifier {
	return &AsyncNotifier{
		whatsapp: whatsapp,
		logRepo:  logRepo,
		timeout:  60 * time.Second,
	}
}

// Dispatch fires the send-and-log in a new goroutine. participantID
// and notificationType wire up the log row; phone and message are the
// gateway payload.
func (n *AsyncNotifier) Dispatch(
	participantID uuid.UUID,
	notificationType valueobject.NotificationType,
	phone string,
	message string,
) {
	if phone == "" || message == "" {
		// Nothing to send. Still log so the host UI can show that a
		// participant was skipped (e.g. no WA number on file).
		n.logSync(participantID, notificationType, message, "", "no whatsapp number or empty message")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
		defer cancel()
		messageID, err := n.whatsapp.SendNotification(ctx, phone, message)
		if err != nil {
			n.logSync(participantID, notificationType, message, "", err.Error())
			return
		}
		n.logSync(participantID, notificationType, message, messageID, "")
	}()
}

// logSync writes the notification_logs row inline (no goroutine,
// short context) so the caller can confidently return once Dispatch
// returns and still have the queued row visible on the next read.
func (n *AsyncNotifier) logSync(
	participantID uuid.UUID,
	notificationType valueobject.NotificationType,
	message, messageID, errMsg string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := entity.NotificationStatusQueued
	var msgIDPtr *string
	var errMsgPtr *string
	if errMsg != "" {
		status = entity.NotificationStatusFailed
		errMsgPtr = &errMsg
	}
	if messageID != "" {
		msgIDPtr = &messageID
	}
	_ = n.logRepo.Create(ctx, &entity.NotificationLog{
		LogID:             uuid.New(),
		ParticipantID:     participantID,
		NotificationType:  notificationType,
		WhatsAppMessageID: msgIDPtr,
		MessageContent:    message,
		SentAt:            time.Now(),
		Status:            status,
		ErrorMessage:      errMsgPtr,
	})
}
