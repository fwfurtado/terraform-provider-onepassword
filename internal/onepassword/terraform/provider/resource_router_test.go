package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderRouterResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_router" "test" {
  vault        = "vault"
  name         = "Router"
  server_ip    = "192.168.1.1"
  airport_id   = "ABC123"
  network_name = "HomeNet"

  attached_storage_password {
    random {
      length = 12
    }
  }

  base_station {
    name = "router.local"
    password {
      random {
        length = 12
      }
    }
  }

  wireless {
    security_type = "wpa2"
    passphrase {
      random {
        length = 12
      }
    }
  }
}
`
	runResourceTest(t, config, "onepassword_router.test",
		resource.TestCheckResourceAttr("onepassword_router.test", "name", "Router"),
		resource.TestCheckResourceAttr("onepassword_router.test", "server_ip", "192.168.1.1"),
		resource.TestCheckResourceAttr("onepassword_router.test", "network_name", "HomeNet"),
	)
}
