package infrastructure

import (
	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/router"
	"github.com/glennprays/log"
)

type App struct {
	Config *config.Config
	Logger *log.Logger
	Router *router.Router
}
