package payment

import (
	"context"
	"math"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/letpai-backend/internal/usecase/idresolve"
)

// PaymentPageResponse represents the public payment page data.
//
// BillItems is now filtered to only items that include this participant
// (or items with no explicit assignment, treated as "everyone").
// `your_share` on each item is the per-item amount THIS participant
// pays — useful so the participant sees exactly what they owe and why,
// not just the bill's total which gets divided across multiple people.
// PaymentPageBankAccount is a per-account row on the public page.
// Same shape as BankAccountItem in the session-detail use case but
// duplicated here to avoid an inter-usecase import.
type PaymentPageBankAccount struct {
	BankName      *string `json:"bank_name,omitempty"`
	AccountNumber *string `json:"account_number,omitempty"`
	AccountHolder *string `json:"account_holder,omitempty"`
}

// PaymentPageFeeBreakdown shows this participant's fee breakdown.
type PaymentPageFeeBreakdown struct {
	ItemsTotal         float64 `json:"items_total"`
	ServiceChargeShare float64 `json:"service_charge_share"`
	TaxShare           float64 `json:"tax_share"`
	Total              float64 `json:"total"`
}

// PaymentPageBillImage is a simplified bill image for the public page.
type PaymentPageBillImage struct {
	BillImageID  string  `json:"bill_image_id"`
	ImageURL     string  `json:"image_url"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
	FileName     string  `json:"file_name"`
}

type PaymentPageResponse struct {
	SessionID         string                    `json:"session_id"`
	SessionSlug       string                    `json:"session_slug"`
	ParticipantID     string                    `json:"participant_id"`
	ParticipantSlug   string                    `json:"participant_slug"`
	SessionName       string                    `json:"session_name"`
	ParticipantName   string                    `json:"participant_name"`
	TotalAmount       float64                   `json:"total_amount"`
	ShareAmount       float64                   `json:"share_amount"`
	Currency          string                    `json:"currency"`
	BillItems         []BillItemInfo            `json:"bill_items"`
	PaymentStatus     string                    `json:"payment_status"`
	PaymentProofURL   *string                   `json:"payment_proof_url,omitempty"`
	RejectionReason   *string                   `json:"rejection_reason,omitempty"`
	BankName          *string                   `json:"bank_name,omitempty"`
	BankAccountNumber *string                   `json:"bank_account_number,omitempty"`
	BankAccountHolder *string                   `json:"bank_account_holder,omitempty"`
	BankAccounts      []*PaymentPageBankAccount `json:"bank_accounts"`
	LinkExpiresAt     string                    `json:"link_expires_at"`
	IsExpired         bool                      `json:"is_expired"`
	BillImages        []*PaymentPageBillImage   `json:"bill_images,omitempty"`
	FeeBreakdown      *PaymentPageFeeBreakdown  `json:"fee_breakdown,omitempty"`
}

// BillItemInfo carries both the bill's full amount and THIS
// participant's slice of it. `your_share` is what the participant
// actually owes for the line; `shared_with` counts how many people
// the bill was divided across so the UI can render "Rp 50000 / 3
// people = Rp 16667 your share".
type BillItemInfo struct {
	BillItemID  string  `json:"bill_item_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    *string `json:"category,omitempty"`
	YourShare   float64 `json:"your_share"`
	SharedWith  int     `json:"shared_with"`
}

// GetPaymentPageUseCase handles getting public payment page data
type GetPaymentPageUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	billItemRepo    ports.BillItemRepository
	contactRepo     ports.ContactRepository
	bankAccountRepo ports.SessionBankAccountRepository
	billImageRepo   ports.BillImageRepository
	imageService    *service.ImageService
}

// NewGetPaymentPageUseCase creates a new get payment page use case
func NewGetPaymentPageUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	billItemRepo ports.BillItemRepository,
	contactRepo ports.ContactRepository,
	bankAccountRepo ports.SessionBankAccountRepository,
	billImageRepo ports.BillImageRepository,
	imageService *service.ImageService,
) *GetPaymentPageUseCase {
	return &GetPaymentPageUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		billItemRepo:    billItemRepo,
		contactRepo:     contactRepo,
		bankAccountRepo: bankAccountRepo,
		billImageRepo:   billImageRepo,
		imageService:    imageService,
	}
}

