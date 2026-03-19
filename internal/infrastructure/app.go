package infrastructure

import (
	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/router"
	"github.com/glennprays/log"
	"github.com/jmoiron/sqlx"
)

type App struct {
	Config *config.Config
	Logger *log.Logger
	DB     *sqlx.DB
	Router *router.Router
}
