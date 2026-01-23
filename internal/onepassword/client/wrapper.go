package client

import (
	"context"
	"log/slog"

	"github.com/1password/onepassword-sdk-go"
)

type ClientWrapper struct {
	inner  *onepassword.Client
	logger *slog.Logger
}

func NewServiceAccount(ctx context.Context, version string, token string) (*ClientWrapper, error) {
	inner, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo(IntegrationName, version),
	)

	if err != nil {
		return nil, err
	}

	return &ClientWrapper{
		inner: inner,
	}, nil
}

func NewDesktopAppIntegration(ctx context.Context, version string, accountName string) (*ClientWrapper, error) {
	inner, err := onepassword.NewClient(
		ctx,
		onepassword.WithDesktopAppIntegration(accountName),
		onepassword.WithIntegrationInfo(IntegrationName, IntegrationVersion),
	)

	if err != nil {
		return nil, err
	}

	return &ClientWrapper{
		inner: inner,
	}, nil
}
