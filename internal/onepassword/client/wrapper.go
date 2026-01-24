package client

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
)

type opClient interface {
	Secrets() onepassword.SecretsAPI
	Items() onepassword.ItemsAPI
	Vaults() onepassword.VaultsAPI
}

// ClientWrapper wraps the 1Password SDK client with helper methods.
type ClientWrapper struct {
	inner opClient
}

func newClientWrapper(inner opClient) *ClientWrapper {
	return &ClientWrapper{inner: inner}
}

// NewServiceAccount creates a client using a service account token.
func NewServiceAccount(ctx context.Context, version string, token string) (*ClientWrapper, error) {
	inner, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo(IntegrationName, version),
	)

	if err != nil {
		return nil, err
	}

	return newClientWrapper(inner), nil
}

// NewDesktopAppIntegration creates a client using the 1Password desktop integration.
func NewDesktopAppIntegration(ctx context.Context, version string, accountName string) (*ClientWrapper, error) {
	inner, err := onepassword.NewClient(
		ctx,
		onepassword.WithDesktopAppIntegration(accountName),
		onepassword.WithIntegrationInfo(IntegrationName, version),
	)

	if err != nil {
		return nil, err
	}

	return newClientWrapper(inner), nil
}
