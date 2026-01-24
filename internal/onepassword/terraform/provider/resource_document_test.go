package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderDocumentResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_document" "test" {
  vault = "vault"
  name  = "Document"

  document {
    filename = "example.txt"
    content  = "example content"
  }
}
`
	runResourceTest(t, config, "onepassword_document.test",
		resource.TestCheckResourceAttr("onepassword_document.test", "name", "Document"),
	)
}
