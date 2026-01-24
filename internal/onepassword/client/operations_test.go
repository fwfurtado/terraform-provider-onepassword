package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/1password/onepassword-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientWrapperGetSecretByReference(t *testing.T) {
	client, fixture := newTestClient(t)

	tests := []struct {
		name      string
		reference string
		want      string
		wantErr   bool
	}{
		{
			name:      "without_section",
			reference: fmt.Sprintf("op://%s/%s/%s", fixture.VaultName, fixture.ItemTitle, fixture.UsernameField),
			want:      fixture.UsernameValue,
		},
		{
			name:      "with_section",
			reference: fmt.Sprintf("op://%s/%s/%s/%s", fixture.VaultName, fixture.ItemTitle, fixture.SectionName, fixture.PasswordField),
			want:      fixture.PasswordValue,
		},
		{
			name:      "missing_field",
			reference: fmt.Sprintf("op://%s/%s/%s", fixture.VaultName, fixture.ItemTitle, "missing"),
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, err := client.GetSecretByReference(context.Background(), test.reference)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, value)
		})
	}
}

func TestClientWrapperGetVault(t *testing.T) {
	client, fixture := newTestClient(t)

	tests := []struct {
		name    string
		title   string
		wantID  string
		wantErr bool
	}{
		{
			name:   "by_title",
			title:  fixture.VaultName,
			wantID: fixture.VaultID,
		},
		{
			name:    "missing",
			title:   "missing-vault",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vaultID, err := client.GetVault(context.Background(), test.title)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantID, vaultID)
		})
	}
}

func TestClientWrapperGetItemOverview(t *testing.T) {
	client, fixture := newTestClient(t)

	vaultID, err := client.GetVault(context.Background(), fixture.VaultName)
	require.NoError(t, err)

	tests := []struct {
		name    string
		item    string
		wantErr bool
	}{
		{
			name: "existing_item",
			item: fixture.ItemTitle,
		},
		{
			name:    "missing_item",
			item:    "missing-item",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			overview, err := client.GetItemOverview(context.Background(), vaultID, test.item)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, overview)
			assert.Equal(t, fixture.ItemTitle, overview.Title)
			assert.Equal(t, fixture.ItemID, overview.ID)
			assert.Equal(t, onepassword.ItemCategoryLogin, overview.Category)
			assert.NotEmpty(t, overview.Websites)
		})
	}
}

func TestClientWrapperGetItem(t *testing.T) {
	client, fixture := newTestClient(t)

	tests := []struct {
		name    string
		itemID  string
		wantErr bool
	}{
		{
			name:   "existing_item",
			itemID: fixture.ItemID,
		},
		{
			name:    "missing_item",
			itemID:  "missing-item",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item, err := client.GetItem(context.Background(), fixture.VaultID, test.itemID)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, item)
			assert.Equal(t, fixture.ItemID, item.ID)
			assert.Equal(t, fixture.ItemTitle, item.Title)
		})
	}
}

func TestClientWrapperValidateSecretReference(t *testing.T) {
	client, _ := newTestClient(t)

	tests := []struct {
		name      string
		reference string
		wantErr   bool
	}{
		{
			name:      "valid_reference",
			reference: "op://vault/item/field",
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
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClientWrapperGeneratePassword(t *testing.T) {
	client, _ := newTestClient(t)

	recipe := onepassword.NewPasswordRecipeTypeVariantRandom(&onepassword.PasswordRecipeRandomInner{
		IncludeDigits:  true,
		IncludeSymbols: true,
		Length:         24,
	})

	password, err := client.GeneratePassword(context.Background(), recipe)
	require.NoError(t, err)
	require.NotEmpty(t, password)
}

func TestClientWrapperCreateUpdateDeleteItem(t *testing.T) {
	client, fixture := newTestClient(t)

	tests := []struct {
		name     string
		category onepassword.ItemCategory
		fields   []onepassword.ItemField
	}{
		{
			name:     "login",
			category: onepassword.ItemCategoryLogin,
			fields: []onepassword.ItemField{
				field("username", onepassword.ItemFieldTypeText, "tester", nil),
				field("password", onepassword.ItemFieldTypeConcealed, "initial-secret", nil),
			},
		},
		{
			name:     "secure_note",
			category: onepassword.ItemCategorySecureNote,
			fields: []onepassword.ItemField{
				field("note", onepassword.ItemFieldTypeText, "example note", nil),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			itemName := fmt.Sprintf("tf-provider-test-%s-%d", test.name, time.Now().UnixNano())
			notes := fmt.Sprintf("created by terraform provider tests (%s)", test.name)

			params := onepassword.ItemCreateParams{
				Category: test.category,
				VaultID:  fixture.VaultID,
				Title:    itemName,
				Notes:    &notes,
				Fields:   test.fields,
			}

			createdItem, err := client.CreateItem(context.Background(), params)
			require.NoError(t, err)
			require.NotNil(t, createdItem)
			require.NotEmpty(t, createdItem.ID)

			createdItem.Notes = fmt.Sprintf("updated by terraform provider tests (%s)", test.name)
			updatedItem, err := client.UpdateItem(context.Background(), *createdItem)
			require.NoError(t, err)
			require.Equal(t, createdItem.ID, updatedItem.ID)
			require.Equal(t, createdItem.Notes, updatedItem.Notes)

			require.NoError(t, client.DeleteItem(context.Background(), fixture.VaultID, createdItem.ID))

			_, err = client.GetItem(context.Background(), fixture.VaultID, createdItem.ID)
			require.Error(t, err)
		})
	}
}

func field(label string, fieldType onepassword.ItemFieldType, value string, details *onepassword.ItemFieldDetails) onepassword.ItemField {
	return onepassword.ItemField{
		ID:        label,
		Title:     label,
		FieldType: fieldType,
		Value:     value,
		Details:   details,
	}
}
