package client

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/cache"
)

type opClient interface {
	Secrets() onepassword.SecretsAPI
	Items() onepassword.ItemsAPI
	Vaults() onepassword.VaultsAPI
}

// ClientWrapper wraps the 1Password SDK client with helper methods.
type ClientWrapper struct {
	inner opClient
	cache *cache.Cache
}

func newClientWrapper(inner opClient, cacheStore *cache.Cache) *ClientWrapper {
	return &ClientWrapper{
		inner: inner,
		cache: cacheStore,
	}
}

// NewServiceAccount creates a client using a service account token.
func NewServiceAccount(ctx context.Context, version string, token string, cacheStore *cache.Cache) (*ClientWrapper, error) {
	inner, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo(IntegrationName, version),
	)

	if err != nil {
		return nil, err
	}

	return newClientWrapper(inner, cacheStore), nil
}

// NewDesktopAppIntegration creates a client using the 1Password desktop integration.
func NewDesktopAppIntegration(ctx context.Context, version string, accountName string, cacheStore *cache.Cache) (*ClientWrapper, error) {
	inner, err := onepassword.NewClient(
		ctx,
		onepassword.WithDesktopAppIntegration(accountName),
		onepassword.WithIntegrationInfo(IntegrationName, version),
	)

	if err != nil {
		return nil, err
	}

	return newClientWrapper(inner, cacheStore), nil
}
