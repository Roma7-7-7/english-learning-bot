package config

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
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
