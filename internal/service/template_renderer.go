package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// TemplateRenderer loads admin-managed message templates from the
// MessageTemplateRepository and renders them with Go text/template
// against an arbitrary struct or map. Lookups are by key (e.g.
// "session_notification"); the body is interpreted as text/template
// syntax (`{{.ParticipantName}}` etc).
//
// We deliberately don't cache here — admin edits should take effect
// immediately. If template traffic ever grows hot, swap in a
// short-TTL cache keyed by (key, updated_at).
type TemplateRenderer struct {
	repo ports.MessageTemplateRepository
}

func NewTemplateRenderer(repo ports.MessageTemplateRepository) *TemplateRenderer {
	return &TemplateRenderer{repo: repo}
}

// Render looks up `key`, parses the body, and executes it with `data`.
// On any error it returns "" so the caller can substitute a hardcoded
// fallback — we never want a missing template to brick a notification.
func (r *TemplateRenderer) Render(ctx context.Context, key string, data any) (string, error) {
	t, err := r.repo.FindByKey(ctx, key)
	if err != nil {
		return "", fmt.Errorf("template %q lookup failed: %w", key, err)
	}
	tmpl, err := template.New(key).Parse(t.Body)
	if err != nil {
		return "", fmt.Errorf("template %q parse failed: %w", key, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template %q execute failed: %w", key, err)
	}
	return buf.String(), nil
}
