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
		envKey  string
		factory func(context.Context, string) (*ClientWrapper, error)
	}{
		{
			name:   "service_account",
			envKey: "OP_SERVICE_ACCOUNT_TOKEN",
			factory: func(ctx context.Context, version string) (*ClientWrapper, error) {
				return NewServiceAccount(ctx, version, os.Getenv("OP_SERVICE_ACCOUNT_TOKEN"))
			},
		},
		{
			name:   "desktop_app",
			envKey: "OP_ACCOUNT_NAME",
			factory: func(ctx context.Context, version string) (*ClientWrapper, error) {
				return NewDesktopAppIntegration(ctx, version, os.Getenv("OP_ACCOUNT_NAME"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if os.Getenv(test.envKey) == "" {
				t.Skipf("missing %s in .env", test.envKey)
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
