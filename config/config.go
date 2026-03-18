package config

import (
	"os"
	"reflect"
	"strings"

	"github.com/creasty/defaults"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Env     Environment `mapstructure:"ENV" default:"development"`
	AppName string      `mapstructure:"APP_NAME" default:"golang-clean-architecture"`
	AppPort int         `mapstructure:"APP_PORT" default:"3000"`

	LogLevel  string `mapstructure:"LOG_LEVEL" default:"debug"`
	LogOutput string `mapstructure:"LOG_OUTPUT" default:"stdout"`

	DBHost     string `mapstructure:"DB_HOST" default:"localhost"`
	DBPort     int    `mapstructure:"DB_PORT" default:"5432"`
	DBUser     string `mapstructure:"DB_USER" default:"postgres"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
}

type Environment string

const (
	DEV     Environment = "development"
	STAGING Environment = "staging"
	PROD    Environment = "production"
)

func (e Environment) String() string {
	return string(e)
}

func Load() (*Config, error) {
	// Create config instance
	cfg := &Config{}

	// Apply defaults from struct tags
	if err := defaults.Set(cfg); err != nil {
		return nil, err
	}

	envStr := strings.ToLower(os.Getenv("ENV"))
	env := Environment(envStr)
	if env == "" {
		env = DEV
	}

	// Load .env file
	if env == DEV {
		_ = godotenv.Load(".env")
	}

	// Configure Viper to read from environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Auto-bind each struct field by key
	t := reflect.TypeOf(cfg).Elem()
	for i := range t.NumField() {
		field := t.Field(i)
		key := field.Tag.Get("mapstructure")
		if key != "" {
			err := viper.BindEnv(key)
			if err != nil {
				return nil, err
			}
		}
	}

	// Unmarshal environment variables into config
	// This will override defaults with actual env values
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
