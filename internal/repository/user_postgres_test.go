package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestPostgresUserRepositoryUpdatePersistsPasswordHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer db.Close()

	user := &entity.User{
		UserID:         uuid.New(),
		WhatsAppNumber: "6281234567890",
		PasswordHash:   "$2a$04$updated-password-hash",
		FullName:       "Raka",
		AvatarURL:      "https://example.com/raka.jpg",
		IsVerified:     true,
		UpdatedAt:      time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC),
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE users
		SET password_hash = $2, full_name = $3, avatar_url = $4, is_verified = $5, updated_at = $6
		WHERE user_id = $1 AND deleted_at IS NULL
	`)).
		WithArgs(user.UserID, user.PasswordHash, user.FullName, user.AvatarURL, user.IsVerified, user.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := &PostgresUserRepository{db: sqlx.NewDb(db, "sqlmock")}
	if err := repo.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
