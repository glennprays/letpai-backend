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
		{"sent advances from pending", NotificationStatusPending, "message.sent", NotificationStatusSent, true},
		{"duplicate sent is no-op", NotificationStatusSent, "message.sent", "", false},
		{"late queued event never downgrades sent", NotificationStatusSent, "message.queued", "", false},
		{"queued event on pending is a no-op (carries no new info)", NotificationStatusPending, "message.queued", "", false},
		{"failed applies after sent", NotificationStatusSent, "message.failed", NotificationStatusFailed, true},
		{"failed applies from sending", NotificationStatusSending, "message.failed", NotificationStatusFailed, true},
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
