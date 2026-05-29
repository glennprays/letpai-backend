package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// MessageTemplateRepository is the persistence port for admin-managed
// message templates. The renderer reads via FindByKey at send time;
// admin endpoints use the rest of the CRUD surface.
type MessageTemplateRepository interface {
	FindByKey(ctx context.Context, key string) (*entity.MessageTemplate, error)
	FindAll(ctx context.Context) ([]*entity.MessageTemplate, error)
	Update(ctx context.Context, t *entity.MessageTemplate) error
}
