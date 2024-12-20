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

	defaultPublishInterval = time.Hour
)

type (
	Config struct {
		Env             string
		TelegramToken   string
		AllowedChatIDs  []int64
		DBURL           string
		PublishInterval time.Duration
		Location        *time.Location
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
	res.Location = loc
	return validate(res, nil)
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

	return &Config{
		TelegramToken:   telegramTokenEnvVar,
		AllowedChatIDs:  allowedChatIDs,
		DBURL:           dbURLEnvVar,
		PublishInterval: publishInterval,
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
			aws.String("/english-learning-bot/prod/publish-interval"),
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
		}
	}

	publishInterval, err := parsePublishInterval(publishIntervalStr, defaultPublishInterval)
	if err != nil {
		return nil, fmt.Errorf("parse publish interval: %w", err)
	}

	return &Config{
		TelegramToken:   telegramToken,
		AllowedChatIDs:  allowedChatIDs,
		DBURL:           dbURL,
		PublishInterval: publishInterval,
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
