package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderDatabaseResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_database" "test" {
  vault    = "vault"
  name     = "Database"
  type     = "postgresql"
  server   = "db.example.com"
  port     = 5432
  database = "app"
  username = "dbuser"

  password {
    random {
      length  = 14
      digits  = true
      symbols = false
    }
  }
}
`
	runResourceTest(t, config, "onepassword_database.test",
		resource.TestCheckResourceAttr("onepassword_database.test", "name", "Database"),
		resource.TestCheckResourceAttr("onepassword_database.test", "server", "db.example.com"),
		resource.TestCheckResourceAttr("onepassword_database.test", "port", "5432"),
	)
}
