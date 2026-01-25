resource "onepassword_api_credentials" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name = "Example API Credentials"

  username = "api_user"
  credential {
    random {
      length  = 40
      symbols = false
      digits  = true
    }
  }
  type       = "bearer-token"
  filename   = "api_credentials.json"
  valid_from = "2025-01-01"
  expires    = "2026-01-01"
  hostname   = "api.example.com"
}
