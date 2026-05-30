// Package idresolve bridges the slug-or-UUID URL boundary. URLs now
// carry the short PublicSlug; older messages in the wild still use
// the 36-char UUID. Use cases call ResolveSession / ResolveParticipant
// instead of FindByID directly so we can keep both shapes working
// through the compat window.
package idresolve

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/pkg/slug"
)

// ResolveSession returns the session matching either a public_slug
// (preferred) or a UUID (legacy). userID is forwarded to the repo
// methods; pass "" for public lookups.
func ResolveSession(ctx context.Context, repo ports.SessionRepository, id, userID string) (*entity.Session, error) {
	if slug.Is(id) {
		return repo.FindBySlug(ctx, id, userID)
	}
	return repo.FindByID(ctx, id, userID)
}

// ResolveParticipant returns the participant matching either a slug
// or a UUID. No userID — the ACL is enforced upstream via the
// session lookup.
func ResolveParticipant(ctx context.Context, repo ports.ParticipantRepository, id string) (*entity.SessionParticipant, error) {
	if slug.Is(id) {
		return repo.FindBySlug(ctx, id)
	}
	return repo.FindByID(ctx, id)
}
