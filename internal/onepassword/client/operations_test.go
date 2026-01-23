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
		fields       []onepassword.ItemField
	}{
		{
			name:         "login",
			category:     onepassword.ItemCategoryLogin,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("username", onepassword.ItemFieldTypeText, "tester", nil),
				field("password", onepassword.ItemFieldTypeConcealed, "initial-secret", nil),
				field("otp", onepassword.ItemFieldTypeTOTP, "otpauth://totp/Example?secret=JBSWY3DPEHPK3PXP", nil),
			},
		},
		{
			name:         "secure_note",
			category:     onepassword.ItemCategorySecureNote,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("note", onepassword.ItemFieldTypeText, "example note", nil),
			},
		},
		{
			name:         "credit_card",
			category:     onepassword.ItemCategoryCreditCard,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("cardholder", onepassword.ItemFieldTypeText, "Ada Lovelace", nil),
				field("number", onepassword.ItemFieldTypeCreditCardNumber, "4111111111111111", nil),
				field("type", onepassword.ItemFieldTypeCreditCardType, "visa", nil),
				field("expiry", onepassword.ItemFieldTypeMonthYear, "01/2030", nil),
			},
		},
		{
			name:         "crypto_wallet",
			category:     onepassword.ItemCategoryCryptoWallet,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("wallet", onepassword.ItemFieldTypeText, "Main Wallet", nil),
				field("address", onepassword.ItemFieldTypeText, "0x0000000000000000000000000000000000000000", nil),
			},
		},
		{
			name:         "identity",
			category:     onepassword.ItemCategoryIdentity,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("first name", onepassword.ItemFieldTypeText, "Ada", nil),
				field("last name", onepassword.ItemFieldTypeText, "Lovelace", nil),
				field("address", onepassword.ItemFieldTypeAddress, "", addressDetails()),
			},
		},
		{
			name:         "password",
			category:     onepassword.ItemCategoryPassword,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("password", onepassword.ItemFieldTypeConcealed, "initial-secret", nil),
			},
		},
		{
			name:         "document",
			category:     onepassword.ItemCategoryDocument,
			expectCreate: true,
		},
		{
			name:         "api_credentials",
			category:     onepassword.ItemCategoryAPICredentials,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("client_id", onepassword.ItemFieldTypeText, "client-id", nil),
				field("client_secret", onepassword.ItemFieldTypeConcealed, "client-secret", nil),
			},
		},
		{
			name:         "bank_account",
			category:     onepassword.ItemCategoryBankAccount,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("bank_name", onepassword.ItemFieldTypeText, "Example Bank", nil),
				field("account_number", onepassword.ItemFieldTypeText, "0001234567", nil),
				field("routing_number", onepassword.ItemFieldTypeText, "110000000", nil),
			},
		},
		{
			name:         "database",
			category:     onepassword.ItemCategoryDatabase,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("hostname", onepassword.ItemFieldTypeText, "db.example.com", nil),
				field("port", onepassword.ItemFieldTypeText, "5432", nil),
				field("database", onepassword.ItemFieldTypeText, "app", nil),
				field("username", onepassword.ItemFieldTypeText, "dbuser", nil),
				field("password", onepassword.ItemFieldTypeConcealed, "dbpassword", nil),
			},
		},
		{
			name:         "driver_license",
			category:     onepassword.ItemCategoryDriverLicense,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("number", onepassword.ItemFieldTypeText, "D1234567", nil),
				field("state", onepassword.ItemFieldTypeText, "CA", nil),
				field("expiry", onepassword.ItemFieldTypeMonthYear, "01/2030", nil),
			},
		},
		{
			name:         "email",
			category:     onepassword.ItemCategoryEmail,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("email", onepassword.ItemFieldTypeEmail, "user@example.com", nil),
				field("password", onepassword.ItemFieldTypeConcealed, "email-secret", nil),
			},
		},
		{
			name:         "medical_record",
			category:     onepassword.ItemCategoryMedicalRecord,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("record_id", onepassword.ItemFieldTypeText, "MR-12345", nil),
				field("provider", onepassword.ItemFieldTypeText, "Example Hospital", nil),
			},
		},
		{
			name:         "membership",
			category:     onepassword.ItemCategoryMembership,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("member_id", onepassword.ItemFieldTypeText, "MEM-12345", nil),
				field("organization", onepassword.ItemFieldTypeText, "Example Club", nil),
			},
		},
		{
			name:         "outdoor_license",
			category:     onepassword.ItemCategoryOutdoorLicense,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("license_number", onepassword.ItemFieldTypeText, "OUT-12345", nil),
				field("state", onepassword.ItemFieldTypeText, "WA", nil),
			},
		},
		{
			name:         "passport",
			category:     onepassword.ItemCategoryPassport,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("number", onepassword.ItemFieldTypeText, "P1234567", nil),
				field("expiry", onepassword.ItemFieldTypeMonthYear, "12/2030", nil),
			},
		},
		{
			name:         "rewards",
			category:     onepassword.ItemCategoryRewards,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("membership_number", onepassword.ItemFieldTypeText, "RW-12345", nil),
				field("program", onepassword.ItemFieldTypeText, "Example Rewards", nil),
			},
		},
		{
			name:         "router",
			category:     onepassword.ItemCategoryRouter,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("hostname", onepassword.ItemFieldTypeText, "router.local", nil),
				field("username", onepassword.ItemFieldTypeText, "admin", nil),
				field("password", onepassword.ItemFieldTypeConcealed, "router-secret", nil),
			},
		},
		{
			name:         "server",
			category:     onepassword.ItemCategoryServer,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("hostname", onepassword.ItemFieldTypeText, "server.example.com", nil),
				field("username", onepassword.ItemFieldTypeText, "root", nil),
				field("password", onepassword.ItemFieldTypeConcealed, "server-secret", nil),
			},
		},
		{
			name:         "ssh_key",
			category:     onepassword.ItemCategorySSHKey,
			expectCreate: true,
			fields: []onepassword.ItemField{
				sshKeyField(),
			},
		},
		{
			name:         "social_security_number",
			category:     onepassword.ItemCategorySocialSecurityNumber,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("ssn", onepassword.ItemFieldTypeText, "123-45-6789", nil),
			},
		},
		{
			name:         "software_license",
			category:     onepassword.ItemCategorySoftwareLicense,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("license_key", onepassword.ItemFieldTypeText, "AAAA-BBBB-CCCC-DDDD", nil),
				field("product", onepassword.ItemFieldTypeText, "Example App", nil),
			},
		},
		{
			name:         "person",
			category:     onepassword.ItemCategoryPerson,
			expectCreate: true,
			fields: []onepassword.ItemField{
				field("first name", onepassword.ItemFieldTypeText, "Ada", nil),
				field("last name", onepassword.ItemFieldTypeText, "Lovelace", nil),
				field("address", onepassword.ItemFieldTypeAddress, "", addressDetails()),
			},
		},
		{
			name:         "unsupported",
			category:     onepassword.ItemCategoryUnsupported,
			expectCreate: false,
		},
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

			if len(test.fields) > 0 {
				params.Fields = test.fields
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

func field(label string, fieldType onepassword.ItemFieldType, value string, details *onepassword.ItemFieldDetails) onepassword.ItemField {
	return onepassword.ItemField{
		ID:        fieldID(label),
		Title:     label,
		FieldType: fieldType,
		Value:     value,
		Details:   details,
	}
}

func fieldID(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "" {
		return fmt.Sprintf("field_%d", time.Now().UnixNano())
	}
	return normalized
}

func addressDetails() *onepassword.ItemFieldDetails {
	address := onepassword.AddressFieldDetails{
		Street:  "123 Main St",
		City:    "Anytown",
		State:   "CA",
		Zip:     "12345",
		Country: "USA",
	}
	details := onepassword.NewItemFieldDetailsTypeVariantAddress(&address)
	return &details
}

func sshKeyDetails() *onepassword.ItemFieldDetails {
	sshKey := onepassword.SSHKeyAttributes{
		PublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8...",
		Fingerprint: "SHA256:1234567890123456789012345678901234567890123456789012345678901234",
		KeyType:     "ssh-rsa",
	}
	details := onepassword.NewItemFieldDetailsTypeVariantSSHKey(&sshKey)
	return &details
}

func sshKeyField() onepassword.ItemField {
	return onepassword.ItemField{
		ID:        "private_key",
		Title:     "private key",
		FieldType: onepassword.ItemFieldTypeSSHKey,
		Value: "-----BEGIN OPENSSH PRIVATE KEY-----\n" +
			"b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\n" +
			"QyNTUxOQAAACCL63pk8JPB71PYZaW0pz+/PZpAI2hiG48Re9kdRqXscAAAAJjMi7o7zIu6\n" +
			"OwAAAAtzc2gtZWQyNTUxOQAAACCL63pk8JPB71PYZaW0pz+/PZpAI2hiG48Re9kdRqXscA\n" +
			"AAAEC7/J8iRePLnRzmPAyH9HZ2GU7pFo86ekYnISM+Mx6uSIvremTwk8HvU9hlpbSnP789\n" +
			"mkAjaGIbjxF72R1GpexwAAAAE2Z3ZnVydGFkb0Bmdy1taW5pcGMBAg==\n" +
			"-----END OPENSSH PRIVATE KEY-----",
		Details: sshKeyDetails(),
	}
}
