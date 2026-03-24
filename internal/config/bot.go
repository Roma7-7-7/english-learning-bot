package config

import (
	"context"
	"errors"
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

	Bot struct {
		Dev            bool              `default:"false"`
		TelegramToken  string            `envconfig:"TELEGRAM_TOKEN" default:""`
		AllowedChatIDs []int64           `envconfig:"ALLOWED_CHAT_IDS" default:""`
		DBPath         string            `envconfig:"DB_PATH" default:"./data/english_learning.db?cache=shared&mode=rwc"`
		Schedule       WordCheckSchedule `envconfig:"SCHEDULE"`
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
	if conf.DBPath == "" {
		return nil, errors.New("db url is required")
	}

	errs := make([]string, 0, 10) //nolint:mnd // 10 is a reasonable default value
	if conf.DBPath == "" {
		errs = append(errs, "db url is required")
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
