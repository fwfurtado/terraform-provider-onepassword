package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/1password/onepassword-sdk-go"
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

func TestClientWrapperGetItem(t *testing.T) {
	client := newTestClient(t)

	vault := getEnvOrSkip(t, "OP_TEST_VAULT")
	item := getEnvOrSkip(t, "OP_TEST_ITEM")

	vaultID, err := client.GetVault(context.Background(), vault)
	if err != nil {
		t.Fatalf("failed to get vault: %v", err)
	}

	overview, err := client.GetItemOverview(context.Background(), vaultID, item)
	if err != nil {
		t.Fatalf("failed to get item overview: %v", err)
	}

	fullItem, err := client.GetItem(context.Background(), vaultID, overview.ID)
	if err != nil {
		t.Fatalf("failed to get item: %v", err)
	}

	if fullItem.ID == "" {
		t.Fatalf("item ID is empty")
	}
}

func TestClientWrapperValidateSecretReference(t *testing.T) {
	client := newTestClient(t)

	tests := []struct {
		name      string
		reference string
		wantErr   bool
	}{
		{
			name:      "valid_reference",
			reference: "op://vault/item/field",
			wantErr:   false,
		},
		{
			name:      "invalid_reference",
			reference: "invalid-reference",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := client.ValidateSecretReference(context.Background(), test.reference)
			if test.wantErr && err == nil {
				t.Fatalf("expected error but got none")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestClientWrapperGeneratePassword(t *testing.T) {
	client := newTestClient(t)

	recipe := onepassword.NewPasswordRecipeTypeVariantRandom(&onepassword.PasswordRecipeRandomInner{
		IncludeDigits:  true,
		IncludeSymbols: true,
		Length:         32,
	})

	password, err := client.GeneratePassword(context.Background(), recipe)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	if password == "" {
		t.Fatalf("generated password is empty")
	}
}

func TestClientWrapperCreateUpdateDeleteItem(t *testing.T) {
	client := newTestClient(t)

	vault := getEnvOrSkip(t, "OP_TEST_VAULT")
	vaultID, err := client.GetVault(context.Background(), vault)
	if err != nil {
		t.Fatalf("failed to get vault: %v", err)
	}

	itemName := fmt.Sprintf("tf-provider-test-%d", time.Now().UnixNano())
	notes := "created by terraform provider tests"

	createdItem, err := client.CreateItem(context.Background(), onepassword.ItemCreateParams{
		Category: onepassword.ItemCategoryLogin,
		VaultID:  vaultID,
		Title:    itemName,
		Notes:    &notes,
		Fields: []onepassword.ItemField{
			{
				ID:        "username",
				Title:     "username",
				FieldType: onepassword.ItemFieldTypeText,
				Value:     "tester",
			},
			{
				ID:        "password",
				Title:     "password",
				FieldType: onepassword.ItemFieldTypeConcealed,
				Value:     "initial-secret",
			},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "not sufficient permissions") {
			t.Skipf("service account lacks item permissions: %v", err)
		}
		t.Fatalf("failed to create item: %v", err)
	}

	if createdItem.ID == "" {
		t.Fatalf("created item ID is empty")
	}

	itemDeleted := false
	t.Cleanup(func() {
		if itemDeleted {
			return
		}
		_ = client.DeleteItem(context.Background(), vaultID, createdItem.ID)
	})

	createdItem.Notes = "updated by terraform provider tests"
	updatedItem, err := client.UpdateItem(context.Background(), *createdItem)
	if err != nil {
		if strings.Contains(err.Error(), "not sufficient permissions") {
			t.Skipf("service account lacks item permissions: %v", err)
		}
		t.Fatalf("failed to update item: %v", err)
	}

	if updatedItem.Notes == "" {
		t.Fatalf("updated item notes is empty")
	}

	if err := client.DeleteItem(context.Background(), vaultID, createdItem.ID); err != nil {
		if strings.Contains(err.Error(), "not sufficient permissions") {
			t.Skipf("service account lacks item permissions: %v", err)
		}
		t.Fatalf("failed to delete item: %v", err)
	}
	itemDeleted = true
}
