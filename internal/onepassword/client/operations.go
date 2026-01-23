package client

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/samber/lo"
)

func (c *ClientWrapper) GetSecretByReference(ctx context.Context, reference string) (string, error) {
	return c.inner.Secrets().Resolve(ctx, reference)
}

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

func (c *ClientWrapper) GetItem(ctx context.Context, vaultID, itemID string) (*onepassword.Item, error) {
	item, err := c.inner.Items().Get(ctx, vaultID, itemID)
	if err != nil {
		return nil, err
	}

	return &item, nil
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
