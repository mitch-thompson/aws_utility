package awsInterface

import (
	"aws_utility/pkg/logger"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/credentials"
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
	clientID      string
	clientSecret  string
}

type Account struct {
	AccountID   string
	AccountName string
}

type Role struct {
	RoleName  string
	AccountID string
}

type AuthenticationInfo struct {
	DeviceCode              string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int32
	Interval                int32
}

func NewAWSInterface(ssoStartURL string) (*AWSInterface, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	ssoClient := sso.NewFromConfig(cfg)
	ssooidcClient := ssooidc.NewFromConfig(cfg)

	awsInterface := &AWSInterface{
		cfg:           cfg,
		ssoClient:     ssoClient,
		ssooidcClient: ssooidcClient,
		ssoStartURL:   ssoStartURL,
	}

	return awsInterface, nil
}

func (a *AWSInterface) AssumeRole(accountID, roleName string) error {
	input := &sso.GetRoleCredentialsInput{
		AccountId:   aws.String(accountID),
		RoleName:    aws.String(roleName),
		AccessToken: aws.String(a.ssoToken),
	}

	output, err := a.ssoClient.GetRoleCredentials(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to get role credentials: %v", err)
	}

	// Create a new AWS config with the role credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			*output.RoleCredentials.AccessKeyId,
			*output.RoleCredentials.SecretAccessKey,
			*output.RoleCredentials.SessionToken,
		)),
	)
	if err != nil {
		return fmt.Errorf("failed to create new AWS config: %v", err)
	}

	// Update the AWS config and create a new Lambda client
	a.cfg = cfg
	a.lambdaClient = lambda.NewFromConfig(cfg)

	return nil
}

func (a *AWSInterface) RegisterClient() error {
	logger.Info("Starting RegisterClient()")
	registerClientInput := &ssooidc.RegisterClientInput{
		ClientName: aws.String("AWSUtility"),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	}

	logger.Info("Running with input: ")
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*1)
	registerClientOutput, err := a.ssooidcClient.RegisterClient(ctx, registerClientInput)
	if err != nil {
		return fmt.Errorf("failed to register client: %v", err)
	}
	logger.Info("Registration completed")

	a.clientID = *registerClientOutput.ClientId
	a.clientSecret = *registerClientOutput.ClientSecret

	return nil
}

func (a *AWSInterface) StartAuthentication() (*AuthenticationInfo, error) {
	if a.clientID == "" || a.clientSecret == "" {
		return nil, fmt.Errorf("client not registered, call RegisterClient() first")
	}

	startDeviceAuthOutput, err := a.ssooidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(a.clientID),
		ClientSecret: aws.String(a.clientSecret),
		StartUrl:     aws.String(a.ssoStartURL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start device authorization: %v", err)
	}

	return &AuthenticationInfo{
		DeviceCode:              *startDeviceAuthOutput.DeviceCode,
		UserCode:                *startDeviceAuthOutput.UserCode,
		VerificationURI:         *startDeviceAuthOutput.VerificationUri,
		VerificationURIComplete: *startDeviceAuthOutput.VerificationUriComplete,
		ExpiresIn:               startDeviceAuthOutput.ExpiresIn,
		Interval:                startDeviceAuthOutput.Interval,
	}, nil
}

func (a *AWSInterface) PollForToken(authInfo *AuthenticationInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(authInfo.ExpiresIn)*time.Second)
	defer cancel()

	ticker := time.NewTicker(time.Duration(authInfo.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("authentication timed out")
		case <-ticker.C:
			createTokenOutput, err := a.ssooidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
				ClientId:     aws.String(a.clientID),
				ClientSecret: aws.String(a.clientSecret),
				DeviceCode:   aws.String(authInfo.DeviceCode),
				GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
			})
			if err != nil {
				// todo
				//var pendingAuthErr
				//if errors.As(err, &pendingAuthErr) {
				continue // User hasn't authorized yet, keep polling
				//}
				//return fmt.Errorf("failed to create token: %v", err)
			}

			a.ssoToken = *createTokenOutput.AccessToken
			a.tokenExpiry = time.Now().Add(time.Duration(createTokenOutput.ExpiresIn) * time.Second)
			return nil
		}
	}
}

func (a *AWSInterface) ListAccounts() ([]Account, error) {
	input := &sso.ListAccountsInput{
		AccessToken: aws.String(a.ssoToken),
	}

	var accounts []Account
	for {
		output, err := a.ssoClient.ListAccounts(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %v", err)
		}

		for _, account := range output.AccountList {
			accounts = append(accounts, Account{
				AccountID:   *account.AccountId,
				AccountName: *account.AccountName,
			})
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return accounts, nil
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

func (a *AWSInterface) ListRoles(accountID string) ([]Role, error) {
	input := &sso.ListAccountRolesInput{
		AccessToken: aws.String(a.ssoToken),
		AccountId:   aws.String(accountID),
	}

	var roles []Role
	for {
		output, err := a.ssoClient.ListAccountRoles(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list roles: %v", err)
		}

		for _, role := range output.RoleList {
			roles = append(roles, Role{
				RoleName:  *role.RoleName,
				AccountID: accountID,
			})
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return roles, nil
}
