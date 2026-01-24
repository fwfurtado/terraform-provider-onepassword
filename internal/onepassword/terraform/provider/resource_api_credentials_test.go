package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderAPICredentialsResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_api_credentials" "test" {
  vault    = "vault"
  name     = "API Credentials"
  username = "api-user"
  type     = "bearer-token"
  filename = "api.json"
  hostname = "api.example.com"

  credential {
    random {
      length  = 16
      digits  = true
      symbols = false
    }
  }
}
`
	runResourceTest(t, config, "onepassword_api_credentials.test",
		resource.TestCheckResourceAttr("onepassword_api_credentials.test", "name", "API Credentials"),
		resource.TestCheckResourceAttr("onepassword_api_credentials.test", "username", "api-user"),
		resource.TestCheckResourceAttr("onepassword_api_credentials.test", "type", "bearer-token"),
		resource.TestCheckResourceAttr("onepassword_api_credentials.test", "filename", "api.json"),
	)
}
