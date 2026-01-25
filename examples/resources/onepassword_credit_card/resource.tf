resource "onepassword_credit_card" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name = "Example Credit Card"

  cardholder = "Ada Lovelace"
  number     = "4111111111111111"
  type       = "visa"
  expiry     = "01/2030"
}
