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

	WebAPI struct {
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

	Web struct {
		DB       DB
		API      WebAPI `json:"api"`
		Telegram Telegram
		Server   Server
	}
)

func NewWeb(env Env) (Web, error) {
	if env == EnvProd {
		return Web{}, errors.New("web environment is prod")
	}

	return Web{
		DB: DB{
			URL: os.Getenv("DB_URL"),
		},
		API: WebAPI{
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
				Issuer:   "english-learning-web",
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
