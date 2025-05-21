package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type (
	WordCheckSchedule struct {
		PublishInterval time.Duration `default:"15m"`
		HourFrom        int           `default:"9"`
		HourTo          int           `default:"21"`
		Location        string        `default:"Europe/Kyiv"`
	}

	Bot struct {
		Dev            bool    `default:"false"`
		TelegramToken  string  `envconfig:"TELEGRAM_TOKEN" required:"true"`
		AllowedChatIDs []int64 `envconfig:"ALLOWED_CHAT_IDS" required:"true"`
		DBURL          string  `envconfig:"DB_URL" default:""`
		Schedule       WordCheckSchedule
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

func GetBot() (*Bot, error) {
	res := &Bot{}
	if err := envconfig.Process("BOT", res); err != nil {
		return nil, fmt.Errorf("parse bot environment: %w", err)
	}

	if !res.Dev {
		if err := setBotProdConfig(res); err != nil {
			return nil, fmt.Errorf("set bot prod config: %w", err)
		}
	}

	return validateBot(res)
}

func validateBot(conf *Bot) (*Bot, error) {
	if conf.DBURL == "" {
		return nil, fmt.Errorf("db url is required")
	}

	errs := make([]string, 0, 10) //nolint:mnd // 10 is a reasonable default value
	if conf.DBURL == "" {
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

func setBotProdConfig(target *Bot) error {
	parameters, err := FetchAWSParams(
		"/english-learning-bot/prod/telegram-token",
		"/english-learning-bot/prod/allowed-chat-ids",
		"/english-learning-bot/prod/db-url",
	)
	if err != nil {
		return fmt.Errorf("get parameters: %w", err)
	}

	for name, value := range parameters {
		switch name {
		case "/english-learning-bot/prod/telegram-token":
			target.TelegramToken = value
		case "/english-learning-bot/prod/allowed-chat-ids":
			target.AllowedChatIDs, err = parseChatIDs(value)
			if err != nil {
				return err
			}
		case "/english-learning-bot/prod/db-url":
			target.DBURL = value
		}
	}

	return nil
}
