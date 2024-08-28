package awsinterface

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AWSInterface struct {
	lambdaClient *lambda.Client
	stsClient    *sts.Client
	currentRole  string
}

func NewAWSInterface() (*AWSInterface, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &AWSInterface{
		lambdaClient: lambda.NewFromConfig(cfg),
		stsClient:    sts.NewFromConfig(cfg),
	}, nil
}

func (a *AWSInterface) AssumeRole(roleArn string) error {
	provider := stscreds.NewAssumeRoleProvider(a.stsClient, roleArn)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		return err
	}

	a.lambdaClient = lambda.NewFromConfig(cfg)
	a.currentRole = roleArn
	return nil
}

func (a *AWSInterface) ListLambdaFunctions() ([]string, error) {
	var functionNames []string
	var marker *string

	for {
		input := &lambda.ListFunctionsInput{
			Marker: marker,
		}

		result, err := a.lambdaClient.ListFunctions(context.TODO(), input)
		if err != nil {
			return nil, err
		}

		for _, function := range result.Functions {
			functionNames = append(functionNames, *function.FunctionName)
		}

		if result.NextMarker == nil {
			break
		}
		marker = result.NextMarker
	}

	return functionNames, nil
}

func (a *AWSInterface) InvokeLambda(functionName string, payload interface{}) ([]byte, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      jsonPayload,
	}

	result, err := a.lambdaClient.Invoke(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	return result.Payload, nil
}
