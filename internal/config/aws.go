package config

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

func FetchAWSParams(ctx context.Context, keys ...string) (map[string]string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)
	withDecryption := true
	parameters, err := ssmClient.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          keys,
		WithDecryption: &withDecryption,
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
