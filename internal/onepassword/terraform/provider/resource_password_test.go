package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderPasswordResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_password" "test" {
  vault = "vault"
  name  = "Test Password"

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
	runResourceTest(t, config, "onepassword_password.test",
		resource.TestCheckResourceAttr("onepassword_password.test", "name", "Test Password"),
	)
}
