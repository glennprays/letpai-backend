package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func TestPostgresBillItemRepositoryCreatePersistsNameFromDescription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer db.Close()

	item := entity.NewBillItem(uuid.New(), "Nasi", 5000, nil, nil)
	repo := NewPostgresBillItemRepository(sqlx.NewDb(db, "sqlmock"))

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO bill_items (bill_item_id, session_id, name, description, amount, category, includes_service_charge, includes_tax, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`)).
		WithArgs(
			item.BillItemID,
			item.SessionID,
			"Nasi",
			"Nasi",
			5000.0,
			nil,
			false,
			false,
			item.CreatedAt,
			item.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
