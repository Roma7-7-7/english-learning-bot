package config

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"         //nolint:staticcheck // to be fixed by https://github.com/Roma7-7-7/english-learning-bot/issues/74
	"github.com/aws/aws-sdk-go/aws/session" //nolint:staticcheck // to be fixed by https://github.com/Roma7-7-7/english-learning-bot/issues/74
	"github.com/aws/aws-sdk-go/service/ssm" //nolint:staticcheck // to be fixed by https://github.com/Roma7-7-7/english-learning-bot/issues/74
)

func FetchAWSParams(keys ...string) (map[string]string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create aws session: %w", err)
	}

	ssmClient := ssm.New(sess, aws.NewConfig())
	parameters, err := ssmClient.GetParameters(&ssm.GetParametersInput{
		Names:          aws.StringSlice(keys),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("get parameters: %w", err)
	}

	params := make(map[string]string)
	for _, param := range parameters.Parameters {
		params[*param.Name] = *param.Value
	}

	if len(params) != len(keys) {
		missingKeys := make([]string, 0)
		for _, key := range keys {
			if _, exists := params[key]; !exists {
				missingKeys = append(missingKeys, key)
			}
		}

		return params, fmt.Errorf("missing parameter values: %v", missingKeys)
	}

	return params, nil
}
