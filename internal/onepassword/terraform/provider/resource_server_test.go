package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderServerResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_server" "test" {
  vault    = "vault"
  name     = "Server"
  url      = "https://server.example.com"
  username = "root"

  password {
    random {
      length = 12
    }
  }

  admin_console {
    url      = "https://server.example.com/admin"
    username = "admin"
    password {
      random {
        length = 12
      }
    }
  }

  hosting_provider {
    name    = "ExampleHost"
    website = "https://example-host.com"
    support {
      email = "support@example.com"
    }
  }
}
`
	runResourceTest(t, config, "onepassword_server.test",
		resource.TestCheckResourceAttr("onepassword_server.test", "name", "Server"),
		resource.TestCheckResourceAttr("onepassword_server.test", "url", "https://server.example.com"),
		resource.TestCheckResourceAttr("onepassword_server.test", "username", "root"),
	)
}
