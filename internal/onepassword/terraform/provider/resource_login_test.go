package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderLoginResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_login" "test" {
  vault    = "vault"
  name     = "Test Login"
  username = "user"

  password {
    random {
      length  = 12
      digits  = true
      symbols = false
    }
  }

  websites = {
    main = {
      url = "https://example.com"
    }
  }
}
`
	runResourceTest(t, config, "onepassword_login.test",
		resource.TestCheckResourceAttr("onepassword_login.test", "vault", "vault"),
		resource.TestCheckResourceAttr("onepassword_login.test", "name", "Test Login"),
		resource.TestCheckResourceAttr("onepassword_login.test", "username", "user"),
	)
}
