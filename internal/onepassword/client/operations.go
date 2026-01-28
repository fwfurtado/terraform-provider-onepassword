package client

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/samber/lo"
)

// GetSecretByReference resolves a secret reference to its value.
func (c *ClientWrapper) GetSecretByReference(ctx context.Context, reference string) (string, error) {
	if c.cache != nil {
		if value, ok := c.cache.GetSecret(reference); ok {
			return value, nil
		}
	}

	value, err := withRetry(ctx, func(ctx context.Context) (string, error) {
		return c.inner.Secrets().Resolve(ctx, reference)
	})
	if err != nil {
		return "", err
	}

	if c.cache != nil {
		c.cache.SetSecret(reference, value)
	}

	return value, nil
}

// GetVault finds a vault by title and returns its ID.
func (c *ClientWrapper) GetVault(ctx context.Context, title string) (string, error) {
	if c.cache != nil {
		if cachedID, ok := c.cache.GetVaultID(title); ok {
			return cachedID, nil
		}
	}

	decryptDetails := true
	vaults, err := withRetry(ctx, func(ctx context.Context) ([]onepassword.VaultOverview, error) {
		return c.inner.Vaults().List(ctx, onepassword.VaultListParams{
			DecryptDetails: &decryptDetails,
		})
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

	if c.cache != nil {
		c.cache.SetVaultID(title, vault.ID)
	}

	return vault.ID, nil
}

// GetItemOverview fetches an item overview by vault and item title.
func (c *ClientWrapper) GetItemOverview(ctx context.Context, vaultID, name string) (*onepassword.ItemOverview, error) {
	if c.cache != nil {
		if cachedItems, ok := c.cache.GetItemOverviews(vaultID); ok {
			item, found := lo.Find(cachedItems, func(item onepassword.ItemOverview) bool {
				return item.Title == name
			})
			if !found {
				return nil, fmt.Errorf("item not found")
			}
			return &item, nil
		}
	}

	items, err := withRetry(ctx, func(ctx context.Context) ([]onepassword.ItemOverview, error) {
		return c.inner.Items().List(ctx, vaultID)
	})
	if err != nil {
		return nil, err
	}

	if c.cache != nil {
		c.cache.SetItemOverviews(vaultID, items)
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
	item, err := withRetry(ctx, func(ctx context.Context) (onepassword.Item, error) {
		return c.inner.Items().Get(ctx, vaultID, itemID)
	})
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
	item, err := withRetry(ctx, func(ctx context.Context) (onepassword.Item, error) {
		return c.inner.Items().Create(ctx, params)
	})
	if err != nil {
		return nil, err
	}

	if c.cache != nil {
		c.cache.InvalidateItemOverviews(params.VaultID)
	}

	return &item, nil
}

// UpdateItem updates an existing item.
func (c *ClientWrapper) UpdateItem(ctx context.Context, item onepassword.Item) (*onepassword.Item, error) {
	updatedItem, err := withRetry(ctx, func(ctx context.Context) (onepassword.Item, error) {
		return c.inner.Items().Put(ctx, item)
	})
	if err != nil {
		return nil, err
	}

	if c.cache != nil {
		c.cache.InvalidateItemOverviews(item.VaultID)
	}

	return &updatedItem, nil
}

// DeleteItem deletes an item by vault and item ID.
func (c *ClientWrapper) DeleteItem(ctx context.Context, vaultID, itemID string) error {
	_, err := withRetry(ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.inner.Items().Delete(ctx, vaultID, itemID)
	})
	if err != nil {
		return err
	}

	if c.cache != nil {
		c.cache.InvalidateItemOverviews(vaultID)
	}

	return nil
}
