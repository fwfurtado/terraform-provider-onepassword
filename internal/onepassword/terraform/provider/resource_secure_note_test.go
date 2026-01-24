package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderSecureNoteResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_secure_note" "test" {
  vault = "vault"
  name  = "Secure Note"
  note  = "example note"
}
`
	runResourceTest(t, config, "onepassword_secure_note.test",
		resource.TestCheckResourceAttr("onepassword_secure_note.test", "name", "Secure Note"),
		resource.TestCheckResourceAttr("onepassword_secure_note.test", "note", "example note"),
	)
}
