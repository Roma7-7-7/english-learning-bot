package config

import (
	"errors"
	"os"
	"time"
)

type (
	DB struct {
		URL string `json:"url"`
	}

	CORS struct {
		AllowOrigins []string `json:"allow_origins"`
	}

	JWT struct {
		Issuer   string   `json:"issuer"`
		Audience []string `json:"audience"`
		Secret   string   `json:"secret"`
	}

	Cookie struct {
		Path            string        `json:"path"`
		Domain          string        `json:"domain"`
		AuthExpiresIn   time.Duration `json:"auth_expires_in"`
		AccessExpiresIn time.Duration `json:"access_expires_in"`
	}

	HTTP struct {
		Timeout   time.Duration `json:"timeout"`
		RateLimit float64       `json:"rate_limit"`
		CORS      CORS          `json:"cors"`
		Cookie    Cookie        `json:"cookie"`
		JWT       JWT           `json:"jwt"`
	}

	Server struct {
		ReadHeaderTimeout time.Duration `json:"read_header_timeout"`
		Addr              string        `json:"addr"`
	}

	Telegram struct {
		Token string
	}

	API struct {
		DB       DB
		HTTP     HTTP `json:"http"`
		Telegram Telegram
		Server   Server
	}
)

func NewAPI(env Env) (API, error) {
	if env == EnvProd {
		return API{}, errors.New("api environment is prod")
	}

	return API{
		DB: DB{
			URL: os.Getenv("DB_URL"),
		},
		HTTP: HTTP{
			Timeout:   10 * time.Second, //nolint:mnd // ignore mnd
			RateLimit: 25,               //nolint:mnd // ignore mnd
			CORS: CORS{
				AllowOrigins: []string{"http://localhost:5173"},
			},
			Cookie: Cookie{
				Path:            "/",
				Domain:          "localhost",
				AuthExpiresIn:   15 * time.Minute, //nolint:mnd // ignore mnd
				AccessExpiresIn: 24 * time.Hour,   //nolint:mnd // ignore mnd
			},
			JWT: JWT{
				Issuer:   "english-learning-api",
				Audience: []string{"http://localhost:8080"},
				Secret:   os.Getenv("JWT_SECRET"),
			},
		},
		Telegram: Telegram{
			Token: os.Getenv("TELEGRAM_TOKEN"),
		},
		Server: Server{
			ReadHeaderTimeout: 10 * time.Second, //nolint:mnd // ignore mnd
			Addr:              ":8080",
		},
	}, nil
}
