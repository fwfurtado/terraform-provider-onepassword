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
				if shouldSkipNotFound(err) {
					t.Skipf("missing test item or field: %v", err)
				}
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
		if shouldSkipNotFound(err) {
			t.Skipf("missing test item: %v", err)
		}
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
		if shouldSkipNotFound(err) {
			t.Skipf("missing test item: %v", err)
		}
		t.Fatalf("failed to get item overview: %v", err)
	}

	fullItem, err := client.GetItem(context.Background(), vaultID, overview.ID)
	if err != nil {
		if shouldSkipNotFound(err) {
			t.Skipf("missing test item: %v", err)
		}
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

	tests := []struct {
		name         string
		category     onepassword.ItemCategory
		expectCreate bool
	}{
		{name: "login", category: onepassword.ItemCategoryLogin, expectCreate: true},
		{name: "secure_note", category: onepassword.ItemCategorySecureNote, expectCreate: true},
		{name: "credit_card", category: onepassword.ItemCategoryCreditCard, expectCreate: true},
		{name: "crypto_wallet", category: onepassword.ItemCategoryCryptoWallet, expectCreate: true},
		{name: "identity", category: onepassword.ItemCategoryIdentity, expectCreate: true},
		{name: "password", category: onepassword.ItemCategoryPassword, expectCreate: true},
		{name: "document", category: onepassword.ItemCategoryDocument, expectCreate: true},
		{name: "api_credentials", category: onepassword.ItemCategoryAPICredentials, expectCreate: true},
		{name: "bank_account", category: onepassword.ItemCategoryBankAccount, expectCreate: true},
		{name: "database", category: onepassword.ItemCategoryDatabase, expectCreate: true},
		{name: "driver_license", category: onepassword.ItemCategoryDriverLicense, expectCreate: true},
		{name: "email", category: onepassword.ItemCategoryEmail, expectCreate: true},
		{name: "medical_record", category: onepassword.ItemCategoryMedicalRecord, expectCreate: true},
		{name: "membership", category: onepassword.ItemCategoryMembership, expectCreate: true},
		{name: "outdoor_license", category: onepassword.ItemCategoryOutdoorLicense, expectCreate: true},
		{name: "passport", category: onepassword.ItemCategoryPassport, expectCreate: true},
		{name: "rewards", category: onepassword.ItemCategoryRewards, expectCreate: true},
		{name: "router", category: onepassword.ItemCategoryRouter, expectCreate: true},
		{name: "server", category: onepassword.ItemCategoryServer, expectCreate: true},
		{name: "ssh_key", category: onepassword.ItemCategorySSHKey, expectCreate: true},
		{name: "social_security_number", category: onepassword.ItemCategorySocialSecurityNumber, expectCreate: true},
		{name: "software_license", category: onepassword.ItemCategorySoftwareLicense, expectCreate: true},
		{name: "person", category: onepassword.ItemCategoryPerson, expectCreate: true},
		{name: "unsupported", category: onepassword.ItemCategoryUnsupported, expectCreate: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			itemName := fmt.Sprintf("tf-provider-test-%s-%d", test.name, time.Now().UnixNano())
			notes := fmt.Sprintf("created by terraform provider tests (%s)", test.name)

			params := onepassword.ItemCreateParams{
				Category: test.category,
				VaultID:  vaultID,
				Title:    itemName,
				Notes:    &notes,
			}

			if test.category == onepassword.ItemCategoryLogin || test.category == onepassword.ItemCategoryPassword {
				params.Fields = []onepassword.ItemField{
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
				}
			}

			if test.category == onepassword.ItemCategoryDocument {
				params.Document = &onepassword.DocumentCreateParams{
					Name:    "example.txt",
					Content: []byte("example"),
				}
			}

			createdItem, err := client.CreateItem(context.Background(), params)
			if err != nil {
				if strings.Contains(err.Error(), "not sufficient permissions") {
					t.Skipf("service account lacks item permissions: %v", err)
				}
				if shouldSkipRateLimit(err) {
					t.Skipf("rate limited by 1Password: %v", err)
				}
				if !test.expectCreate {
					return
				}
				t.Fatalf("failed to create item: %v", err)
			}

			if !test.expectCreate {
				t.Fatalf("expected create to fail for %s but it succeeded", test.name)
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

			createdItem.Notes = fmt.Sprintf("updated by terraform provider tests (%s)", test.name)
			updatedItem, err := client.UpdateItem(context.Background(), *createdItem)
			if err != nil {
				if strings.Contains(err.Error(), "not sufficient permissions") {
					t.Skipf("service account lacks item permissions: %v", err)
				}
				if shouldSkipRateLimit(err) {
					t.Skipf("rate limited by 1Password: %v", err)
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
				if shouldSkipRateLimit(err) {
					t.Skipf("rate limited by 1Password: %v", err)
				}
				t.Fatalf("failed to delete item: %v", err)
			}
			itemDeleted = true
		})
	}
}

func shouldSkipNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "item not found") ||
		strings.Contains(message, "no item matched the secret reference query")
}

func shouldSkipRateLimit(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "rate limit exceeded")
}
