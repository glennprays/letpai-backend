package logger

import (
	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/log"
)

func ProviderLogger(config *config.Config) *log.Logger {
	logger, err := log.New(log.Config{
		Service: config.AppName,
		Env:     config.Env.String(),
		Level:   log.Level(config.LogLevel),
		Output:  log.OutputType(config.LogOutput),
	})
	if err != nil {
		panic(err)
	}

	return logger
}
