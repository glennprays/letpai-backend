package session

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// MaxBankAccountsPerSession is the FE-side and server-side cap. 5 is
// the design call from the rollout plan; bump if hosts ask. The check
// is enforced here (use case) so the repo doesn't need to know.
const MaxBankAccountsPerSession = 5

// BankAccountInput is one row from the FE form. All three fields are
// optional individually but at least one must be non-blank
// (mirrors the CHECK constraint in migration 000023).
type BankAccountInput struct {
	BankName      *string `json:"bank_name,omitempty"      validate:"omitempty,max=80"`
	AccountNumber *string `json:"account_number,omitempty" validate:"omitempty,max=40"`
	AccountHolder *string `json:"account_holder,omitempty" validate:"omitempty,max=80"`
}

// ReplaceBankAccountsRequest carries the full list. Empty array
// clears every row.
type ReplaceBankAccountsRequest struct {
	Accounts []BankAccountInput `json:"accounts" validate:"required,max=5,dive"`
}

// BankAccountItem is the wire shape returned to the FE.
type BankAccountItem struct {
	AccountID     string  `json:"account_id"`
	Ordinal       int     `json:"ordinal"`
	BankName      *string `json:"bank_name,omitempty"`
	AccountNumber *string `json:"account_number,omitempty"`
	AccountHolder *string `json:"account_holder,omitempty"`
}

// ReplaceBankAccountsResponse echoes the freshly-persisted list.
type ReplaceBankAccountsResponse struct {
	Accounts []*BankAccountItem `json:"accounts"`
}

// ReplaceBankAccountsUseCase swaps the entire bank-accounts list for
// a session in one transaction (BulkReplace) and dual-writes into the
// legacy sessions.bank_* columns during the compat window.
//
// Dual-write rules (per the rollout plan):
//   - 1 account → mirror into sessions.bank_*
//   - 2+ accounts → NULL the legacy columns (legacy readers see
//     "no single account configured" rather than a stale first-of-many)
//   - 0 accounts → NULL the legacy columns
type ReplaceBankAccountsUseCase struct {
	sessionRepo     ports.SessionRepository
	bankAccountRepo ports.SessionBankAccountRepository
}

func NewReplaceBankAccountsUseCase(
	sessionRepo ports.SessionRepository,
	bankAccountRepo ports.SessionBankAccountRepository,
) *ReplaceBankAccountsUseCase {
	return &ReplaceBankAccountsUseCase{
		sessionRepo:     sessionRepo,
		bankAccountRepo: bankAccountRepo,
	}
}

func (uc *ReplaceBankAccountsUseCase) Execute(ctx context.Context, userID, sessionID string, req *ReplaceBankAccountsRequest) (*ReplaceBankAccountsResponse, error) {
	if req == nil {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("request is required"))
	}
	if len(req.Accounts) > MaxBankAccountsPerSession {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("too many bank accounts (max 5)"))
	}

	// Auth: must own the session.
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Convert + drop fully-blank rows. The repo also normalises but
	// doing it here lets the count below match what actually persists.
	live := make([]*entity.SessionBankAccount, 0, len(req.Accounts))
	for _, a := range req.Accounts {
		ent := &entity.SessionBankAccount{
			BankName:      a.BankName,
			AccountNumber: a.AccountNumber,
			AccountHolder: a.AccountHolder,
		}
		ent.Normalize()
		if ent.IsEmpty() {
			continue
		}
		live = append(live, ent)
	}

	if err := uc.bankAccountRepo.BulkReplace(ctx, sessionID, live); err != nil {
		return nil, err
	}

	// Dual-write into the legacy single-row columns. Reads always
	// load from the new table; only fall back to legacy if the new
	// table is empty (pre-migration data that somehow slipped the
	// backfill). When the host configures 2+ accounts we explicitly
	// NULL the legacy fields so old clients can't render a stale
	// first-of-many.
	switch len(live) {
	case 0:
		session.SetBankInfo(strPtr(""), strPtr(""), strPtr(""))
	case 1:
		a := live[0]
		session.SetBankInfo(ptrOrEmpty(a.BankName), ptrOrEmpty(a.AccountNumber), ptrOrEmpty(a.AccountHolder))
	default:
		session.SetBankInfo(strPtr(""), strPtr(""), strPtr(""))
	}
	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		// Best-effort: the new-table write already succeeded; reads
		// will be correct. Just log via error bubbling.
		return nil, err
	}

	// Echo the fresh list back. Re-read so the response contains the
	// generated account_id values + ordinals.
	fresh, err := uc.bankAccountRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	items := make([]*BankAccountItem, 0, len(fresh))
	for _, a := range fresh {
		items = append(items, &BankAccountItem{
			AccountID:     a.AccountID.String(),
			Ordinal:       a.Ordinal,
			BankName:      a.BankName,
			AccountNumber: a.AccountNumber,
			AccountHolder: a.AccountHolder,
		})
	}
	return &ReplaceBankAccountsResponse{Accounts: items}, nil
}

func strPtr(s string) *string { return &s }

func ptrOrEmpty(p *string) *string {
	if p == nil {
		s := ""
		return &s
	}
	return p
}
