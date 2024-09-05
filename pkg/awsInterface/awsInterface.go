package awsInterface

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

type AWSInterface struct {
	cfg           aws.Config
	ssoClient     *sso.Client
	ssooidcClient *ssooidc.Client
	lambdaClient  *lambda.Client
	ssoToken      string
	tokenExpiry   time.Time
	ssoStartURL   string
}

func NewAWSInterface(ssoProfileName string) (*AWSInterface, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(ssoProfileName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	ssoClient := sso.NewFromConfig(cfg)
	ssooidcClient := ssooidc.NewFromConfig(cfg)
	lambdaClient := lambda.NewFromConfig(cfg)

	awsInterface := &AWSInterface{
		cfg:           cfg,
		ssoClient:     ssoClient,
		ssooidcClient: ssooidcClient,
		lambdaClient:  lambdaClient,
	}

	return awsInterface, nil
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
			return nil, fmt.Errorf("failed to list Lambda functions: %v", err)
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

func (a *AWSInterface) InvokeLambda(functionName string, payload []byte) ([]byte, error) {
	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payload,
	}

	result, err := a.lambdaClient.Invoke(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Lambda function: %v", err)
	}

	return result.Payload, nil
}
