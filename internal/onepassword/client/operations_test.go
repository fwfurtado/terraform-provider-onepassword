package client

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestClientWrapperGetSecretByReference(t *testing.T) {
	client := newTestClient(t)

	vault := getEnvOrSkip(t, "OP_TEST_VAULT")
	item := getEnvOrSkip(t, "OP_TEST_ITEM")
	field := getEnvOrSkip(t, "OP_TEST_FIELD")
	section := os.Getenv("OP_TEST_SECTION")

	tests := []struct {
		name    string
		section string
	}{
		{
			name:    "without_section",
			section: "",
		},
		{
			name:    "with_section",
			section: section,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "with_section" && test.section == "" {
				t.Skip("missing OP_TEST_SECTION in .env")
			}

			var (
				value string
				err   error
			)

			if test.section == "" {
				value, err = client.GetSecretByReference(context.Background(), fmt.Sprintf("op://%s/%s/%s", vault, item, field))
			} else {
				value, err = client.GetSecretByReference(context.Background(), fmt.Sprintf("op://%s/%s/%s/%s", vault, item, test.section, field))
			}

			if err != nil {
				t.Fatalf("failed to get item field: %v", err)
			}

			if value == "" {
				t.Log("field value is empty")
			}
		})
	}
}

func TestClientWrapperGetVault(t *testing.T) {
	client := newTestClient(t)

	vault := getEnvOrSkip(t, "OP_TEST_VAULT")

	tests := []struct {
		name  string
		title string
	}{
		{
			name:  "by_title",
			title: vault,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vaultID, err := client.GetVault(context.Background(), test.title)
			if err != nil {
				t.Fatalf("failed to get vault: %v", err)
			}

			if vaultID == "" {
				t.Fatalf("vault ID is empty")
			}
		})
	}
}

func TestClientWrapperGetItemOverview(t *testing.T) {
	client := newTestClient(t)

	vault := getEnvOrSkip(t, "OP_TEST_VAULT")
	item := getEnvOrSkip(t, "OP_TEST_ITEM")

	vaultID, err := client.GetVault(context.Background(), vault)
	if err != nil {
		t.Fatalf("failed to get vault: %v", err)
	}

	if vaultID == "" {
		t.Fatalf("vault ID is empty")
	}

	overview, err := client.GetItemOverview(context.Background(), vaultID, item)
	if err != nil {
		t.Fatalf("failed to get item overview: %v", err)
	}

	if overview == nil {
		t.Fatalf("item overview is nil")
	}

	if overview.ID == "" {
		t.Fatalf("item overview ID is empty")
	}

	if overview.Title == "" {
		t.Fatalf("item overview title is empty")
	}

	if overview.Category == "" {
		t.Fatalf("item overview category is empty")
	}

	if len(overview.Websites) == 0 {
		t.Fatalf("item overview websites is empty")
	}
}
