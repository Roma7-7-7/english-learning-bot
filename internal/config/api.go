package config

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type (
	DB struct {
		Path string `required:"false" default:"./data/english_learning.db?cache=shared&mode=rwc"`
	}

	CORS struct {
		AllowOrigins []string `envconfig:"ALLOW_ORIGINS" required:"true"`
	}

	JWT struct {
		Issuer   string   `envconfig:"ISSUER" default:"english-learning-api"`
		Audience []string `envconfig:"AUDIENCE" required:"true"`
		Secret   string   `envconfig:"SECRET" required:"false"`
	}

	Cookie struct {
		Path            string        `envconfig:"CPATH" default:"/"` // not using PATH here because it may conflict with os.Path
		Domain          string        `envconfig:"DOMAIN" required:"true"`
		AuthExpiresIn   time.Duration `envconfig:"AUTH_EXPIRES_IN" default:"24h"`
		AccessExpiresIn time.Duration `envconfig:"ACCESS_EXPIRES_IN" default:"720h"`
	}

	HTTP struct {
		ProcessTimeout time.Duration `envconfig:"PROCESS_TIMEOUT" default:"10s"`
		RateLimit      float64       `envconfig:"RATE_LIMIT" default:"25"`
		CORS           CORS
		Cookie         Cookie
		JWT            JWT
	}

	Server struct {
		ReadHeaderTimeout time.Duration `envconfig:"READ_HEADER_TIMEOUT" default:"10s"`
		Addr              string        `envconfig:"ADDR" default:":8080"`
	}

	Telegram struct {
		Token          string  `required:"false"`
		AllowedChatIDs []int64 `envconfig:"ALLOWED_CHAT_IDS" required:"false"`
	}

	BuildInfo struct {
		Version   string
		BuildTime string
	}

	API struct {
		Dev       bool `envconfig:"DEV" default:"false"`
		DB        DB
		HTTP      HTTP
		Telegram  Telegram
		Server    Server
		BuildInfo BuildInfo
	}
)

func NewAPI(ctx context.Context) (*API, error) {
	res := &API{}
	if err := envconfig.Process("API", res); err != nil {
		return nil, fmt.Errorf("parse api environment: %w", err)
	}

	if err := validateAPI(res); err != nil {
		return nil, fmt.Errorf("validate api config: %w", err)
	}

	return res, nil
}

func validateAPI(target *API) error {
	if target.DB.Path == "" {
		return errors.New("db url is required")
	}
	if target.HTTP.JWT.Secret == "" {
		return errors.New("jwt secret is required")
	}
	if target.Telegram.Token == "" {
		return errors.New("telegram token is required")
	}
	if len(target.Telegram.AllowedChatIDs) == 0 {
		return errors.New("allowed chat ids are required")
	}

	return nil
}
