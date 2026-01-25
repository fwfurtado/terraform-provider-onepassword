resource "onepassword_ssh_key" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name = "Example Generated SSH Key"

  private_key {
    type      = "ed25519"
    generated = true
  }
}
