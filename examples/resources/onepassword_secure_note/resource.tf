resource "onepassword_secure_note" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name  = "Example Secure Note"
  notes = <<EOF
Example secure note

Without section
EOF
}
