package onepasswordprovider

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	"github.com/1password/onepassword-sdk-go"
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	mock "github.com/ovechkin-dm/mockio/v2/mock"
)

const baseProviderConfig = `
provider "onepassword" {
  service_account = {
    token = "test"
  }
}
`

type stubOpClient struct {
	secrets onepassword.SecretsAPI
	items   onepassword.ItemsAPI
	vaults  onepassword.VaultsAPI
}

func (s *stubOpClient) Secrets() onepassword.SecretsAPI {
	return s.secrets
}

func (s *stubOpClient) Items() onepassword.ItemsAPI {
	return s.items
}

func (s *stubOpClient) Vaults() onepassword.VaultsAPI {
	return s.vaults
}

type mockProvider struct {
	*OnePasswordProvider
	client *client.ClientWrapper
}

func (p *mockProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	config := &providerConfig{
		client:      p.client,
		defaultTags: nil,
	}
	resp.ResourceData = config
	resp.DataSourceData = config
	resp.EphemeralResourceData = config
}

func newTestClientWrapper(t *testing.T, secrets onepassword.SecretsAPI, items onepassword.ItemsAPI, vaults onepassword.VaultsAPI) *client.ClientWrapper {
	t.Helper()

	wrapper := &client.ClientWrapper{}
	inner := &stubOpClient{secrets: secrets, items: items, vaults: vaults}

	field := reflect.ValueOf(wrapper).Elem().FieldByName("inner")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(inner))

	return wrapper
}

func runResourceTest(t *testing.T, config string, resourceAddr string, checks ...resource.TestCheckFunc) {
	t.Helper()

	ctrl := mock.NewMockController(t)
	secrets := mock.Mock[onepassword.SecretsAPI](ctrl)
	items := mock.Mock[onepassword.ItemsAPI](ctrl)
	vaults := mock.Mock[onepassword.VaultsAPI](ctrl)

	mock.WhenDouble(vaults.List(mock.AnyContext(), mock.Any[onepassword.VaultListParams]())).
		ThenReturn([]onepassword.VaultOverview{{ID: "vault-id", Title: "vault"}}, nil)
	mock.WhenDouble(items.Create(mock.AnyContext(), mock.Any[onepassword.ItemCreateParams]())).
		ThenReturn(onepassword.Item{ID: "item-id", VaultID: "vault-id", Title: "Test Item"}, nil)
	mock.WhenDouble(items.Get(mock.AnyContext(), mock.Exact("vault-id"), mock.Exact("item-id"))).
		ThenReturn(onepassword.Item{ID: "item-id", VaultID: "vault-id", Title: "Test Item"}, nil)
	mock.WhenSingle(items.Delete(mock.AnyContext(), mock.Exact("vault-id"), mock.Exact("item-id"))).
		ThenReturn(nil)

	wrapper := newTestClientWrapper(t, secrets, items, vaults)
	testProvider := &mockProvider{
		OnePasswordProvider: New("test"),
		client:              wrapper,
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"onepassword": providerserver.NewProtocol6WithError(testProvider),
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					append([]resource.TestCheckFunc{
						resource.TestCheckResourceAttrSet(resourceAddr, "id"),
					}, checks...)...,
				),
			},
		},
	})
}
