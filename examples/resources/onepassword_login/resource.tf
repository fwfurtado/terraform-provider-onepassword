resource "onepassword_login" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name     = "Grafana"
  username = "grafana"
  password {
    random {
      length  = 32
      digits  = true
      symbols = true
    }
  }

  websites = {
    "website" = {
      url               = "https://grafana.com"
      autofill_behavior = "anywhere-on-website"
    }
  }

  notes = "This is a note"
}
