package service

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	glog "github.com/glennprays/log"
	"github.com/google/uuid"
)

// NotificationWorker drains the notification_logs outbox: it periodically
// claims due rows, sends them through the WhatsApp gateway, and records the
// result with bounded exponential-backoff retries. Replacing the old
// fire-and-forget goroutine, this makes delivery durable (survives restarts),
// retryable (transient gateway failures don't permanently fail a send), and
// observable (every row's status reflects reality).
type NotificationWorker struct {
	whatsapp    *WhatsAppService
	logRepo     ports.NotificationLogRepository
	logger      *glog.Logger
	workerID    string
	interval    time.Duration
	batchSize   int
	sendTimeout time.Duration
	stuckAfter  time.Duration
}

// NewNotificationWorker constructs a worker with production-sensible defaults.
func NewNotificationWorker(whatsapp *WhatsAppService, logRepo ports.NotificationLogRepository, logger *glog.Logger) *NotificationWorker {
	return &NotificationWorker{
		whatsapp:    whatsapp,
		logRepo:     logRepo,
		logger:      logger,
		workerID:    uuid.New().String(),
		interval:    5 * time.Second,
		batchSize:   10,
		sendTimeout: 60 * time.Second,
		stuckAfter:  5 * time.Minute,
	}
}

// Run drives the worker loop until ctx is cancelled. Intended to be launched in
// its own goroutine from main and stopped on graceful shutdown.
func (w *NotificationWorker) Run(ctx context.Context) {
	w.logger.Info(w.workerID, "notification worker started", map[string]any{
		"interval":   w.interval.String(),
		"batch_size": w.batchSize,
	})
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.Info(w.workerID, "notification worker stopped", nil)
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick performs one drain cycle: requeue rows orphaned by a crashed worker,
// claim a batch of due rows, and process each.
func (w *NotificationWorker) tick(ctx context.Context) {
	if _, err := w.logRepo.RequeueStuck(ctx, time.Now().Add(-w.stuckAfter)); err != nil {
		w.logger.Error(w.workerID, "requeue stuck notifications failed", map[string]any{"error": err.Error()})
	}

	logs, err := w.logRepo.ClaimPending(ctx, w.workerID, w.batchSize)
	if err != nil {
		w.logger.Error(w.workerID, "claim pending notifications failed", map[string]any{"error": err.Error()})
		return
	}
	for _, l := range logs {
		w.process(ctx, l)
	}
}

// process sends a single claimed row and records the outcome. A panic in the
// send path is recovered so one bad row can't take down the worker; the row is
// left in 'sending' and will be requeued by the stuck-reaper.
func (w *NotificationWorker) process(ctx context.Context, l *entity.NotificationLog) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error(w.workerID, "panic while sending notification", map[string]any{
				"log_id": l.LogID.String(),
				"panic":  fmt.Sprintf("%v", r),
			})
		}
	}()

	phone := ""
	if l.Phone != nil {
		phone = *l.Phone
	}
	if phone == "" {
		// Nothing to send to; terminal so the host UI shows it as failed
		// rather than spinning forever.
		_ = w.logRepo.MarkDead(ctx, l.LogID.String(), l.Attempts, "no whatsapp number on file")
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, w.sendTimeout)
	defer cancel()

	messageID, err := w.whatsapp.SendNotification(sendCtx, phone, l.MessageContent)
	if err != nil {
		attempts := l.Attempts + 1
		if attempts >= l.MaxAttempts {
			_ = w.logRepo.MarkDead(ctx, l.LogID.String(), attempts, err.Error())
			w.logger.Error(w.workerID, "notification dead-lettered", map[string]any{
				"log_id":   l.LogID.String(),
				"attempts": attempts,
				"error":    err.Error(),
			})
			return
		}
		next := time.Now().Add(NextBackoff(attempts))
		_ = w.logRepo.MarkRetry(ctx, l.LogID.String(), attempts, next, err.Error())
		return
	}

	if err := w.logRepo.MarkSent(ctx, l.LogID.String(), messageID); err != nil {
		w.logger.Error(w.workerID, "mark notification sent failed", map[string]any{
			"log_id": l.LogID.String(),
			"error":  err.Error(),
		})
	}
}
