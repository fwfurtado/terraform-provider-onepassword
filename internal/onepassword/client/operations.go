package client

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/samber/lo"
)

// GetSecretByReference resolves a secret reference to its value.
func (c *ClientWrapper) GetSecretByReference(ctx context.Context, reference string) (string, error) {
	return c.inner.Secrets().Resolve(ctx, reference)
}

// GetVault finds a vault by title and returns its ID.
func (c *ClientWrapper) GetVault(ctx context.Context, title string) (string, error) {

	decryptDetails := true
	vaults, err := c.inner.Vaults().List(ctx, onepassword.VaultListParams{
		DecryptDetails: &decryptDetails,
	})

	if err != nil {
		return "", err
	}

	vault, found := lo.Find(vaults, func(vault onepassword.VaultOverview) bool {
		return vault.Title == title
	})

	if !found {
		return "", fmt.Errorf("vault not found")
	}

	return vault.ID, nil
}

// GetItemOverview fetches an item overview by vault and item title.
func (c *ClientWrapper) GetItemOverview(ctx context.Context, vaultID, name string) (*onepassword.ItemOverview, error) {
	items, err := c.inner.Items().List(ctx, vaultID)
	if err != nil {
		return nil, err
	}

	item, found := lo.Find(items, func(item onepassword.ItemOverview) bool {
		return item.Title == name
	})

	if !found {
		return nil, fmt.Errorf("item not found")
	}

	return &item, nil
}

// GetItem fetches a full item by vault and item ID.
func (c *ClientWrapper) GetItem(ctx context.Context, vaultID, itemID string) (*onepassword.Item, error) {
	item, err := c.inner.Items().Get(ctx, vaultID, itemID)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// ValidateSecretReference validates the secret reference syntax.
func (c *ClientWrapper) ValidateSecretReference(ctx context.Context, reference string) error {
	return onepassword.Secrets.ValidateSecretReference(ctx, reference)
}

// GeneratePassword generates a password based on a recipe.
func (c *ClientWrapper) GeneratePassword(ctx context.Context, recipe onepassword.PasswordRecipe) (string, error) {
	response, err := onepassword.Secrets.GeneratePassword(ctx, recipe)

	if err != nil {
		return "", err
	}
	return response.Password, nil
}

// CreateItem creates a new item in the specified vault.
func (c *ClientWrapper) CreateItem(ctx context.Context, params onepassword.ItemCreateParams) (*onepassword.Item, error) {
	item, err := c.inner.Items().Create(ctx, params)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// UpdateItem updates an existing item.
func (c *ClientWrapper) UpdateItem(ctx context.Context, item onepassword.Item) (*onepassword.Item, error) {
	updatedItem, err := c.inner.Items().Put(ctx, item)
	if err != nil {
		return nil, err
	}

	return &updatedItem, nil
}

// DeleteItem deletes an item by vault and item ID.
func (c *ClientWrapper) DeleteItem(ctx context.Context, vaultID, itemID string) error {
	return c.inner.Items().Delete(ctx, vaultID, itemID)
}

// func (c *ClientWrapper) GeneratePassword(ctx context.Context) (string, error) {

// 	c.inner.Items().Create(ctx, onepassword.ItemCreateParams{
// 		Title:    "Test",
// 		Category: onepassword.ItemCategorySSHKey,
// 		Fields: []onepassword.ItemField{
// 			{
// 				Title: "Password",
// 				Value: "password",
// 			},
// 		},
// 	})

// 	return "", nil
// }
