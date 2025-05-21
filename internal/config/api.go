package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type (
	DB struct {
		URL string `envconfig:"DB_URL" required:"true"`
	}

	CORS struct {
		AllowOrigins []string `envconfig:"ALLOW_ORIGINS" required:"true"`
	}

	JWT struct {
		Issuer   string   `envconfig:"ISSUER" default:"english-learning-api"`
		Audience []string `envconfig:"AUDIENCE" required:"true"`
		Secret   string   `envconfig:"SECRET" required:"true"`
	}

	Cookie struct {
		Path            string        `envconfig:"CPATH" default:"/"` // not using PATH here because it may conflict with os.Path
		Domain          string        `envconfig:"DOMAIN" required:"true"`
		AuthExpiresIn   time.Duration `envconfig:"AUTH_EXPIRES_IN" default:"15m"`
		AccessExpiresIn time.Duration `envconfig:"ACCESS_EXPIRES_IN" default:"24h"`
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
		Token string `envconfig:"TELEGRAM_TOKEN" required:"true"`
	}

	API struct {
		Dev      bool `envconfig:"DEV" default:"false"`
		DB       DB
		HTTP     HTTP
		Telegram Telegram
		Server   Server
	}
)

func NewAPI() (API, error) {
	var res API
	if err := envconfig.Process("API", &res); err != nil {
		return API{}, fmt.Errorf("parse api environment: %w", err)
	}

	if !res.Dev {
		return API{}, errors.New("prod is not supported yet")
	}

	return res, nil
}
