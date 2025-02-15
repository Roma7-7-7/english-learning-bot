package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const (
	DefaultAWSRegion = "eu-central-1"

	EnvDev  = "dev"
	EnvProd = "prod"

	defaultPublishInterval = 10 * time.Minute
	defaultHourFrom        = 9
	defaultHourTo          = 21
)

type (
	WordCheckSchedule struct {
		PublishInterval time.Duration
		HourFrom        int
		HourTo          int
		Location        *time.Location
	}
	Config struct {
		Env               string
		TelegramToken     string
		AllowedChatIDs    []int64
		DBURL             string
		WordCheckSchedule WordCheckSchedule
	}

	configBuilderFn func(string) (*Config, error)
)

func GetConfig() (*Config, error) {
	env := os.Getenv("ENV")
	if env == "" {
		env = EnvProd
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = DefaultAWSRegion
	}

	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		return nil, fmt.Errorf("load location: %w", err)
	}

	var confBuilder configBuilderFn
	switch {
	case env == EnvDev:
		confBuilder = getDevConfig
	case env == EnvProd:
		confBuilder = getProdConfig
	default:
		return nil, fmt.Errorf("unknown environment: %s", env)
	}

	res, err := confBuilder(region)
	if err != nil {
		return nil, err
	}
	res.Env = env
	res.WordCheckSchedule.Location = loc
	return validate(res)
}

func validate(conf *Config) (*Config, error) {
	errs := make([]string, 0, 10) //nolint:mnd // 10 is a reasonable default value
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
	if conf.WordCheckSchedule.PublishInterval == 0 {
		errs = append(errs, "publish interval is required")
	}
	if conf.WordCheckSchedule.HourFrom < 0 || conf.WordCheckSchedule.HourFrom > 23 {
		errs = append(errs, fmt.Sprintf("hour from %d must be in range 0-23", conf.WordCheckSchedule.HourFrom))
	}
	if conf.WordCheckSchedule.HourTo < 0 || conf.WordCheckSchedule.HourTo > 23 {
		errs = append(errs, fmt.Sprintf("hour to %d must be in range 0-23", conf.WordCheckSchedule.HourTo))
	}
	if conf.WordCheckSchedule.HourFrom >= conf.WordCheckSchedule.HourTo {
		errs = append(errs, fmt.Sprintf("hour from %d must be less than hour to %d", conf.WordCheckSchedule.HourFrom, conf.WordCheckSchedule.HourTo))
	}
	if conf.WordCheckSchedule.Location == nil {
		errs = append(errs, "location is required")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %s", strings.Join(errs, ", "))
	}

	return conf, nil
}

func getDevConfig(string) (*Config, error) {
	telegramTokenEnvVar := os.Getenv("TELEGRAM_TOKEN")
	dbURLEnvVar := os.Getenv("DB_URL")
	allowedChatIDs, err := parseChatIDs(os.Getenv("ALLOWED_CHAT_IDS"))
	if err != nil {
		return nil, err
	}
	publishInterval, err := parsePublishInterval(os.Getenv("PUBLISH_INTERVAL"), defaultPublishInterval)
	if err != nil {
		return nil, fmt.Errorf("parse publish interval: %w", err)
	}
	wordCheckScheduleFrom, err := parseInt(os.Getenv("WORD_CHECK_SCHEDULE_FROM"), defaultHourFrom)
	if err != nil {
		return nil, fmt.Errorf("parse word check schedule from: %w", err)
	}
	wordCheckScheduleTo, err := parseInt(os.Getenv("WORD_CHECK_SCHEDULE_TO"), defaultHourTo)
	if err != nil {
		return nil, fmt.Errorf("parse word check schedule to: %w", err)
	}

	return &Config{
		TelegramToken:  telegramTokenEnvVar,
		AllowedChatIDs: allowedChatIDs,
		DBURL:          dbURLEnvVar,
		WordCheckSchedule: WordCheckSchedule{
			PublishInterval: publishInterval,
			HourFrom:        wordCheckScheduleFrom,
			HourTo:          wordCheckScheduleTo,
		},
	}, nil
}

func getProdConfig(region string) (*Config, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("create aws session: %w", err)
	}

	ssmClient := ssm.New(sess, aws.NewConfig().WithRegion(region))
	parameters, err := ssmClient.GetParameters(&ssm.GetParametersInput{
		Names: []*string{
			aws.String("/english-learning-bot/prod/telegram-token"),
			aws.String("/english-learning-bot/prod/allowed-chat-ids"),
			aws.String("/english-learning-bot/prod/db-url"),
			aws.String("/english-learning-bot/prod/word-check-schedule/publish-interval"),
			aws.String("/english-learning-bot/prod/word-check-schedule/hour-from"),
			aws.String("/english-learning-bot/prod/word-check-schedule/hour-to"),
		},
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("get parameters: %w", err)
	}

	telegramToken := ""
	var allowedChatIDs []int64
	dbURL := ""
	publishIntervalStr := ""
	scheduleHourFrom := defaultHourFrom
	scheduleHourTo := defaultHourTo
	for _, param := range parameters.Parameters {
		switch *param.Name {
		case "/english-learning-bot/prod/telegram-token":
			telegramToken = *param.Value
		case "/english-learning-bot/prod/allowed-chat-ids":
			allowedChatIDs, err = parseChatIDs(*param.Value)
			if err != nil {
				return nil, err
			}
		case "/english-learning-bot/prod/db-url":
			dbURL = *param.Value
		case "/english-learning-bot/prod/publish-interval":
			publishIntervalStr = *param.Value
		case "/english-learning-bot/prod/word-check-schedule/hour-from":
			scheduleHourFrom, err = strconv.Atoi(*param.Value)
			if err != nil {
				return nil, fmt.Errorf("parse word check schedule from: %w", err)
			}
		case "/english-learning-bot/prod/word-check-schedule/hour-to":
			scheduleHourTo, err = strconv.Atoi(*param.Value)
			if err != nil {
				return nil, fmt.Errorf("parse word check schedule to: %w", err)
			}
		}
	}

	publishInterval, err := parsePublishInterval(publishIntervalStr, defaultPublishInterval)
	if err != nil {
		return nil, fmt.Errorf("parse publish interval: %w", err)
	}

	return &Config{
		TelegramToken:  telegramToken,
		AllowedChatIDs: allowedChatIDs,
		DBURL:          dbURL,
		WordCheckSchedule: WordCheckSchedule{
			PublishInterval: publishInterval,
			HourFrom:        scheduleHourFrom,
			HourTo:          scheduleHourTo,
		},
	}, nil
}

func parseChatIDs(chatIDsStr string) ([]int64, error) {
	if chatIDsStr == "" {
		return nil, nil
	}

	chatIDStrings := strings.Split(chatIDsStr, ",")
	chatIDs := make([]int64, 0, len(chatIDStrings))
	for _, chatIDString := range chatIDStrings {
		chatID, err := strconv.ParseInt(chatIDString, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse chat IDs: invalid chat ID %s: %w", chatIDString, err)
		}
		chatIDs = append(chatIDs, chatID)
	}

	return chatIDs, nil
}

func parsePublishInterval(publishIntervalStr string, def time.Duration) (time.Duration, error) {
	if publishIntervalStr == "" {
		return def, nil
	}
	return time.ParseDuration(publishIntervalStr)
}

func parseInt(val string, def int) (int, error) {
	if val == "" {
		return def, nil
	}
	return strconv.Atoi(val)
}
