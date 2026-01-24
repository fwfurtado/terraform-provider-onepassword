package onepasswordprovider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestProviderCreditCardResourceWithMockClient(t *testing.T) {
	config := baseProviderConfig + `
resource "onepassword_credit_card" "test" {
  vault      = "vault"
  name       = "Test Card"
  cardholder = "Ada Lovelace"
  number     = "4111111111111111"
  type       = "visa"
  expiry     = "01/2030"
}
`
	runResourceTest(t, config, "onepassword_credit_card.test",
		resource.TestCheckResourceAttr("onepassword_credit_card.test", "name", "Test Card"),
		resource.TestCheckResourceAttr("onepassword_credit_card.test", "cardholder", "Ada Lovelace"),
		resource.TestCheckResourceAttr("onepassword_credit_card.test", "type", "visa"),
		resource.TestCheckResourceAttr("onepassword_credit_card.test", "expiry", "01/2030"),
	)
}
