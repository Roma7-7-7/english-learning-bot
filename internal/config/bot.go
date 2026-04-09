package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type (
	WordCheckSchedule struct {
		PublishInterval time.Duration `envconfig:"PUBLISH_INTERVAL" default:"15m"`
		HourFrom        int           `envconfig:"HOUR_FROM" default:"9"`
		HourTo          int           `envconfig:"HOUR_TO" default:"22"`
		Location        string        `envconfig:"LOCATION" default:"Europe/Kyiv"`
	}

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

	Bot struct {
		Dev       bool              `default:"false"`
		DB        DB                `envconfig:"DB"`
		Telegram  Telegram          `envconfig:"TELEGRAM"`
		Schedule  WordCheckSchedule `envconfig:"SCHEDULE"`
		HTTP      HTTP              `envconfig:"HTTP"`
		Server    Server            `envconfig:"SERVER"`
		BuildInfo BuildInfo
	}
)

func (s WordCheckSchedule) TimeLocation() (*time.Location, error) {
	loc, err := time.LoadLocation(s.Location)
	if err != nil {
		return nil, fmt.Errorf("load location: %w", err)
	}
	return loc, nil
}

func (s WordCheckSchedule) MustTimeLocation() *time.Location {
	loc, err := s.TimeLocation()
	if err != nil {
		panic(fmt.Sprintf("failed to load location %s: %v", s.Location, err))
	}
	return loc
}

func GetBot(ctx context.Context) (*Bot, error) {
	res := &Bot{}
	if err := envconfig.Process("BOT", res); err != nil {
		return nil, fmt.Errorf("parse bot environment: %w", err)
	}

	return validateBot(res)
}

func validateBot(conf *Bot) (*Bot, error) {
	errs := make([]string, 0, 10) //nolint:mnd // 10 is a reasonable default value

	if conf.DB.Path == "" {
		errs = append(errs, "db path is required")
	}
	if conf.Telegram.Token == "" {
		errs = append(errs, "telegram token is required")
	}
	if len(conf.Telegram.AllowedChatIDs) == 0 {
		errs = append(errs, "allowed chat ids are required")
	}
	if conf.HTTP.JWT.Secret == "" {
		errs = append(errs, "jwt secret is required")
	}
	if conf.Schedule.PublishInterval == 0 {
		errs = append(errs, "publish interval is required")
	}
	if conf.Schedule.HourFrom < 0 || conf.Schedule.HourFrom > 23 {
		errs = append(errs, fmt.Sprintf("hour from %d must be in range 0-23", conf.Schedule.HourFrom))
	}
	if conf.Schedule.HourTo < 0 || conf.Schedule.HourTo > 23 {
		errs = append(errs, fmt.Sprintf("hour to %d must be in range 0-23", conf.Schedule.HourTo))
	}
	if conf.Schedule.HourFrom >= conf.Schedule.HourTo {
		errs = append(errs, fmt.Sprintf("hour from %d must be less than hour to %d", conf.Schedule.HourFrom, conf.Schedule.HourTo))
	}
	if _, err := conf.Schedule.TimeLocation(); err != nil {
		errs = append(errs, fmt.Sprintf("invalid timezone: %s", err))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %s", strings.Join(errs, ", "))
	}

	return conf, nil
}