// Execute returns payment page data for a participant (public endpoint, no auth required)
func (uc *GetPaymentPageUseCase) Execute(ctx context.Context, participantID string) (*PaymentPageResponse, error) {
	// Get participant
	participant, err := idresolve.ResolveParticipant(ctx, uc.participantRepo, participantID)
	if err != nil {
		return nil, err
	}

	// Get session
	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), "")
	if err != nil {
		return nil, err
	}

	// Get participant name
	participantName := ""
	if participant.ContactID != nil {
		contact, err := uc.contactRepo.FindByID(ctx, participant.ContactID.String(), session.UserID.String())
		if err == nil {
			participantName = contact.Name
		}
	} else {
		participantName = participant.CustomName
	}

	// Get bill items for the session
	billItems, err := uc.billItemRepo.FindBySessionID(ctx, participant.SessionID.String())
	if err != nil {
		return nil, err
	}

	// All participants are needed so we can correctly compute "shared
	// with N people" for bills with no explicit assignment (legacy
	// "everyone" semantics).
	allParticipants, err := uc.participantRepo.FindBySessionID(ctx, participant.SessionID.String())
	if err != nil {
		return nil, err
	}
	totalParticipants := len(allParticipants)
	if totalParticipants == 0 {
		totalParticipants = 1 // defensive — shouldn't happen if this participant exists
	}

	pidStr := participant.ParticipantID.String()
	billItemInfos := make([]BillItemInfo, 0, len(billItems))

	// Fee-aware accumulators for this participant
	itemsTotal := 0.0
	serviceChargeBase := 0.0
	taxBase := 0.0
	hasFees := session.ServiceChargePercentage > 0 || session.TaxPercentage > 0

	for _, bill := range billItems {
		// Decide whether this bill applies to this participant and how
		// many people it's split across. An empty ParticipantIDs list
		// means "everyone in the session"; a populated list means only
		// the listed participants share the bill.
		isMine := false
		sharedWith := 0
		if len(bill.ParticipantIDs) == 0 {
			isMine = true
			sharedWith = totalParticipants
		} else {
			for _, pid := range bill.ParticipantIDs {
				if pid.String() == pidStr {
					isMine = true
				}
			}
			sharedWith = len(bill.ParticipantIDs)
		}
		if !isMine {
			continue
		}
		yourShare := math.Floor(bill.Amount / float64(sharedWith))
		billItemInfos = append(billItemInfos, BillItemInfo{
			BillItemID:  bill.BillItemID.String(),
			Description: bill.Description,
			Amount:      bill.Amount,
			Category:    bill.Category,
			YourShare:   yourShare,
			SharedWith:  sharedWith,
		})

		itemsTotal += yourShare
		if hasFees {
			if bill.IncludesServiceCharge {
				serviceChargeBase += yourShare
			}
			if bill.IncludesTax {
				taxBase += yourShare
			}
		}
	}

	// Compute fee breakdown if session has fees
	var feeBreakdown *PaymentPageFeeBreakdown
	if hasFees {
		scShare := math.Round(serviceChargeBase * session.ServiceChargePercentage / 100)
		taxShare := math.Round(taxBase * session.TaxPercentage / 100)
		feeBreakdown = &PaymentPageFeeBreakdown{
			ItemsTotal:         math.Round(itemsTotal),
			ServiceChargeShare: scShare,
			TaxShare:           taxShare,
			Total:              math.Round(itemsTotal) + scShare + taxShare,
		}
	}

	// Bill images (best-effort)
	var billImageItems []*PaymentPageBillImage
	if uc.billImageRepo != nil && uc.imageService != nil {
		billImages, imgErr := uc.billImageRepo.FindBySessionID(ctx, participant.SessionID.String())
		if imgErr == nil && len(billImages) > 0 {
			billImageItems = make([]*PaymentPageBillImage, 0, len(billImages))
			for _, img := range billImages {
				s3Key := "bill-images/" + img.ImageURL
				signedURL, _ := uc.imageService.GetPresignedGetURL(ctx, s3Key, 15*time.Minute)
				if signedURL == "" {
					signedURL = img.ImageURL // fallback to raw key
				}
				billImageItems = append(billImageItems, &PaymentPageBillImage{
					BillImageID:  img.BillImageID.String(),
					ImageURL:     signedURL,
					ThumbnailURL: img.ThumbnailURL,
					FileName:     img.FileName,
				})
			}
		}
	}

	// Bank accounts — same read-fallback pattern as GetSessionDetail.
	bankAccounts, _ := uc.bankAccountRepo.FindBySessionID(ctx, participant.SessionID.String())
	if len(bankAccounts) == 0 {
		if anyNonNil(session.BankName, session.BankAccountNumber, session.BankAccountHolder) {
			bankAccounts = []*entity.SessionBankAccount{{
				Ordinal:       0,
				BankName:      session.BankName,
				AccountNumber: session.BankAccountNumber,
				AccountHolder: session.BankAccountHolder,
			}}
		}
	}
	bankItems := make([]*PaymentPageBankAccount, 0, len(bankAccounts))
	for _, a := range bankAccounts {
		bankItems = append(bankItems, &PaymentPageBankAccount{
			BankName:      a.BankName,
			AccountNumber: a.AccountNumber,
			AccountHolder: a.AccountHolder,
		})
	}

	// Calculate link expiry (7 days from session creation or completion)
	linkExpiresAt := session.CreatedAt.Add(7 * 24 * time.Hour)
	isExpired := time.Now().After(linkExpiresAt)

	return &PaymentPageResponse{
		SessionID:         session.SessionID.String(),
		SessionSlug:       session.PublicSlug,
		ParticipantID:     participant.ParticipantID.String(),
		ParticipantSlug:   participant.PublicSlug,
		SessionName:       session.Title,
		ParticipantName:   participantName,
		TotalAmount:       session.TotalAmount,
		ShareAmount:       participant.ShareAmount,
		Currency:          session.Currency,
		BillItems:         billItemInfos,
		PaymentStatus:     participant.PaymentStatus.String(),
		PaymentProofURL:   participant.PaymentProofURL,
		RejectionReason:   participant.RejectionReason,
		BankName:          session.BankName,
		BankAccountNumber: session.BankAccountNumber,
		BankAccountHolder: session.BankAccountHolder,
		BankAccounts:      bankItems,
		LinkExpiresAt:     linkExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		IsExpired:         isExpired,
		BillImages:        billImageItems,
		FeeBreakdown:      feeBreakdown,
	}, nil
}

func anyNonNil(name, number, holder *string) bool {
	for _, p := range []*string{name, number, holder} {
		if p != nil && *p != "" {
			return true
		}
	}
	return false
}
