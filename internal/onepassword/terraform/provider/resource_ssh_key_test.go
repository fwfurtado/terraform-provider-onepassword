package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderSSHKeyResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_ssh_key" "test" {
  vault = "vault"
  name  = "SSH Key"

  private_key {
    generated = true
    type      = "ed25519"
  }
}
`
	runResourceTest(t, config, "onepassword_ssh_key.test",
		resource.TestCheckResourceAttr("onepassword_ssh_key.test", "name", "SSH Key"),
		resource.TestCheckResourceAttrSet("onepassword_ssh_key.test", "public_key"),
		resource.TestCheckResourceAttrSet("onepassword_ssh_key.test", "fingerprint"),
		resource.TestCheckResourceAttrSet("onepassword_ssh_key.test", "key_type"),
	)
}
