package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderSoftwareLicenseResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_software_license" "test" {
  vault = "vault"
  name  = "Software License"

  software {
    version     = "1.0.0"
    license_key = "AAAA-BBBB-CCCC"
  }

  customer {
    licensed_to      = "Example Customer"
    registered_email = "customer@example.com"
  }

  publisher {
    name    = "Vendor"
    website = "https://vendor.example.com"
  }

  order {
    number        = "12345"
    total         = 99.95
    purchase_date = "2025-01-01"
  }
}
`
	runResourceTest(t, config, "onepassword_software_license.test",
		resource.TestCheckResourceAttr("onepassword_software_license.test", "name", "Software License"),
	)
}
