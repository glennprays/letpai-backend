//go:build wireinject
// +build wireinject

package infrastructure

import (
	"github.com/google/wire"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/handler"
	"github.com/glennprays/letpai-backend/internal/router"
	"github.com/glennprays/letpai-backend/pkg/logger"
)

var CoreSet = wire.NewSet(
	config.Load,
	logger.ProviderLogger,
)

var ApiSet = wire.NewSet(
	handler.NewHealthHandler,
	router.NewRouter,
)

func InitializeApp() (*App, error) {
	wire.Build(
		CoreSet,
		ApiSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
