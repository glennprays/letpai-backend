package admin

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
	"github.com/google/uuid"
)

// RegisterGatewayRequest carries the admin-entered WhatsApp number this
// workspace sends from.
type RegisterGatewayRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,phone"`
}

// RegisterGatewayResponse reports the registered number and when the
// minted JWT expires (gateway JWTs last ~1 year).
type RegisterGatewayResponse struct {
	PhoneNumber    string `json:"phone_number"`
	TokenExpiresAt string `json:"token_expires_at,omitempty"`
}

// RegisterGatewayUseCase mints a gateway JWT for the admin-entered
// phone number and persists it encrypted (AES-256-GCM) in
// whatsapp_configs.gateway_token — replacing the broken assumption
// that WHATSAPP_API_KEY holds a pre-minted JWT (it holds the /register
// secret).
type RegisterGatewayUseCase struct {
	whatsappSvc *service.WhatsAppService
	configRepo  ports.WhatsAppConfigRepository
	cfg         *config.Config
}

func NewRegisterGatewayUseCase(
	whatsappSvc *service.WhatsAppService,
	configRepo ports.WhatsAppConfigRepository,
	cfg *config.Config,
) *RegisterGatewayUseCase {
	return &RegisterGatewayUseCase{
		whatsappSvc: whatsappSvc,
		configRepo:  configRepo,
		cfg:         cfg,
	}
}

// Execute registers the phone with the WAGA gateway and stores the
// sealed token. Steps:
//
//  1. Normalize the phone to bare digits (strip '+', spaces, dashes).
//  2. If a config row exists for a different non-empty phone, best-effort
//     logout the old device session (WAGA has one session per secret).
//  3. POST /register with WHATSAPP_API_KEY as the secret.
//  4. Seal the JWT and upsert the row (repo COALESCEs on conflict).
//  5. Load the fresh token onto the shared client.
func (uc *RegisterGatewayUseCase) Execute(ctx context.Context, req *RegisterGatewayRequest) (*RegisterGatewayResponse, error) {
	phone := normalizeGatewayPhone(req.PhoneNumber)

	if uc.cfg.WhatsAppAPIKey == "" {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("WHATSAPP_API_KEY is not configured; cannot register with the gateway"))
	}

	existing, err := uc.configRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.PhoneNumber != "" && existing.PhoneNumber != phone {
		// Old session unpair is best-effort: a failure here must not
		// block the new registration (the gateway will error with its
		// own message if it actually conflicts).
		if logoutErr := uc.whatsappSvc.Logout(ctx); logoutErr != nil {
			_ = logoutErr // logged via the gateway error surface only
		}
	}

	resp, err := uc.whatsappSvc.Register(ctx, phone, uc.cfg.WhatsAppAPIKey)
	if err != nil {
		// Surface auth failures with the gateway's own message so the
		// admin sees "wrong secret" rather than a generic 500.
		if errors.Is(err, waga.ErrUnauthorized) || errors.Is(err, waga.ErrForbidden) {
			return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("gateway rejected registration: %w", err))
		}
		return nil, err
	}

	box, err := secretBoxFromConfig(uc.cfg.WhatsAppTokenEncKey)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	sealed, err := box.Seal(resp.Token)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to encrypt gateway token: %w", err))
	}

	now := time.Now()
	cfgRow := &entity.WhatsAppConfig{
		ConfigID:         existingConfigID(existing),
		PhoneNumber:      phone,
		GatewayToken:     sealed,
		ConnectionStatus: entity.ConnectionStatusQRPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := uc.configRepo.CreateOrUpdate(ctx, cfgRow); err != nil {
		return nil, err
	}
	// CreateOrUpdate never writes gateway_token (its INSERT column list
	// omits it and the upsert COALESCEs), so persist the sealed JWT via
	// the dedicated update. Same transactionality caveat as before: the
	// two statements are separate, a crash between them leaves a row
	// with a stale/absent token — the admin simply re-registers.
	if err := uc.configRepo.UpdateToken(ctx, phone, sealed); err != nil {
		return nil, err
	}

	uc.whatsappSvc.SetToken(resp.Token)

	out := &RegisterGatewayResponse{PhoneNumber: phone}
	if exp, ok := service.ParseJWTExpiry(resp.Token); ok {
		out.TokenExpiresAt = exp.Format(time.RFC3339)
	}
	return out, nil
}

// normalizeGatewayPhone strips separators, keeping bare digits so the
// stored value matches the 628… MSISDN convention.
func normalizeGatewayPhone(in string) string {
	var b strings.Builder
	for _, r := range in {
		switch r {
		case '+', ' ', '-', '(', ')':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// secretBoxFromConfig builds the AES box from the base64-encoded env
// key.
func secretBoxFromConfig(encKeyB64 string) (*service.SecretBox, error) {
	if encKeyB64 == "" {
		return nil, errors.New("WHATSAPP_TOKEN_ENC_KEY is not set; cannot store the gateway token encrypted")
	}
	key, err := base64.StdEncoding.DecodeString(encKeyB64)
	if err != nil {
		return nil, fmt.Errorf("WHATSAPP_TOKEN_ENC_KEY is not valid base64: %w", err)
	}
	return service.NewSecretBox(key)
}

// existingConfigID keeps the singleton row's ID when present so the
// upsert stays a true update. New rows mint a fresh UUID.
func existingConfigID(existing *entity.WhatsAppConfig) uuid.UUID {
	if existing != nil {
		return existing.ConfigID
	}
	return uuid.New()
}
