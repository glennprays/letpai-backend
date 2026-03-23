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

	JWTSecret           string `mapstructure:"JWT_SECRET" default:"your-secret-key-change-in-production"`
	JWTExpiryHours      int    `mapstructure:"JWT_EXPIRY_HOURS" default:"168"` // 7 days
	OTPExpiryMinutes    int    `mapstructure:"OTP_EXPIRY_MINUTES" default:"5"`

	WhatsAppGatewayURL    string `mapstructure:"WHATSAPP_GATEWAY_URL" default:"http://localhost:8080"`
	WhatsAppAPIKey         string `mapstructure:"WHATSAPP_API_KEY" default:""`
	WhatsAppWebhookSecret string `mapstructure:"WHATSAPP_WEBHOOK_SECRET" default:""`

	RedisHost     string `mapstructure:"REDIS_HOST" default:"localhost"`
	RedisPort     int    `mapstructure:"REDIS_PORT" default:"6379"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD" default:""`
	RedisDB       int    `mapstructure:"REDIS_DB" default:"0"`

	// S3/AWS Configuration for image upload
	AWSEndpoint      string `mapstructure:"AWS_ENDPOINT" default:""` // Optional: for S3-compatible services
	AWSRegion        string `mapstructure:"AWS_REGION" default:"us-east-1"`
	AWSAccessID      string `mapstructure:"AWS_ACCESS_KEY_ID"`
	AWSSecret        string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	S3BucketName     string `mapstructure:"S3_BUCKET_NAME" default:"letpai-uploads"`
	CDNURL           string `mapstructure:"CDN_URL" default:""`
	EnableWebP       bool   `mapstructure:"ENABLE_WEBP" default:"true"`
	WebPQuality      int    `mapstructure:"WEBP_QUALITY" default:"85"`
	MaxImageSizeMB   int    `mapstructure:"MAX_IMAGE_SIZE_MB" default:"5"`
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
