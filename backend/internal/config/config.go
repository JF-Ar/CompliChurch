package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	DatabaseURL       string `envconfig:"DATABASE_URL" required:"true"`
	JWTPrivateKeyPath string `envconfig:"JWT_PRIVATE_KEY_PATH" required:"true"`
	JWTPublicKeyPath  string `envconfig:"JWT_PUBLIC_KEY_PATH" required:"true"`
	JWTAccessTTL      int    `envconfig:"JWT_ACCESS_TTL_MINUTES" default:"15"`
	JWTRefreshTTLDays int    `envconfig:"JWT_REFRESH_TTL_DAYS" default:"30"`
	ResendAPIKey      string `envconfig:"RESEND_API_KEY"`
	EmailFrom         string `envconfig:"EMAIL_FROM" default:"noreply@igreaorganizada.com.br"`
	R2AccountID       string `envconfig:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `envconfig:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `envconfig:"R2_SECRET_ACCESS_KEY"`
	R2BucketName      string `envconfig:"R2_BUCKET_NAME"`
	R2PublicURL       string `envconfig:"R2_PUBLIC_URL"`
	Port              string `envconfig:"PORT" default:"8080"`
	Env               string `envconfig:"ENV" default:"development"`
	LogLevel          string `envconfig:"LOG_LEVEL" default:"info"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
