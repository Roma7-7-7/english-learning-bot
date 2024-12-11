package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

type Config struct {
	Env             string
	TelegramToken   string
	AllowedChatIDs  []int64
	DBURL           string
	PublishInterval time.Duration
	Location        *time.Location
}

func GetConfig() (*Config, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = EnvProd
	}

	switch {
	case env == EnvDev:
		return validate(getDevConfig())
	case env == EnvProd:
		return validate(getProdConfig())
	default:
		return nil, fmt.Errorf("unknown environment: %s", env)
	}
}

func validate(conf *Config, err error) (*Config, error) {
	if err != nil {
		return nil, err
	}

	errs := make([]string, 0, 6) //nolint:mnd // 6 is a reasonable default value
	if conf.Env != EnvDev && conf.Env != EnvProd {
		errs = append(errs, fmt.Sprintf("unknown environment: %s", conf.Env))
	}
	if conf.TelegramToken == "" {
		errs = append(errs, "telegram token is required")
	}
	if len(conf.AllowedChatIDs) == 0 {
		errs = append(errs, "allowed chat ids is required")
	}
	if conf.DBURL == "" {
		errs = append(errs, "db url is required")
	}
	if conf.PublishInterval == 0 {
		errs = append(errs, "publish interval is required")
	}
	if conf.Location == nil {
		errs = append(errs, "location is required")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %s", strings.Join(errs, ", "))
	}

	return conf, nil
}

func getDevConfig() (*Config, error) {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		return nil, fmt.Errorf("load location: %w", err)
	}

	telegramTokenEnvVar := os.Getenv("TELEGRAM_TOKEN")
	allowedChatIDs := make([]int64, 0, 10) //nolint:mnd // 10 is a reasonable default value
	allowedChatIDsEnvVar := os.Getenv("ALLOWED_CHAT_IDS")
	if allowedChatIDsEnvVar != "" {
		chatIDStrings := strings.Split(allowedChatIDsEnvVar, ",")
		for _, chatIDString := range chatIDStrings {
			var chatID int64
			chatID, err = strconv.ParseInt(chatIDString, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid chat ID %s: %w", chatIDString, err)
			}
			allowedChatIDs = append(allowedChatIDs, chatID)
		}
	}
	dbURLEnvVar := os.Getenv("DB_URL")
	publishIntervalEnvVar := os.Getenv("PUBLISH_INTERVAL")
	if publishIntervalEnvVar == "" {
		publishIntervalEnvVar = "1h"
	}
	publishInterval, err := time.ParseDuration(publishIntervalEnvVar)
	if err != nil {
		return nil, fmt.Errorf("parse publish interval: %w", err)
	}

	return &Config{
		Env:             EnvDev,
		TelegramToken:   telegramTokenEnvVar,
		AllowedChatIDs:  allowedChatIDs,
		DBURL:           dbURLEnvVar,
		PublishInterval: publishInterval,
		Location:        loc,
	}, nil
}

func getProdConfig() (*Config, error) {
	return nil, errors.New("not implemented")
}
