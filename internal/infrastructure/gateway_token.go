package infrastructure

import (
	"context"
	"encoding/base64"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/repository"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/log"
	"github.com/jmoiron/sqlx"
)

// RestoreGatewayToken rehydrates the WhatsApp service's gateway JWT at
// boot. Precedence:
//
//  1. restored-from-db: decrypt whatsapp_configs.gateway_token with
//     WHATSAPP_TOKEN_ENC_KEY and SetToken it. This is the normal path —
//     the token was minted by RegisterGatewayUseCase via the admin UI.
//  2. env-bootstrap: fall back to the optional WHATSAPP_GATEWAY_TOKEN
//     env (pre-minted JWT escape hatch for the first admin bootstrap,
//     when no row exists yet).
//  3. none-registered-yet: no token anywhere; the admin registers via
//     /admin/whatsapp. Send paths will 401 until then — expected.
//
// A decrypt failure never blocks boot: it logs loudly and continues
// tokenless. Recovery is re-registering the phone from the admin UI
// once the correct key is in place.
func RestoreGatewayToken(cfg *config.Config, db *sqlx.DB, svc *service.WhatsAppService, logger *log.Logger) {
	bootID := "gateway-token"
	repo := repository.NewPostgresWhatsAppConfigRepository(db)
	row, err := repo.Get(context.Background())
	if err != nil {
		logger.Error(bootID, "gateway token restore: failed to read whatsapp_configs", map[string]any{
			"error": err.Error(),
		})
		return
	}

	if row != nil && row.GatewayToken != "" {
		if cfg.WhatsAppTokenEncKey == "" {
			logger.Error(bootID, "gateway token restore: stored token exists but WHATSAPP_TOKEN_ENC_KEY is not set; continuing tokenless (re-register to recover)", nil)
			return
		}
		key, decErr := base64.StdEncoding.DecodeString(cfg.WhatsAppTokenEncKey)
		if decErr != nil {
			logger.Error(bootID, "gateway token restore: WHATSAPP_TOKEN_ENC_KEY is not valid base64; continuing tokenless", map[string]any{
				"error": decErr.Error(),
			})
			return
		}
		box, boxErr := service.NewSecretBox(key)
		if boxErr != nil {
			logger.Error(bootID, "gateway token restore: bad encryption key; continuing tokenless", map[string]any{
				"error": boxErr.Error(),
			})
			return
		}
		token, openErr := box.Open(row.GatewayToken)
		if openErr != nil {
			logger.Error(bootID, "gateway token restore: decrypt failed (wrong or rotated key?); continuing tokenless (re-register to recover)", map[string]any{
				"error": openErr.Error(),
			})
			return
		}
		svc.SetToken(token)
		logger.Info(bootID, "gateway token restored-from-db", map[string]any{
			"phone_number": row.PhoneNumber,
		})
		return
	}

	if cfg.WhatsAppGatewayToken != "" {
		svc.SetToken(cfg.WhatsAppGatewayToken)
		logger.Info(bootID, "gateway token env-bootstrap", nil)
		return
	}

	logger.Info(bootID, "gateway token none-registered-yet; waiting for admin registration at /admin/whatsapp", nil)
}
