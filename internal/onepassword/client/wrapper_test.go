package client

import (
	"context"
	"os"
	"testing"
)

func TestClientWrapperFactories(t *testing.T) {
	loadDotEnv(t)

	tests := []struct {
		name    string
		factory func(context.Context, string) (*ClientWrapper, error)
	}{
		{
			name: "service_account",
			factory: func(ctx context.Context, version string) (*ClientWrapper, error) {
				return NewServiceAccount(ctx, version, os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if os.Getenv("OP_SERVICE_ACCOUNT_TOKEN") == "" {
				t.Skip("missing OP_SERVICE_ACCOUNT_TOKEN in .env")
			}

			client, err := test.factory(context.Background(), "test")
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			if client == nil {
				t.Fatalf("client is nil")
			}
		})
	}
}
