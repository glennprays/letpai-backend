package admintemplates

import (
	"context"
	"errors"
	"strings"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// TemplateItem is the wire-shape of a single message template surfaced
// to the admin UI. Mirrors entity.MessageTemplate but uses string IDs
// so the FE doesn't have to deal with uuid.UUID.
type TemplateItem struct {
	TemplateID  string   `json:"template_id"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Body        string   `json:"body"`
	Variables   []string `json:"variables"`
	UpdatedAt   string   `json:"updated_at"`
}

// ListTemplatesUseCase fetches every message template for the admin
// management page.
type ListTemplatesUseCase struct {
	repo ports.MessageTemplateRepository
}

func NewListTemplatesUseCase(repo ports.MessageTemplateRepository) *ListTemplatesUseCase {
	return &ListTemplatesUseCase{repo: repo}
}

func (uc *ListTemplatesUseCase) Execute(ctx context.Context) ([]*TemplateItem, error) {
	tmpls, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*TemplateItem, 0, len(tmpls))
	for _, t := range tmpls {
		out = append(out, toItem(t))
	}
	return out, nil
}

// UpdateTemplateRequest is the body for PUT /admin/templates/:key.
// `key` itself is path-scoped and not editable.
type UpdateTemplateRequest struct {
	Name        *string  `json:"name" validate:"omitempty,min=1,max=120"`
	Description *string  `json:"description" validate:"omitempty,max=400"`
	Body        *string  `json:"body" validate:"omitempty,min=1,max=4000"`
	Variables   []string `json:"variables" validate:"omitempty,dive,min=1,max=40"`
}

// UpdateTemplateUseCase updates a single template by key. Validates
// that the new body parses as Go text/template before persisting so a
// broken template never makes it to the runtime renderer.
type UpdateTemplateUseCase struct {
	repo ports.MessageTemplateRepository
}

func NewUpdateTemplateUseCase(repo ports.MessageTemplateRepository) *UpdateTemplateUseCase {
	return &UpdateTemplateUseCase{repo: repo}
}

func (uc *UpdateTemplateUseCase) Execute(ctx context.Context, key string, req *UpdateTemplateRequest) (*TemplateItem, error) {
	existing, err := uc.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, domain.NewError(domain.ErrBadRequest, errors.New("name cannot be empty"))
		}
		existing.Name = trimmed
	}
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if desc == "" {
			existing.Description = nil
		} else {
			existing.Description = &desc
		}
	}
	if req.Body != nil {
		// Parse-check so a malformed template body is rejected at
		// edit time instead of bricking the renderer at send time.
		if err := validateTemplateBody(*req.Body); err != nil {
			return nil, domain.NewError(domain.ErrBadRequest, err)
		}
		existing.Body = *req.Body
	}
	if req.Variables != nil {
		existing.Variables = req.Variables
	}
	if err := uc.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return toItem(existing), nil
}

func toItem(t *entity.MessageTemplate) *TemplateItem {
	return &TemplateItem{
		TemplateID:  t.TemplateID.String(),
		Key:         t.Key,
		Name:        t.Name,
		Description: t.Description,
		Body:        t.Body,
		Variables:   t.Variables,
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
