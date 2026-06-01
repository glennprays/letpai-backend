package admintemplates

import (
	"context"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// TestSendRequest is the body for POST /admin/templates/:key/test-send.
type TestSendRequest struct {
	PhoneNumber string            `json:"phone_number" validate:"required,min=8,max=15"`
	Variables   map[string]string `json:"variables,omitempty"`
}

// TestSendResponse contains the rendered preview and send status.
type TestSendResponse struct {
	Rendered  string `json:"rendered"`
	MessageID string `json:"message_id,omitempty"`
	Status    string `json:"status"` // "sent" or "failed"
}

// sampleDefaults maps variable names to friendly placeholder values
// used when the admin doesn't supply custom values.
var sampleDefaults = map[string]string{
	"ParticipantName": "John",
	"SessionName":     "Dinner at Padella",
	"MakerName":       "Sarah",
	"Total":           "350.000",
	"Share":           "87.500",
	"URL":             "https://letpai.app/payment/abc123",
	"Code":            "123456",
	"ExpiresMinutes":  "5",
}

// TestSendUseCase renders a template with sample (or custom) data and
// sends it to an admin-chosen phone number via the WhatsApp gateway.
// This lets admins preview exactly what their templates look like on
// a real device before broadcasting to participants.
type TestSendUseCase struct {
	repo      ports.MessageTemplateRepository
	renderer  *service.TemplateRenderer
	whatsapp  *service.WhatsAppService
}

func NewTestSendUseCase(
	repo ports.MessageTemplateRepository,
	renderer *service.TemplateRenderer,
	whatsapp *service.WhatsAppService,
) *TestSendUseCase {
	return &TestSendUseCase{
		repo:     repo,
		renderer: renderer,
		whatsapp: whatsapp,
	}
}

func (uc *TestSendUseCase) Execute(ctx context.Context, key string, req *TestSendRequest) (*TestSendResponse, error) {
	// Verify the template exists.
	tmpl, err := uc.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("template %q not found", key))
	}

	// Build variable map: custom values override sample defaults.
	vars := make(map[string]string)
	for _, v := range tmpl.Variables {
		if custom, ok := req.Variables[v]; ok {
			vars[v] = custom
		} else if def, ok := sampleDefaults[v]; ok {
			vars[v] = def
		} else {
			vars[v] = v // Unknown var: use its name as placeholder
		}
	}

	// Render the template. The renderer expects a struct, but our vars
	// are dynamic — use a map[string]string which Go text/template
	// handles natively (accessed as {{.Key}}).
	rendered, err := uc.renderer.Render(ctx, key, vars)
	if err != nil || rendered == "" {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("template render failed: %v", err))
	}

	resp := &TestSendResponse{
		Rendered: rendered,
		Status:   "failed",
	}

	// Attempt to send via WhatsApp gateway. We still return the
	// rendered preview even on send failure so the admin can see it.
	msgID, sendErr := uc.whatsapp.SendNotification(ctx, req.PhoneNumber, rendered)
	if sendErr == nil {
		resp.MessageID = msgID
		resp.Status = "sent"
	}

	return resp, nil
}
